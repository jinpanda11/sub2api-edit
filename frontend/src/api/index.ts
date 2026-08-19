/**
 * API Client for Sub2API Backend
 * Central export point for all API modules
 */

// Re-export the HTTP client
export { apiClient } from './client'

// Auth API
export { authAPI, isTotp2FARequired, type LoginResponse } from './auth'

// User APIs
export { keysAPI } from './keys'
export { usageAPI } from './usage'
export { userAPI } from './user'
export { redeemAPI, type RedeemHistoryItem } from './redeem'
export { lotteryAPI, type LotteryStatus, type LotteryDrawResult, type LotteryRecordItem } from './lottery'
export { rankingAPI, type ConsumptionRanking, type ConsumptionRankingEntry } from './ranking'
export { generateImage, editImage, listAvailableModels, type ImageGenerateResult, type GeneratedImage } from './imagePlayground'
export { paymentAPI } from './payment'
export { userGroupsAPI } from './groups'
export { userChannelsAPI } from './channels'
export * as batchImageAPI from './batchImage'
export { totpAPI } from './totp'
export { passkeyAPI, type PasskeyCredentialSummary } from './passkey'
export { default as announcementsAPI } from './announcements'
export { channelMonitorUserAPI } from './channelMonitor'

// Admin APIs
export { adminAPI } from './admin'

// Default export
export { default } from './client'
