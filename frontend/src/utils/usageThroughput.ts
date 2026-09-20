import type { UsageLog } from '@/types'
import { isImageUsage } from '@/utils/billingMode'
import { textOutputTokens } from '@/utils/imageUsage'

export type UsageThroughputRow = Pick<
  UsageLog,
  'output_tokens' | 'image_output_tokens' | 'duration_ms' | 'first_token_ms' | 'image_count' | 'billing_mode'
>

/**
 * Calculate generation-phase output throughput, excluding time to first token.
 * Rows without a reliable text-generation window are intentionally omitted.
 */
export const calculateGenerationTokensPerSecond = (
  row: UsageThroughputRow | null | undefined,
): number | null => {
  if (!row || isImageUsage(row)) return null

  const outputTokens = textOutputTokens(row)
  const durationMs = row.duration_ms
  const firstTokenMs = row.first_token_ms
  if (
    !Number.isFinite(outputTokens) ||
    outputTokens <= 0 ||
    durationMs == null ||
    firstTokenMs == null ||
    !Number.isFinite(durationMs) ||
    !Number.isFinite(firstTokenMs)
  ) {
    return null
  }

  const generationMs = durationMs - firstTokenMs
  if (generationMs <= 0) return null

  const tokensPerSecond = outputTokens * 1000 / generationMs
  return Number.isFinite(tokensPerSecond) && tokensPerSecond > 0 ? tokensPerSecond : null
}

export const formatGenerationTokensPerSecond = (
  row: UsageThroughputRow | null | undefined,
): string => {
  const tokensPerSecond = calculateGenerationTokensPerSecond(row)
  return tokensPerSecond == null ? '—' : `${tokensPerSecond.toFixed(1)} t/s`
}
