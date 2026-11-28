import assert from 'node:assert/strict'
import { spawn } from 'node:child_process'
import { createRequire } from 'node:module'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'
const require=createRequire(resolve(process.env.FONU_UI_TEST_ROOT,'package.json'))
const {chromium}=require('playwright')
const base='http://127.0.0.1:4173'
const server=spawn(process.execPath,['node_modules/vite/bin/vite.js','preview','--host','127.0.0.1','--port','4173','--strictPort'],{stdio:'inherit'})
let browser, page
try{
 for(let i=0;i<80;i++){try{if((await fetch(base)).ok)break}catch{}if(i===79)throw Error('Vite preview failed');await new Promise(r=>setTimeout(r,200))}
 await mkdir('ui-screenshots',{recursive:true})
 browser=await chromium.launch({headless:true})
 page=await browser.newPage({viewport:{width:1440,height:980},deviceScaleFactor:1})
 const errors=[]
 page.on('pageerror',err=>{errors.push(err.message);console.error('Browser page error:',err.message)})
 page.on('console',msg=>{if(msg.type()==='error'&&msg.text().startsWith('[fonu]'))errors.push(msg.text())})
 let rules=[
  {id:1,name:'飞牛影视',domain:'speedtest.muxui.com',upstream:'http://192.168.8.3:8005',listen_port:8009,entry_id:2,enabled:true,security:{china_only:true}},
  {id:2,name:'飞牛影视',domain:'s.muxui.com',upstream:'http://192.168.8.3:8005',listen_port:8009,entry_id:2,enabled:false,security:{basic_auth:{enabled:true,username:'test',has_password:true}}},
  {id:3,name:'飞牛影视',domain:'fonu.roven.cc',upstream:'http://192.168.8.3:8005',listen_port:8009,entry_id:2,enabled:true,security:{}},
  {id:4,name:'飞牛 NAS',domain:'nas.example.com',upstream:'http://192.168.1.10:5666',listen_port:443,entry_id:1,enabled:true,security:{}},
  {id:5,name:'iKuai',domain:'ikuai.example.com',upstream:'https://192.168.1.1',listen_port:443,entry_id:1,enabled:true,security:{china_only:true}},
  {id:6,name:'相册',domain:'photo.example.com',upstream:'http://192.168.1.10:2283',listen_port:443,entry_id:1,enabled:true,security:{}}
 ].map(r=>({...r,hosts:[{id:r.id,hostname:r.domain}],listen_ipv4:true,listen_ipv6:false,https_enabled:true,http_redirect:true,sort_order:r.id,created_at:'2026-11-26T00:00:00Z',updated_at:'2026-11-26T00:00:00Z'}))
 const entries=[{id:1,name:'默认入口',listen_port:443,rule_count:3},{id:2,name:'影音入口',listen_port:8009,rule_count:3}].map(e=>({...e,listen_ipv4:true,listen_ipv6:false,https_enabled:true,http_redirect:true,sort_order:e.id,created_at:'',updated_at:''}))
 let nextId=7, failBatch=true
 let business={groups:[{id:1,name:'影音娱乐'},{id:2,name:'常用服务'}],members:[{rule_id:1,group_id:1},{rule_id:2,group_id:1},{rule_id:3,group_id:1},{rule_id:4,group_id:2},{rule_id:6,group_id:2}]}
 const traffic=rules.map(r=>({rule_id:r.id,connections:r.enabled?2:0,upload_rate:65536,download_rate:1048576,upload_total:1073741824,download_total:5368709120}))
 const mutations=[],batches=[],deletes=[]
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
  else if(url.pathname==='/api/certificates')data=[{id:1,domain:'example.com',name:'测试证书',domains:['*.example.com'],cert_path:'/test/cert.pem',key_path:'/test/key.pem',expires_at:new Date(Date.now()+86400000*60).toISOString(),status:'valid'}]
  else if(url.pathname==='/api/proxies/traffic')data=traffic
  else if(url.pathname==='/api/proxies/traffic/snapshot')data={rules:traffic,total:{rule_id:0,connections:2,upload_rate:65536,download_rate:1048576,upload_total:1073741824,download_total:5368709120},sampled_at:new Date().toISOString(),active_window_seconds:65,rate_window_seconds:5}
  else if(/^\/api\/proxies\/\d+\/diagnostics$/.test(url.pathname))data=method==='POST'?{id:1,upstream:'http://192.168.1.10:5666',checked_at:new Date().toISOString(),state:'reachable',duration_ms:10,http_status:200,detail:'上游已响应 HTTP'}:[]
  else if(url.pathname==='/api/discovery/scan')data=[{name:'测试服务',port:8080,host:'127.0.0.1',upstream:'http://127.0.0.1:8080',detected:true,platform:'test'}]
  else if(url.pathname==='/api/proxies/batch'&&method==='POST'){
   const payload=route.request().postDataJSON();batches.push(payload)
   if(failBatch){failBatch=false;return route.fulfill({status:400,json:{error:'模拟发布失败'}})}
   const make=(item,id)=>({...rules[0],...item,id,domain:item.hosts[0],hosts:item.hosts.map((hostname,i)=>({id:i+1,hostname})),security:item.security??{},sort_order:id,entry_id:item.entry_id??1})
   const updated=payload.update_id?[make(payload.update,payload.update_id)]:[]
   const created=payload.create.map(item=>make(item,nextId++))
   rules=rules.map(r=>updated.find(x=>x.id===r.id)||r).concat(created)
   data=[...updated,...created]
  }
  else if(url.pathname==='/api/proxy-groups'){
   if(method==='POST'){const change=route.request().postDataJSON();if(change.action==='create')business={...business,groups:[...business.groups,{id:3,name:change.name}]}}
   data=business
  }else if(/^\/api\/proxies\/\d+$/.test(url.pathname)&&method==='PUT'){
   const id=Number(url.pathname.split('/').pop()),change=route.request().postDataJSON()
   mutations.push({id,change});rules=rules.map(r=>r.id===id?{...r,...change}:r);data=rules.find(r=>r.id===id)
  }else if(/^\/api\/proxies\/\d+$/.test(url.pathname)&&method==='DELETE'){
   const id=Number(url.pathname.split('/').pop());deletes.push(id);rules=rules.filter(r=>r.id!==id);return route.fulfill({status:204,body:''})
  }else if(url.pathname.endsWith('/favicon'))return route.fulfill({status:404,body:''})
  else if(url.pathname==='/api/settings')data={}
  return route.fulfill({json:data})
 })
 await page.addInitScript(()=>{localStorage.setItem('fonu-theme','light');localStorage.setItem('fonu-nav-collapsed','true')})
 await page.goto(base+'/proxies')
 await page.locator('.summary-identity').first().waitFor()
 assert.equal(await page.locator('.service-block').count(),4)
 await page.getByRole('button',{name:'端口分组 / 监听入口'}).click()
 const listenerManager=page.locator('.listener-manager')
 await listenerManager.waitFor()
 assert.equal(await listenerManager.locator('.proxy-entry-card').count(),2)
 await listenerManager.getByRole('button',{name:'关闭',exact:true}).click()
 await listenerManager.waitFor({state:'hidden'})
 await page.locator('.summary-identity').first().click()
 assert.equal(await page.locator('.service-row--child').count(),3)
 assert.match(await page.locator('.summary-identity').first().innerText(),/3 条独立规则/)
 assert.equal(await page.locator('.service-row--child').first().getByRole('link',{name:'在新窗口打开'}).getAttribute('href'),'https://speedtest.muxui.com:8009')
 assert.equal(await page.locator('.service-row--child').first().getByRole('link',{name:'在新窗口打开'}).getAttribute('target'),'_blank')
 await page.screenshot({path:'ui-screenshots/desktop-list.png',fullPage:true})
 const authRow=page.locator('.service-row--child').filter({hasText:'s.muxui.com'})
 await authRow.getByRole('button',{name:'编辑规则'}).click()
 const edit=page.locator('.editor-frame').filter({has:page.getByRole('heading',{name:'编辑代理',exact:true})})
 await edit.getByText('证书列表中没有未过期的匹配证书',{exact:true}).waitFor()
 await edit.getByRole('tab',{name:'安全设置'}).click()
 assert.equal(await edit.getByRole('textbox',{name:'代理认证用户名'}).inputValue(),'test')
 await edit.getByRole('button',{name:'取消',exact:true}).click()
 await edit.waitFor({state:'hidden'})
 await page.locator('.service-row--child').filter({hasText:'fonu.roven.cc'}).getByRole('button',{name:'编辑规则'}).click()
 await edit.getByRole('tab',{name:'安全设置'}).click()
 await edit.getByText('访问认证',{exact:true}).click()
 assert.equal(await edit.locator('.security-option').filter({hasText:'Basic Auth'}).getByRole('switch').getAttribute('aria-checked'),'false')
 await edit.getByRole('button',{name:'取消',exact:true}).click()
 await edit.waitFor({state:'hidden'})
 // Each domain rule has its own switch. Only the selected rule is updated.
 await Promise.all([page.waitForResponse(r=>r.url().endsWith('/api/proxies/2')&&r.request().method()==='PUT'),page.locator('.service-row--child').nth(1).locator('[role=switch]').click()])
 await page.waitForFunction(()=>document.querySelectorAll('.service-row--child [role=switch]')[1]?.getAttribute('aria-checked')==='true')
 assert.deepEqual(mutations,[{id:2,change:{enabled:true}}])
 await page.getByRole('button',{name:'卡片',exact:true}).click()
 await page.locator('.service-blocks--cards').waitFor()
 await page.locator('.summary-identity').first().click()
 const bounds=await page.locator('.service-block').evaluateAll(els=>els.map(el=>({top:el.getBoundingClientRect().top,bottom:el.getBoundingClientRect().bottom})))
 for(const b of bounds)for(const other of bounds)if(Math.abs(b.top-other.top)<2)assert.ok(Math.abs(b.bottom-other.bottom)<2,JSON.stringify(bounds))
 await page.screenshot({path:'ui-screenshots/desktop-cards.png',fullPage:true})
 await page.locator('.summary-identity').first().click()
 await page.locator('.expanded-panel').waitFor()
 assert.equal(await page.locator('.expanded-panel .service-row--child').count(),3)
 assert.equal(await page.locator('.expanded-panel .service-row--card').count(),0)
 await page.screenshot({path:'ui-screenshots/desktop-cards-expanded.png',fullPage:true})
 await page.locator('.summary-identity').first().click()
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
 // A long domain cannot push the favicon or link controls to another line.
 const longRule=rules.find(r=>r.id===6)
 longRule.domain='very-long-service-subdomain-for-responsive-layout.example.net'
 longRule.hosts=[{id:6,hostname:longRule.domain}]
 await page.reload()
 await page.locator('.service-row--card').filter({hasText:longRule.domain}).waitFor()
 const longCard=page.locator('.service-row--card').filter({hasText:longRule.domain})
 const layout=await longCard.evaluate(el=>{
  const box=s=>el.querySelector(s).getBoundingClientRect()
  return {icon:box('.service-favicon'),name:box('.service-name'),host:box('.primary-host'),open:box('.open-link'),copy:box('[aria-label="复制链接"]')}
 })
 assert.ok(Math.abs(layout.icon.top-layout.name.top)<5,JSON.stringify(layout))
 assert.ok(Math.abs(layout.open.top-layout.copy.top)<5,JSON.stringify(layout))
 assert.equal(await longCard.getByRole('link',{name:'在新窗口打开'}).getAttribute('href'),'https://'+longRule.domain)
 await page.screenshot({path:'ui-screenshots/narrow-long-domain.png',fullPage:true})
 await page.evaluate(()=>{localStorage.setItem('fonu-theme','dark')})
 await page.setViewportSize({width:1440,height:980})
 // A second init script switches to dark on the next navigation.
 await page.addInitScript(()=>localStorage.setItem('fonu-theme','dark'))
 await page.reload()
 await page.locator('.service-blocks--cards').waitFor()
 await page.screenshot({path:'ui-screenshots/desktop-dark.png',fullPage:true})
 // Editor: committed values persist across tabs and failed saves.
 await page.getByRole('button',{name:'新增代理',exact:true}).first().click()
 const editor=page.locator('.editor-frame').filter({has:page.getByRole('heading',{name:'新增代理',exact:true})})
 await editor.waitFor()
 await editor.getByPlaceholder('例如：飞牛影视').click()
 assert.ok(await editor.getByPlaceholder('例如：飞牛影视').evaluate(el=>document.activeElement===el),'rule name input must retain focus inside the modal')
 await editor.getByPlaceholder('例如：飞牛影视').pressSequentially('New service')
 assert.equal(await editor.getByPlaceholder('例如：飞牛影视').inputValue(),'New service')
 await editor.getByPlaceholder('例如：飞牛影视').fill('新服务')
 await editor.getByPlaceholder('http://192.168.1.100:8005').fill('http://127.0.0.1:8080')
 assert.equal(await editor.getByPlaceholder('例如：飞牛影视').inputValue(),'新服务')
 assert.equal(await editor.getByPlaceholder('http://192.168.1.100:8005').inputValue(),'http://127.0.0.1:8080')
 await editor.getByText('有未保存的配置',{exact:true}).waitFor()
 await editor.getByText('监听端口',{exact:true}).waitFor({state:'visible'})
 const domain=editor.getByRole('textbox',{name:'前端域名'})
 await domain.click()
 await domain.pressSequentially('new.example.net')
 assert.equal(await domain.inputValue(),'new.example.net')
 await editor.getByRole('tab',{name:'安全设置'}).click()
 await editor.getByRole('tab',{name:'基础配置'}).focus()
 await editor.getByRole('tab',{name:'基础配置'}).press('Enter')
 assert.equal(await editor.getByRole('tab',{name:'基础配置'}).getAttribute('aria-selected'),'true')
 assert.equal(await domain.inputValue(),'new.example.net')
 await domain.fill('')
 await editor.getByPlaceholder('例如：飞牛影视').waitFor({state:'visible'})
 assert.equal(await editor.getByPlaceholder('例如：飞牛影视').inputValue(),'新服务')
 assert.equal(await editor.getByPlaceholder('http://192.168.1.100:8005').inputValue(),'http://127.0.0.1:8080')
 await editor.getByRole('button',{name:'创建代理',exact:true}).click()
 await editor.locator('.editor-save-error').getByText('至少需要一个前端域名').waitFor()
 await page.screenshot({path:'ui-screenshots/editor-validation.png',fullPage:true})
 await editor.getByRole('tab',{name:'安全设置'}).click()
 await editor.getByRole('button',{name:'一键推荐'}).click()
 await editor.getByText('基础安全响应头',{exact:true}).waitFor()
 await editor.getByText('请求防护',{exact:true}).click()
 await editor.getByText('敏感路径保护',{exact:true}).waitFor()
 await page.screenshot({path:'ui-screenshots/editor-security.png',fullPage:true})
 await editor.getByText('上游代理设置',{exact:true}).click()
 await page.screenshot({path:'ui-screenshots/editor-advanced.png',fullPage:true})
 await editor.getByRole('button',{name:'取消',exact:true}).click()
 await page.getByRole('button',{name:'放弃更改',exact:true}).click()
 await editor.waitFor({state:'hidden'})
 // Copy source and independently editable destination; no automatic commit.
 await page.locator('.service-row').filter({hasText:'nas.example.com'}).getByRole('button',{name:'复制规则'}).click()
 const copy=page.locator('.editor-frame').filter({has:page.getByRole('heading',{name:'复制代理规则'})})
 await copy.getByText('复制来源：飞牛 NAS').waitFor()
 assert.equal(await copy.getByPlaceholder('http://192.168.1.100:8005').inputValue(),'http://192.168.1.10:5666')
 assert.equal(batches.length,0)
 await page.screenshot({path:'ui-screenshots/editor-copy.png',fullPage:true})
 await copy.getByRole('button',{name:'取消',exact:true}).click()
 await copy.waitFor({state:'hidden'})
 // Discovery failure does not drop selected service or edited domain.
 await page.getByRole('button',{name:'发现服务',exact:true}).click()
 const discovery=page.locator('.editor-frame').filter({has:page.getByRole('heading',{name:'发现内网服务'})})
 await discovery.locator('.scan-result').waitFor()
 await page.waitForFunction(()=>!document.querySelector('.scan-status'))
 assert.equal(await discovery.getByPlaceholder('nas.example.com').inputValue(),'')
 assert.equal(await discovery.getByRole('checkbox',{name:'选择 测试服务'}).getAttribute('aria-checked'),'false')
 await discovery.getByRole('checkbox',{name:'选择 测试服务'}).click()
 await discovery.getByPlaceholder('nas.example.com').click()
 assert.ok(await discovery.getByPlaceholder('nas.example.com').evaluate(el=>document.activeElement===el),'discovery domain input must retain focus')
 await discovery.getByPlaceholder('nas.example.com').fill('')
 await discovery.getByPlaceholder('nas.example.com').pressSequentially('found.muxui.com')
 await discovery.getByPlaceholder('服务名称').click()
 await discovery.getByPlaceholder('服务名称').fill('')
 await discovery.getByPlaceholder('服务名称').pressSequentially('Discovered service')
 assert.equal(await discovery.getByPlaceholder('服务名称').inputValue(),'Discovered service')
 await discovery.getByPlaceholder('服务名称').fill('发现服务测试')
 await discovery.getByRole('button',{name:'重新扫描'}).click()
 await discovery.getByRole('button',{name:'创建 1 条规则'}).waitFor()
 await page.waitForFunction(()=>!document.querySelector('.scan-status'))
 assert.equal(await discovery.getByPlaceholder('nas.example.com').inputValue(),'found.muxui.com')
 assert.equal(await discovery.getByPlaceholder('服务名称').inputValue(),'发现服务测试')
 await discovery.getByRole('button',{name:'创建 1 条规则'}).click()
 await discovery.getByText('模拟发布失败',{exact:true}).waitFor()
 assert.deepEqual(batches[0].create[0].hosts,['found.muxui.com'])
 assert.equal(batches[0].create[0].name,'发现服务测试')
 assert.equal(await discovery.getByPlaceholder('nas.example.com').inputValue(),'found.muxui.com')
 await page.screenshot({path:'ui-screenshots/editor-discovery.png',fullPage:true})
 await discovery.getByRole('button',{name:'创建 1 条规则'}).click()
 await discovery.waitFor({state:'hidden'})
 assert.deepEqual(batches[1],batches[0])
 await page.locator('.service-row').filter({hasText:'found.muxui.com'}).waitFor()
 // Delete confirmation only submits the selected rule ID.
 await page.locator('.service-row').filter({hasText:'found.muxui.com'}).getByRole('button',{name:'更多操作'}).click()
 await page.getByText('删除规则',{exact:true}).click()
 const del=page.locator('.editor-frame').filter({has:page.getByRole('heading',{name:'删除独立代理规则'})})
 await del.getByText('found.muxui.com',{exact:true}).waitFor()
 await page.screenshot({path:'ui-screenshots/editor-delete.png',fullPage:true})
 await del.getByRole('button',{name:'确认删除'}).click()
 await del.waitFor({state:'hidden'})
 assert.deepEqual(deletes,[7])
 // Responsive editor uses one scroll body and fixed navigation/footer.
 await page.setViewportSize({width:520,height:780})
 await page.getByRole('button',{name:'新增代理',exact:true}).first().click()
 await editor.waitFor()
 const frameBounds=await editor.boundingBox()
 assert.ok(frameBounds.width<=520&&frameBounds.height<=780,JSON.stringify(frameBounds))
 await page.screenshot({path:'ui-screenshots/narrow-editor.png',fullPage:true})
 await editor.getByRole('button',{name:'取消',exact:true}).click()
 await page.setViewportSize({width:1440,height:980})
 await page.locator('.service-row').filter({hasText:'nas.example.com'}).getByRole('button',{name:'详情',exact:true}).click()
 await page.getByRole('tab',{name:'运行诊断',exact:true}).click()
 const diag=page.locator('.diagnostics-panel')
 await diag.getByText('尚未检测当前目标地址').waitFor()
 assert.equal(await diag.locator('[role=switch]').getAttribute('aria-checked'),'false')
 await diag.getByRole('button',{name:'立即检测'}).click()
 await diag.locator('.diag-latest').getByText('HTTP 可达').waitFor()
 await page.screenshot({path:'ui-screenshots/diagnostics.png',fullPage:true})
 // More than ten services must remain in one scrollable workspace.
 const extra=Array.from({length:11},(_,i)=>({...rules[0],id:100+i,name:'额外服务 '+i,domain:'extra'+i+'.example.net',hosts:[{id:100+i,hostname:'extra'+i+'.example.net'}],security:{}}))
 rules=rules.concat(extra)
 await page.reload()
 await page.waitForFunction(()=>document.querySelectorAll('.service-block').length>=15)
 assert.equal(await page.locator('.workspace .n-pagination').count(),0)
 assert.deepEqual(errors,[])
 console.log('Proxy workspace: independent toggles, domain search, group creation, list/cards, narrow width and screenshot capture passed')
}catch(error){
 if(page){
  await page.screenshot({path:'ui-screenshots/failure.png',fullPage:true}).catch(()=>{})
  console.error('Visible dialogs/tabs:',await page.locator('[role=dialog]:visible, [role=tab]:visible, .n-tabs-tab:visible').evaluateAll(els=>els.map(el=>({tag:el.tagName,role:el.getAttribute('role'),label:el.getAttribute('aria-label'),text:el.textContent?.trim().slice(0,80)}))).catch(()=>[]))
 }
 throw error
}finally{
 await browser?.close()
 server.kill('SIGTERM')
}
