import type {ProxyRule,ProxyTraffic,ProxyBusinessState} from '../api/types'
export interface ServiceBlock {key:string;name:string;groupId:number;rules:ProxyRule[]}
export function buildServiceBlocks(rules:ProxyRule[],business:ProxyBusinessState):ServiceBlock[]{
  const memberships=new Map(business.members.map(m=>[m.rule_id,m.group_id]))
  const groups=new Map<string,ServiceBlock>()
  for(const rule of rules){
    const name=rule.name?.trim()
    const groupId=memberships.get(rule.id)??0
    // Visual grouping only; never modify, merge or infer configuration equality.
    const key=name?JSON.stringify([groupId,name,rule.upstream.trim()]):JSON.stringify(['rule',rule.id])
    const block=groups.get(key)
    if(block)block.rules.push(rule)
    else groups.set(key,{key,name:name||rule.domain||rule.hosts?.[0]?.hostname||'规则 #'+rule.id,groupId,rules:[rule]})
  }
  return [...groups.values()]
}
export function sumServiceTraffic(rules:ProxyRule[],traffic:Record<number,ProxyTraffic>){
  return rules.reduce((sum,rule)=>{
    const t=traffic[rule.id]
    if(t){sum.connections+=t.connections;sum.upload_rate+=t.upload_rate;sum.download_rate+=t.download_rate;sum.upload_total+=t.upload_total;sum.download_total+=t.download_total}
    return sum
  },{connections:0,upload_rate:0,download_rate:0,upload_total:0,download_total:0})
}
export function moveServiceRules(rules:ProxyRule[],source:number[],target:number[]):ProxyRule[]{
  const from=new Set(source),to=new Set(target)
  if(!source.length||!target.length||source.some(id=>to.has(id)))return rules
  const moving=rules.filter(r=>from.has(r.id))
  const rest=rules.filter(r=>!from.has(r.id))
  const first=rest.findIndex(r=>to.has(r.id))
  if(first<0||moving.length!==from.size)return rules
  const before=rules.findIndex(r=>from.has(r.id))>rules.findIndex(r=>to.has(r.id))
  const at=before?first:rest.reduce((last,r,i)=>to.has(r.id)?i+1:last,first)
  return [...rest.slice(0,at),...moving,...rest.slice(at)]
}
