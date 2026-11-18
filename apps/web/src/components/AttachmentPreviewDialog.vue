<script setup lang="ts">
import { computed, nextTick, ref } from "vue"
import { NpButton, toast } from "@ningpm/ui"
import {
  parseExcel,
  previewKind,
  renderDocx,
  type ExcelSheetView,
  type PreviewKind,
} from "../lib/preview"
import { downloadResource, fetchResourceBlob } from "../lib/upload"

const open = ref(false)
const busy = ref(false)
const name = ref("")
const kind = ref<PreviewKind>("none")
const objectUrl = ref("")
const docxHost = ref<HTMLElement | null>(null)
const excelSheets = ref<ExcelSheetView[]>([])
const excelTab = ref(0)

const activeExcel = computed(() => excelSheets.value[excelTab.value] || null)

function resetContent() {
  if (objectUrl.value) URL.revokeObjectURL(objectUrl.value)
  objectUrl.value = ""
  excelSheets.value = []
  excelTab.value = 0
  if (docxHost.value) docxHost.value.innerHTML = ""
}

function close() {
  open.value = false
  busy.value = false
  name.value = ""
  kind.value = "none"
  resetContent()
}

/** Open preview for a resource; non-previewable types fall back to download. */
async function show(opts: { name: string; url: string; contentType?: string }) {
  const nextKind = previewKind(opts.name, opts.contentType || "")
  if (nextKind === "none") {
    try {
      await downloadResource(opts.url, opts.name)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "下载失败")
    }
    return
  }

  resetContent()
  name.value = opts.name
  kind.value = nextKind
  open.value = true
  busy.value = true
  try {
    await nextTick()
    const blob = await fetchResourceBlob(opts.url)
    if (nextKind === "image" || nextKind === "pdf") {
      objectUrl.value = URL.createObjectURL(blob)
      return
    }
    if (nextKind === "docx") {
      await nextTick()
      if (!docxHost.value) throw new Error("预览容器未就绪")
      await renderDocx(blob, docxHost.value)
      return
    }
    if (nextKind === "excel") {
      excelSheets.value = await parseExcel(blob)
      excelTab.value = 0
    }
  } catch (e) {
    close()
    toast.error(e instanceof Error ? e.message : "预览失败")
  } finally {
    busy.value = false
  }
}

defineExpose({ show, close })
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="preview-mask" @click.self="close">
      <div class="preview-panel" :class="`is-${kind}`" role="dialog" aria-modal="true">
        <header>
          <strong>{{ name }}</strong>
          <NpButton size="sm" variant="ghost" @click="close">关闭</NpButton>
        </header>
        <div class="preview-body">
          <p v-if="busy" class="preview-loading">加载预览…</p>
          <img
            v-show="!busy && kind === 'image' && objectUrl"
            :src="objectUrl"
            :alt="name"
          />
          <iframe
            v-show="!busy && kind === 'pdf' && objectUrl"
            class="pdf-frame"
            :src="objectUrl"
            title="PDF 预览"
          />
          <div
            v-show="kind === 'docx'"
            ref="docxHost"
            class="docx-host"
            :class="{ 'is-busy': busy }"
          />
          <div v-show="kind === 'excel'" class="excel-host" :class="{ 'is-busy': busy }">
            <div v-if="excelSheets.length > 1" class="excel-tabs">
              <button
                v-for="(s, i) in excelSheets"
                :key="s.name"
                type="button"
                class="excel-tab"
                :class="{ 'is-on': excelTab === i }"
                @click="excelTab = i"
              >
                {{ s.name }}
              </button>
            </div>
            <div v-if="activeExcel" class="excel-sheet" v-html="activeExcel.html" />
            <p v-else-if="!busy" class="muted">空工作簿</p>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.preview-mask {
  position: fixed;
  inset: 0;
  z-index: 80;
  display: grid;
  place-items: center;
  padding: 24px;
  background: rgba(15, 23, 42, 0.55);
}

.preview-panel {
  width: min(960px, 100%);
  max-height: min(90vh, 900px);
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  border-radius: 14px;
  background: #fff;
  box-shadow: 0 24px 64px rgba(15, 23, 42, 0.28);
  overflow: hidden;
}

.preview-panel.is-docx,
.preview-panel.is-excel,
.preview-panel.is-pdf {
  width: min(1100px, 100%);
}

.preview-panel header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-shrink: 0;
}

.preview-body {
  position: relative;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.preview-loading {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: grid;
  place-items: center;
  margin: 0;
  background: rgba(255, 255, 255, 0.82);
  color: var(--np-text-muted);
  font-size: var(--np-text-sm);
}

.preview-panel img {
  width: 100%;
  max-height: calc(90vh - 96px);
  object-fit: contain;
  border-radius: 8px;
  background: #0f172a0a;
}

.pdf-frame {
  width: 100%;
  height: calc(90vh - 96px);
  border: 1px solid var(--np-border);
  border-radius: 8px;
  background: #f8fafc;
}

.docx-host,
.excel-host {
  flex: 1;
  min-height: 0;
  overflow: auto;
  border: 1px solid var(--np-border);
  border-radius: 8px;
  background: #f8fafc;
}

.docx-host {
  padding: 12px;
}

.docx-host.is-busy,
.excel-host.is-busy {
  opacity: 0.35;
}

.docx-host :deep(.np-docx) {
  background: #fff;
  padding: 16px;
}

.excel-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 10px;
  border-bottom: 1px solid var(--np-border);
  background: #fff;
}

.excel-tab {
  border: 1px solid var(--np-border);
  background: #fff;
  border-radius: 999px;
  padding: 4px 10px;
  font: inherit;
  font-size: var(--np-text-xs);
  cursor: pointer;
}

.excel-tab.is-on {
  border-color: var(--np-accent);
  color: var(--np-accent);
  background: #eff6ff;
}

.excel-sheet {
  padding: 10px;
  overflow: auto;
}

.excel-sheet :deep(table) {
  border-collapse: collapse;
  background: #fff;
  font-size: 12px;
}

.excel-sheet :deep(td),
.excel-sheet :deep(th) {
  border: 1px solid #e5e7eb;
  padding: 4px 8px;
  white-space: nowrap;
}

.muted {
  color: var(--np-text-muted);
  font-size: var(--np-text-sm);
  padding: 16px;
}
</style>
