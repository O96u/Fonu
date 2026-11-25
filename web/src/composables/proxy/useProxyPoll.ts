// Serialize polling and stop scheduling when the owning view is closed.
export function createProxyPoll(task: () => Promise<void>, interval: number) {
  let timer: ReturnType<typeof setTimeout> | undefined
  let generation = 0
  let pending = false
  function stop() {
    generation++
    if (timer !== undefined) clearTimeout(timer)
    timer = undefined
  }
  function start(immediate = false) {
    stop()
    const token = generation
    const tick = async () => {
      if (token !== generation) return
      if (!document.hidden && !pending) {
        pending = true
        try { await task() } catch { /* polling is best effort */ }
        finally { pending = false }
      }
      if (token === generation) timer = setTimeout(tick, interval)
    }
    if (immediate) void tick()
    else timer = setTimeout(tick, interval)
  }
  return { start, stop }
}
