/**
 * Daily consumption ranking (public) API
 */

import { apiClient } from './client'

export interface ConsumptionRankingEntry {
  rank: number
  masked_email: string
  amount: number
  total_tokens: number
}

export interface ConsumptionRanking {
  date: string
  timezone: string
  enabled: boolean
  list: ConsumptionRankingEntry[]
  updated_at: string
}

/**
 * Get today's top-20 consumption ranking (public, no auth required)
 */
export async function getConsumptionRanking(): Promise<ConsumptionRanking> {
  const { data } = await apiClient.get<ConsumptionRanking>('/public/ranking/consumption')
  return data
}

export const rankingAPI = {
  getConsumptionRanking
}

export default rankingAPI
