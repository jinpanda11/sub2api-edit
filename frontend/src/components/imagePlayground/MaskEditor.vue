<template>
  <BaseDialog :show="show" :title="title" width="wide" @close="emit('close')">
    <div class="p-6">
      <p class="mb-4 text-sm text-gray-500 dark:text-dark-400">
        {{ t('imagePlayground.maskHint') }}
      </p>

      <!-- 画布区 -->
      <div
        ref="containerRef"
        class="relative mx-auto w-full touch-none select-none overflow-hidden rounded-xl border border-gray-200 bg-gray-900 dark:border-dark-700"
        :style="{ maxWidth: `${maxWidth}px` }"
      >
        <img v-if="image" :src="image" alt="" class="block h-auto w-full" @load="onImageLoaded" />
        <canvas ref="maskCanvasRef" class="absolute inset-0 h-full w-full" :width="canvasWidth" :height="canvasHeight"></canvas>
      </div>

      <!-- 工具栏 -->
      <div class="mt-4 flex flex-wrap items-center gap-4">
        <div class="flex items-center gap-2 rounded-xl border border-gray-200 p-1 dark:border-dark-700">
          <button
            type="button"
            class="rounded-lg px-3 py-1.5 text-sm font-medium transition"
            :class="tool === 'brush' ? 'bg-primary-500 text-white' : 'text-gray-600 hover:bg-gray-100 dark:text-dark-300 dark:hover:bg-dark-800'"
            @click="tool = 'brush'"
          >
            {{ t('imagePlayground.maskBrush') }}
          </button>
          <button
            type="button"
            class="rounded-lg px-3 py-1.5 text-sm font-medium transition"
            :class="tool === 'eraser' ? 'bg-gray-700 text-white' : 'text-gray-600 hover:bg-gray-100 dark:text-dark-300 dark:hover:bg-dark-800'"
            @click="tool = 'eraser'"
          >
            {{ t('imagePlayground.maskEraser') }}
          </button>
        </div>

        <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-dark-300">
          {{ t('imagePlayground.maskBrushSize') }}
          <input v-model.number="brushSize" type="range" min="4" max="80" step="2" class="w-32 accent-primary-500" />
          <span class="w-8 text-gray-400">{{ brushSize }}</span>
        </label>

        <button type="button" class="btn btn-secondary" @click="clearMask">
          {{ t('imagePlayground.maskClear') }}
        </button>
      </div>

      <!-- 操作按钮 -->
      <div class="mt-6 flex justify-end gap-2">
        <button type="button" class="btn btn-secondary" @click="emit('close')">
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="btn btn-primary" :disabled="!maskDirty" @click="confirm">
          {{ t('imagePlayground.maskApply') }}
        </button>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'

const props = defineProps<{
  show: boolean
  image: string
  title: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'confirm', maskBlob: Blob): void
}>()

const { t } = useI18n()

const containerRef = ref<HTMLDivElement | null>(null)
const maskCanvasRef = ref<HTMLCanvasElement | null>(null)
const canvasWidth = ref(0)
const canvasHeight = ref(0)
const maxWidth = ref(640)

const tool = ref<'brush' | 'eraser'>('brush')
const brushSize = ref(32)
const maskDirty = ref(false)

let bgCanvas: HTMLCanvasElement | null = null
let drawing = false
let lastX = 0
let lastY = 0

function onImageLoaded(e: Event) {
  const img = e.target as HTMLImageElement
  const container = containerRef.value
  if (!container || !maskCanvasRef.value) return

  const containerWidth = Math.min(container.clientWidth || maxWidth.value, maxWidth.value)
  const scale = containerWidth / img.naturalWidth
  canvasWidth.value = Math.round(img.naturalWidth * scale)
  canvasHeight.value = Math.round(img.naturalHeight * scale)

  // 背景层：绘制原图
  bgCanvas = document.createElement('canvas')
  bgCanvas.width = canvasWidth.value
  bgCanvas.height = canvasHeight.value
  const bgCtx = bgCanvas.getContext('2d')
  if (bgCtx) bgCtx.drawImage(img, 0, 0, canvasWidth.value, canvasHeight.value)

  // 遮罩层：透明底，白笔绘制
  const ctx = maskCanvasRef.value.getContext('2d')
  if (ctx) {
    ctx.clearRect(0, 0, canvasWidth.value, canvasHeight.value)
    ctx.lineCap = 'round'
    ctx.lineJoin = 'round'
  }
  maskDirty.value = false
}

function canvasPos(e: PointerEvent): { x: number; y: number } {
  const canvas = maskCanvasRef.value!
  const rect = canvas.getBoundingClientRect()
  return {
    x: ((e.clientX - rect.left) / rect.width) * canvasWidth.value,
    y: ((e.clientY - rect.top) / rect.height) * canvasHeight.value,
  }
}

function startDraw(e: PointerEvent) {
  if (!maskCanvasRef.value) return
  drawing = true
  const pos = canvasPos(e)
  lastX = pos.x
  lastY = pos.y
  drawStroke(e)
}

function drawStroke(e: PointerEvent) {
  if (!drawing || !maskCanvasRef.value) return
  const ctx = maskCanvasRef.value.getContext('2d')
  if (!ctx) return
  const pos = canvasPos(e)

  // 画笔：source-over 白色（编辑区）；橡皮：destination-out 擦除
  ctx.globalCompositeOperation = tool.value === 'brush' ? 'source-over' : 'destination-out'
  ctx.strokeStyle = tool.value === 'brush' ? '#ffffff' : 'rgba(0,0,0,1)'
  ctx.lineWidth = brushSize.value
  ctx.beginPath()
  ctx.moveTo(lastX, lastY)
  ctx.lineTo(pos.x, pos.y)
  ctx.stroke()
  lastX = pos.x
  lastY = pos.y
  maskDirty.value = true
}

function stopDraw() {
  drawing = false
}

function clearMask() {
  const canvas = maskCanvasRef.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  if (ctx) ctx.clearRect(0, 0, canvasWidth.value, canvasHeight.value)
  maskDirty.value = false
}

function confirm() {
  const canvas = maskCanvasRef.value
  if (!canvas) return
  // 导出：白色笔迹绘制在黑色底上（OpenAI 约定：白色区域将被编辑）
  const out = document.createElement('canvas')
  out.width = canvasWidth.value
  out.height = canvasHeight.value
  const ctx = out.getContext('2d')
  if (!ctx) return
  ctx.fillStyle = '#000000'
  ctx.fillRect(0, 0, out.width, out.height)
  ctx.drawImage(canvas, 0, 0)
  out.toBlob(
    (blob) => {
      if (blob) emit('confirm', blob)
    },
    'image/png',
  )
}

watch(
  () => props.show,
  (show) => {
    if (show) {
      maskDirty.value = false
      drawing = false
      tool.value = 'brush'
    }
  },
)

onMounted(() => {
  // 指针事件绑定在画布上
  const canvas = maskCanvasRef.value
  if (canvas) {
    canvas.addEventListener('pointerdown', startDraw)
    canvas.addEventListener('pointermove', drawStroke)
    canvas.addEventListener('pointerup', stopDraw)
    canvas.addEventListener('pointerleave', stopDraw)
  }
})
</script>
