export type AccessLogEntry = {
  time: string
  host: string
  method: string
  path: string
  status: number
  ms: string
  client: string
  upstream: string
}

const accessLineReWithPort =
  /^(\S+)\s+(\S+)\s+(\d+)\s+(\S+)\s+(\S+)\s+(\d{3})\s+([\d.]+)\s+(\S+)\s+(\S+)(?:\s+(\d+|-)\s+(\d+|-))?(?:\s+"(?:[^"\\]|\\.)*")?$/
const accessLineReLegacy =
  /^(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\d{3})\s+([\d.]+)\s+(\S+)\s+(\S+)(?:\s+(\d+|-)\s+(\d+|-))?(?:\s+"(?:[^"\\]|\\.)*")?$/

export function parseAccessLog(line: string): AccessLogEntry | null {
  let m = line.match(accessLineReWithPort)
  if (m) {
    const [, time, host, , method, path, status, rt, client, upstream] = m
    return buildEntry(time, host, method, path, status, rt, client, upstream)
  }
  m = line.match(accessLineReLegacy)
  if (!m) return null
  const [, time, host, method, path, status, rt, client, upstream] = m
  return buildEntry(time, host, method, path, status, rt, client, upstream)
}

function buildEntry(
  time: string,
  host: string,
  method: string,
  path: string,
  status: string,
  rt: string,
  client: string,
  upstream: string,
): AccessLogEntry {
  return {
    time,
    host,
    method,
    path,
    status: Number(status),
    ms: (parseFloat(rt) * 1000).toFixed(1),
    client,
    upstream,
  }
}

