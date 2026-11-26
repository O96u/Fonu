import assert from 'node:assert/strict'
import test from 'node:test'
import {readFile} from 'node:fs/promises'
import ts from 'typescript'
const source=await readFile(new URL('../src/utils/proxyServices.ts',import.meta.url),'utf8')
const {outputText}=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.ESNext,target:ts.ScriptTarget.ES2022}})
const {buildServiceBlocks,sumServiceTraffic,moveServiceRules}=await import('data:text/javascript;base64,'+Buffer.from(outputText).toString('base64'))
const a={id:1,name:'飞牛影视',upstream:'http://127.0.0.1:8005',enabled:true,security:{china_only:true}}
const b={...a,id:2,enabled:false,security:{}}
const c={...a,id:3,upstream:'http://127.0.0.1:8080'}
const d={...a,id:4,name:''}
const e={...a,id:5}
const state={groups:[{id:1,name:'影视'}],members:[{rule_id:5,group_id:1}]}
test('visual grouping retains independent rule IDs, policies and states',()=>{
 const blocks=buildServiceBlocks([a,b,c,d,e],state)
 assert.equal(blocks.length,4)
 assert.deepEqual(blocks[0].rules,[a,b])
 assert.equal(blocks[0].rules[1].enabled,false)
 assert.deepEqual(blocks[0].rules[1].security,{})
 assert.equal(blocks[3].groupId,1)
})
test('traffic aggregation counts each independent rule once',()=>{
 const traffic={1:{connections:3,upload_rate:1,download_rate:2,upload_total:4,download_total:8},2:{connections:2,upload_rate:2,download_rate:3,upload_total:6,download_total:9}}
 assert.deepEqual(sumServiceTraffic([a,b],traffic),{connections:5,upload_rate:3,download_rate:5,upload_total:10,download_total:17})
})
test('moving service blocks preserves all IDs and relative internal order',()=>{
 const input=[a,b,c,d,e]
 assert.deepEqual(moveServiceRules(input,[1,2],[3]).map(r=>r.id),[3,1,2,4,5])
 assert.deepEqual(moveServiceRules(input,[3],[1,2]).map(r=>r.id),[3,1,2,4,5])
 assert.equal(moveServiceRules(input,[1,2],[2]),input)
})
