<template>
<div class="proxy-detail__pane proxy-detail__pane--overview">
            <div class="overview-strip">
              <div class="overview-strip__item">
                <span class="overview-strip__label">域名</span>
                <span class="overview-strip__value" :title="hosts.join('、')">
                  {{ hosts.join('、') }}
                </span>
              </div>
              <div class="overview-strip__item">
                <span class="overview-strip__label">监听</span>
                <span class="overview-strip__value">{{ listen }}</span>
              </div>
              <div class="overview-strip__item overview-strip__item--wide">
                <span class="overview-strip__label">目标</span>
                <span class="overview-strip__value mono" :title="selectedRule.upstream">{{ selectedRule.upstream }}</span>
              </div>
              <div class="overview-strip__item">
                <span class="overview-strip__label">协议</span>
                <span class="overview-strip__value">
                  <n-tag size="small" :type="selectedRule.https_enabled ? 'success' : 'default'" :bordered="false" round>
                    {{ selectedRule.https_enabled ? 'HTTPS' : 'HTTP' }}
                  </n-tag>
                </span>
              </div>
            </div>

            <div class="overview-security">
              <span class="overview-security__label">安全策略</span>
              <div v-if="selectedSecurityFeatures.length" class="overview-security__tags">
                <n-tag
                  v-for="tag in selectedSecurityFeatures"
                  :key="tag"
                  size="small"
                  round
                  :bordered="false"
                >
                  {{ tag }}
                </n-tag>
              </div>
              <span v-else class="overview-security__empty">未启用</span>
            </div>

            <div class="overview-main">
              <section class="overview-card overview-card--chart">
                <div class="overview-card__head">
                  <h4>实时流量 <span class="overview-card__sub">最近 5 分钟</span></h4>
                  <span class="live-badge">{{trafficError?'采样失败':sampledAt?'实时':'未采样'}}</span>
                </div>
                <div class="traffic-live-legend traffic-live-legend--compact">
                  <span class="traffic-live-legend__item traffic-live-legend__item--up">
                    <span class="traffic-live-legend__dot" />
                    上传 {{ selectedTraffic?formatRate(selectedTraffic.upload_rate):"—" }}
                  </span>
                  <span class="traffic-live-legend__item traffic-live-legend__item--down">
                    <span class="traffic-live-legend__dot" />
                    下载 {{ selectedTraffic?formatRate(selectedTraffic.download_rate):"—" }}
                  </span>
                </div>
                <small class="overview-card__sub">仅显示本次详情打开后的采样；空白表示未采样，速率来自已完成访问日志。</small>
                <MiniTrafficChart
                  class="overview-chart"
                  :labels="trafficChart.labels"
                  :upload="trafficChart.upload"
                  :download="trafficChart.download"
                />
              </section>

              <aside class="overview-side">
                <section class="overview-metrics">
                  <div class="overview-metric">
                    <div class="overview-metric__label">总上传</div>
                    <div class="overview-metric__value">{{ selectedTraffic?formatBytes(selectedTraffic.upload_total):"—" }}</div>
                  </div>
                  <div class="overview-metric">
                    <div class="overview-metric__label">总下载</div>
                    <div class="overview-metric__value">{{ selectedTraffic?formatBytes(selectedTraffic.download_total):"—" }}</div>
                  </div>
                  <div class="overview-metric">
                    <div class="overview-metric__label">当前上传</div>
                    <div class="overview-metric__value">{{ selectedTraffic?formatRate(selectedTraffic.upload_rate):"—" }}</div>
                  </div>
                  <div class="overview-metric">
                    <div class="overview-metric__label">当前下载</div>
                    <div class="overview-metric__value">{{ selectedTraffic?formatRate(selectedTraffic.download_rate):"—" }}</div>
                  </div>
                </section>

                <section class="overview-card overview-card--clients">
                  <div class="overview-card__head">
                    <h4>
                      最近访问
                      <span class="overview-card__sub">65 秒内 · {{ clientRows.length }} 个</span>
                    </h4>
                    <n-button size="tiny" quaternary :loading="clientsLoading" @click="emit('refreshClients')">刷新</n-button>
                  </div>
                  <div v-if="clientsError" class="overview-empty">{{clientsError}} · {{clientRows.length?'保留上次数据':'暂无数据'}}</div>
                  <div class="overview-client-list">
                    <template v-if="clientRows.length > 0">
                      <div v-for="row in pagedClients" :key="row.ip" class="overview-client-row">
                        <span class="mono">{{ row.ip }}</span>
                        <span class="overview-client-row__time">{{ formatRelativeTime(row.last_seen) }}</span>
                      </div>
                    </template>
                    <p v-else class="overview-empty">{{clientsLoading?'读取中…':clientsError?'未获取到数据':'暂无访问记录'}}</p>
                  </div>
                  <n-pagination v-if="clientRows.length>10" v-model:page="page" :page-count="pageCount" :page-slot="5" size="small"/>
                  <small class="overview-card__sub">近期 IP 不是在线人数或 TCP 连接数。</small>
                </section>
              </aside>
            </div>
        </div>
</template>
<script setup lang="ts">
import {computed,ref,watch} from 'vue'
import {NTag,NButton,NPagination} from 'naive-ui'
import MiniTrafficChart from '../MiniTrafficChart.vue'
import type {ProxyRule,ProxyTraffic,ProxyClientConn} from '../../api/types'
import {formatBytes,formatRate,formatRelativeTime} from '../../utils/format'
const props=defineProps<{
 selectedRule:ProxyRule
 hosts:string[]
 listen:string
 selectedSecurityFeatures:string[]
 selectedTraffic?:ProxyTraffic
 trafficChart:{labels:string[];upload:(number|null)[];download:(number|null)[]}
 clientsError:string
 trafficError:string
 sampledAt:string
 clientRows:ProxyClientConn[]
 clientsLoading:boolean
}>()
const page=ref(1)
const pageCount=computed(()=>Math.max(1,Math.ceil(props.clientRows.length/10)))
const pagedClients=computed(()=>props.clientRows.slice((page.value-1)*10,page.value*10))
watch(pageCount,n=>page.value=Math.min(page.value,n))
watch(()=>props.selectedRule.id,()=>page.value=1)
const emit=defineEmits<{refreshClients:[]}>()
</script>
<style scoped>
.proxy-detail__pane {
  width: 100%;
  box-sizing: border-box;
  padding: var(--fonu-space-4) var(--fonu-space-5) var(--fonu-space-5);
}

.proxy-detail__pane--overview {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-4);
}

.overview-strip {
  display: grid;
  grid-template-columns: minmax(0, 1.1fr) minmax(0, 0.8fr) minmax(0, 1.4fr) auto;
  gap: 12px 16px;
  padding: 12px 14px;
  background: var(--fonu-bg);
  border: 1px solid var(--fonu-border);
  border-radius: 10px;
  flex-shrink: 0;
}

.overview-security {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 12px;
  padding: 10px 14px;
  background: var(--fonu-bg);
  border: 1px solid var(--fonu-border);
  border-radius: 10px;
  flex-shrink: 0;
}

.overview-security__label {
  font-size: 11px;
  color: var(--fonu-text-muted);
  letter-spacing: 0.02em;
  flex-shrink: 0;
}

.overview-security__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
}

.overview-security__empty {
  font-size: 13px;
  color: var(--fonu-text-muted);
}

.overview-strip__item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.overview-strip__label {
  font-size: 11px;
  color: var(--fonu-text-muted);
  letter-spacing: 0.02em;
}

.overview-strip__value {
  font-size: 13px;
  font-weight: 500;
  color: var(--fonu-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.overview-main {
  display: grid;
  grid-template-columns: minmax(0, 1.45fr) minmax(0, 1fr);
  gap: var(--fonu-space-4);
  flex: 0 0 auto;
  align-items: stretch;
}

.overview-side {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
  min-height: 100%;
}

.overview-card {
  border: 1px solid var(--fonu-border);
  border-radius: 10px;
  background: var(--fonu-surface);
  padding: 14px 16px;
}

.overview-card__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-3);
  margin-bottom: 10px;
}

.overview-card__head h4 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}

.overview-card__sub {
  margin-left: 6px;
  font-size: 12px;
  font-weight: 400;
  color: var(--fonu-text-muted);
}

.overview-card--chart {
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.overview-chart {
  flex: none;
  height: 200px;
  overflow: hidden;
}

.overview-metrics {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  grid-template-rows: 1fr 1fr;
  gap: 10px;
  flex-shrink: 0;
}

.overview-metric {
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 12px 14px;
  border-radius: 10px;
  background: var(--fonu-bg);
  border: 1px solid var(--fonu-border);
}

.overview-metric__label {
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.overview-metric__value {
  margin-top: 4px;
  font-size: 18px;
  font-weight: 600;
  line-height: 1.2;
  color: var(--fonu-text);
}

.overview-card--clients {
  flex: 1 1 auto;
  min-height: 112px;
  display: flex;
  flex-direction: column;
  margin: 0;
  padding-bottom: 12px;
  overflow: hidden;
}

.overview-client-list {
  flex: 1 1 auto;
  min-height: 48px;
  overflow-y: auto;
  margin: 0 -2px;
  padding: 0 2px;
  scrollbar-width: thin;
  scrollbar-color: rgba(148, 163, 184, 0.35) transparent;
}

.overview-client-list:hover {
  scrollbar-color: rgba(148, 163, 184, 0.55) var(--fonu-bg-muted);
}

.overview-client-list::-webkit-scrollbar {
  width: 6px;
}

.overview-client-list::-webkit-scrollbar-thumb {
  background: rgba(148, 163, 184, 0.25);
  border-radius: 4px;
}

.overview-client-list:hover::-webkit-scrollbar-thumb {
  background: rgba(148, 163, 184, 0.5);
}

.overview-client-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-3);
  padding: 7px 10px;
  border-radius: 8px;
  font-size: 12px;
}

.overview-client-row + .overview-client-row {
  margin-top: 2px;
}

.overview-client-row:hover {
  background: var(--fonu-bg);
}

.overview-client-row__time {
  flex-shrink: 0;
  color: var(--fonu-text-muted);
  font-size: 11px;
}

.overview-empty {
  margin: 0;
  flex: 1;
  min-height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  color: var(--fonu-text-muted);
}

.traffic-live-legend--compact {
  margin-bottom: 6px;
}

.proxy-detail__pane--logs,
.proxy-detail__pane--nginx {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding-top: var(--fonu-space-3);
}

.live-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: var(--fonu-success);
  font-weight: 500;
}

.live-badge__dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--fonu-success);
  animation: live-pulse 1.5s ease-in-out infinite;
}

.traffic-live-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  margin-bottom: 8px;
  font-size: 12px;
  color: var(--fonu-text-secondary);
}

.traffic-live-legend__item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.traffic-live-legend__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.traffic-live-legend__item--up .traffic-live-legend__dot {
  background: var(--fonu-brand);
}

.traffic-live-legend__item--down .traffic-live-legend__dot {
  background: var(--fonu-chart-secondary);
}

.mono { font-family: var(--fonu-mono); }
@media(max-width:767px){
.overview-strip{grid-template-columns:1fr 1fr}
.overview-main{grid-template-columns:1fr}
.overview-card--clients{min-height:100px}
}
.proxy-detail__pane--overview{overflow:visible;flex:none;padding:14px;gap:12px}.overview-strip{grid-template-columns:repeat(2,minmax(0,1fr));text-align:left}.overview-strip__value{white-space:normal;overflow-wrap:anywhere}.overview-main{grid-template-columns:1fr;min-height:0}.overview-side{display:grid;gap:10px;min-height:0}.overview-client-list{overflow:visible;max-height:none}.overview-card--clients{min-height:0;overflow:visible}.overview-chart{height:180px}
</style>
