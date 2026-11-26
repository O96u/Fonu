<template>
  <div class="draft-field" :class="{'draft-field--small':small}">
    <textarea v-if="type==='textarea'" :value="value" :placeholder="placeholder" :disabled="disabled" :rows="rows??2" :maxlength="maxlength" :aria-label="label" @input="onInput"/>
    <input v-else :type="type==='password'&&!revealed?'password':'text'" :value="value" :placeholder="placeholder" :disabled="disabled" :maxlength="maxlength" :aria-label="label" :autocomplete="type==='password'?'new-password':undefined" @input="onInput" @keydown.enter.prevent="emit('enter')"/>
    <button v-if="type==='password'" type="button" :disabled="disabled" :aria-label="revealed?'隐藏密码':'显示密码'" @click="revealed=!revealed">{{revealed?'隐藏':'显示'}}</button>
  </div>
</template>
<script setup lang="ts">
import {ref} from 'vue'
defineProps<{value:string;label?:string;placeholder?:string;type?:'text'|'password'|'textarea';rows?:number;maxlength?:number;disabled?:boolean;small?:boolean}>()
const emit=defineEmits<{'update:value':[value:string];enter:[]}>()
const revealed=ref(false)
function onInput(event:Event){emit('update:value',(event.target as HTMLInputElement|HTMLTextAreaElement).value)}
</script>
<style scoped>
.draft-field{display:flex;align-items:center;width:100%;min-width:0;box-sizing:border-box;border:1px solid var(--fonu-border);border-radius:var(--fonu-radius-sm,6px);background:var(--fonu-surface);color:var(--fonu-text);transition:border-color .15s}
.draft-field:focus-within{border-color:var(--fonu-brand);box-shadow:0 0 0 2px var(--fonu-brand-soft)}
.draft-field input,.draft-field textarea{flex:1;width:100%;min-width:0;box-sizing:border-box;min-height:34px;border:0;background:none;color:inherit;font:inherit;padding:6px 11px;outline:none}.draft-field textarea{resize:vertical;line-height:1.5}.draft-field--small input{min-height:28px;padding:4px 9px;font-size:12px}.draft-field input::placeholder,.draft-field textarea::placeholder{color:var(--fonu-text-muted)}.draft-field input:disabled,.draft-field textarea:disabled{opacity:.65;cursor:not-allowed}.draft-field button{border:0;background:none;color:var(--fonu-text-secondary);padding:6px 9px;cursor:pointer;font:inherit;font-size:12px;flex-shrink:0}
</style>