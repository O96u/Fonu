<script setup lang="ts">
import { ref, watch } from 'vue'
import { NIcon } from 'naive-ui'
import { GitNetworkOutline } from '@vicons/ionicons5'
const props = defineProps<{ ruleId?: number }>()
const failed = ref(false)
watch(() => props.ruleId, () => { failed.value = false })
</script>
<template><span class="service-favicon"><img v-if="ruleId && !failed" :key="ruleId" :src="'/api/proxies/'+ruleId+'/favicon'" alt="" loading="lazy" decoding="async" @error="failed=true"/><n-icon v-else :component="GitNetworkOutline"/></span></template>
<style scoped>
.service-favicon{display:grid;place-items:center;flex-shrink:0;width:25px;height:25px;border-radius:7px;background:var(--fonu-brand-soft);color:var(--fonu-brand-text);overflow:hidden}
.service-favicon img{width:20px;height:20px;object-fit:contain}
</style>
