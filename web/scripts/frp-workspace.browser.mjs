import assert from 'node:assert/strict'
import { spawn } from 'node:child_process'
import { createRequire } from 'node:module'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const require = createRequire(resolve(process.env.FONU_UI_TEST_ROOT, 'package.json'))
const { chromium } = require('playwright')
const base = 'http://127.0.0.1:4174'
const server = spawn(process.execPath, ['node_modules/vite/bin/vite.js', 'preview', '--host', '127.0.0.1', '--port', '4174', '--strictPort'], { stdio: 'inherit' })
let browser
try {
  for (let i = 0; i < 80; i++) {
    try { if ((await fetch(base)).ok) break } catch {}
    if (i === 79) throw Error('Vite preview failed')
    await new Promise(r => setTimeout(r, 200))
  }
  await mkdir('ui-screenshots', { recursive: true })
  browser = await chromium.launch({ headless: true })
  const page = await browser.newPage({ viewport: { width: 1440, height: 980 } })
  const errors = [], saves = []
  page.on('pageerror', err => errors.push(err.message))
  let cfg = {
    enabled: true, server_addr: 'frp.example.com', server_port: 7000,
    auth_token: '********', has_auth_token: true, tls_enabled: true,
    custom_domains: ['nas.example.com'],
    tcp_proxies: [
      { id: 'ssh1', name: 'ssh', protocol: 'tcp', local_ip: '192.168.1.10', local_port: 22, remote_port: 60022, enabled: true },
      { id: 'dns1', name: 'dns', protocol: 'udp', local_ip: '192.168.1.10', local_port: 53, remote_port: 6053, enabled: true }
    ]
  }
  const status = { enabled: true, connected: true, process_running: true, message: '已连接', http_gateway_enabled: true, https_gateway_enabled: true, http_gateway_state: 'registered', https_gateway_state: 'registered', tunnel_states: { ssh1: 'registered', dns1: 'failed' }, server_endpoint: 'frp.example.com:7000' }
  const meta = { status, frps_config: 'token = "********"', nginx_http_port: 80, nginx_https_port: 443 }
  let restartCount = 0, failSave = false
  await page.route('**/*', async route => {
    const url = new URL(route.request().url())
    if (url.origin !== base) return route.abort()
    if (!url.pathname.startsWith('/api/')) return route.continue()
    const method = route.request().method()
    let data = []
    if (url.pathname === '/api/auth/status') data = { initialized: true, authenticated: true, username: 'admin' }
    else if (url.pathname === '/api/version') data = { version: '2.0.0-beta.25', nginx_http_port: 80, nginx_https_port: 443 }
    else if (url.pathname === '/api/settings') data = {}
    else if (url.pathname === '/api/frp' && method === 'GET') data = { ...cfg, ...meta }
    else if (url.pathname === '/api/frp' && method === 'PUT') {
      const payload = route.request().postDataJSON()
      saves.push(payload)
      if (failSave) { failSave = false; return route.fulfill({ status: 400, json: { error: '模拟保存失败' } }) }
      cfg = { ...cfg, ...payload, auth_token: '********', has_auth_token: true }
      data = { message: '配置已保存', config: cfg, ...meta }
    } else if (url.pathname === '/api/frp/status') data = status
    else if (url.pathname === '/api/frp/restart') { restartCount++; data = { message: 'FRP 已重启', status } }
    else if (url.pathname === '/api/frp/domain-candidates') data = { domains: ['nas.example.com', 'media.example.com'] }
    else if (url.pathname === '/api/frp/logs') data = ['[INFO] login success', '[WARN] test warning']
    else if (url.pathname === '/api/frp/diagnostics') data = [{ name: '控制端口', state: 'success', detail: 'TCP 可连接' }]
    return route.fulfill({ json: data })
  })
  await page.addInitScript(() => { localStorage.setItem('fonu-theme','light'); localStorage.setItem('fonu-nav-collapsed','true') })
  await page.goto(base + '/frp')
  await page.locator('.domain-row').first().waitFor()
  assert.equal(await page.locator('.domain-row').count(), 1)
  assert.equal(await page.locator('.domain-row a').getAttribute('href'), 'https://nas.example.com')
  await page.getByRole('tab', { name: '连接设置' }).click()
  await page.getByPlaceholder('127.0.0.1 或 frp.example.com').fill('draft.example.com')
  await page.getByRole('tab', { name: 'Web 穿透' }).click()
  await page.getByRole('button', { name: '同步域名' }).click()
  await page.getByRole('button', { name: '导入 1 个域名' }).click()
  await page.locator('.domain-row').filter({ hasText: 'media.example.com' }).waitFor()
  assert.equal(saves.at(-1).server_addr, 'frp.example.com')
  assert.equal(saves.at(-1).auth_token, undefined)
  await page.getByRole('tab', { name: '连接设置' }).click()
  assert.equal(await page.getByPlaceholder('127.0.0.1 或 frp.example.com').inputValue(), 'draft.example.com')
  const before = saves.length
  await page.getByRole('button', { name: '重启 FRP 服务' }).click()
  await page.waitForFunction(() => !document.querySelector('.status-footer .n-button--loading'))
  assert.equal(restartCount, 1)
  assert.equal(saves.length, before)
  await page.getByRole('tab', { name: '端口穿透' }).click()
  await page.locator('.frp-table-mode').waitFor({ state: 'visible' })
  await page.screenshot({ path: 'ui-screenshots/frp-desktop.png', fullPage: true })
  await page.setViewportSize({ width: 520, height: 780 })
  await page.locator('.frp-card-mode').waitFor({ state: 'visible' })
  assert.equal(await page.locator('.tunnel-card').count(), 2)
  await page.getByPlaceholder('搜索名称、地址、端口').fill('60022')
  await page.waitForFunction(() => document.querySelectorAll('.tunnel-card').length === 1)
  await page.getByPlaceholder('搜索名称、地址、端口').fill('')
  await page.waitForFunction(() => document.querySelectorAll('.tunnel-card').length === 2)
  const ssh = page.locator('.tunnel-card').filter({ hasText: '60022' })
  failSave = true
  await Promise.all([page.waitForResponse(r => r.url().endsWith('/api/frp') && r.request().method() === 'PUT'), ssh.getByRole('switch').click()])
  await page.waitForFunction(() => document.querySelector('.tunnel-card [role=switch]')?.getAttribute('aria-checked') === 'true')
  await page.screenshot({ path: 'ui-screenshots/frp-narrow.png', fullPage: true })
  const sizes = await page.locator('.tunnel-card').evaluateAll(els => els.map(el => ({ w: el.clientWidth, scroll: el.scrollWidth })))
  assert.ok(sizes.every(x => x.scroll <= x.w + 2), JSON.stringify(sizes))
  await page.getByRole('tab', { name: '运行日志' }).click()
  await page.getByRole('button', { name: '开始检测' }).click()
  await page.getByText('TCP 可连接', { exact: true }).waitFor()
  await page.getByPlaceholder('搜索日志').fill('test warning')
  await page.waitForFunction(() => document.querySelector('.log-view--panel')?.textContent.includes('test warning') && !document.querySelector('.log-view--panel')?.textContent.includes('login success'))
  await page.screenshot({ path: 'ui-screenshots/frp-diagnostics.png', fullPage: true })
  assert.deepEqual(errors, [])
} finally {
  await browser?.close()
  server.kill()
}
