<template>
  <div
    class="nginx-code-editor"
    :class="{
      'nginx-code-editor--readonly': readonly,
      'nginx-code-editor--embedded': embedded,
    }"
  >
    <Codemirror
      :model-value="modelValue"
      :placeholder="placeholder"
      :style="editorStyle"
      :autofocus="false"
      :indent-with-tab="true"
      :tab-size="4"
      :extensions="extensions"
      @update:model-value="onUpdate"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Codemirror } from 'vue-codemirror'
import { EditorState } from '@codemirror/state'
import { EditorView, lineNumbers, highlightActiveLine, highlightActiveLineGutter } from '@codemirror/view'
import { StreamLanguage, syntaxHighlighting, defaultHighlightStyle } from '@codemirror/language'
import { nginx } from '@codemirror/legacy-modes/mode/nginx'
import { oneDark } from '@codemirror/theme-one-dark'
import { useTheme } from '../composables/useTheme'

const props = withDefaults(
  defineProps<{
    modelValue: string
    readonly?: boolean
    embedded?: boolean
    placeholder?: string
    minHeight?: string
  }>(),
  {
    readonly: false,
    embedded: false,
    placeholder: '',
    minHeight: '',
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const { isDark } = useTheme()

const nginxLanguage = StreamLanguage.define(nginx)

const lightTheme = EditorView.theme(
  {
    '&': {
      backgroundColor: 'var(--fonu-surface)',
      color: 'var(--fonu-text)',
    },
    '.cm-content': {
      caretColor: 'var(--fonu-brand)',
      fontFamily: 'var(--fonu-mono)',
      fontSize: '12px',
      lineHeight: '1.7',
    },
    '.cm-gutters': {
      backgroundColor: 'var(--fonu-bg-muted)',
      color: 'var(--fonu-text-muted)',
      borderRight: '1px solid var(--fonu-border)',
    },
    '.cm-activeLineGutter': {
      backgroundColor: 'var(--fonu-bg)',
      color: 'var(--fonu-text-secondary)',
    },
    '.cm-activeLine': {
      backgroundColor: 'var(--fonu-brand-soft)',
    },
    '.cm-selectionBackground, &.cm-focused .cm-selectionBackground, ::selection': {
      backgroundColor: 'color-mix(in srgb, var(--fonu-brand) 22%, transparent) !important',
    },
    '.cm-cursor, .cm-dropCursor': {
      borderLeftColor: 'var(--fonu-brand)',
    },
    '&.cm-focused': {
      outline: 'none',
    },
  },
  { dark: false },
)

const editorStyle = computed(() => {
  if (props.embedded) {
    return { height: '100%' }
  }
  if (props.minHeight) {
    return { height: '100%', minHeight: props.minHeight }
  }
  return { height: '100%', minHeight: '240px' }
})

const extensions = computed(() => {
  const base = [
    lineNumbers(),
    highlightActiveLineGutter(),
    highlightActiveLine(),
    nginxLanguage,
    EditorView.lineWrapping,
  ]
  if (isDark.value) {
    base.push(oneDark)
  } else {
    base.push(lightTheme, syntaxHighlighting(defaultHighlightStyle, { fallback: true }))
  }
  if (props.readonly) {
    base.push(EditorState.readOnly.of(true), EditorView.editable.of(false))
  }
  return base
})

function onUpdate(value: string) {
  if (!props.readonly) {
    emit('update:modelValue', value)
  }
}
</script>

<style scoped>
.nginx-code-editor {
  border: 1px solid var(--fonu-border);
  border-radius: var(--fonu-radius-sm);
  overflow: hidden;
  background: var(--fonu-surface);
}

.nginx-code-editor--embedded {
  flex: 1 1 0;
  min-height: 0;
}

.nginx-code-editor :deep(.cm-editor) {
  height: 100%;
}

.nginx-code-editor :deep(.cm-scroller) {
  overflow: auto;
  scrollbar-width: thin;
  scrollbar-color: color-mix(in srgb, var(--fonu-text-muted) 35%, transparent) transparent;
}

.nginx-code-editor:hover :deep(.cm-scroller) {
  scrollbar-color: color-mix(in srgb, var(--fonu-text-muted) 65%, transparent) var(--fonu-bg-muted);
}

.nginx-code-editor :deep(.cm-scroller)::-webkit-scrollbar {
  width: 7px;
  height: 7px;
}

.nginx-code-editor :deep(.cm-scroller)::-webkit-scrollbar-track {
  background: transparent;
}

.nginx-code-editor :deep(.cm-scroller)::-webkit-scrollbar-thumb {
  background: color-mix(in srgb, var(--fonu-text-muted) 30%, transparent);
  border-radius: 4px;
}

.nginx-code-editor:hover :deep(.cm-scroller)::-webkit-scrollbar-thumb {
  background: color-mix(in srgb, var(--fonu-text-muted) 55%, transparent);
}

.nginx-code-editor--readonly :deep(.cm-cursor) {
  display: none;
}

.nginx-code-editor--readonly :deep(.cm-content) {
  caret-color: transparent;
}
</style>
