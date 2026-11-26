<template>
  <n-data-table
    class="proxy-table"
    :class="{'proxy-table--nested':nested,'proxy-table--sortable':sortable,'proxy-table--compact':compact}"
    :columns="columns"
    :data="data"
    :bordered="false"
    size="small"
    :scroll-x="scrollX"
    :row-key="(r:ProxyRule)=>r.id"
    :row-props="rowProps"
  />
</template>
<script setup lang="ts">
import {NDataTable,type DataTableColumns} from 'naive-ui'
import type {ProxyRule} from '../../api/types'
defineProps<{nested?:boolean;compact?:boolean;sortable:boolean;columns:DataTableColumns<ProxyRule>;data:ProxyRule[];scrollX?:number;rowProps:(r:ProxyRule)=>{class:string;'data-rule-id':string}}>()
</script>
<style scoped>
.proxy-table--nested {
  width: 100%;
}

.proxy-table--nested :deep(.n-data-table-wrapper) {
  padding: 0;
}

.proxy-table--nested :deep(.n-data-table-base-table) {
  width: 100%;
  table-layout: fixed;
}

.proxy-table--nested :deep(.n-data-table-th) {
  background: var(--fonu-bg, #f8fafc);
  font-size: 12px;
  color: var(--fonu-text-secondary);
}

.proxy-table--nested :deep(.n-data-table-td) {
  padding-top: 10px;
  padding-bottom: 10px;
}

.proxy-table--nested :deep(.domain-cell__main) {
  font-size: 13px;
}

.proxy-table--nested :deep(.domain-cell__link) {
  font-size: 13px;
  font-weight: 500;
  flex: 0 1 auto;
  max-width: calc(100% - 52px);
}

.proxy-table--nested :deep(.domain-cell__link-row) {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
}

.proxy-table--nested :deep(.domain-cell--compact) {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-height: 0;
}

.proxy-table--nested :deep(.domain-cell__main--one-line) {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.35;
}

.proxy-table--nested :deep(.domain-cell__hosts-line) {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  line-height: 1.35;
}

.proxy-table--nested :deep(.domain-cell--hosts .domain-cell__links) {
  margin-top: 0;
}

.proxy-table--nested :deep(.domain-cell__fallback-host) {
  font-size: 12px;
}

.proxy-table--sortable :deep(.proxy-drag-handle:active) {
  cursor: grabbing;
}

.proxy-table-wrap {
  width: 100%;
  overflow-x: auto;
}

.proxy-table { width: 100%; }

.proxy-table :deep(.proxy-row--active td) {
  background: var(--fonu-bg-muted);
}

.proxy-table :deep(.proxy-row--disabled td) {
  background: var(--fonu-surface, #fff);
}

.proxy-table :deep(.n-data-table-td--fixed-right),
.proxy-table :deep(.n-data-table-td--fixed-left) {
  background: var(--fonu-surface, #fff);
}

.proxy-table :deep(.n-data-table-th--fixed-right),
.proxy-table :deep(.n-data-table-th--fixed-left) {
  background: var(--fonu-bg, #f8fafc);
}

.proxy-table :deep(.n-data-table-tr:hover .n-data-table-td--fixed-right),
.proxy-table :deep(.n-data-table-tr:hover .n-data-table-td--fixed-left) {
  background: var(--fonu-bg, #f8fafc);
}

.proxy-table :deep(.proxy-row--active .n-data-table-td--fixed-right),
.proxy-table :deep(.proxy-row--active .n-data-table-td--fixed-left) {
  background: var(--fonu-bg-muted);
}

.proxy-table :deep(.domain-cell__main) {
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
}

.proxy-table :deep(.domain-cell__links) {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: 4px;
}

.proxy-table :deep(.domain-cell__link-row) {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  max-width: 100%;
}

.proxy-table :deep(.domain-cell__link) {
  flex: 0 1 auto;
  max-width: calc(100% - 52px);
  min-width: 0;
  font-size: 12px;
  font-family: var(--fonu-mono);
  color: var(--fonu-brand-text);
  text-decoration: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.proxy-table :deep(.domain-cell__link:hover) {
  color: var(--fonu-brand-hover);
  text-decoration: underline;
  text-decoration-skip-ink: none;
  text-underline-offset: 2px;
}

.proxy-table :deep(.domain-cell__link-action) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 22px;
  height: 22px;
  padding: 0;
  border: none;
  background: none;
  color: var(--fonu-brand-text);
  cursor: pointer;
  border-radius: 4px;
}

.proxy-table :deep(.domain-cell__link-action:hover) {
  color: var(--fonu-brand-hover);
  background: var(--fonu-brand-soft);
}

.proxy-table :deep(.domain-cell__more) {
  align-self: flex-start;
  margin-top: 2px;
  padding: 0;
  border: none;
  background: none;
  font: inherit;
  font-size: 12px;
  color: var(--fonu-brand-text);
  cursor: pointer;
}

.proxy-table :deep(.domain-cell__more:hover) {
  color: var(--fonu-brand-hover);
  text-decoration: underline;
  text-decoration-skip-ink: none;
  text-underline-offset: 2px;
}

.proxy-table :deep(.proto-tags) {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.proxy-table{min-width:0}
.proxy-table :deep(.n-data-table-base-table){table-layout:fixed;width:100%}
.proxy-table :deep(.n-data-table-th){background:var(--fonu-bg-muted);color:var(--fonu-text-secondary);font-size:12px;font-weight:500}
.proxy-table :deep(.n-data-table-td){padding:9px 10px;vertical-align:middle}
.proxy-table :deep(.proxy-service-cell){display:flex;align-items:flex-start;gap:9px;min-width:0}
.proxy-table :deep(.proxy-service-cell__content){display:grid;gap:3px;min-width:0;flex:1}
.proxy-table :deep(.proxy-service-name){border:0;background:transparent;padding:0;color:var(--fonu-text);font:inherit;font-weight:600;text-align:left;cursor:pointer;overflow-wrap:anywhere}
.proxy-table :deep(.proxy-service-name:hover){color:var(--fonu-brand-text)}
.proxy-table :deep(.proxy-service-entry),.proxy-table :deep(.proxy-cell-muted){font-size:11px;color:var(--fonu-text-secondary);overflow-wrap:anywhere}
.proxy-table :deep(.proxy-cell-lines){display:grid;gap:4px;min-width:0}
.proxy-table :deep(.proxy-target){font-size:12px;font-family:var(--fonu-mono);color:var(--fonu-text-secondary);overflow-wrap:anywhere}
.proxy-table :deep(.proxy-traffic-cell){font-size:11px;font-variant-numeric:tabular-nums;overflow-wrap:anywhere}
.proxy-table :deep(.proxy-security-tags){display:flex;flex-wrap:wrap;gap:3px}
.proxy-table :deep(.proxy-security-tags .n-tag){max-width:100%;font-size:11px}
.proxy-table :deep(.proxy-drag-handle){cursor:grab;color:var(--fonu-text-secondary);display:flex;align-items:center}
.proxy-table--compact :deep(.n-data-table-td){padding:8px 6px;vertical-align:top}
.proxy-table--compact :deep(.proxy-mobile-service){display:grid;gap:5px;min-width:0}
.proxy-table--compact :deep(.proxy-mobile-meta){display:flex;flex-wrap:wrap;align-items:center;gap:5px}
.proxy-table--compact :deep(.proxy-mobile-actions){display:grid;gap:8px;justify-items:end}
.proxy-table--compact :deep(.proxy-mobile-actions .n-space){display:grid!important;grid-template-columns:repeat(2, minmax(0,1fr));gap:2px!important}
.proxy-table :deep(.domain-cell__link){color:var(--fonu-brand-text)}
.proxy-table :deep(.domain-cell__link-row){display:flex;min-width:0}
.proxy-table :deep(.domain-cell__link){max-width:none;flex:1;min-width:0}
.proxy-table :deep(.domain-cell__link-action){flex:0 0 19px;width:19px;height:20px}
.proxy-table--compact :deep(.proxy-mobile-actions .n-button){min-width:26px}
.proxy-table--compact :deep(.proxy-service-cell){gap:5px}
.proxy-table--compact :deep(.proxy-service-cell .service-favicon){width:21px;height:21px}
</style>
