package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fonu/fonu/internal/settings"
)

type LogBlacklistStatusInput struct {
	Host string `json:"host"`
	Port int `json:"port"`
	IP string `json:"ip"`
}

type LogBlacklistStatus struct {
	Scope string `json:"scope,omitempty"`
	RuleID int64 `json:"rule_id,omitempty"`
	IP string `json:"ip,omitempty"`
	Exact bool `json:"exact"`
	Covered bool `json:"covered"`
	GlobalExact bool `json:"global_exact"`
	GlobalCovered bool `json:"global_covered"`
	Error string `json:"error,omitempty"`
}

// Read current policy once for the visible page, not once per historical log.
// This reports configured membership, not a verdict on a historical request.
func (s *ProxyService) LogBlacklistStatuses(ctx context.Context, inputs []LogBlacklistStatusInput) ([]LogBlacklistStatus,error) {
	if len(inputs)>100{return nil,fmt.Errorf("单次最多查询 100 条黑名单状态")}
	out:=make([]LogBlacklistStatus,len(inputs))
	if len(inputs)==0{return out,nil}
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	if err:=s.recoverPendingLocked(ctx);err!=nil{return nil,err}
	rules,err:=s.store.List(ctx)
	if err!=nil{return nil,err}
	if s.settings==nil{return nil,fmt.Errorf("全局安全设置不可用")}
	raw,err:=s.settings.Get(ctx,settings.KeyGlobalIPBlacklist)
	if err!=nil{return nil,err}
	var global []string
	if strings.TrimSpace(raw)!=""{
		if err=json.Unmarshal([]byte(raw),&global);err!=nil{return nil,fmt.Errorf("全局黑名单格式无效")}
	}
	for i,input:=range inputs{
		ip,err:=accessBlacklistIP(input.IP)
		if err!=nil{out[i].Error=err.Error();continue}
		status:=LogBlacklistStatus{IP:ip.String()}
		status.GlobalExact,status.GlobalCovered=blacklistCoverage(global,ip)
		rule,found,err:=logBlacklistRule(rules,input.Host,input.Port)
		if err!=nil{status.Error=err.Error();out[i]=status;continue}
		if found{
			status.Scope="rule";status.RuleID=rule.ID
			status.Exact,status.Covered=blacklistCoverage(rule.Security.IPBlacklist,ip)
		}else{
			status.Scope="global";status.Exact=status.GlobalExact;status.Covered=status.GlobalCovered
		}
		out[i]=status
	}
	return out,nil
}
