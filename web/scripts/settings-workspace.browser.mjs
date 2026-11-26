import assert from 'node:assert/strict'
import { spawn } from 'node:child_process'
import { createRequire } from 'node:module'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'
const require=createRequire(resolve(process.env.FONU_UI_TEST_ROOT,'package.json'))
const {chromium}=require('playwright')
const base='http://127.0.0.1:4176'
const server=spawn(process.execPath,['node_modules/vite/bin/vite.js','preview','--host','127.0.0.1','--port','4176','--strictPort'],{stdio:'inherit'})
let browser,page
try {
 for(let i=0;i<80;i++){try{if((await fetch(base)).ok)break}catch{}if(i===79)throw Error('Vite preview failed');await new Promise(r=>setTimeout(r,200))}
 await mkdir('ui-screenshots',{recursive:true})
 browser=await chromium.launch({headless:true})
 page=await browser.newPage({viewport:{width:1440,height:980}})
 const errors=[]
 page.on('pageerror',err=>errors.push(err.message))
 const saved={theme:'system',timezone:'Asia/Shanghai',ddns_check_interval_minutes:'5',log_retention_days:'30',analytics_retention_days:'30',notify_type:'telegram',notify_telegram_json:JSON.stringify({chat_id:'123',proxy_url:'',has_bot_token:true}),acme_email:'',zerossl_eab_kid:'********',zerossl_eab_hmac_key:'********'}
 const writes=[]
 let fail=false
 await page.route('**/*',async route=>{
  const url=new URL(route.request().url())
  if(url.origin!==base)return route.abort()
  if(!url.pathname.startsWith('/api/'))return route.continue()
  let data={}
  if(url.pathname==='/api/auth/status')data={initialized:true,authenticated:true,username:'admin'}
  else if(url.pathname==='/api/version')data={version:'2.0.0-beta.42'}
  else if(url.pathname==='/api/settings'){
   if(route.request().method()==='PUT'){
    const body=route.request().postDataJSON();writes.push(body)
    if(fail)return route.fulfill({status:400,json:{error:'模拟保存失败'}})
    Object.assign(saved,body)
   }
   data=saved
  }
  else if(url.pathname==='/api/settings/china-cidr')data={entry_count_v4:0,entry_count_v6:0}
  else if(url.pathname==='/api/settings/nginx/global')data={mode:'auto',content:'',generated_framework:'http {}',generated_snippet:'# auto',backups:[]}
  return route.fulfill({json:data})
 })
 await page.addInitScript(()=>{localStorage.setItem('fonu-theme','system');localStorage.setItem('fonu-nav-collapsed','true')})
 await page.goto(base+'/settings')
 const save=page.getByTestId('settings-save-bar').getByRole('button',{name:'保存设置',exact:true})
 // Wait for the specific PUT and durable clean state, not overlapping transient toasts.
 const saveBar=page.getByTestId('settings-save-bar')
 async function saveAndWait(expectedStatus=200){
  const count=writes.length
  const [response]=await Promise.all([
   page.waitForResponse(r=>new URL(r.url()).pathname==='/api/settings'&&r.request().method()==='PUT'),
   save.click(),
  ])
  assert.equal(response.status(),expectedStatus)
  assert.equal(writes.length,count+1)
  if(expectedStatus===200){
   await saveBar.getByText('当前配置已保存',{exact:true}).waitFor()
   await page.waitForFunction(()=>document.querySelector('[data-testid="settings-save-bar"] button:last-child')?.disabled===true)
  }else{
   await saveBar.getByText('当前分类有未保存修改',{exact:true}).waitFor()
   await page.waitForFunction(()=>document.querySelector('[data-testid="settings-save-bar"] button:last-child')?.disabled===false)
  }
 }

 await save.waitFor()
 await page.waitForFunction(()=>document.querySelector('[data-testid="settings-save-bar"] button:last-child')?.disabled)
 await page.screenshot({path:'ui-screenshots/settings-general-desktop.png',fullPage:true})
 const ddns=page.locator('.settings-field').filter({hasText:'DDNS 检查周期'}).locator('input')
 await ddns.fill('15');await ddns.blur()
 await page.getByTestId('settings-tab-acme').click()
 const email=page.getByPlaceholder('admin@example.com',{exact:true})
 await email.fill('invalid-email')
 await page.getByTestId('settings-tab-general').click()
 assert.equal(await ddns.inputValue(),'15')
 await saveAndWait()
 assert.deepEqual(writes.at(-1),{ddns_check_interval_minutes:'15'})
 await page.getByTestId('settings-tab-acme').click()
 assert.equal(await email.inputValue(),'invalid-email')
 const zero=page.locator('.provider-row').filter({hasText:'ZeroSSL'})
 await zero.getByRole('button',{name:'编辑',exact:true}).click()
 await page.getByText('关闭后保留本次输入', {exact:false}).waitFor()
 await page.screenshot({path:'ui-screenshots/settings-acme-editor.png',fullPage:true})
 await page.getByRole('button',{name:'完成编辑',exact:true}).click()
 await page.getByRole('button',{name:'取消修改',exact:true}).click()
 assert.equal(await email.inputValue(),'')
 await page.getByTestId('settings-tab-logs').click()
 const retention=page.locator('.settings-field').filter({hasText:'统计保留天数'}).locator('input')
 await retention.fill('45');await retention.blur();fail=true
 await saveAndWait(400)
 await page.getByText('模拟保存失败',{exact:true}).waitFor()
 assert.equal(await retention.inputValue(),'45')
 fail=false;await saveAndWait()
 assert.deepEqual(writes.at(-1),{analytics_retention_days:'45'})
 await page.getByTestId('settings-tab-backup').click()
 assert.equal(await page.getByTestId('settings-save-bar').count(),0)
 await page.getByTestId('settings-tab-notify').click()
 assert.equal(await page.locator('.notify-channel-choice').count(),9)
 assert.equal(await page.getByTestId('notify-channel-telegram').getAttribute('aria-pressed'),'true')
 assert.equal(await page.getByPlaceholder('-1001234567890').inputValue(),'123')
 await page.getByTestId('notify-enabled').click()
 assert.equal(await page.getByTestId('notify-enabled').getAttribute('aria-checked'),'false')
 await saveAndWait()
 assert.equal(writes.at(-1).notify_type,'')
 assert.equal(JSON.parse(writes.at(-1).notify_webhook_json).preferred_type,'telegram')
 await page.getByTestId('notify-enabled').click()
 await page.getByTestId('notify-channel-wecom').click()
 await page.getByText('自建应用',{exact:true}).click()
 await page.getByPlaceholder('1000002').waitFor()
 await page.getByPlaceholder('应用标识').fill('corp-test')
 await page.getByPlaceholder('1000002').fill('1000002')
 await page.getByPlaceholder('zhangsan|lisi').fill('test-user')
 await page.getByPlaceholder('请输入应用密钥').fill('test-secret')
 await page.screenshot({path:'ui-screenshots/settings-wecom-app.png',fullPage:true})
 await page.getByRole('button',{name:'取消修改',exact:true}).click()
 assert.equal(await page.getByTestId('notify-channel-telegram').getAttribute('aria-pressed'),'true')
 await page.setViewportSize({width:520,height:750})
 await page.getByTestId('settings-tab-general').click()
 await page.screenshot({path:'ui-screenshots/settings-narrow.png',fullPage:true})
 assert.deepEqual(errors,[])
} catch(error){if(page)await page.screenshot({path:'ui-screenshots/settings-failure.png',fullPage:true}).catch(()=>{});throw error}
finally{await browser?.close();server.kill()}
