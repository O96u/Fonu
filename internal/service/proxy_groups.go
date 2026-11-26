package service

import (
 "context"
 "database/sql"
 "fmt"
 "strings"
 "unicode/utf8"
)

type ProxyBusinessGroup struct {
 ID int64 `json:"id"`
 Name string `json:"name"`
}
type ProxyBusinessMember struct {
 RuleID int64 `json:"rule_id"`
 GroupID int64 `json:"group_id"`
}
type ProxyBusinessState struct {
 Groups []ProxyBusinessGroup `json:"groups"`
 Members []ProxyBusinessMember `json:"members"`
}
type ProxyBusinessChange struct {
 Action string `json:"action"`
 ID int64 `json:"id"`
 Name string `json:"name"`
 RuleIDs []int64 `json:"rule_ids"`
 IDs []int64 `json:"ids"`
}

func readProxyBusiness(ctx context.Context, tx *sql.Tx)(ProxyBusinessState,error){
 out:=ProxyBusinessState{Groups:[]ProxyBusinessGroup{},Members:[]ProxyBusinessMember{}}
 rows,err:=tx.QueryContext(ctx,"SELECT id,name FROM proxy_business_groups ORDER BY sort_order,id")
 if err!=nil{return out,err}
 for rows.Next(){var g ProxyBusinessGroup;if err=rows.Scan(&g.ID,&g.Name);err!=nil{rows.Close();return out,err};out.Groups=append(out.Groups,g)}
 err=rows.Err();rows.Close();if err!=nil{return out,err}
 rows,err=tx.QueryContext(ctx,"SELECT rule_id,group_id FROM proxy_business_members ORDER BY rule_id")
 if err!=nil{return out,err};defer rows.Close()
 for rows.Next(){var m ProxyBusinessMember;if err=rows.Scan(&m.RuleID,&m.GroupID);err!=nil{return out,err};out.Members=append(out.Members,m)}
 return out,rows.Err()
}
func(s *ProxyService) ProxyBusiness(ctx context.Context)(ProxyBusinessState,error){
 tx,err:=s.db.BeginTx(ctx,nil);if err!=nil{return ProxyBusinessState{},err};defer tx.Rollback()
 out,err:=readProxyBusiness(ctx,tx);if err!=nil{return out,err};return out,tx.Commit()
}
// Display metadata is deliberately independent of Nginx publication and rule configuration.
func(s *ProxyService) ChangeProxyBusiness(ctx context.Context,in ProxyBusinessChange)(ProxyBusinessState,error){
 var empty ProxyBusinessState
 s.applyMu.Lock();defer s.applyMu.Unlock()
 tx,err:=s.db.BeginTx(ctx,nil);if err!=nil{return empty,err};defer tx.Rollback()
 name:=strings.TrimSpace(in.Name)
 switch in.Action {
 case "create","rename":
  if name==""||utf8.RuneCountInString(name)>60||strings.ContainsAny(name,"\r\n\t"){return empty,fmt.Errorf("分组名称须为 1–60 字符")}
  if in.Action=="create"{
   var count int
   if err=tx.QueryRowContext(ctx,"SELECT count(*) FROM proxy_business_groups").Scan(&count);err!=nil{return empty,err}
   if count>=100{return empty,fmt.Errorf("最多创建 100 个分组")}
   _,err=tx.ExecContext(ctx,"INSERT INTO proxy_business_groups(name,sort_order) VALUES(?,(SELECT coalesce(max(sort_order),0)+1 FROM proxy_business_groups))",name)
  }else{
   var result sql.Result
   result,err=tx.ExecContext(ctx,"UPDATE proxy_business_groups SET name=? WHERE id=?",name,in.ID)
   if err==nil{if n,e:=result.RowsAffected();e!=nil||n!=1{return empty,fmt.Errorf("分组不存在")}}
  }
 case "delete":
  _,err=tx.ExecContext(ctx,"DELETE FROM proxy_business_groups WHERE id=?",in.ID)
 case "assign":
  if in.ID<0||len(in.RuleIDs)==0||len(in.RuleIDs)>500{return empty,fmt.Errorf("无效的分组或规则数量")}
  if in.ID>0{var id int64;if err=tx.QueryRowContext(ctx,"SELECT id FROM proxy_business_groups WHERE id=?",in.ID).Scan(&id);err!=nil{return empty,fmt.Errorf("分组不存在")}}
  for _,id:=range in.RuleIDs{
   var exists int64
   if err=tx.QueryRowContext(ctx,"SELECT id FROM proxy_rules WHERE id=?",id).Scan(&exists);err!=nil{return empty,fmt.Errorf("规则不存在")}
   if in.ID==0{_,err=tx.ExecContext(ctx,"DELETE FROM proxy_business_members WHERE rule_id=?",id)}else{
    _,err=tx.ExecContext(ctx,"INSERT INTO proxy_business_members(rule_id,group_id) VALUES(?,?) ON CONFLICT(rule_id) DO UPDATE SET group_id=excluded.group_id",id,in.ID)
   };if err!=nil{return empty,err}
  }
 case "reorder":
  current,e:=readProxyBusiness(ctx,tx);if e!=nil{return empty,e}
  if len(current.Groups)!=len(in.IDs){return empty,fmt.Errorf("分组列表已变化，请刷新")}
  valid:=map[int64]bool{};for _,g:=range current.Groups{valid[g.ID]=true}
  for i,id:=range in.IDs{if !valid[id]{return empty,fmt.Errorf("无效分组排序")};delete(valid,id);if _,err=tx.ExecContext(ctx,"UPDATE proxy_business_groups SET sort_order=? WHERE id=?",i,id);err!=nil{return empty,err}}
 default:return empty,fmt.Errorf("未知分组操作")
 }
 if err!=nil{return empty,err}
 out,err:=readProxyBusiness(ctx,tx);if err!=nil{return empty,err};return out,tx.Commit()
}
