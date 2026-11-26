package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fonu/fonu/internal/nginx"
)

type proxyJournal struct {
	Token string `json:"token"`
	Files nginx.PublicationFiles `json:"files"`
	// Only CIDR updates need to snapshot the potentially large IP database.
	CIDR *cidrSnapshot `json:"cidr,omitempty"`
}

func(s *ProxyService) journalPath()string{return filepath.Join(s.cfg.NginxDir(),"pending-publication.json")}

type cidrSnapshot struct {
	Exists bool `json:"exists"`
	Body []byte `json:"body,omitempty"`
}

func(s *ProxyService) beginProxyJournal()(proxyJournal,error){
	return s.beginProxyJournalWithCIDR(nil)
}

func(s *ProxyService) beginProxyJournalWithCIDR(cidr *cidrSnapshot)(proxyJournal,error){
	var j proxyJournal
	j.CIDR=cidr
	if _,err:=os.Stat(s.journalPath());err==nil{return j,fmt.Errorf("存在未恢复的反代配置发布记录")}else if !os.IsNotExist(err){return j,err}
	token:=make([]byte,16)
	if _,err:=rand.Read(token);err!=nil{return j,err}
	j.Token=hex.EncodeToString(token)
	files,err:=nginx.SnapshotPublicationFiles(s.cfg)
	if err!=nil{return j,err}
	j.Files=files
	data,err:=json.Marshal(j)
	if err!=nil{return j,err}
	if err=nginx.WriteCustomAtomic(s.journalPath(),string(data));err!=nil{return j,err}
	return j,nil
}

func(s *ProxyService) clearProxyJournal()error{
	err:=os.Remove(s.journalPath())
	if os.IsNotExist(err){return nil}
	return err
}

func(s *ProxyService) restoreJournalFiles(j proxyJournal) error {
	if j.CIDR!=nil {
		path:=s.cfg.ChinaCIDRPath()
		if j.CIDR.Exists {
			tmp,err:=writeCIDRTemp(filepath.Dir(path),j.CIDR.Body)
			if err!=nil{return err}
			defer os.Remove(tmp)
			if err=os.Rename(tmp,path);err!=nil{return err}
		} else if err:=os.Remove(path);err!=nil&&!os.IsNotExist(err){return err}
	}
	return nginx.RestorePublicationFiles(s.cfg,j.Files)
}

func(s *ProxyService) abortProxyJournal(j proxyJournal,cause error)error{
	if err:=s.restoreJournalFiles(j);err!=nil{return errors.Join(cause,fmt.Errorf("恢复发布文件失败：%w",err))}
	return errors.Join(cause,s.clearProxyJournal())
}

// Keep the journal if publication failed: a subsequent recovery must verify
// disk and running configuration against the committed database.
func(s *ProxyService) retainFailedPublication(j proxyJournal,cause error)error{
	if err:=s.restoreJournalFiles(j);err!=nil{return errors.Join(cause,err)}
	return cause
}

// The token is committed in the same transaction as rule/global mode changes.
func(s *ProxyService) recoverPendingLocked(ctx context.Context)error{
	data,err:=os.ReadFile(s.journalPath())
	if os.IsNotExist(err){return nil}
	if err!=nil{return err}
	var j proxyJournal
	if err=json.Unmarshal(data,&j);err!=nil{return fmt.Errorf("配置恢复记录损坏：%w",err)}
	if len(j.Token)!=32{return fmt.Errorf("无效的配置恢复记录")}
	var committed string
	err=s.db.QueryRowContext(ctx,"SELECT token FROM proxy_publication_commit WHERE id=1").Scan(&committed)
	if err!=nil&&!errors.Is(err,sql.ErrNoRows){return err}
	if committed!=j.Token{
		if err:=s.restoreJournalFiles(j);err!=nil{return err}
	}
	if err:=s.applyNginxLocked(ctx);err!=nil{return fmt.Errorf("恢复反代配置失败：%w",err)}
	return s.clearProxyJournal()
}
