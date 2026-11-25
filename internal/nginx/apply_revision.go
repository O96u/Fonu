package nginx

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

// Custom snippets can include external files whose dependencies are unknown;
// never skip their reload based on a partial digest.
func configRevision(cfg config.Config, content string, rules []proxy.Rule, certs []CertSource, opts GenerateOptions) (string,bool) {
	h := sha256.New()
	fmt.Fprintln(h,content)
	safe := opts.GlobalCustomMode != "custom" && !globalCustomShouldInclude(cfg,opts)
	paths := map[string]bool{cfg.NginxMimeTypes:true}
	if opts.ChinaCIDRAvailable { paths[cfg.ChinaCIDRPath()] = true }
	for _,r:=range rules {
		if !r.Enabled {continue}
		if r.NginxMode=="custom"{safe=false;continue}
		if r.Security.BasicAuthEnabled(){paths[HtpasswdPath(cfg,r.ID)]=true}
		if r.HTTPSEnabled{
			for _,g:=range r.PortGroups(){for _,host:=range g.Hostnames{
				c:=findCertificateForHosts(cfg.CertsDir(),[]string{host},certs)
				if c==nil{safe=false;continue}
				paths[c.CertPath]=true;paths[c.KeyPath]=true
			}}
		}
		if r.Security.ProxySSLVerifyConfigured&&!r.Security.ProxySSLVerifyOff{
			if path:=systemCABundle();path!=""{paths[path]=true}
		}
	}
	names:=make([]string,0,len(paths))
	for path:=range paths{names=append(names,path)}
	sort.Strings(names)
	for _,path:=range names{
		fmt.Fprintf(h,"%q\n",path)
		f,err:=os.Open(path)
		if err!=nil{safe=false;continue}
		sum:=sha256.New()
		n,err:=io.Copy(sum,io.LimitReader(f,64<<20))
		_ = f.Close()
		if err!=nil||n>=64<<20{safe=false;continue}
		fmt.Fprintf(h,"%x\n",sum.Sum(nil))
	}
	return fmt.Sprintf("%x",h.Sum(nil)),safe
}

func workersOverlap(a,b map[string]bool)bool{
	for k:=range a{if b[k]{return true}}
	return false
}

func (m *Manager) waitForWorkers(ctx context.Context, previous map[string]bool)(int,map[string]bool,error){
	ctx,cancel:=context.WithTimeout(ctx,8*time.Second)
	defer cancel()
	tick:=time.NewTicker(100*time.Millisecond)
	defer tick.Stop()
	var last map[string]bool
	for{
		pid,err:=readPIDFile(m.cfg.NginxPIDFile)
		if err==nil&&m.ownsMaster(pid){
			current:=activeWorkers(pid)
			fresh:=map[string]bool{}
			for k:=range current{if !previous[k]{fresh[k]=true}}
			if workersOverlap(last,fresh){return pid,fresh,nil}
			last=fresh
		}else{last=nil}
		select{case <-ctx.Done():return 0,nil,fmt.Errorf("未确认 Nginx 新 worker 就绪：%w",ctx.Err());case <-tick.C:}
	}
}

type ApplyHistoryEntry struct {
	State string `json:"state"`
	Revision string `json:"revision,omitempty"`
	CreatedAt string `json:"created_at"`
	Unchanged bool `json:"unchanged,omitempty"`
}

func (m *Manager) historyPath()string{return filepath.Join(m.cfg.NginxDir(),"apply-history.json")}
func (m *Manager) readApplyHistory()[]ApplyHistoryEntry{
	data,err:=os.ReadFile(m.historyPath());if err!=nil{return nil}
	var rows []ApplyHistoryEntry
	if json.Unmarshal(data,&rows)!=nil{return nil}
	if len(rows)>5{return rows[:5]}
	return rows
}
func (m *Manager) appendApplyHistory(status ApplyStatus,unchanged bool)error{
	if unchanged{return nil}
	rows:=append([]ApplyHistoryEntry{{State:status.State,Revision:status.Desired,CreatedAt:status.UpdatedAt}},m.readApplyHistory()...)
	if len(rows)>5{rows=rows[:5]}
	data,err:=json.Marshal(rows);if err!=nil{return err}
	return WriteCustomAtomic(m.historyPath(),string(data))
}
