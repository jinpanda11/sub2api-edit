import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  dataUrlToBlob,
  downloadGeneratedImage,
  imageToBlob,
  editImageAsync,
  generateImage,
  generateImageAsync,
  normalizeImageTaskResult,
  pollImageTask,
} from './imagePlayground'

const originalFetch = globalThis.fetch

function response(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
}

afterEach(() => {
  globalThis.fetch = originalFetch
  vi.restoreAllMocks()
})

describe('image playground response handling', () => {
  it('preserves the upstream image MIME for base64 results', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(response({
      created: 1,
      data: [{ b64_json: 'aGVsbG8=', mime_type: 'image/webp' }],
    }))

    const result = await generateImage('sk-test', { model: 'gpt-image-2', prompt: 'cat' })

    expect(result.data[0].mimeType).toBe('image/webp')
    expect(result.data[0].dataUrl).toBe('data:image/webp;base64,aGVsbG8=')
    expect((globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][1]).toMatchObject({
      headers: { Authorization: 'Bearer sk-test', 'Content-Type': 'application/json' },
    })
  })

  it('normalizes a b64_json value that already contains a data URL prefix', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(response({
      created: 1,
      data: [{ b64_json: 'data:image/png;base64,aGVsbG8=' }],
    }))

    const result = await generateImage('sk-test', { model: 'gpt-image-2', prompt: 'cat' })

    expect(result.data[0].dataUrl).toBe('data:image/png;base64,aGVsbG8=')
  })

  it('uses b64_json-compatible normalization when the upstream omits MIME metadata', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(response({
      created: 1,
      data: [{ b64_json: 'aGVsbG8=' }],
    }))

    const result = await generateImage('sk-test', { model: 'gpt-image-2', prompt: 'cat' })

    expect(result.data[0].dataUrl).toBe('data:image/png;base64,aGVsbG8=')
    expect(result.data[0].mimeType).toBe('image/png')
  })

  it('keeps URL-only results available as a download source', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(response({
      created: 1,
      data: [{ url: 'https://cdn.example/image.jpg', output_format: 'jpeg', proxy_url: '/v1/images/tasks/imgtask_1/images/0' }],
    }))

    const result = await generateImage('sk-test', { model: 'gpt-image-2', prompt: 'cat' })

    expect(result.data[0].url).toBe('https://cdn.example/image.jpg')
    expect(result.data[0].mimeType).toBe('image/jpeg')
  })
})

describe('async image tasks', () => {
  it('submits generation tasks and reads the retry interval', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(response({ id: 'imgtask_1', task_id: 'imgtask_1', status: 'processing' }, {
      status: 202,
      headers: { 'Content-Type': 'application/json', 'Retry-After': '5' },
    }))

    const result = await generateImageAsync('sk-test', { model: 'gpt-image-2', prompt: 'cat' })

    expect(result.task.task_id).toBe('imgtask_1')
    expect(result.retryAfterSeconds).toBe(5)
    expect((globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toContain('/v1/images/generations/async')
    expect((globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][1]).toMatchObject({
      method: 'POST',
      headers: { Authorization: 'Bearer sk-test', 'Content-Type': 'application/json' },
    })
  })

  it('submits edit tasks as multipart requests', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(response({ id: 'imgtask_2', task_id: 'imgtask_2', status: 'processing' }, { status: 202 }))
    const image = new Blob(['image'], { type: 'image/png' })

    const result = await editImageAsync('sk-test', { image, prompt: 'edit', model: 'gpt-image-2' })
    const request = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0]

    expect(result.task.id).toBe('imgtask_2')
    expect(request[0]).toContain('/v1/images/edits/async')
    expect(request[1]).toMatchObject({ method: 'POST', headers: { Authorization: 'Bearer sk-test' } })
    expect(request[1].body).toBeInstanceOf(FormData)
  })

  it('polls and normalizes a completed task result', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(response({
      id: 'imgtask_3', task_id: 'imgtask_3', status: 'completed', created_at: 1, expires_at: 2,
      result: { created: 1, data: [{ b64_json: 'data:image/webp;base64,aGVsbG8=', mime_type: 'image/webp', proxy_url: '/v1/images/tasks/imgtask_3/images/0' }] },
    }))

    const task = await pollImageTask('sk-test', 'imgtask_3')
    const result = normalizeImageTaskResult(task.task)

    expect(task.retryAfterSeconds).toBe(3)
    expect(result.data[0].dataUrl).toBe('data:image/webp;base64,aGVsbG8=')
    expect(result.data[0].proxyUrl).toBe('/v1/images/tasks/imgtask_3/images/0')
    expect((globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toContain('/v1/images/tasks/imgtask_3')
  })

  it('exposes async feature errors with the HTTP status', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(response({ error: { type: 'not_found_error', message: 'async image tasks are not enabled' } }, { status: 404 }))

    await expect(generateImageAsync('sk-test', { model: 'gpt-image-2', prompt: 'cat' })).rejects.toMatchObject({
      status: 404,
      message: 'async image tasks are not enabled',
    })
  })
})

describe('image playground downloads', () => {
  it('converts data URLs to typed blobs', () => {
    const blob = dataUrlToBlob('data:image/jpeg;base64,aGVsbG8=')
    expect(blob.type).toBe('image/jpeg')
  })

  it('downloads a same-origin proxy image with the API key', async () => {
    const imageBlob = new Blob(['image'], { type: 'image/jpeg' })
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      blob: vi.fn().mockResolvedValue(imageBlob),
    })

    await imageToBlob({ proxyUrl: '/v1/images/tasks/imgtask_1/images/0', url: 'https://r2.example/image.jpg' }, { apiKey: 'sk-test' })

    expect(globalThis.fetch).toHaveBeenCalledWith('/v1/images/tasks/imgtask_1/images/0', {
      headers: { Authorization: 'Bearer sk-test' },
      cache: 'no-store',
    })
  })

  it('downloads a remote image as a local blob', async () => {
    const imageBlob = new Blob(['image'], { type: 'image/jpeg' })
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      blob: vi.fn().mockResolvedValue(imageBlob),
    })
    const createObjectURL = vi.fn(() => 'blob:image')
    const revokeObjectURL = vi.fn()
    Object.defineProperty(URL, 'createObjectURL', { configurable: true, value: createObjectURL })
    Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: revokeObjectURL })
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    await downloadGeneratedImage({ url: 'https://cdn.example/image.jpg', mimeType: 'image/jpeg' })

    expect(globalThis.fetch).toHaveBeenCalledWith('https://cdn.example/image.jpg', {
      headers: undefined,
      cache: 'no-store',
    })
    expect(createObjectURL).toHaveBeenCalledWith(imageBlob)
    expect(click).toHaveBeenCalledOnce()
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:image')
  })

  it('rejects non-image remote responses', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(new Response('challenge', {
      status: 200,
      headers: { 'Content-Type': 'text/html' },
    }))

    await expect(imageToBlob({ url: 'https://cdn.example/image.jpg' })).rejects.toThrow('not an image')
  })
})
