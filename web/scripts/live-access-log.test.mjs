import assert from 'node:assert/strict'
import test from 'node:test'
import { parseAccessLog } from '../src/utils/liveAccessLog.ts'
test('parses current Nginx format including request length, bytes and user agent',()=>{
 const r=parseAccessLog('2026-11-26T14:09:31+08:00 speedtest.muxui.com 8009 GET /favicon.ico 200 0.002 192.168.8.1 192.168.8.3:8005 224 4555 "Go-http-client/1.1"')
 assert.equal(r.status,200);assert.equal(r.path,'/favicon.ico');assert.equal(r.ms,'2.0');assert.equal(r.client,'192.168.8.1')
})
test('parses legacy format and quoted user agent with spaces',()=>{
 assert.equal(parseAccessLog('2026-11-26T14:09:31+08:00 example.com GET / 301 0.000 192.0.2.1 -').status,301)
 assert.equal(parseAccessLog('2026-11-26T14:09:31+08:00 example.com 443 GET / 403 0.001 192.0.2.1 - 123 456 "Mozilla/5.0 (Test browser)"').status,403)
 assert.equal(parseAccessLog('unrecognized line'),null)
})
