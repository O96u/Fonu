import assert from 'node:assert/strict'
import { spawn } from 'node:child_process'
import { createRequire } from 'node:module'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'
const require=createRequire(resolve(process.env.FONU_UI_TEST_ROOT,'package.json'))
const {chromium}=require('playwright')
const base='http://127.0.0.1:4175'
const server=spawn(process.execPath,['node_modules/vite/bin/vite.js','preview','--host','127.0.0.1','--port','4175','--strictPort'],{stdio:'inherit'})
let browser,page
try {
 for(let i=0;i<80;i++){try{if((await fetch(base)).ok)break}catch{}if(i===79)throw Error('Vite preview failed');await new Promise(r=>setTimeout(r,200))}
 await mkdir('ui-screenshots',{recursive:true})
 browser=await chromium.launch({headless:true})
 page=await browser.newPage({viewport:{width:1440,height:980}})
 const errors=[]
 page.on('pageerror',err=>errors.push(err.message))
 const rows=[{time:'2026-11-26T14:09:31+08:00',domain:'nas.example.com',server_port:443,method:'GET',path:'/login',status:403,response_time:0.002,client_ip:'192.0.2.1',upstream:'-',geo:'海外'},{time:'2026-11-26T14:09:32+08:00',domain:'media.example.com',method:'GET',path:'/',status:200,response_time:0.01,client_ip:'192.0.2.2',upstream:'127.0.0.1:80'}]
 await page.route('**/*',async route=>{
 const url=new URL(route.request().url())
 if(url.origin!==base)return route.abort()
 if(!url.pathname.startsWith('/api/'))return route.continue()
 let data=[]
 if(url.pathname==='/api/auth/status')data={initialized:true,authenticated:true,username:'admin'}
 else if(url.pathname==='/api/version')data={version:'2.0.0-beta.35',nginx_http_port:80,nginx_https_port:443}
 else if(url.pathname==='/api/settings')data={}
 else if(url.pathname==='/api/logs/access')data=rows.filter(r=>!url.searchParams.get('ip')||r.client_ip===url.searchParams.get('ip'))
 else if(url.pathname==='/api/logs/system')data=[{time:rows[0].time,level:'INFO',module:'SYSTEM',message:'Fonu started'}]
 else if(url.pathname==='/api/logs/error')data=['2026/11/26 14:09:31 [error] upstream failed']
 else if(url.pathname==='/api/frp/logs')data=['2026-11-26 14:09:31 [I] [client] login success']
 return route.fulfill({json:data})
 })
 await page.addInitScript(()=>{localStorage.setItem('fonu-theme','light');localStorage.setItem('fonu-nav-collapsed','true')})
 await page.goto(base+'/logs?tab=access')
 const row=page.locator('.log-table tbody tr').filter({hasText:'/login'})
 await row.waitFor()
 assert.equal(await row.locator('.http-status-badge').innerText(),'403')
 await page.screenshot({path:'ui-screenshots/log-workspace-desktop.png',fullPage:true})
 await row.click()
 await page.getByTestId('log-detail-fields').waitFor()
 assert.match(await page.getByTestId('log-detail-fields').innerText(),/192\.0\.2\.1/)
 await page.screenshot({path:'ui-screenshots/log-detail.png',fullPage:true})
 await page.getByRole('button',{name:'筛选此 IP',exact:true}).click()
 await page.waitForURL('**/logs?tab=access&ip=192.0.2.1')
 await page.waitForFunction(()=>document.querySelectorAll('.log-table tbody tr').length===1)
 await page.getByTestId('log-source-frp').click()
 await page.getByText('login success',{exact:true}).waitFor()
 await page.getByTestId('log-source-system').click()
 await page.getByText('Fonu started',{exact:true}).waitFor()
 await page.getByPlaceholder('关键词搜索').fill('not-present')
 await page.getByText('没有匹配的日志',{exact:true}).waitFor()
 await page.setViewportSize({width:520,height:780})
 await page.screenshot({path:'ui-screenshots/log-workspace-narrow.png',fullPage:true})
 assert.deepEqual(errors,[])
} catch(error){if(page)await page.screenshot({path:'ui-screenshots/log-workspace-failure.png',fullPage:true}).catch(()=>{});throw error}
finally{await browser?.close();server.kill()}
