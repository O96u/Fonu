<template><StatusBadge class="http-status-badge" :kind="httpStatusKind(code)" :text="String(code)" :dot="false" /></template>
<script setup lang="ts">
import StatusBadge from './StatusBadge.vue'
import { httpStatusKind } from '../utils/status'
defineProps<{ code: number }>()
</script>
<style scoped>
.http-status-badge { min-width:42px; box-sizing:border-box; justify-content:center; text-align:center; padding:4px 8px; font-variant-numeric:tabular-nums; font-weight:600; }
</style>
