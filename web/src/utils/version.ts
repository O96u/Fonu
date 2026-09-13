import { GITHUB_REPO } from '../constants/app'

export interface LatestRelease {
  version: string
  url: string
}

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
  if (!latest || !current || current === 'dev') {
    return false
  }
  return compareVersions(latest, current) > 0
}

export async function fetchLatestRelease(): Promise<LatestRelease | null> {
  try {
    const response = await fetch(`https://api.github.com/repos/${GITHUB_REPO}/releases/latest`, {
      headers: { Accept: 'application/vnd.github+json' },
    })
    if (!response.ok) {
      return null
    }

    const data = (await response.json()) as { tag_name?: string; html_url?: string }
    if (!data.tag_name || !data.html_url) {
      return null
    }

    return {
      version: data.tag_name.replace(/^v/i, ''),
      url: data.html_url,
    }
  } catch {
    return null
  }
}
