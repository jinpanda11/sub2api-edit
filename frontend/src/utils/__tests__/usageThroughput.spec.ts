import { describe, expect, it } from 'vitest'

import {
  calculateGenerationTokensPerSecond,
  formatGenerationTokensPerSecond,
} from '@/utils/usageThroughput'

const textRow = (overrides: Record<string, unknown> = {}) => ({
  output_tokens: 191,
  image_output_tokens: 0,
  duration_ms: 4250,
  first_token_ms: 1760,
  image_count: 0,
  billing_mode: 'token',
  ...overrides,
})

describe('usage throughput', () => {
  it('calculates generation-phase tokens per second after first token', () => {
    expect(calculateGenerationTokensPerSecond(textRow())).toBeCloseTo(76.7068, 4)
    expect(formatGenerationTokensPerSecond(textRow())).toBe('76.7 t/s')
  })

  it.each([
    { name: 'missing first token time', row: textRow({ first_token_ms: null }) },
    { name: 'missing duration', row: textRow({ duration_ms: null }) },
    { name: 'zero output tokens', row: textRow({ output_tokens: 0 }) },
    { name: 'non-positive generation window', row: textRow({ duration_ms: 1760 }) },
    { name: 'image usage', row: textRow({ image_count: 1, billing_mode: 'image', output_tokens: 200 }) },
  ])('returns no rate for $name', ({ row }) => {
    expect(calculateGenerationTokensPerSecond(row)).toBeNull()
    expect(formatGenerationTokensPerSecond(row)).toBe('—')
  })

  it('excludes image output tokens from text throughput', () => {
    const row = textRow({ output_tokens: 200, image_output_tokens: 50 })
    expect(calculateGenerationTokensPerSecond(row)).toBeCloseTo(60.241, 3)
  })
})
