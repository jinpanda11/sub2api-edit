/**
 * Admin Lottery Statistics API
 * Daily blind-box lottery stats for administrators.
 */

import { apiClient } from '../client'

export interface LotteryDailyAggregate {
  date: string
  draws: number
  participants: number
  total_amount: number
}

export interface LotteryDailySummary {
  total_draws: number
  total_participants: number
  total_amount: number
  avg_amount: number
}

export interface LotteryDailyStats {
  start_date: string
  end_date: string
  summary: LotteryDailySummary
  daily: LotteryDailyAggregate[]
}

/**
 * Get daily lottery statistics.
 * @param startDate Optional start date (YYYY-MM-DD)
 * @param endDate Optional end date (YYYY-MM-DD)
 */
export async function getDailyStats(startDate?: string, endDate?: string): Promise<LotteryDailyStats> {
  const { data } = await apiClient.get<LotteryDailyStats>('/admin/lottery/stats', {
    params: {
      ...(startDate ? { start_date: startDate } : {}),
      ...(endDate ? { end_date: endDate } : {})
    }
  })
  return data
}

export const adminLotteryAPI = {
  getDailyStats
}

export default adminLotteryAPI
