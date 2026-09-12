import { h, type VNode } from 'vue'
import { NButton, NSpace } from 'naive-ui'
import type { ButtonProps } from 'naive-ui'

export interface TableRowAction {
  label: string
  onClick: () => void
  type?: ButtonProps['type']
  /** 为 false 时隐藏该操作，默认显示 */
  show?: boolean
}

/** 表格操作列：所有按钮平铺展示，不使用下拉菜单 */
export function renderTableRowActions(actions: TableRowAction[]): VNode {
  const buttons = actions
    .filter((action) => action.show !== false)
    .map((action) =>
      h(
        NButton,
        {
          size: 'small',
          quaternary: true,
          type: action.type,
          onClick: action.onClick,
        },
        () => action.label,
      ),
    )

  return h(NSpace, { size: 4, wrap: false }, () => buttons)
}
