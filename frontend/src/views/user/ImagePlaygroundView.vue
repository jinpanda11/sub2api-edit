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
            <div v-if="results.length" class="grid grid-cols-1 gap-4 p-6 sm:grid-cols-2 xl:grid-cols-3">
              <div v-for="(img, idx) in results" :key="idx" class="group relative overflow-hidden rounded-xl border border-gray-100 bg-gray-50 dark:border-dark-700 dark:bg-dark-800">
                <img
                  :src="img.dataUrl || img.url"
                  alt="generated"
                  class="aspect-square w-full object-cover"
                  loading="lazy"
                />
                <!-- 操作层 -->
                <div class="absolute inset-0 flex flex-col justify-end gap-1 bg-gradient-to-t from-black/70 via-transparent to-transparent p-3 opacity-0 transition-opacity group-hover:opacity-100">
                  <div class="flex flex-wrap gap-1.5">
                    <button type="button" class="rounded-lg bg-white/90 px-2 py-1 text-xs font-medium text-gray-800 hover:bg-white" @click="useAsReference(img)">
                      {{ t('imagePlayground.actionEdit') }}
                    </button>
                    <button type="button" class="rounded-lg bg-white/90 px-2 py-1 text-xs font-medium text-gray-800 hover:bg-white" @click="openMaskEditorFromResult(img)">
                      {{ t('imagePlayground.actionMask') }}
                    </button>
                    <button type="button" class="rounded-lg bg-white/90 px-2 py-1 text-xs font-medium text-gray-800 hover:bg-white" @click="downloadImage(img)">
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
          <button type="button" class="btn btn-secondary" @click="downloadImage(previewRecord!.images[0])">
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
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { keysAPI } from '@/api'
import {
  editImage,
  generateImage,
  listAvailableModels,
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
function dataUrlToBlob(dataUrl: string): Blob {
  const [meta, base64] = dataUrl.split(',')
  const mime = meta.match(/data:(.*?);/)?.[1] || 'image/png'
  const bytes = atob(base64)
  const arr = new Uint8Array(bytes.length)
  for (let i = 0; i < bytes.length; i++) arr[i] = bytes.charCodeAt(i)
  return new Blob([arr], { type: mime })
}

async function handleGenerate() {
  if (!selectedKey.value || !prompt.value.trim()) return
  if (mode.value === 'edit' && !referenceBlob.value) return

  generating.value = true
  errorMessage.value = ''
  try {
    const commonParams = {
      model: model.value.trim(),
      prompt: prompt.value.trim(),
      n: count.value,
      size: resolvedSize.value,
      quality: quality.value,
      output_format: outputFormat.value,
      moderation: moderation.value,
      seed: seed.value ?? undefined,
    }
    let result
    if (mode.value === 'generation') {
      result = await generateImage(selectedKey.value.key, {
        ...commonParams,
        transparent_output: transparentOutput.value,
        ...(outputFormat.value !== 'png' ? { output_compression: outputCompression.value } : {}),
      })
    } else {
      result = await editImage(selectedKey.value.key, {
        image: referenceBlob.value!,
        mask: maskBlob.value,
        prompt: prompt.value.trim(),
        model: model.value.trim(),
        n: count.value,
        size: resolvedSize.value,
      })
    }
    results.value = result.data
    galleryTab.value = 'results'

    // 存入画廊（本地 IndexedDB）
    if (result.data.length) {
      await galleryAdd({
        id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
        prompt: prompt.value.trim(),
        model: model.value.trim(),
        size: resolvedSize.value,
        quality: quality.value,
        mode: mode.value,
        images: result.data.map((d) => ({ dataUrl: d.dataUrl || '', url: d.url, revised_prompt: d.revised_prompt })),
        favorite: false,
        createdAt: Date.now(),
      })
      await loadGallery()
    }
    appStore.showSuccess(t('imagePlayground.generateSuccess'))
  } catch (error: any) {
    errorMessage.value = error?.message || t('imagePlayground.generateFailed')
    appStore.showError(t('imagePlayground.generateFailed'))
  } finally {
    generating.value = false
  }
}

// ---------- 图片操作 ----------
function useAsReference(img: GeneratedImage) {
  const dataUrl = img.dataUrl
  if (!dataUrl) return
  referenceImage.value = dataUrl
  referenceBlob.value = dataUrlToBlob(dataUrl)
  maskBlob.value = null
  mode.value = 'edit'
  prompt.value = ''
}

function openMaskEditorFromResult(img: GeneratedImage) {
  if (!img.dataUrl) return
  useAsReference(img)
  maskEditorImage.value = img.dataUrl
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

function downloadImage(img: GeneratedImage) {
  const dataUrl = img.dataUrl
  if (!dataUrl) return
  const link = document.createElement('a')
  link.href = dataUrl
  link.download = `sub2api-image-${Date.now()}.png`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
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
</script>
