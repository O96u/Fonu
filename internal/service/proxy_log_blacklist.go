package service

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/fonu/fonu/internal/proxy"
)

type LogBlacklistRequest struct {
	Host string `json:"host"`
	Port int `json:"port"`
	IP string `json:"ip"`
	Action string `json:"action"`
	Scope string `json:"scope,omitempty"`
	RuleID int64 `json:"rule_id,omitempty"`
	Confirmed bool `json:"confirmed"`
}

type LogBlacklistResult struct {
	Scope string `json:"scope"`
	RuleID int64 `json:"rule_id,omitempty"`
	RuleName string `json:"rule_name,omitempty"`
	IP string `json:"ip"`
	Exact bool `json:"exact"`
	Covered bool `json:"covered"`
	Changed bool `json:"changed"`
	CustomConfig bool `json:"custom_config"`
	WhitelistMode bool `json:"whitelist_mode"`
}

func logBlacklistHost(host string) string {
	host=strings.TrimSpace(host)
	if h,_,err:=net.SplitHostPort(host);err==nil{host=h}
	return strings.ToLower(strings.TrimSuffix(strings.Trim(host,"[]"),"."))
}

// Nginx exact names outrank wildcard names; longest leading wildcard wins.
// A known host with missing port cannot safely distinguish listeners.
func logBlacklistRule(rules []proxy.Rule, host string, port int) (proxy.Rule,bool,error) {
	host=logBlacklistHost(host)
	best:=-1
	var matches []proxy.Rule
	for _,rule:=range rules{
		if !rule.Enabled{continue}
		score:=-1
		for _,ep:=range rule.Endpoints(){
			name:=logBlacklistHost(ep.Hostname)
			if host==""{continue}
			if name==host{
				if port<1||port>65535{return proxy.Rule{},false,fmt.Errorf("日志缺少有效监听端口，无法确认匹配服务，请在反代页面手动管理")}
				if ep.Port==port{score=10000}
			}else if strings.HasPrefix(name,"*.")&&strings.HasSuffix(host,name[1:])&&len(host)>len(name)-1{
				if _,err:=netip.ParseAddr(host);err==nil{continue}
				if port<1||port>65535{return proxy.Rule{},false,fmt.Errorf("日志缺少有效监听端口，无法确认匹配服务，请在反代页面手动管理")}
				if ep.Port==port&&len(name)>score{score=len(name)}
			}
		}
		if score<0{continue}
		if score>best{best=score;matches=nil}
		if score==best{matches=append(matches,rule)}
	}
	if len(matches)==0{return proxy.Rule{},false,nil}
	if len(matches)!=1{return proxy.Rule{},false,fmt.Errorf("匹配到多个反代规则，请手动确认，不会自动扩大封禁范围")}
	if matches[0].NginxMode=="custom"{return proxy.Rule{},false,fmt.Errorf("匹配规则使用手动 Nginx 配置，请手动管理黑名单，不会自动扩大封禁范围")}
	return matches[0],true,nil
}

// Inspect and mutation independently resolve the endpoint under the publication
// lock. A stale confirmation must never switch from a rule to global or back.
func (s *ProxyService) LogBlacklist(ctx context.Context, req LogBlacklistRequest) (LogBlacklistResult,error) {
	ip,err:=accessBlacklistIP(req.IP)
	if err!=nil{return LogBlacklistResult{},err}
	if req.Action!="inspect"&&req.Action!="block"&&req.Action!="unblock"{return LogBlacklistResult{},fmt.Errorf("无效的黑名单操作")}
	if req.Action!="inspect"&&!req.Confirmed{return LogBlacklistResult{},fmt.Errorf("请先确认黑名单操作及作用范围")}
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	if err=s.recoverPendingLocked(ctx);err!=nil{return LogBlacklistResult{},err}
	rules,err:=s.store.List(ctx)
	if err!=nil{return LogBlacklistResult{},err}
	rule,found,err:=logBlacklistRule(rules,req.Host,req.Port)
	if err!=nil{return LogBlacklistResult{},err}
	if found{
		if req.Action!="inspect"&&(req.Scope!="rule"||req.RuleID!=rule.ID){return LogBlacklistResult{},fmt.Errorf("黑名单作用范围或规则已变化，请重新确认")}
		res,err:=s.accessBlacklistLocked(ctx,AccessBlacklistRequest{Action:req.Action,RuleID:rule.ID},rule,ip)
		if err!=nil{return LogBlacklistResult{},err}
		return LogBlacklistResult{Scope:"rule",RuleID:res.RuleID,RuleName:res.RuleName,IP:res.IP,Exact:res.Exact,Covered:res.Covered,Changed:res.Changed,WhitelistMode:res.WhitelistMode},nil
	}
	if req.Action!="inspect"&&(req.Scope!="global"||req.RuleID!=0){return LogBlacklistResult{},fmt.Errorf("黑名单作用范围已变化，请重新确认")}
	res,err:=s.globalBlacklistLocked(ctx,GlobalBlacklistRequest{IP:ip.String(),Action:req.Action,Confirmed:req.Confirmed},ip)
	if err!=nil{return LogBlacklistResult{},err}
	return LogBlacklistResult{Scope:"global",IP:res.IP,Exact:res.Exact,Covered:res.Covered,Changed:res.Changed,CustomConfig:res.CustomConfig},nil
}
