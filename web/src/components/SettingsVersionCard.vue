<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '../api/client'
import { GITHUB_RELEASES_URL } from '../constants/app'
import {
  FALLBACK_RELEASE_NOTES,
  fetchRecentReleases,
  isNewerVersion,
  LOCAL_PREVIEW_RELEASE,
  pickReleaseNotes,
  renderReleaseMarkdown,
  withLocalPreview,
  type ReleaseNote,
} from '../utils/version'
import FonuCard from './FonuCard.vue'

const SHOW_COUNT = 5

const currentVersion = ref(__APP_VERSION__)
const notes = ref<ReleaseNote[]>(
  pickReleaseNotes(withLocalPreview(FALLBACK_RELEASE_NOTES), __APP_VERSION__, SHOW_COUNT),
)
const fromGitHub = ref(false)

function sameVersion(a: string, b: string): boolean {
  return a.replace(/^v/i, '').trim() === b.replace(/^v/i, '').trim()
}

function isReleaseVersion(version: string): boolean {
  return /^\d+\.\d+/.test(version.replace(/^v/i, '').trim())
}

const latest = computed(() => notes.value[0])
const hasUpdate = computed(
  () => !!latest.value && isNewerVersion(latest.value.version, currentVersion.value),
)
const currentIsOlder = computed(
  () => notes.value.length > SHOW_COUNT && sameVersion(notes.value[notes.value.length - 1]?.version ?? '', currentVersion.value),
)

onMounted(async () => {
  try {
    const { version } = await api.getVersion()
    if (version && isReleaseVersion(version)) currentVersion.value = version
  } catch {
    /* 使用构建期版本 */
  }
  const remote = await fetchRecentReleases(20)
  const source = withLocalPreview(remote?.length ? remote : FALLBACK_RELEASE_NOTES)
  fromGitHub.value = !!remote?.length
  notes.value = pickReleaseNotes(source, currentVersion.value, SHOW_COUNT)
})
</script>

<template>
  <FonuCard title="更新日志" subtitle="最近五次更新。更早的到 GitHub 看。" class="version-card">
    <div class="version-card__bar">
      <div class="version-card__current">
        <span class="version-card__current-label">当前版本</span>
        <span class="version-card__current-value">v{{ currentVersion }}</span>
        <span v-if="hasUpdate && latest" class="version-card__badge version-card__badge--update">有新版本 v{{ latest.version }}</span>
        <span v-else class="version-card__badge">已是最新</span>
      </div>
      <a v-if="hasUpdate && latest" class="version-card__update-link" :href="latest.url" target="_blank" rel="noreferrer">
        前往 GitHub 查看
      </a>
    </div>

    <ol class="version-card__list">
      <li v-for="item in notes" :key="item.version" class="version-card__item">
        <div class="version-card__item-head">
          <a class="version-card__tag" :href="item.url" target="_blank" rel="noreferrer">v{{ item.version }}</a>
          <span v-if="item.date" class="version-card__date">{{ item.date }}</span>
          <span v-if="sameVersion(item.version, currentVersion)" class="version-card__badge">当前版本</span>
        </div>
        <div class="version-card__md" v-html="renderReleaseMarkdown(item.markdown)" />
      </li>
    </ol>

    <p v-if="currentIsOlder" class="version-card__hint">当前版本早于最近五次，已附在列表末尾。</p>
    <p v-if="LOCAL_PREVIEW_RELEASE" class="version-card__hint">v{{ LOCAL_PREVIEW_RELEASE.version }} 只在本地预览，没有发布到 GitHub。</p>
    <p class="version-card__hint">{{ fromGitHub ? '其余更新说明来自 GitHub Releases。' : '暂时连不上 GitHub，下面是本地保存的说明。' }}</p>
    <a class="version-card__more" :href="GITHUB_RELEASES_URL" target="_blank" rel="noreferrer">
      更早的更新在 GitHub Releases 查看
    </a>
  </FonuCard>
</template>

<style scoped>
.version-card__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
  min-width: 0;
}

.version-card__current {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.version-card__update-link {
  flex-shrink: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--fonu-brand);
  text-decoration: none;
  white-space: nowrap;
}

.version-card__update-link:hover {
  text-decoration: underline;
}

.version-card__current-label {
  font-size: 13px;
  color: var(--fonu-text-muted);
}

.version-card__current-value {
  font-size: 20px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: var(--fonu-text);
}

.version-card__list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.version-card__item {
  padding-top: 14px;
  border-top: 1px solid var(--fonu-border);
}

.version-card__item-head {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.version-card__tag {
  font-weight: 650;
  color: var(--fonu-text);
  text-decoration: none;
}

.version-card__tag:hover {
  color: var(--fonu-brand);
}

.version-card__date {
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.version-card__badge {
  font-size: 11px;
  line-height: 1;
  padding: 3px 8px;
  border-radius: 999px;
  color: var(--fonu-brand);
  background: var(--fonu-brand-soft, rgba(16, 185, 129, 0.12));
  white-space: nowrap;
}

.version-card__badge--update {
  color: #e11d48;
  background: color-mix(in srgb, #ef4444 12%, transparent);
}

.version-card__md {
  margin-top: 8px;
  color: var(--fonu-text-secondary);
  font-size: 13px;
  line-height: 1.65;
}

.version-card__md :deep(h1),
.version-card__md :deep(h2),
.version-card__md :deep(h3),
.version-card__md :deep(h4) {
  margin: 10px 0 4px;
  font-size: 13px;
  font-weight: 650;
  color: var(--fonu-text);
}

.version-card__md :deep(p) {
  margin: 6px 0;
}

.version-card__md :deep(ul) {
  margin: 6px 0;
  padding-left: 18px;
}

.version-card__md :deep(li + li) {
  margin-top: 4px;
}

.version-card__md :deep(code) {
  padding: 0 4px;
  border-radius: 4px;
  background: color-mix(in srgb, var(--fonu-text) 6%, transparent);
  font-family: var(--fonu-mono, ui-monospace, monospace);
  font-size: 12px;
}

.version-card__md :deep(pre) {
  margin: 8px 0;
  padding: 10px 12px;
  overflow: auto;
  border-radius: 8px;
  background: color-mix(in srgb, var(--fonu-text) 6%, transparent);
}

.version-card__md :deep(pre code) {
  padding: 0;
  background: transparent;
}

.version-card__md :deep(a) {
  color: var(--fonu-brand);
  text-decoration: none;
}

.version-card__md :deep(a:hover) {
  text-decoration: underline;
}

.version-card__hint {
  margin: 14px 0 0;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.version-card__more {
  display: inline-block;
  margin-top: 16px;
  font-size: 13px;
  color: var(--fonu-brand);
  text-decoration: none;
}

.version-card__more:hover {
  text-decoration: underline;
}
</style>
