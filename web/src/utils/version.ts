import { GITHUB_REPO } from '../constants/app'

const GITHUB_RELEASES = `https://github.com/${GITHUB_REPO}/releases`

export interface LatestRelease {
  version: string
  url: string
}

export interface ReleaseNote {
  version: string
  date: string
  url: string
  /** GitHub Release 正文，保留 Markdown。 */
  markdown: string
}

/** 非 null 时只在本地假装有这条新版本，不发布到 GitHub。 */
export const LOCAL_PREVIEW_RELEASE: ReleaseNote | null = null

export function withLocalPreview(notes: ReleaseNote[]): ReleaseNote[] {
  const preview = LOCAL_PREVIEW_RELEASE
  if (!preview) return notes
  if (notes.some((item) => item.version === preview.version)) return notes
  return [preview, ...notes]
}

/** 连不上 GitHub 时仍能展示最近五次发版说明。 */
export const FALLBACK_RELEASE_NOTES: ReleaseNote[] = [
  {
    version: '1.3.8',
    date: '2026-11-10',
    url: `${GITHUB_RELEASES}/tag/v1.3.8`,
    markdown: [
      '### 变更',
      '',
      '- DDNS 与证书申请新增华为云 DNS',
      '',
      '### 修复',
      '',
      '- 腾讯云、DNSPod 申请证书时 TXT 的 TTL 改为 600 秒',
    ].join('\n'),
  },
  {
    version: '1.3.7',
    date: '2026-11-06',
    url: `${GITHUB_RELEASES}/tag/v1.3.7`,
    markdown: [
      '### 修复',
      '',
      '- 根域名与泛域名同时申请时，同一 `_acme-challenge` 保留两条 TXT，不再互相覆盖',
      '- 反代详情流量图与最近访问留在卡片内；停用规则操作列不再透出底色',
      '- 内网穿透地址与端口同一行；状态区改为两行各两张卡片',
    ].join('\n'),
  },
  {
    version: '1.3.6',
    date: '2026-11-06',
    url: `${GITHUB_RELEASES}/tag/v1.3.6`,
    markdown: [
      '### 修复',
      '',
      '- DNSPod ACME：写入验证 TXT 前先清除同主机名下旧记录，修复传播超时',
    ].join('\n'),
  },
  {
    version: '1.3.5',
    date: '2026-11-06',
    url: `${GITHUB_RELEASES}/tag/v1.3.5`,
    markdown: [
      '### 变更',
      '',
      '- 证书映射说明按 Docker / 飞牛 Root 包 / 普通包区分',
      '- 仪表盘去掉实时流量块；访问地图可缩放平移，窄屏布局优化',
      '- 异常访问、FRP 页在较窄视口下纵向排列',
      '',
      '### 修复',
      '',
      '- HTTP 下导出备份改为直接下载',
    ].join('\n'),
  },
  {
    version: '1.3.4',
    date: '2026-11-06',
    url: `${GITHUB_RELEASES}/tag/v1.3.4`,
    markdown: [
      '### 修复',
      '',
      '- 证书 DNS-01 修复根域名与独立子域 zone（Cloudflare、腾讯云、DNSPod、阿里云）',
      '',
      '### 变更',
      '',
      '- GitHub Issue 固定反馈表单',
    ].join('\n'),
  },
  {
    version: '1.3.3',
    date: '2026-11-04',
    url: `${GITHUB_RELEASES}/tag/v1.3.3`,
    markdown: [
      '### 变更',
      '',
      '- 反代默认带上客户端 Host、Origin 和 Referer',
      '- 飞牛额外提供 root 安装包，只在 GitHub Release，不上架飞牛商店',
      '',
      '### 修复',
      '',
      '- 备份改为数据库快照；恢复先校验，再在下次启动时替换',
      '- 恢复完成后自动退出并重新登录',
    ].join('\n'),
  },
  {
    version: '1.3.2',
    date: '2026-11-02',
    url: `${GITHUB_RELEASES}/tag/v1.3.2`,
    markdown: [
      '### 变更',
      '',
      '- 仪表盘离开后保留页面，回来先显示上次结果',
      '- 首页统计合成一次请求，并缓存 15 秒',
      '- 独立 IP、拦截、疑似扫描补上较昨日',
      '',
      '### 修复',
      '',
      '- 访问日志不再被重复计入今天',
      '- 平均耗时不再被 WebSocket 长连接拉高',
    ].join('\n'),
  },
  {
    version: '1.3.1',
    date: '2026-11-02',
    url: `${GITHUB_RELEASES}/tag/v1.3.1`,
    markdown: [
      '### 新增',
      '',
      '- 访问日志增量写入 SQLite，首页展示请求、独立 IP、拦截和访问地图',
      '- 今天的 5xx、4xx、耗时和慢请求，以及问题 URL',
      '- 系统设置增加 **版本** 页，展示最近五次更新说明',
      '',
      '### 变更',
      '',
      '- 首页去掉横幅和重复统计；登录页 logo 改为小图',
    ].join('\n'),
  },
  {
    version: '1.3.0',
    date: '2026-11-02',
    url: `${GITHUB_RELEASES}/tag/v1.3.0`,
    markdown: ['### 新增', '', '- 首页动态感知、拦截记录，以及离线 IP 归属地', '- 侧栏版本号；检测到 GitHub 新版本时显示更新提示', '', '### 变更', '', '- 访问日志增加 User-Agent；构建工具链升级到 Go 1.27.3'].join('\n'),
  },
]

function parseVersion(version: string): number[] {
  const clean = version.replace(/^v/i, '').split('-')[0]
  return clean.split('.').map((part) => Number.parseInt(part, 10) || 0)
}

export function compareVersions(a: string, b: string): number {
  const left = parseVersion(a)
  const right = parseVersion(b)
  const length = Math.max(left.length, right.length)

  for (let i = 0; i < length; i += 1) {
    const diff = (left[i] ?? 0) - (right[i] ?? 0)
    if (diff !== 0) {
      return diff > 0 ? 1 : -1
    }
  }

  return 0
}

export function isNewerVersion(latest: string, current: string): boolean {
  if (!latest) {
    return false
  }
  const cur = current.replace(/^v/i, '').trim()
  if (!cur || cur === 'dev') {
    // 本地 dev 构建：与 GitHub 最新正式版对比，便于侧栏更新提示联调
    return compareVersions(latest, '0.0.0') > 0
  }
  return compareVersions(latest, cur) > 0
}

export async function fetchLatestRelease(): Promise<LatestRelease | null> {
  try {
    const response = await fetch(`https://api.github.com/repos/${GITHUB_REPO}/releases/latest`, {
      headers: { Accept: 'application/vnd.github+json' },
    })
    if (!response.ok) {
      return previewLatest()
    }

    const data = (await response.json()) as { tag_name?: string; html_url?: string }
    if (!data.tag_name || !data.html_url) {
      return previewLatest()
    }

    const remote = {
      version: data.tag_name.replace(/^v/i, ''),
      url: data.html_url,
    }
    const preview = LOCAL_PREVIEW_RELEASE
    if (preview && compareVersions(preview.version, remote.version) > 0) {
      return { version: preview.version, url: preview.url }
    }
    return remote
  } catch {
    return previewLatest()
  }
}

function previewLatest(): LatestRelease | null {
  if (!LOCAL_PREVIEW_RELEASE) return null
  return { version: LOCAL_PREVIEW_RELEASE.version, url: LOCAL_PREVIEW_RELEASE.url }
}

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function renderInline(escaped: string): string {
  return escaped
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\[([^\]]+)\]\((https?:\/\/[^)\s]+)\)/g, '<a href="$2" target="_blank" rel="noreferrer">$1</a>')
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    .replace(/__([^_]+)__/g, '<strong>$1</strong>')
}

/** 把 Release 正文渲染成预览。先转义 HTML，再还原常用 Markdown。 */
export function renderReleaseMarkdown(source: string): string {
  const lines = escapeHtml(source.replace(/\r\n/g, '\n')).split('\n')
  const out: string[] = []
  let inList = false
  let inCode = false
  const closeList = () => {
    if (!inList) return
    out.push('</ul>')
    inList = false
  }
  for (const raw of lines) {
    const line = raw.trim()
    if (line.startsWith('```')) {
      closeList()
      if (inCode) {
        out.push('</code></pre>')
        inCode = false
      } else {
        out.push('<pre><code>')
        inCode = true
      }
      continue
    }
    if (inCode) {
      out.push(`${raw}\n`)
      continue
    }
    if (!line || line.startsWith('&lt;!--')) {
      closeList()
      continue
    }
    const heading = line.match(/^(#{1,6})\s+(.+)/)
    if (heading) {
      closeList()
      const level = Math.min(heading[1].length, 4)
      out.push(`<h${level}>${renderInline(heading[2])}</h${level}>`)
      continue
    }
    const item = line.match(/^[-*]\s+(.+)/)
    if (item) {
      if (!inList) {
        out.push('<ul>')
        inList = true
      }
      out.push(`<li>${renderInline(item[1])}</li>`)
      continue
    }
    closeList()
    out.push(`<p>${renderInline(line)}</p>`)
  }
  closeList()
  if (inCode) out.push('</code></pre>')
  return out.join('') || '<p>详见 GitHub Release 说明</p>'
}

/**
 * 默认取最新 limit 条。当前版本比这更老时，把它补在列表末尾，避免「当前版本」标识被截掉。
 */
export function pickReleaseNotes(all: ReleaseNote[], current: string, limit = 5): ReleaseNote[] {
  const top = all.slice(0, limit)
  const cur = current.replace(/^v/i, '').trim()
  if (!/^\d+\.\d+/.test(cur)) return top
  if (top.some((item) => item.version.replace(/^v/i, '') === cur)) return top
  const found = all.find((item) => item.version.replace(/^v/i, '') === cur)
  return found ? [...top, found] : top
}

/** 最近若干次发布，供版本页挑选。失败时返回 null，由页面改用本地说明。 */
export async function fetchRecentReleases(limit = 20): Promise<ReleaseNote[] | null> {
  try {
    const response = await fetch(`https://api.github.com/repos/${GITHUB_REPO}/releases?per_page=${limit}`, {
      headers: { Accept: 'application/vnd.github+json' },
    })
    if (!response.ok) return null
    const data = (await response.json()) as {
      draft?: boolean
      tag_name?: string
      html_url?: string
      body?: string
      published_at?: string
    }[]
    const notes = data
      .filter((item) => !item.draft && item.tag_name && item.html_url)
      .map((item) => ({
        version: item.tag_name!.replace(/^v/i, ''),
        date: (item.published_at ?? '').slice(0, 10),
        url: item.html_url!,
        markdown: (item.body ?? '').trim() || '详见 GitHub Release 说明',
      }))
    return notes.length ? notes : null
  } catch {
    return null
  }
}
