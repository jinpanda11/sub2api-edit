<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-6">
      <!-- ===================== 顶栏 ===================== -->
      <div class="card">
        <div class="flex flex-col gap-4 p-5 lg:flex-row lg:items-center lg:justify-between">
          <div class="flex items-center gap-3">
            <div
              class="flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-primary-500 to-primary-600 text-white shadow-md shadow-primary-500/25"
            >
              <Icon name="beaker" size="lg" />
            </div>
            <div>
              <h1 class="text-lg font-bold text-gray-900 dark:text-white">
                {{ t('imagePlayground.title') }}
              </h1>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('imagePlayground.subtitle') }}
              </p>
            </div>
          </div>

          <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
            <span class="text-sm text-gray-500 dark:text-dark-400">
              {{ t('imagePlayground.balance') }}:
              <span class="font-semibold text-primary-600 dark:text-primary-400">
                US${{ (authStore.user?.balance ?? 0).toFixed(2) }}
              </span>
            </span>

            <!-- API Key 选择 -->
            <select v-model="selectedKeyId" class="input w-full sm:w-72" @change="onKeyChange">
              <option value="" disabled>{{ t('imagePlayground.selectKey') }}</option>
              <option v-for="k in usableKeys" :key="k.id" :value="k.id">
                {{ k.name || k.key.slice(0, 12) }}…（{{ k.key.slice(0, 8) }}…）
              </option>
            </select>

            <!-- 模型选择 -->
            <div class="relative w-full sm:w-72">
              <input
                v-model="model"
                type="text"
                list="image-model-options"
                class="input w-full"
                :placeholder="t('imagePlayground.modelPlaceholder')"
              />
              <datalist id="image-model-options">
                <option v-for="m in modelOptions" :key="m" :value="m"></option>
              </datalist>
            </div>
          </div>
        </div>
      </div>

      <!-- ===================== 主体 ===================== -->
      <div class="grid gap-6 lg:grid-cols-3">
        <!-- 左：参数面板 -->
        <div class="card h-fit lg:col-span-1">
          <div class="p-6">
            <!-- 模式切换 -->
            <div class="mb-5 inline-flex w-full rounded-lg border border-gray-200 bg-gray-50 p-1 dark:border-dark-700 dark:bg-dark-900/40">
              <button
                type="button"
                class="flex-1 rounded-md px-3 py-2 text-sm font-medium transition"
                :class="mode === 'generation' ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300' : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'"
                @click="mode = 'generation'"
              >
                {{ t('imagePlayground.modeGeneration') }}
              </button>
              <button
                type="button"
                class="flex-1 rounded-md px-3 py-2 text-sm font-medium transition"
                :class="mode === 'edit' ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300' : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'"
                @click="mode = 'edit'"
              >
                {{ t('imagePlayground.modeEdit') }}
              </button>
            </div>

            <!-- 提示词 -->
            <label class="input-label">{{ t('imagePlayground.prompt') }}</label>
            <textarea
              v-model="prompt"
              rows="4"
              class="input mt-1.5 w-full resize-y"
              :placeholder="t('imagePlayground.promptPlaceholder')"
            ></textarea>

            <!-- 图生图：参考图 -->
            <div v-if="mode === 'edit'" class="mt-5 space-y-3">
              <label class="input-label">{{ t('imagePlayground.referenceImage') }}</label>
              <ImageUpload v-model="referenceImage" :max-size="10 * 1024 * 1024" />
              <button
                type="button"
                class="btn btn-secondary w-full"
                :disabled="!referenceImage"
                @click="openMaskEditor"
              >
                <Icon name="edit" size="md" class="mr-2" />
                {{ maskBlob ? t('imagePlayground.maskEditAgain') : t('imagePlayground.maskEdit') }}
              </button>
              <p v-if="maskBlob" class="text-xs text-emerald-600 dark:text-emerald-400">
                {{ t('imagePlayground.maskReady') }}
              </p>
            </div>

            <!-- 参数 -->
            <div class="mt-5 space-y-4">
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <label class="input-label">{{ t('imagePlayground.count') }}</label>
                  <input v-model.number="count" type="number" min="1" max="4" class="input mt-1.5" />
                </div>
                <div>
                  <label class="input-label">{{ t('imagePlayground.quality') }}</label>
                  <select v-model="quality" class="input mt-1.5">
                    <option value="auto">{{ t('imagePlayground.qualityAuto') }}</option>
                    <option value="high">{{ t('imagePlayground.qualityHigh') }}</option>
                    <option value="medium">{{ t('imagePlayground.qualityMedium') }}</option>
                    <option value="low">{{ t('imagePlayground.qualityLow') }}</option>
                  </select>
                </div>
              </div>

              <!-- 尺寸：档位 × 比例 -->
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <label class="input-label">{{ t('imagePlayground.sizeTier') }}</label>
                  <select v-model="sizeTier" class="input mt-1.5">
                    <option value="auto">{{ t('imagePlayground.sizeAuto') }}</option>
                    <option value="1K">1K</option>
                    <option value="2K">2K</option>
                    <option value="4K">4K</option>
                  </select>
                </div>
                <div>
                  <label class="input-label">{{ t('imagePlayground.sizeRatio') }}</label>
                  <select v-model="sizeRatio" class="input mt-1.5" :disabled="sizeTier === 'auto'">
                    <option v-for="r in SIZE_RATIOS" :key="r" :value="r">{{ r }}</option>
                  </select>
                </div>
              </div>
              <p v-if="sizeTier !== 'auto'" class="text-xs text-gray-400 dark:text-dark-500">
                {{ resolvedSize }}
              </p>

              <!-- 输出格式 / 审核 / 种子 -->
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <label class="input-label">{{ t('imagePlayground.outputFormat') }}</label>
                  <select v-model="outputFormat" class="input mt-1.5">
                    <option value="png">PNG</option>
                    <option value="jpeg">JPEG</option>
                    <option value="webp">WebP</option>
                  </select>
                </div>
                <div>
                  <label class="input-label">{{ t('imagePlayground.moderation') }}</label>
                  <select v-model="moderation" class="input mt-1.5">
                    <option value="auto">{{ t('imagePlayground.moderationAuto') }}</option>
                    <option value="low">{{ t('imagePlayground.moderationLow') }}</option>
                  </select>
                </div>
              </div>

              <!-- jpeg/webp 压缩率 -->
              <div v-if="outputFormat !== 'png'" class="flex items-center justify-between gap-3">
                <label class="input-label mb-0">
                  {{ t('imagePlayground.outputCompression') }}
                </label>
                <div class="flex items-center gap-2">
                  <input v-model.number="outputCompression" type="range" min="0" max="100" step="1" class="w-32 accent-primary-500" />
                  <span class="w-8 text-xs text-gray-400">{{ outputCompression }}</span>
                </div>
              </div>

              <!-- 透明背景 / 种子 -->
              <div class="grid grid-cols-2 items-end gap-4">
                <div class="flex items-center justify-between rounded-xl border border-gray-100 bg-gray-50 px-3 py-2.5 dark:border-dark-700 dark:bg-dark-800">
                  <label class="text-sm text-gray-600 dark:text-dark-300">
                    {{ t('imagePlayground.transparentOutput') }}
                  </label>
                  <Toggle v-model="transparentOutput" />
                </div>
                <div>
                  <label class="input-label">{{ t('imagePlayground.seed') }}</label>
                  <input
                    v-model.number="seed"
                    type="number"
                    min="0"
                    :placeholder="t('imagePlayground.seedPlaceholder')"
                    class="input mt-1.5"
                  />
                </div>
              </div>
            </div>

            <!-- 生成按钮 -->
            <button
              class="btn btn-primary mt-6 w-full py-3 text-base"
              :disabled="generating || !selectedKeyId || !prompt.trim() || (mode === 'edit' && !referenceImage)"
              @click="handleGenerate"
            >
              <svg
                v-if="generating"
                class="-ml-1 mr-2 h-5 w-5 animate-spin"
                fill="none"
                viewBox="0 0 24 24"
              >
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
              </svg>
              {{ generating ? t('imagePlayground.generating') : t('imagePlayground.generate') }}
            </button>

            <p v-if="errorMessage" class="mt-3 rounded-lg bg-red-50 px-3 py-2 text-xs text-red-600 dark:bg-red-900/20 dark:text-red-400">
              {{ errorMessage }}
            </p>
          </div>
        </div>

        <!-- 右：结果 / 画廊 -->
        <div class="space-y-6 lg:col-span-2">
          <!-- Tab 切换 -->
          <div class="flex items-center justify-between gap-3">
            <div class="inline-flex rounded-lg border border-gray-200 bg-gray-50 p-1 dark:border-dark-700 dark:bg-dark-900/40">
              <button
                type="button"
                class="rounded-md px-4 py-1.5 text-sm font-medium transition"
                :class="galleryTab === 'results' ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300' : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'"
                @click="galleryTab = 'results'"
              >
                {{ t('imagePlayground.tabResults') }}（{{ results.length }}）
              </button>
              <button
                type="button"
                class="rounded-md px-4 py-1.5 text-sm font-medium transition"
                :class="galleryTab === 'gallery' ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300' : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'"
                @click="galleryTab = 'gallery'"
              >
                {{ t('imagePlayground.tabGallery') }}（{{ gallery.length }}）
              </button>
            </div>
            <button
              v-if="galleryTab === 'gallery'"
              type="button"
              class="btn btn-secondary btn-sm"
              @click="loadGallery"
            >
              <Icon name="refresh" size="sm" class="mr-1.5" />
              {{ t('imagePlayground.refresh') }}
            </button>
          </div>

          <!-- 本次结果 -->
          <div v-if="galleryTab === 'results'" class="card">
            <div v-if="generating || lastGenerationDurationSeconds !== null" class="border-b border-gray-100 px-6 py-3 text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
              <span v-if="generating">{{ formatGenerationDuration(generationElapsedSeconds, 'elapsed') }}</span>
              <span v-else>{{ formatGenerationDuration(lastGenerationDurationSeconds!, 'completed') }}</span>
            </div>
            <div v-if="results.length" class="grid grid-cols-1 gap-4 p-6 sm:grid-cols-2 xl:grid-cols-3">
              <div v-for="(img, idx) in results" :key="idx" class="group relative overflow-hidden rounded-xl border border-gray-100 bg-gray-50 dark:border-dark-700 dark:bg-dark-800">
                <img
                  :src="img.dataUrl || img.url"
                  alt="generated"
                  class="aspect-square w-full cursor-pointer object-cover"
                  loading="lazy"
                  @click="openResultsPreview"
                />
                <!-- 操作层 -->
                <div class="pointer-events-none absolute inset-0 flex flex-col justify-end gap-1 bg-gradient-to-t from-black/70 via-transparent to-transparent p-3 opacity-0 transition-opacity group-hover:opacity-100">
                  <div class="flex flex-wrap gap-1.5">
                    <button type="button" class="pointer-events-auto rounded-lg bg-white/90 px-2 py-1 text-xs font-medium text-gray-800 hover:bg-white" @click="useAsReference(img)">
                      {{ t('imagePlayground.actionEdit') }}
                    </button>
                    <button type="button" class="pointer-events-auto rounded-lg bg-white/90 px-2 py-1 text-xs font-medium text-gray-800 hover:bg-white" @click="openMaskEditorFromResult(img)">
                      {{ t('imagePlayground.actionMask') }}
                    </button>
                    <button type="button" :disabled="downloadingImage" class="pointer-events-auto rounded-lg bg-white/90 px-2 py-1 text-xs font-medium text-gray-800 hover:bg-white disabled:cursor-not-allowed disabled:opacity-60" @click="downloadImage(img)">
                      {{ t('imagePlayground.actionDownload') }}
                    </button>
                  </div>
                  <p v-if="img.revised_prompt" class="mt-1 line-clamp-2 text-[10px] text-white/80">{{ img.revised_prompt }}</p>
                </div>
              </div>
            </div>
            <div v-else class="flex flex-col items-center gap-3 py-16 text-center">
              <Icon name="beaker" size="xl" class="text-gray-300 dark:text-dark-600" />
              <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('imagePlayground.noResults') }}</p>
            </div>
          </div>

          <!-- 画廊 -->
          <div v-else class="space-y-4">
            <div v-if="gallery.length" class="card">
              <div class="grid grid-cols-1 gap-4 p-6 sm:grid-cols-2 xl:grid-cols-3">
                <div
                  v-for="record in gallery"
                  :key="record.id"
                  class="overflow-hidden rounded-xl border border-gray-100 bg-gray-50 dark:border-dark-700 dark:bg-dark-800"
                >
                  <img
                    :src="record.images[0]?.dataUrl || record.images[0]?.url"
                    alt=""
                    class="aspect-square w-full cursor-pointer object-cover"
                    loading="lazy"
                    @click="previewRecord = record"
                  />
                  <div class="flex items-center justify-between gap-2 p-3">
                    <div class="min-w-0">
                      <p class="truncate text-xs text-gray-600 dark:text-dark-300">{{ record.prompt }}</p>
                      <p class="mt-0.5 text-[10px] text-gray-400">
                        {{ record.model }} · {{ formatDateTime(new Date(record.createdAt)) }}
                      </p>
                    </div>
                    <div class="flex flex-shrink-0 items-center gap-1.5">
                      <button
                        type="button"
                        class="text-sm transition"
                        :class="record.favorite ? 'text-amber-500' : 'text-gray-400 hover:text-amber-500'"
                        @click="toggleFavorite(record)"
                      >
                        <Icon name="sparkles" size="md" />
                      </button>
                      <button
                        type="button"
                        class="text-sm text-gray-400 transition hover:text-red-500"
                        @click="removeRecord(record)"
                      >
                        <Icon name="trash" size="md" />
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <div v-else class="card flex flex-col items-center gap-3 py-16 text-center">
              <Icon name="grid" size="xl" class="text-gray-300 dark:text-dark-600" />
              <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('imagePlayground.noGallery') }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 遮罩编辑器 -->
    <MaskEditor
      :show="maskEditorOpen"
      :image="maskEditorImage"
      :title="t('imagePlayground.maskTitle')"
      @close="maskEditorOpen = false"
      @confirm="onMaskConfirm"
    />

    <!-- 画廊大图预览 -->
    <BaseDialog :show="previewRecord !== null" :title="previewRecord?.prompt || ''" width="wide" @close="previewRecord = null">
      <div class="p-6">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <img
            v-for="(img, idx) in previewRecord?.images || []"
            :key="idx"
            :src="img.dataUrl || img.url"
            alt=""
            class="w-full rounded-xl border border-gray-100 object-cover dark:border-dark-700"
          />
        </div>
        <div class="mt-4 flex justify-end gap-2">
          <button type="button" :disabled="downloadingImage" class="btn btn-secondary" @click="downloadImage(previewRecord!.images[0])">
            {{ t('imagePlayground.actionDownload') }}
          </button>
          <button type="button" class="btn btn-primary" @click="useAsReference(previewRecord!.images[0]); previewRecord = null">
            {{ t('imagePlayground.actionEdit') }}
          </button>
        </div>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { keysAPI } from '@/api'
import {
  dataUrlToBlob,
  editImageAsync,
  imageToBlob,
  generateImageAsync,
  listAvailableModels,
  normalizeImageTaskResult,
  pollImageTask,
  downloadGeneratedImage,
  type GeneratedImage,
} from '@/api/imagePlayground'
import {
  galleryAdd,
  galleryList,
  galleryRemove,
  galleryToggleFavorite,
  type GalleryRecord,
} from '@/lib/imageGallery'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ImageUpload from '@/components/common/ImageUpload.vue'
import Toggle from '@/components/common/Toggle.vue'
import MaskEditor from '@/components/imagePlayground/MaskEditor.vue'
import { formatDateTime } from '@/utils/format'
import type { ApiKey } from '@/types'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

// ---------- 尺寸预设（与原版 GPT Image Playground 一致） ----------
export type SizeTier = '1K' | '2K' | '4K'
export type SizeRatio = '1:1' | '3:2' | '2:3' | '16:9' | '9:16' | '4:3' | '3:4' | '21:9'

const SIZE_RATIOS: SizeRatio[] = ['1:1', '3:2', '2:3', '16:9', '9:16', '4:3', '3:4', '21:9']
const SIZE_PRESETS: Record<SizeTier, Record<SizeRatio, string>> = {
  '1K': {
    '1:1': '1024x1024',
    '3:2': '1536x1024',
    '2:3': '1024x1536',
    '16:9': '1280x720',
    '9:16': '720x1280',
    '4:3': '1024x768',
    '3:4': '768x1024',
    '21:9': '1280x544',
  },
  '2K': {
    '1:1': '2048x2048',
    '3:2': '2160x1440',
    '2:3': '1440x2160',
    '16:9': '2560x1440',
    '9:16': '1440x2560',
    '4:3': '2048x1536',
    '3:4': '1536x2048',
    '21:9': '2560x1088',
  },
  '4K': {
    '1:1': '2880x2880',
    '3:2': '3456x2304',
    '2:3': '2304x3456',
    '16:9': '3840x2160',
    '9:16': '2160x3840',
    '4:3': '3200x2400',
    '3:4': '2400x3200',
    '21:9': '3840x1600',
  },
}

// 拉取 /v1/models 失败时的内置图片模型清单
const FALLBACK_IMAGE_MODELS = [
  'gpt-image-1',
  'gpt-image-2',
  'dall-e-3',
  'gemini-2.5-flash-image',
  'gemini-3-pro-image',
  'imagen-3.0-generate-002',
]

// ---------- 状态 ----------
const keys = ref<ApiKey[]>([])
const selectedKeyId = ref<number | ''>('')
const model = ref('gpt-image-1')
const mode = ref<'generation' | 'edit'>('generation')
const prompt = ref('')
const referenceImage = ref('')
const referenceBlob = ref<Blob | null>(null)
const maskBlob = ref<Blob | null>(null)
const count = ref(1)
const sizeTier = ref<'auto' | SizeTier>('1K')
const sizeRatio = ref<SizeRatio>('1:1')
const quality = ref<'auto' | 'low' | 'medium' | 'high'>('high')
const outputFormat = ref<'png' | 'jpeg' | 'webp'>('png')
const outputCompression = ref(90)
const transparentOutput = ref(false)
const moderation = ref('auto')
const seed = ref<number | null>(null)
const generating = ref(false)
const errorMessage = ref('')
const results = ref<GeneratedImage[]>([])
const gallery = ref<GalleryRecord[]>([])
const galleryTab = ref<'results' | 'gallery'>('results')
const maskEditorOpen = ref(false)
const maskEditorImage = ref('')
const previewRecord = ref<GalleryRecord | null>(null)
const modelOptions = ref<string[]>([])
const generationElapsedSeconds = ref(0)
const lastGenerationDurationSeconds = ref<number | null>(null)

let pollTimer: ReturnType<typeof setTimeout> | null = null
let elapsedTimer: ReturnType<typeof setInterval> | null = null
let requestController: AbortController | null = null
let generationSequence = 0
let generationStartedAt: number | null = null
const MAX_POLL_DURATION_MS = 30 * 60 * 1000

// ---------- 计算属性 ----------
const usableKeys = computed(() =>
  keys.value.filter((k) => k.status === 'active' && (k.group ? k.group.allow_image_generation !== false : true)),
)

const selectedKey = computed(() => keys.value.find((k) => k.id === selectedKeyId.value))

/** 由档位 + 比例计算最终尺寸；auto 表示交给模型决定 */
const resolvedSize = computed(() => {
  if (sizeTier.value === 'auto') return 'auto'
  return SIZE_PRESETS[sizeTier.value][sizeRatio.value]
})

// ---------- 数据加载 ----------
async function loadKeys() {
  try {
    const page = await keysAPI.list(1, 100, { status: 'active' })
    keys.value = page.items
    if (usableKeys.value.length && !selectedKeyId.value) {
      selectedKeyId.value = usableKeys.value[0].id
      await onKeyChange()
    }
  } catch (error) {
    console.error('Failed to load API keys:', error)
    appStore.showError(t('imagePlayground.keyLoadFailed'))
  }
}

async function onKeyChange() {
  modelOptions.value = []
  if (!selectedKey.value) return
  try {
    const all = await listAvailableModels(selectedKey.value.key)
    // 优先过滤图片模型并自动选中第一个；没有图片模型时展示完整列表但保持当前选择
    const imageModels = all.filter((m) => /image|dall-e|gpt-image|imagen|flux/i.test(m))
    if (imageModels.length) {
      modelOptions.value = imageModels
      model.value = imageModels[0]
    } else if (all.length) {
      modelOptions.value = all.slice(0, 50)
    }
  } catch {
    // 拉取失败：回退到内置图片模型清单
    modelOptions.value = [...FALLBACK_IMAGE_MODELS]
  }
}

async function loadGallery() {
  try {
    gallery.value = await galleryList()
  } catch (error) {
    console.error('Failed to load gallery:', error)
  }
}

// ---------- 生成 ----------
function formatGenerationDuration(seconds: number, type: 'elapsed' | 'completed'): string {
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  const key = minutes > 0 ? `generation${type === 'elapsed' ? 'Elapsed' : 'Completed'}Minutes` : `generation${type === 'elapsed' ? 'Elapsed' : 'Completed'}Seconds`
  return t(`imagePlayground.${key}`, { minutes, seconds: remainingSeconds })
}

function updateGenerationElapsed() {
  if (generationStartedAt === null) return
  generationElapsedSeconds.value = Math.floor((Date.now() - generationStartedAt) / 1000)
}

function stopGenerationClock(clearCurrent = true) {
  if (elapsedTimer) {
    clearInterval(elapsedTimer)
    elapsedTimer = null
  }
  generationStartedAt = null
  if (clearCurrent) generationElapsedSeconds.value = 0
}

function startGenerationClock() {
  stopGenerationClock()
  generationStartedAt = Date.now()
  generationElapsedSeconds.value = 0
  elapsedTimer = setInterval(updateGenerationElapsed, 1000)
}

function stopImageTaskPolling() {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
  requestController?.abort()
  requestController = null
}

async function saveGeneratedResult(
  result: { data: GeneratedImage[] },
  snapshot: {
    apiKeyId: number
    prompt: string
    model: string
    size: string
    quality: typeof quality.value
    mode: typeof mode.value
  },
  isCurrent: () => boolean,
) {
  if (!isCurrent()) return
  results.value = result.data
  galleryTab.value = 'results'

  if (result.data.length) {
    await galleryAdd({
      id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      apiKeyId: snapshot.apiKeyId,
      prompt: snapshot.prompt,
      model: snapshot.model,
      size: snapshot.size,
      quality: snapshot.quality,
      mode: snapshot.mode,
      images: result.data.map((d) => ({
        dataUrl: d.dataUrl || '',
        url: d.url,
        proxyUrl: d.proxyUrl,
        apiKeyId: snapshot.apiKeyId,
        mimeType: d.mimeType,
        revised_prompt: d.revised_prompt,
      })),
      favorite: false,
      createdAt: Date.now(),
    })
    if (!isCurrent()) return
    await loadGallery()
  }
}

async function handleGenerate() {
  if (!selectedKey.value || !prompt.value.trim()) return
  if (mode.value === 'edit' && !referenceBlob.value) return

  stopImageTaskPolling()
  const sequence = ++generationSequence
  const controller = new AbortController()
  requestController = controller
  const snapshot = {
    apiKey: selectedKey.value.key,
    apiKeyId: selectedKey.value.id,
    prompt: prompt.value.trim(),
    model: model.value.trim(),
    size: resolvedSize.value,
    quality: quality.value,
    mode: mode.value,
    count: count.value,
    outputFormat: outputFormat.value,
    outputCompression: outputCompression.value,
    moderation: moderation.value,
    transparentOutput: transparentOutput.value,
    seed: seed.value ?? undefined,
    image: referenceBlob.value,
    mask: maskBlob.value,
  }
  let deadline = Date.now() + MAX_POLL_DURATION_MS
  lastGenerationDurationSeconds.value = null
  generating.value = true
  errorMessage.value = ''
  startGenerationClock()

  const isCurrent = () => sequence === generationSequence
  const finishWithError = (message: string) => {
    if (!isCurrent()) return
    stopImageTaskPolling()
    stopGenerationClock()
    errorMessage.value = message
    appStore.showError(t('imagePlayground.generateFailed'))
    generating.value = false
  }

  const poll = async (taskId: string, delaySeconds: number): Promise<void> => {
    if (!isCurrent()) return
    const remaining = deadline - Date.now()
    if (remaining <= 0) {
      finishWithError(t('imagePlayground.generateTimeout'))
      return
    }
    pollTimer = setTimeout(async () => {
      pollTimer = null
      if (!isCurrent()) return
      try {
        const response = await pollImageTask(snapshot.apiKey, taskId, controller.signal)
        if (!isCurrent()) return
        const task = response.task
        if (task.status === 'processing') {
          await poll(task.task_id || task.id, response.retryAfterSeconds)
          return
        }
        if (task.status === 'completed') {
          const result = normalizeImageTaskResult(task)
          result.data = result.data.map((image) => ({ ...image, apiKeyId: snapshot.apiKeyId }))
          await saveGeneratedResult(result, snapshot, isCurrent)
          if (!isCurrent()) return
          updateGenerationElapsed()
          lastGenerationDurationSeconds.value = generationElapsedSeconds.value
          stopGenerationClock(false)
          generating.value = false
          requestController = null
          appStore.showSuccess(t('imagePlayground.generateSuccess'))
          return
        }
        finishWithError(task.error?.message || t('imagePlayground.generateFailed'))
      } catch (error: any) {
        if (error?.name === 'AbortError' || !isCurrent()) return
        finishWithError(error?.message || t('imagePlayground.generateFailed'))
      }
    }, Math.min(Math.max(delaySeconds, 1) * 1000, remaining))
  }

  try {
    const taskResponse = snapshot.mode === 'generation'
      ? await generateImageAsync(snapshot.apiKey, {
          model: snapshot.model,
          prompt: snapshot.prompt,
          n: snapshot.count,
          size: snapshot.size,
          quality: snapshot.quality,
          output_format: snapshot.outputFormat,
          moderation: snapshot.moderation,
          seed: snapshot.seed,
          transparent_output: snapshot.transparentOutput,
          ...(snapshot.outputFormat !== 'png' ? { output_compression: snapshot.outputCompression } : {}),
        }, controller.signal)
      : await editImageAsync(snapshot.apiKey, {
          image: snapshot.image!,
          mask: snapshot.mask,
          prompt: snapshot.prompt,
          model: snapshot.model,
          n: snapshot.count,
          size: snapshot.size,
        }, controller.signal)
    if (!isCurrent()) return
    const taskExpiresAt = taskResponse.task.expires_at * 1000
    if (Number.isFinite(taskExpiresAt) && taskExpiresAt > 0) {
      deadline = Math.min(deadline, taskExpiresAt)
    }
    await poll(taskResponse.task.task_id || taskResponse.task.id, taskResponse.retryAfterSeconds)
  } catch (error: any) {
    if (error?.name === 'AbortError' || !isCurrent()) return
    const message = error?.status === 404
      ? t('imagePlayground.asyncUnavailable')
      : error?.message || t('imagePlayground.generateFailed')
    finishWithError(message)
  }
}

// ---------- 图片操作 ----------
function openResultsPreview() {
  if (!results.value.length) return
  previewRecord.value = {
    id: 'current-results',
    apiKeyId: results.value[0].apiKeyId,
    prompt: prompt.value,
    model: model.value,
    size: resolvedSize.value,
    quality: quality.value,
    mode: mode.value,
    images: results.value.map((image) => ({
      dataUrl: image.dataUrl || '',
      url: image.url,
      proxyUrl: image.proxyUrl,
      apiKeyId: image.apiKeyId,
      mimeType: image.mimeType,
      revised_prompt: image.revised_prompt,
    })),
    favorite: false,
    createdAt: Date.now(),
  }
}

async function useAsReference(img: GeneratedImage) {
  try {
    const keyId = img.apiKeyId
    const apiKey = keyId === undefined
      ? selectedKey.value?.key
      : keys.value.find((key) => key.id === keyId)?.key
    const blob = img.dataUrl
      ? dataUrlToBlob(img.dataUrl)
      : await imageToBlob(img, { apiKey })
    const dataUrl = await new Promise<string>((resolve, reject) => {
      const reader = new FileReader()
      reader.onload = () => resolve(String(reader.result))
      reader.onerror = () => reject(reader.error || new Error('failed to read image'))
      reader.readAsDataURL(blob)
    })
    referenceImage.value = dataUrl
    referenceBlob.value = blob
    maskBlob.value = null
    mode.value = 'edit'
    prompt.value = ''
  } catch (error) {
    console.error('Failed to use image as reference:', error)
    appStore.showError(t('imagePlayground.referenceImageFailed'))
  }
}

watch(referenceImage, (value) => {
  if (!value) {
    referenceBlob.value = null
  } else if (!referenceBlob.value && value.startsWith('data:image/')) {
    referenceBlob.value = dataUrlToBlob(value)
  }
})

async function openMaskEditorFromResult(img: GeneratedImage) {
  await useAsReference(img)
  if (!referenceImage.value) return
  maskEditorImage.value = referenceImage.value
  maskEditorOpen.value = true
}

function openMaskEditor() {
  if (!referenceImage.value) return
  maskEditorImage.value = referenceImage.value
  maskEditorOpen.value = true
}

function onMaskConfirm(mask: Blob) {
  maskBlob.value = mask
  maskEditorOpen.value = false
}

const downloadingImage = ref(false)

async function downloadImage(img: GeneratedImage) {
  if (downloadingImage.value) return
  downloadingImage.value = true
  try {
    const keyId = img.apiKeyId
    const apiKey = keyId === undefined
      ? selectedKey.value?.key
      : keys.value.find((key) => key.id === keyId)?.key
    await downloadGeneratedImage(img, { apiKey })
  } catch (error) {
    console.error('Failed to download image:', error)
    appStore.showError(t('imagePlayground.downloadFailed'))
  } finally {
    downloadingImage.value = false
  }
}

async function toggleFavorite(record: GalleryRecord) {
  const next = !record.favorite
  record.favorite = next
  await galleryToggleFavorite(record.id, next)
}

async function removeRecord(record: GalleryRecord) {
  await galleryRemove(record.id)
  await loadGallery()
}

// ---------- 初始化 ----------
onMounted(() => {
  loadKeys()
  loadGallery()
})

onUnmounted(() => {
  generationSequence++
  stopImageTaskPolling()
  stopGenerationClock()
})
</script>
