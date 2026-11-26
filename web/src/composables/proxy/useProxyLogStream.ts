import { onMounted, onUnmounted, onActivated, onDeactivated, ref } from 'vue'

type StreamState = 'idle' | 'connecting' | 'open' | 'retrying' | 'paused'

/** One stream per visible rule. Reconnect replaces the retained tail rather
 * than appending a duplicate snapshot. Late events are ignored by identity. */
export function useProxyLogStream() {
  const lines = ref<string[]>([])
  const state = ref<StreamState>('idle')
  let ruleId: number | undefined
  let source: EventSource | undefined
  let timer: ReturnType<typeof setTimeout> | undefined
  let attempts = 0
  let active = true
  function disconnect() {
    if (timer !== undefined) clearTimeout(timer)
    timer = undefined
    const old = source
    source = undefined
    old?.close()
  }
  function connect() {
    disconnect()
    if (ruleId === undefined) { state.value = 'idle'; return }
    if (!active || document.hidden) { state.value = 'paused'; return }
    state.value = attempts ? 'retrying' : 'connecting'
    const current = new EventSource(`/api/proxies/${ruleId}/logs/stream?tail=100`, { withCredentials: true })
    source = current
    current.onopen = () => {
      if (source !== current) return
      lines.value = []
      state.value = 'open'
    }
    current.addEventListener('log', (event: MessageEvent<string>) => {
      if (source !== current) return
      attempts = 0
      lines.value.push(event.data)
      if (lines.value.length > 500) lines.value = lines.value.slice(-400)
    })
    current.onerror = () => {
      if (source !== current) return
      disconnect()
      if (ruleId === undefined || !active || document.hidden) { state.value = 'paused'; return }
      state.value = 'retrying'
      const delay = Math.min(30000, 1000 * 2 ** Math.min(attempts++, 5))
      timer = setTimeout(connect, delay)
    }
  }
  function start(id: number) {
    if (ruleId === id && source) return
    ruleId = id
    attempts = 0
    connect()
  }
  function stop() {
    ruleId = undefined
    attempts = 0
    disconnect()
    state.value = 'idle'
  }
  function visibility() {
    if (ruleId === undefined) return
    if (document.hidden || !active) { disconnect(); state.value = 'paused' }
    else if (!source && timer === undefined) connect()
  }
  onMounted(() => document.addEventListener('visibilitychange', visibility))
  onActivated(() => { active = true; visibility() })
  onDeactivated(() => { active = false; visibility() })
  onUnmounted(() => { stop(); document.removeEventListener('visibilitychange', visibility) })
  return { lines, state, start, stop, clear: () => { lines.value = [] } }
}
