import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const { generateImageAsync, pollImageTask, normalizeImageTaskResult, listAvailableModels } = vi.hoisted(() => ({
  generateImageAsync: vi.fn(),
  pollImageTask: vi.fn(),
  normalizeImageTaskResult: vi.fn(),
  listAvailableModels: vi.fn(),
}))
const { keysList } = vi.hoisted(() => ({ keysList: vi.fn() }))
const { galleryList, galleryAdd } = vi.hoisted(() => ({ galleryList: vi.fn(), galleryAdd: vi.fn() }))
const { showError, showSuccess } = vi.hoisted(() => ({ showError: vi.fn(), showSuccess: vi.fn() }))

vi.mock('@/api', () => ({ keysAPI: { list: keysList } }))
vi.mock('@/api/imagePlayground', () => ({
  dataUrlToBlob: vi.fn((value: string) => new Blob([value], { type: 'image/png' })),
  downloadGeneratedImage: vi.fn(),
  editImageAsync: vi.fn(),
  generateImageAsync,
  listAvailableModels,
  normalizeImageTaskResult,
  pollImageTask,
}))
vi.mock('@/lib/imageGallery', () => ({
  galleryList,
  galleryAdd,
  galleryRemove: vi.fn(),
  galleryToggleFavorite: vi.fn(),
}))
vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ user: { balance: 10 } }),
}))
vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess }),
}))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, number>) => {
        if (!params) return key
        return `${key}:${Object.entries(params).map(([name, value]) => `${name}=${value}`).join(',')}`
      },
    }),
  }
})

import ImagePlaygroundView from '../ImagePlaygroundView.vue'

const key = { id: 1, key: 'sk-test', name: 'Test key', status: 'active', group: { allow_image_generation: true } }
const processingTask = { id: 'imgtask_1', task_id: 'imgtask_1', object: 'image.generation.task', status: 'processing', created_at: 1, expires_at: 9999999999 }
const completedTask = { ...processingTask, status: 'completed', result: { created: 1, data: [{ url: 'https://cdn.example/image.png' }] } }

function mountView() {
  return mount(ImagePlaygroundView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Icon: { template: '<span />' },
        BaseDialog: { template: '<div><slot /></div>' },
        ImageUpload: { template: '<div />' },
        Toggle: { template: '<input />' },
        MaskEditor: { template: '<div />' },
      },
    },
  })
}

describe('ImagePlaygroundView async generation', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    keysList.mockReset().mockResolvedValue({ items: [key] })
    galleryList.mockReset().mockResolvedValue([])
    galleryAdd.mockReset().mockResolvedValue(undefined)
    listAvailableModels.mockReset().mockResolvedValue(['gpt-image-2'])
    generateImageAsync.mockReset()
    pollImageTask.mockReset()
    normalizeImageTaskResult.mockReset().mockReturnValue({
      created: 1,
      data: [{ url: 'https://cdn.example/image.png', mimeType: 'image/png' }],
    })
    showError.mockReset()
    showSuccess.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('submits an async task, polls processing, and renders the completed result', async () => {
    generateImageAsync.mockResolvedValue({ task: processingTask, retryAfterSeconds: 1 })
    pollImageTask.mockResolvedValue({ task: completedTask, retryAfterSeconds: 3 })

    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('textarea').setValue('a cat')
    await wrapper.get('button.btn-primary').trigger('click')
    await flushPromises()

    expect(generateImageAsync).toHaveBeenCalledOnce()
    expect(wrapper.get('button.btn-primary').text()).toContain('imagePlayground.generating')
    expect(pollImageTask).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(1000)
    await flushPromises()

    expect(pollImageTask).toHaveBeenCalledWith('sk-test', 'imgtask_1', expect.any(AbortSignal))
    expect(wrapper.find('img[alt="generated"]').exists()).toBe(true)
    expect(wrapper.get('button.btn-primary').text()).toContain('imagePlayground.generate')
    expect(galleryAdd).toHaveBeenCalledOnce()
    expect(showSuccess).toHaveBeenCalledWith('imagePlayground.generateSuccess')
  })

  it('shows elapsed time while processing and freezes the duration on completion', async () => {
    generateImageAsync.mockResolvedValue({ task: processingTask, retryAfterSeconds: 120 })
    pollImageTask.mockResolvedValue({ task: completedTask, retryAfterSeconds: 3 })

    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('textarea').setValue('a cat')
    await wrapper.get('button.btn-primary').trigger('click')
    await flushPromises()

    await vi.advanceTimersByTimeAsync(61_000)
    await flushPromises()
    expect(wrapper.text()).toContain('imagePlayground.generationElapsedMinutes:minutes=1,seconds=1')
    expect(pollImageTask).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(59_000)
    await flushPromises()
    expect(pollImageTask).toHaveBeenCalledOnce()
    expect(wrapper.text()).toContain('imagePlayground.generationCompletedMinutes:minutes=2,seconds=0')
    expect(wrapper.text()).not.toContain('imagePlayground.generationElapsedMinutes')

    await vi.advanceTimersByTimeAsync(10_000)
    expect(wrapper.text()).toContain('imagePlayground.generationCompletedMinutes:minutes=2,seconds=0')
  })

  it('shows a storage configuration message when async tasks are unavailable', async () => {
    const error = Object.assign(new Error('async image tasks are not enabled'), { status: 404 })
    generateImageAsync.mockRejectedValue(error)

    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('textarea').setValue('a cat')
    await wrapper.get('button.btn-primary').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('imagePlayground.asyncUnavailable')
    expect(wrapper.get('button.btn-primary').text()).toContain('imagePlayground.generate')
    expect(pollImageTask).not.toHaveBeenCalled()
  })
})
