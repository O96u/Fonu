<template>
  <n-data-table
    class="proxy-table"
    :class="{'proxy-table--nested':nested,'proxy-table--sortable':sortable}"
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
defineProps<{nested?:boolean;sortable:boolean;columns:DataTableColumns<ProxyRule>;data:ProxyRule[];scrollX?:number;rowProps:(r:ProxyRule)=>{class:string;'data-rule-id':string}}>()
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
  background: color-mix(in srgb, #10b981 8%, var(--fonu-surface, #fff));
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
  background: color-mix(in srgb, #10b981 8%, var(--fonu-surface, #fff));
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
  color: #2563eb;
  text-decoration: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.proxy-table :deep(.domain-cell__link:hover) {
  color: #1d4ed8;
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
  color: #2563eb;
  cursor: pointer;
  border-radius: 4px;
}

.proxy-table :deep(.domain-cell__link-action:hover) {
  color: #1d4ed8;
  background: rgba(37, 99, 235, 0.08);
}

.proxy-table :deep(.domain-cell__more) {
  align-self: flex-start;
  margin-top: 2px;
  padding: 0;
  border: none;
  background: none;
  font: inherit;
  font-size: 12px;
  color: #2563eb;
  cursor: pointer;
}

.proxy-table :deep(.domain-cell__more:hover) {
  color: #1d4ed8;
  text-decoration: underline;
  text-decoration-skip-ink: none;
  text-underline-offset: 2px;
}

.proxy-table :deep(.proto-tags) {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
</style>
