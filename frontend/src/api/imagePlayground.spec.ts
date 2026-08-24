import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  dataUrlToBlob,
  downloadGeneratedImage,
  generateImage,
  imageToBlob,
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
      data: [{ url: 'https://cdn.example/image.jpg', output_format: 'jpeg' }],
    }))

    const result = await generateImage('sk-test', { model: 'gpt-image-2', prompt: 'cat' })

    expect(result.data[0].url).toBe('https://cdn.example/image.jpg')
    expect(result.data[0].mimeType).toBe('image/jpeg')
  })
})

describe('image playground downloads', () => {
  it('converts data URLs to typed blobs', () => {
    const blob = dataUrlToBlob('data:image/jpeg;base64,aGVsbG8=')
    expect(blob.type).toBe('image/jpeg')
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

    expect(globalThis.fetch).toHaveBeenCalledWith('https://cdn.example/image.jpg')
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
