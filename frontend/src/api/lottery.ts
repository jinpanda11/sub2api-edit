/**
 * Daily blind-box lottery API endpoints
 */

import { apiClient } from './client'

export interface LotteryStatus {
  enabled: boolean
  available_count: number
  today_recharge_count: number
  today_recharge_max: number
  login_rewarded_today: boolean
  current_balance: number
  min_prize: number
  max_prize: number
  recharge_unit: number
  login_reward: number
  today: string
  timezone: string
}

export interface LotteryDrawResult {
  prize_amount: number
  new_balance: number
  remaining_count: number
}

export interface LotteryRecordItem {
  id: number
  user_id: number
  amount: number
  balance_after: number
  source: string
  created_at: string
}

export interface LotteryRecordsPage {
  items: LotteryRecordItem[]
  total: number
  page: number
  page_size: number
  pages: number
}

/**
 * Get lottery status (available chances, recharge progress, balance)
 */
export async function getStatus(): Promise<LotteryStatus> {
  const { data } = await apiClient.get<LotteryStatus>('/lottery/status')
  return data
}

/**
 * Draw once (consumes 1 chance, returns prize + new balance)
 */
export async function draw(): Promise<LotteryDrawResult> {
  const { data } = await apiClient.post<LotteryDrawResult>('/lottery/draw')
  return data
}

/**
 * Get paginated lottery history
 */
export async function getRecords(page = 1, pageSize = 20): Promise<LotteryRecordsPage> {
  const { data } = await apiClient.get<LotteryRecordsPage>('/lottery/records', {
    params: { page, page_size: pageSize }
  })
  return data
}

export const lotteryAPI = {
  getStatus,
  draw,
  getRecords
}

export default lotteryAPI
