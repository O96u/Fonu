const DDNS_PROVIDER_LABELS: Record<string, string> = {
  cloudflare: 'Cloudflare',
  dnspod: 'DNSPod',
  alidns: '阿里云 DNS',
  tencentcloud: '腾讯云 DNS',
  volcengine: '火山引擎 DNS',
  dnshe: 'DNSHE',
}

export function ddnsProviderLabel(provider: string): string {
  const key = provider.trim().toLowerCase()
  return DDNS_PROVIDER_LABELS[key] ?? provider
}
