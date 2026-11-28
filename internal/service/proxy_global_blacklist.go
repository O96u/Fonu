package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fonu/fonu/internal/settings"
	"github.com/fonu/fonu/internal/validate"
)

type GlobalBlacklistRequest struct {
	IP string `json:"ip"`
	Action string `json:"action"`
	Confirmed bool `json:"confirmed"`
}

type GlobalBlacklistResult struct {
	IP string `json:"ip"`
	Exact bool `json:"exact"`
	Covered bool `json:"covered"`
	Changed bool `json:"changed"`
	CustomConfig bool `json:"custom_config"`
}

// GlobalBlacklist updates the existing global policy; it never changes a rule's
// local blacklist. All writes use the same publication lock as settings edits.
func (s *ProxyService) GlobalBlacklist(ctx context.Context, req GlobalBlacklistRequest) (GlobalBlacklistResult,error) {
	ip,err:=accessBlacklistIP(req.IP)
	if err!=nil{return GlobalBlacklistResult{},err}
	if req.Action!="inspect"&&req.Action!="block"&&req.Action!="unblock"{return GlobalBlacklistResult{},fmt.Errorf("无效的全局黑名单操作")}
	if req.Action!="inspect"&&!req.Confirmed{return GlobalBlacklistResult{},fmt.Errorf("请先确认全局黑名单操作")}
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	if s.settings==nil{return GlobalBlacklistResult{},fmt.Errorf("全局安全设置不可用")}
	if err:=s.recoverPendingLocked(ctx);err!=nil{return GlobalBlacklistResult{},err}
	raw,err:=s.settings.Get(ctx,settings.KeyGlobalIPBlacklist)
	if err!=nil{return GlobalBlacklistResult{},err}
	list:=make([]string,0)
	if strings.TrimSpace(raw)!="" {
		if err=json.Unmarshal([]byte(raw),&list);err!=nil{return GlobalBlacklistResult{},fmt.Errorf("现有全局黑名单格式无效，请先在系统设置修复")}
	}
	if err=validate.IPOrCIDRList(list);err!=nil{return GlobalBlacklistResult{},fmt.Errorf("现有全局黑名单包含无效条目，请先在系统设置修复：%w",err)}
	mode,err:=s.settings.Get(ctx,settings.KeyNginxGlobalMode)
	if err!=nil{return GlobalBlacklistResult{},err}
	rules,err:=s.store.List(ctx)
	if err!=nil{return GlobalBlacklistResult{},err}
	custom:=mode=="custom"
	for _,rule:=range rules{if rule.Enabled&&rule.NginxMode=="custom"{custom=true;break}}
	exact,covered:=blacklistCoverage(list,ip)
	result:=GlobalBlacklistResult{IP:ip.String(),Exact:exact,Covered:covered,CustomConfig:custom}
	if req.Action=="inspect"||req.Action=="block"&&(exact||covered)||req.Action=="unblock"&&!exact{return result,nil}
	next:=make([]string,0,len(list)+1)
	for _,entry:=range list{
		if req.Action=="unblock"&&exactBlacklistIP(entry,ip){continue}
		next=append(next,entry)
	}
	if req.Action=="block"{next=append(next,ip.String())}
	if err=validate.IPOrCIDRList(next);err!=nil{return GlobalBlacklistResult{},err}
	payload,err:=json.Marshal(next)
	if err!=nil{return GlobalBlacklistResult{},err}
	if err=s.saveSettingsWithPublicationLocked(ctx,map[string]string{settings.KeyGlobalIPBlacklist:string(payload)});err!=nil{return GlobalBlacklistResult{},err}
	result.Exact,result.Covered=blacklistCoverage(next,ip)
	result.Changed=true
	return result,nil
}
