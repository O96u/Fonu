import { h, type Component, type VNode } from 'vue'
import { NButton, NIcon, NSpace, NTooltip } from 'naive-ui'
import type { ButtonProps } from 'naive-ui'

export interface TableRowAction {
  label: string
  onClick: () => void
  type?: ButtonProps['type']
  /** 为 false 时隐藏该操作，默认显示 */
  show?: boolean
  icon?: Component
}

function renderActionButton(action: TableRowAction): VNode {
  const button = h(
    NButton,
    {
      size: 'small',
      quaternary: true,
      type: action.type,
      onClick: (e: MouseEvent) => {
        e.stopPropagation()
        action.onClick()
      },
    },
    action.icon
      ? {
          icon: () => h(NIcon, { component: action.icon, size: 16 }),
        }
      : () => action.label,
  )

  if (!action.icon) return button

  return h(
    NTooltip,
    { trigger: 'hover' },
    {
      trigger: () => button,
      default: () => action.label,
    },
  )
}

/** 表格操作列：所有按钮平铺展示，不使用下拉菜单 */
export function renderTableRowActions(actions: TableRowAction[]): VNode {
  const buttons = actions
    .filter((action) => action.show !== false)
    .map((action) => renderActionButton(action))

  return h(NSpace, { size: 2, wrap: false, align: 'center' }, () => buttons)
}
