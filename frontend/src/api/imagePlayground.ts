/**
 * Image playground API — 用用户自己的 API Key 直连 Sub2API 网关的
 * OpenAI 兼容图片接口（/v1/images/generations、/v1/images/edits）。
 * 计费由网关按 API Key 自动从用户余额扣除，无需额外后端。
 */

import { buildGatewayUrl } from './client'

export type ImageQuality = 'auto' | 'low' | 'medium' | 'high'
export type ImageOutputFormat = 'png' | 'jpeg' | 'webp'

export interface ImageGenerateParams {
  model: string
  prompt: string
  n?: number
  size?: string
  quality?: ImageQuality
  /** gpt-image 系列：输出格式 png/jpeg/webp（原样透传上游） */
  output_format?: ImageOutputFormat
  /** jpeg/webp 压缩率 0-100 */
  output_compression?: number
  /** 内容审核：auto | low */
  moderation?: string
  /** 透明背景（png） */
  transparent_output?: boolean
  /** 随机种子（固定可复现） */
  seed?: number
  response_format?: 'b64_json' | 'url'
}

export interface GeneratedImage {
  /** b64_json 时转成的 data URL；url 时直接使用网关返回的 URL */
  dataUrl?: string
  url?: string
  revised_prompt?: string
}

export interface ImageGenerateResult {
  created: number
  data: GeneratedImage[]
}

/** 图生图/遮罩编辑入参（multipart） */
export interface ImageEditInput {
  image: Blob
  mask?: Blob | null
  prompt: string
  model: string
  n?: number
  size?: string
}

function authHeaders(apiKey: string): HeadersInit {
  return { Authorization: `Bearer ${apiKey}` }
}

function normalizeGeneratedImage(item: { b64_json?: string; url?: string; revised_prompt?: string }): GeneratedImage {
  const out: GeneratedImage = { revised_prompt: item.revised_prompt }
  if (item.b64_json) {
    out.dataUrl = `data:image/png;base64,${item.b64_json}`
  } else if (item.url) {
    out.url = item.url
  }
  return out
}

/**
 * 获取用户 API Key 可用的模型列表（OpenAI 兼容 /v1/models）。
 * 返回模型 id 数组；失败时由调用方回退到内置模型清单。
 */
export async function listAvailableModels(apiKey: string): Promise<string[]> {
  const response = await fetch(buildGatewayUrl('/v1/models'), {
    headers: authHeaders(apiKey),
  })
  if (!response.ok) return []
  const result = await response.json()
  const data: Array<{ id?: string }> = result?.data || []
  return data.map((m) => m.id).filter((id): id is string => Boolean(id))
}

/**
 * 文生图（同步）：POST /v1/images/generations
 * 网关原样透传请求体（仅改写模型名），因此原版 Playground 的
 * output_format / output_compression / moderation / transparent_output / seed
 * 等参数均可直接下发。
 */
export async function generateImage(apiKey: string, params: ImageGenerateParams): Promise<ImageGenerateResult> {
  const response = await fetch(buildGatewayUrl('/v1/images/generations'), {
    method: 'POST',
    headers: authHeaders(apiKey),
    body: JSON.stringify({
      model: params.model,
      prompt: params.prompt,
      n: params.n ?? 1,
      size: params.size,
      quality: params.quality,
      output_format: params.output_format,
      output_compression: params.output_compression,
      moderation: params.moderation,
      transparent_output: params.transparent_output,
      seed: params.seed,
      response_format: params.response_format ?? 'b64_json',
    }),
  })
  if (!response.ok) {
    const body = await response.json().catch(() => null)
    const message = body?.error?.message || `HTTP ${response.status}`
    const error = new Error(message)
    ;(error as any).status = response.status
    ;(error as any).code = body?.error?.code
    throw error
  }
  const result: ImageGenerateResult = await response.json()
  result.data = (result.data || []).map(normalizeGeneratedImage)
  return result
}

/**
 * 图生图 / 遮罩编辑（同步）：POST /v1/images/edits（multipart）
 */
export async function editImage(apiKey: string, input: ImageEditInput): Promise<ImageGenerateResult> {
  const form = new FormData()
  form.append('image', input.image, 'input.png')
  if (input.mask) {
    form.append('mask', input.mask, 'mask.png')
  }
  form.append('prompt', input.prompt)
  form.append('model', input.model)
  form.append('n', String(input.n ?? 1))
  if (input.size) form.append('size', input.size)

  const response = await fetch(buildGatewayUrl('/v1/images/edits'), {
    method: 'POST',
    headers: authHeaders(apiKey),
    body: form,
  })
  if (!response.ok) {
    const body = await response.json().catch(() => null)
    const message = body?.error?.message || `HTTP ${response.status}`
    const error = new Error(message)
    ;(error as any).status = response.status
    ;(error as any).code = body?.error?.code
    throw error
  }
  const result: ImageGenerateResult = await response.json()
  result.data = (result.data || []).map(normalizeGeneratedImage)
  return result
}
