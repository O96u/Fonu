import assert from 'node:assert/strict'
import { spawn } from 'node:child_process'
import { createRequire } from 'node:module'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'
const require=createRequire(resolve(process.env.FONU_UI_TEST_ROOT,'package.json'))
const { chromium }=require('playwright')
const base='http://127.0.0.1:4177'
const server=spawn(process.execPath,['node_modules/vite/bin/vite.js','preview','--host','127.0.0.1','--port','4177','--strictPort'],{stdio:'inherit'})
let browser,page
try {
 for(let i=0;i<80;i++){try{if((await fetch(base)).ok)break}catch{}if(i===79)throw Error('Vite preview failed');await new Promise(r=>setTimeout(r,200))}
 await mkdir('ui-screenshots',{recursive:true})
 browser=await chromium.launch({headless:true})
 page=await browser.newPage({viewport:{width:1720,height:980}})
 const errors=[]
 page.on('pageerror',e=>errors.push(e.message))
 const now=new Date().toISOString()
 const analytics={updated_at:now,timezone:'Asia/Shanghai',range:'today',backfill_active:false,realtime:{},kpi:{total_requests:1200,unique_ips:520,risk_requests:15,yesterday_total:1000,yesterday_unique_ips:400,yesterday_risk:20,bytes_total:1000},series:{labels:['00:00','02:00','04:00'],requests:[30,120,60],unique_ips:[10,55,30],bytes:[0,0,0],granularity:'two_hour',current_bucket:2},by_province:[],by_country:[{country:'中国',requests:1200,intercept:0}],region_visitors:[],top_ips:[],clients:[],status_codes:[],funnel:{safe:1185,blocked_total:15,blocked_403:15,rate_limited_429:0,auth_failed_401:0,scan_suspected:5,yesterday_scan:10},recent_events:[{at:now,service:'demo-service',domain:'demo.example.com',client_ip:'192.0.2.10',geo:'中国',path:'/',status:200,kind:'normal'}],by_service:[{service:'demo-service',requests:1200,bytes:1000,unique_ips:520}]}
 await page.route('**/*',async route=>{
  const url=new URL(route.request().url())
  if(url.origin!==base)return route.abort()
  if(!url.pathname.startsWith('/api/'))return route.continue()
  let data={}
  switch(url.pathname){
   case '/api/auth/status':data={initialized:true,authenticated:true,username:'admin'};break
   case '/api/version':data={version:'2.0.0'};break
   case '/api/settings':data={theme:'light',timezone:'Asia/Shanghai'};break
   case '/api/status':data={public_ipv4:'192.0.2.1',public_ipv6:'2001:db8::1',public_ip_source:'detect',ddns_status:'ok',ddns_count:0,certificate_status:'ok',certificate_count:1,certificate_days:15,proxy_count:1,nginx_status:'running',nginx_available:true,requests_hourly:[],request_today:1200,error_today:0,avg_response_ms:20,started_at:now,uptime_seconds:1000};break
   case '/api/analytics/overview':data=analytics;break
   case '/api/runtime':data={uptime_seconds:1000,rss_bytes:40000000,heap_bytes:4000000,goroutines:25,versions:{fonu:'2.0.0',go:'go1.27.4',nginx:'1.30.8',frpc:'0.72.0'},processes:{},nginx_errors:[]};break
   case '/api/certificates':data=[{id:1,domain:'demo.example.com',days_left:15,status:'ok'}];break
   case '/api/ddns':case '/api/proxies':data=[];break
  }
  return route.fulfill({json:data})
 })
 await page.addInitScript(()=>{localStorage.setItem('fonu-theme','light')})
 await page.goto(base+'/dashboard')
 await page.getByTestId('dashboard-recent').getByText('demo-service',{exact:true}).waitFor()
 await page.locator('.workspace-side').getByText('运行状态',{exact:true}).waitFor()
 for(const width of [1720,1100,760]){
  await page.setViewportSize({width,height:800})
  await page.waitForFunction(()=>{const g=document.querySelector('.workspace-grid');return g&&getComputedStyle(g).gridTemplateColumns.split(' ').length===2})
  const b=await page.evaluate(()=>{
   const rect=s=>{const r=document.querySelector(s).getBoundingClientRect();return {x:r.x,y:r.y,right:r.right,bottom:r.bottom,width:r.width}}
   return {grid:rect('.workspace-grid'),main:rect('.workspace-main'),side:rect('.workspace-side'),trend:rect('.workspace-main .traffic-panel'),source:rect('[data-testid="dashboard-source-rank"]'),service:rect('[data-testid="dashboard-service-rank"]'),recent:rect('[data-testid="dashboard-recent"]'),runtime:rect('.workspace-side > :nth-child(1)'),quick:rect('.workspace-side > :nth-child(2)'),pending:rect('.workspace-side > :nth-child(3)'),scrollWidth:document.querySelector('.workspace-grid').scrollWidth,clientWidth:document.querySelector('.workspace-grid').clientWidth}
  })
  assert.ok(b.main.right<=b.side.x+1&&Math.abs(b.main.y-b.side.y)<=2,JSON.stringify(b))
  assert.ok(b.source.right<=b.service.x+1&&Math.abs(b.source.y-b.service.y)<=2,JSON.stringify(b))
  assert.ok(b.trend.bottom<=b.source.y+1&&b.source.bottom<=b.recent.y+1,JSON.stringify(b))
  assert.ok(Math.abs(b.recent.width-b.main.width)<=2&&b.runtime.bottom<=b.quick.y+1&&b.quick.bottom<=b.pending.y+1,JSON.stringify(b))
  assert.ok(b.scrollWidth<=b.clientWidth+2,JSON.stringify(b))
  await page.screenshot({path:'ui-screenshots/dashboard-'+width+'.png',fullPage:true})
 }
 assert.deepEqual(errors,[])
} catch(e){if(page)await page.screenshot({path:'ui-screenshots/dashboard-failure.png',fullPage:true}).catch(()=>{});throw e}
finally{await browser?.close();server.kill()}
