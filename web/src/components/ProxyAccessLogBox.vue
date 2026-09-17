<template>
  <div
    ref="root"
    class="proxy-log-box"
    :class="{
      'proxy-log-box--embedded': embedded,
      'proxy-log-box--fullscreen': fullscreen,
      'is-empty': lines.length === 0,
    }"
    @scroll="onScroll"
  >
    <template v-for="(item, i) in parsedLines" :key="i">
      <div
        v-if="item.parsed"
        class="proxy-log-entry"
        :class="logEntryStatusClass(item.parsed.status)"
      >
        <time class="proxy-log-entry__time">{{ item.parsed.time }}</time>
        <span class="proxy-log-entry__method" :class="`is-${item.parsed.method.toLowerCase()}`">
          {{ item.parsed.method }}
        </span>
        <span class="proxy-log-entry__path" :title="item.parsed.path">{{ item.parsed.path }}</span>
        <span class="proxy-log-entry__status">{{ item.parsed.status }}</span>
        <span class="proxy-log-entry__ms">{{ item.parsed.ms }}ms</span>
        <span class="proxy-log-entry__client mono">{{ item.parsed.client }}</span>
        <span class="proxy-log-entry__upstream mono" :title="item.parsed.upstream">
          → {{ item.parsed.upstream }}
        </span>
      </div>
      <div v-else class="proxy-log-line proxy-log-line--raw">{{ item.raw }}</div>
    </template>
    <div v-if="lines.length === 0" class="proxy-log-empty">暂无记录，通过反代域名访问后会显示在这里。</div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

type AccessLogEntry = {
  time: string
  host: string
  method: string
  path: string
  status: number
  ms: string
  client: string
  upstream: string
}

const props = defineProps<{
  lines: string[]
  embedded?: boolean
  fullscreen?: boolean
}>()

const emit = defineEmits<{
  scroll: [el: HTMLElement]
}>()

const root = ref<HTMLElement | null>(null)

const accessLineReWithPort =
  /^(\S+)\s+(\S+)\s+(\d+)\s+(\S+)\s+(\S+)\s+(\d{3})\s+([\d.]+)\s+(\S+)\s+(\S+)(?:\s+(\d+)\s+(\d+))?$/
const accessLineReLegacy =
  /^(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\d{3})\s+([\d.]+)\s+(\S+)\s+(\S+)(?:\s+(\d+)\s+(\d+))?$/

function parseAccessLog(line: string): AccessLogEntry | null {
  let m = line.match(accessLineReWithPort)
  if (m) {
    const [, time, , , method, path, status, rt, client, upstream] = m
    return {
      time: time.replace('T', ' ').replace(/([+-]\d{2}:\d{2}|Z)$/, ''),
      host: '',
      method,
      path,
      status: Number(status),
      ms: (parseFloat(rt) * 1000).toFixed(1),
      client,
      upstream,
    }
  }
  m = line.match(accessLineReLegacy)
  if (!m) return null
  const [, time, host, method, path, status, rt, client, upstream] = m
  return {
    time: time.replace('T', ' ').replace(/([+-]\d{2}:\d{2}|Z)$/, ''),
    host,
    method,
    path,
    status: Number(status),
    ms: (parseFloat(rt) * 1000).toFixed(1),
    client,
    upstream,
  }
}

const parsedLines = computed(() =>
  props.lines.map((raw) => ({ raw, parsed: parseAccessLog(raw) })),
)

function logEntryStatusClass(status: number): string {
  if (status >= 500) return 'is-error'
  if (status >= 400) return 'is-warn'
  return ''
}

function onScroll() {
  if (root.value) emit('scroll', root.value)
}

function scrollToBottom() {
  if (root.value) root.value.scrollTop = root.value.scrollHeight
}

defineExpose({ scrollToBottom })
</script>

<style scoped>
.proxy-log-box {
  background: #0f172a;
  color: #e2e8f0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: var(--fonu-space-3);
  font-family: var(--fonu-mono);
  font-size: 12px;
  line-height: 1.7;
  border-radius: var(--fonu-radius-sm);
  scrollbar-width: thin;
  scrollbar-color: rgba(148, 163, 184, 0.2) transparent;
}

.proxy-log-box:hover {
  scrollbar-color: rgba(148, 163, 184, 0.55) rgba(15, 23, 42, 0.35);
}

.proxy-log-box::-webkit-scrollbar {
  width: 7px;
}

.proxy-log-box::-webkit-scrollbar-track {
  background: transparent;
}

.proxy-log-box::-webkit-scrollbar-thumb {
  background: rgba(148, 163, 184, 0.18);
  border-radius: 4px;
}

.proxy-log-box:hover::-webkit-scrollbar-thumb {
  background: rgba(148, 163, 184, 0.55);
}

.proxy-log-box--embedded {
  flex: 1 1 0;
  min-height: 0;
  overflow-y: auto;
}

.proxy-log-box--embedded.is-empty {
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.proxy-log-box--fullscreen {
  flex: 1;
  min-height: 0;
  border-radius: var(--fonu-radius);
}

.proxy-log-entry {
  display: grid;
  grid-template-columns: 132px 52px minmax(0, 1fr) 44px 56px 108px minmax(80px, auto);
  gap: 6px 10px;
  align-items: center;
  padding: 8px 10px;
  border-radius: 6px;
  font-size: 12px;
  line-height: 1.45;
  border: 1px solid transparent;
}

.proxy-log-entry + .proxy-log-entry {
  margin-top: 4px;
}

.proxy-log-entry:hover {
  background: rgba(255, 255, 255, 0.04);
  border-color: rgba(148, 163, 184, 0.12);
}

.proxy-log-entry__time {
  color: #94a3b8;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.proxy-log-entry__method {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.02em;
  background: rgba(59, 130, 246, 0.18);
  color: #93c5fd;
}

.proxy-log-entry__method.is-get { background: rgba(59, 130, 246, 0.18); color: #93c5fd; }
.proxy-log-entry__method.is-post { background: rgba(16, 185, 129, 0.18); color: #6ee7b7; }
.proxy-log-entry__method.is-put,
.proxy-log-entry__method.is-patch { background: rgba(245, 158, 11, 0.18); color: #fcd34d; }
.proxy-log-entry__method.is-delete { background: rgba(239, 68, 68, 0.18); color: #fca5a5; }

.proxy-log-entry__path {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #e2e8f0;
}

.proxy-log-entry__status {
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  text-align: right;
  color: #86efac;
}

.proxy-log-entry.is-warn .proxy-log-entry__status { color: #fbbf24; }
.proxy-log-entry.is-error .proxy-log-entry__status { color: #f87171; }

.proxy-log-entry__ms {
  color: #94a3b8;
  font-variant-numeric: tabular-nums;
  text-align: right;
  white-space: nowrap;
}

.proxy-log-entry__client {
  color: #cbd5e1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.proxy-log-entry__upstream {
  color: #64748b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 11px;
}

.proxy-log-line {
  white-space: pre-wrap;
  word-break: break-all;
  padding: 6px 10px;
  color: #cbd5e1;
}

.proxy-log-line--raw + .proxy-log-entry,
.proxy-log-entry + .proxy-log-line--raw {
  margin-top: 4px;
}

.proxy-log-empty {
  padding: var(--fonu-space-5);
  text-align: center;
  color: #94a3b8;
}

.mono { font-family: var(--fonu-mono); }

@media (max-width: 767px) {
  .proxy-log-entry {
    grid-template-columns: 1fr 1fr;
    gap: 4px 8px;
  }
  .proxy-log-entry__path { grid-column: 1 / -1; }
  .proxy-log-entry__upstream { grid-column: 1 / -1; }
}
</style>
