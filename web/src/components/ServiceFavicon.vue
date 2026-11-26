<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from 'vue'
import { NIcon } from 'naive-ui'
import { GitNetworkOutline } from '@vicons/ionicons5'
const props = defineProps<{ ruleId?: number }>()
const failed = ref(false)
const attempt = ref(0)
let timer: ReturnType<typeof setTimeout>|undefined
function reset(){if(timer)clearTimeout(timer);timer=undefined;failed.value=false;attempt.value=0}
watch(() => props.ruleId, reset)
onBeforeUnmount(()=>{if(timer)clearTimeout(timer)})
function retry(){failed.value=true;if(attempt.value>=2)return;timer=setTimeout(()=>{attempt.value++;failed.value=false},2000*(attempt.value+1))}
</script>
<template><span class="service-favicon"><img v-if="ruleId && !failed" :key="ruleId+'-'+attempt" :src="'/api/proxies/'+ruleId+'/favicon'+(attempt?'?retry='+attempt:'')" alt="" loading="lazy" decoding="async" @error="retry"/><n-icon v-else :component="GitNetworkOutline"/></span></template>
<style scoped>
.service-favicon{display:grid;place-items:center;flex-shrink:0;width:25px;height:25px;border-radius:7px;background:var(--fonu-brand-soft);color:var(--fonu-brand-text);overflow:hidden}
.service-favicon img{width:20px;height:20px;object-fit:contain}
</style>
