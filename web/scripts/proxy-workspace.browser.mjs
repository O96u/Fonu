import assert from 'node:assert/strict'
import { spawn } from 'node:child_process'
import { createRequire } from 'node:module'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'
const require=createRequire(resolve(process.env.FONU_UI_TEST_ROOT,'package.json'))
const {chromium}=require('playwright')
const base='http://127.0.0.1:4173'
const server=spawn(process.execPath,['node_modules/vite/bin/vite.js','preview','--host','127.0.0.1','--port','4173','--strictPort'],{stdio:'inherit'})
let browser
try{
 for(let i=0;i<80;i++){try{if((await fetch(base)).ok)break}catch{}if(i===79)throw Error('Vite preview failed');await new Promise(r=>setTimeout(r,200))}
 await mkdir('ui-screenshots',{recursive:true})
 browser=await chromium.launch({headless:true})
 const page=await browser.newPage({viewport:{width:1440,height:980},deviceScaleFactor:1})
 const errors=[]
 page.on('pageerror',err=>errors.push(err.message))
 page.on('console',msg=>{if(msg.type()==='error'&&msg.text().startsWith('[fonu]'))errors.push(msg.text())})
 let rules=[
  {id:1,name:'飞牛影视',domain:'speedtest.muxui.com',upstream:'http://192.168.8.3:8005',listen_port:8009,entry_id:2,enabled:true,security:{china_only:true}},
  {id:2,name:'飞牛影视',domain:'s.muxui.com',upstream:'http://192.168.8.3:8005',listen_port:8009,entry_id:2,enabled:false,security:{basic_auth:{enabled:true}}},
  {id:3,name:'飞牛影视',domain:'fonu.roven.cc',upstream:'http://192.168.8.3:8005',listen_port:8009,entry_id:2,enabled:true,security:{}},
  {id:4,name:'飞牛 NAS',domain:'nas.example.com',upstream:'http://192.168.1.10:5666',listen_port:443,entry_id:1,enabled:true,security:{}},
  {id:5,name:'iKuai',domain:'ikuai.example.com',upstream:'https://192.168.1.1',listen_port:443,entry_id:1,enabled:true,security:{china_only:true}},
  {id:6,name:'相册',domain:'photo.example.com',upstream:'http://192.168.1.10:2283',listen_port:443,entry_id:1,enabled:true,security:{}}
 ].map(r=>({...r,hosts:[{id:r.id,hostname:r.domain}],listen_ipv4:true,listen_ipv6:false,https_enabled:true,http_redirect:true,sort_order:r.id,created_at:'2026-11-26T00:00:00Z',updated_at:'2026-11-26T00:00:00Z'}))
 const entries=[{id:1,name:'默认入口',listen_port:443,rule_count:3},{id:2,name:'影音入口',listen_port:8009,rule_count:3}].map(e=>({...e,listen_ipv4:true,listen_ipv6:false,https_enabled:true,http_redirect:true,sort_order:e.id,created_at:'',updated_at:''}))
 let business={groups:[{id:1,name:'影音娱乐'},{id:2,name:'常用服务'}],members:[{rule_id:1,group_id:1},{rule_id:2,group_id:1},{rule_id:3,group_id:1},{rule_id:4,group_id:2},{rule_id:6,group_id:2}]}
 const traffic=rules.map(r=>({rule_id:r.id,connections:r.enabled?2:0,upload_rate:65536,download_rate:1048576,upload_total:1073741824,download_total:5368709120}))
 const mutations=[]
 await page.route('**/*',async route=>{
  const url=new URL(route.request().url())
  if(url.origin!==base)return route.abort()
  if(!url.pathname.startsWith('/api/'))return route.continue()
  let data=[]
  const method=route.request().method()
  if(url.pathname==='/api/auth/status')data={initialized:true,authenticated:true,username:'admin'}
  else if(url.pathname==='/api/version')data={version:'2.0.0-beta.23',nginx_http_port:80,nginx_https_port:443}
  else if(url.pathname==='/api/nginx/apply-status')data={state:'confirmed',confirmed_revision:'aabbccddeeff',history:[]}
  else if(url.pathname==='/api/proxies'&&method==='GET')data=rules
  else if(url.pathname==='/api/proxy-entries')data=entries
  else if(url.pathname==='/api/proxies/traffic')data=traffic
  else if(url.pathname==='/api/proxy-groups'){
   if(method==='POST'){const change=route.request().postDataJSON();if(change.action==='create')business={...business,groups:[...business.groups,{id:3,name:change.name}]}}
   data=business
  }else if(/^\/api\/proxies\/\d+$/.test(url.pathname)&&method==='PUT'){
   const id=Number(url.pathname.split('/').pop()),change=route.request().postDataJSON()
   mutations.push({id,change});rules=rules.map(r=>r.id===id?{...r,...change}:r);data=rules.find(r=>r.id===id)
  }else if(url.pathname.endsWith('/favicon'))return route.fulfill({status:404,body:''})
  else if(url.pathname==='/api/settings')data={}
  return route.fulfill({json:data})
 })
 await page.addInitScript(()=>{localStorage.setItem('fonu-theme','light');localStorage.setItem('fonu-nav-collapsed','true')})
 await page.goto(base+'/proxies')
 await page.locator('.summary-identity').first().waitFor()
 assert.equal(await page.locator('.service-block').count(),4)
 await page.locator('.summary-identity').first().click()
 assert.equal(await page.locator('.service-row--child').count(),3)
 assert.match(await page.locator('.summary-identity').first().innerText(),/3 条独立规则/)
 await page.screenshot({path:'ui-screenshots/desktop-list.png',fullPage:true})
 // Each domain rule has its own switch. Only the selected rule is updated.
 await page.locator('.service-row--child').nth(1).locator('[role=switch]').click()
 await page.waitForFunction(()=>document.querySelectorAll('.service-row--child [role=switch]')[1]?.getAttribute('aria-checked')==='true')
 assert.deepEqual(mutations,[{id:2,change:{enabled:true}}])
 await page.getByRole('button',{name:'卡片',exact:true}).click()
 await page.locator('.service-blocks--cards').waitFor()
 await page.screenshot({path:'ui-screenshots/desktop-cards.png',fullPage:true})
 await page.getByRole('button',{name:'列表',exact:true}).click()
 const search=page.locator('.filters input').first()
 await search.fill('fonu.roven.cc')
 await page.waitForFunction(()=>document.querySelectorAll('.service-block').length===1)
 assert.equal(await page.locator('.service-row').count(),1)
 await search.fill('')
 await page.waitForFunction(()=>document.querySelectorAll('.service-block').length===4)
 await page.getByRole('button',{name:'管理业务分组'}).click()
 await page.getByPlaceholder('新分组名称').fill('测试分组')
 await page.getByRole('button',{name:'新增',exact:true}).click()
 await page.locator('.group-manage-row').filter({hasText:'测试分组'}).waitFor()
 await page.getByRole('button',{name:'关闭',exact:true}).click()
 await page.setViewportSize({width:520,height:780})
 await page.waitForTimeout(250)
 const overflow=await page.locator('.workspace').evaluate(el=>({client:el.clientWidth,scroll:el.scrollWidth,blocks:[...el.querySelectorAll('.service-block')].map(b=>({client:b.clientWidth,scroll:b.scrollWidth}))}))
 assert.ok(overflow.scroll<=overflow.client+2,JSON.stringify(overflow))
 assert.ok(overflow.blocks.every(b=>b.scroll<=b.client+2),JSON.stringify(overflow))
 await page.screenshot({path:'ui-screenshots/narrow-list.png',fullPage:true})
 await page.getByRole('button',{name:'卡片',exact:true}).click()
 await page.screenshot({path:'ui-screenshots/narrow-cards.png',fullPage:true})
 await page.evaluate(()=>{localStorage.setItem('fonu-theme','dark')})
 await page.setViewportSize({width:1440,height:980})
 // A second init script switches to dark on the next navigation.
 await page.addInitScript(()=>localStorage.setItem('fonu-theme','dark'))
 await page.reload()
 await page.locator('.service-blocks--cards').waitFor()
 await page.screenshot({path:'ui-screenshots/desktop-dark.png',fullPage:true})
 assert.deepEqual(errors,[])
 console.log('Proxy workspace: independent toggles, domain search, group creation, list/cards, narrow width and screenshot capture passed')
}finally{
 await browser?.close()
 server.kill('SIGTERM')
}
