<template>
  <n-input class="draft-field" :value="value" :type="type==='textarea'?'textarea':type==='password'?'password':'text'" :size="small?'small':'medium'" :placeholder="placeholder" :disabled="disabled" :rows="rows??2" :maxlength="maxlength" :input-props="{'aria-label':label,autocomplete:autocomplete??(type==='password'?'new-password':'off')}" :show-password-on="type==='password'?'click':undefined" @update:value="emit('update:value',$event)" @keydown.enter="onEnter"/>
</template>
<script setup lang="ts">
import {NInput} from 'naive-ui'
const props=defineProps<{value:string;label?:string;autocomplete?:string;placeholder?:string;type?:'text'|'password'|'textarea';rows?:number;maxlength?:number;disabled?:boolean;small?:boolean}>()
const emit=defineEmits<{'update:value':[value:string];enter:[]}>()
function onEnter(event:KeyboardEvent){if(props.type!=='textarea'&&!event.isComposing){event.preventDefault();emit('enter')}}
</script>
<style scoped>
.draft-field{width:100%;min-width:0}
</style>
