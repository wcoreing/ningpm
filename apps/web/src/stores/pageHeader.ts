import { defineStore } from "pinia"
import { ref } from "vue"

export type PageHeaderState = {
  title: string
  badge?: string
  subtitle?: string
}

/** Shell 顶栏标题（项目页等可覆盖路由默认 meta） */
export const usePageHeaderStore = defineStore("pageHeader", () => {
  const override = ref<PageHeaderState | null>(null)

  function set(next: PageHeaderState | null) {
    override.value = next
  }

  function clear() {
    override.value = null
  }

  return { override, set, clear }
})
