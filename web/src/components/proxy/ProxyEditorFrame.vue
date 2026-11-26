<template>
  <!-- Naive UI/vueuc focus trap resolves its modal root by DIV tag. Keep the modal body a div. -->
  <n-modal :show="show" :mask-closable="false" :close-on-esc="!busy" @update:show="value=>{if(!value&&!busy)emit('close')}" transform-origin="center">
    <div class="editor-frame" :class="{'editor-frame--wide':wide}" role="dialog" aria-modal="true" :aria-label="title">
      <header class="editor-head"><div class="editor-head-icon"><n-icon :component="icon"/></div><div class="editor-head-text"><h2>{{title}}</h2><p v-if="subtitle">{{subtitle}}</p></div><n-button quaternary size="small" :disabled="busy" aria-label="关闭" @click="emit('close')"><template #icon><n-icon :component="CloseOutline"/></template></n-button></header>
      <div v-if="$slots.navigation" class="editor-navigation"><slot name="navigation"/></div>
      <div class="editor-body" :aria-busy="busy"><slot/></div>
      <footer v-if="$slots.footer" class="editor-footer"><slot name="footer"/></footer>
    </div>
  </n-modal>
</template>
<script setup lang="ts">
import type {Component} from 'vue'
import {NModal,NButton,NIcon} from 'naive-ui'
import {CloseOutline} from '@vicons/ionicons5'
defineProps<{show:boolean;title:string;subtitle?:string;icon:Component;busy?:boolean;wide?:boolean}>()
const emit=defineEmits<{close:[]}>()
</script>
<style scoped>
.editor-frame{width:min(760px,calc(100vw - 24px));max-height:calc(100dvh - 28px);display:flex;flex-direction:column;background:var(--fonu-surface);color:var(--fonu-text);border:1px solid var(--fonu-border);border-radius:var(--fonu-radius);box-shadow:var(--fonu-shadow-md);overflow:hidden;container:proxy-editor / inline-size;min-width:0;text-align:left}
.editor-frame--wide{width:min(1020px,calc(100vw - 24px))}
.editor-head{display:flex;align-items:center;gap:12px;flex-shrink:0;padding:18px 20px;border-bottom:1px solid var(--fonu-border)}
.editor-head-icon{width:38px;height:38px;display:grid;place-items:center;background:var(--fonu-brand-soft);color:var(--fonu-brand-text);border-radius:9px;font-size:21px;flex-shrink:0}
.editor-head-text{flex:1;min-width:0}.editor-head h2{font-size:18px;line-height:1.35;margin:0;font-weight:650;overflow-wrap:anywhere}.editor-head p{font-size:12px;margin:4px 0 0;color:var(--fonu-text-secondary);overflow-wrap:anywhere}
.editor-navigation{flex-shrink:0;padding:0 20px;background:var(--fonu-surface)}
.editor-body{flex:1 1 auto;min-height:0;min-width:0;overflow-y:auto;overscroll-behavior:contain;padding:16px 20px;scrollbar-gutter:stable}
.editor-footer{display:flex;gap:8px;align-items:center;justify-content:flex-end;flex-wrap:wrap;flex-shrink:0;padding:12px 20px;border-top:1px solid var(--fonu-border);background:var(--fonu-surface)}
@media(max-width:600px){.editor-frame{max-height:calc(100dvh - 16px);width:calc(100vw - 16px)}.editor-head{padding:12px;gap:8px}.editor-head-icon{width:30px;height:30px;font-size:17px}.editor-head h2{font-size:16px}.editor-body{padding:12px;scrollbar-gutter:auto}.editor-navigation{padding:0 12px}.editor-footer{padding:10px 12px}}
</style>
