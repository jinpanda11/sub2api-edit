<template>
  <div class="space-y-6">
    <!-- 标题区 -->
    <div class="card">
      <div class="flex flex-col gap-4 p-6 sm:flex-row sm:items-center sm:justify-between">
        <div class="flex items-center gap-3">
          <div
            class="flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-br from-primary-500 to-primary-600 text-white shadow-md shadow-primary-500/25"
          >
            <Icon name="trophy" size="lg" />
          </div>
          <div>
            <h1 class="text-xl font-bold text-gray-900 dark:text-white">{{ t('ranking.title') }}</h1>
            <p class="mt-0.5 text-sm text-gray-500 dark:text-gray-400">
              {{ t('ranking.dateLabel') }} {{ ranking?.date || '' }}
            </p>
          </div>
        </div>
        <button class="btn btn-secondary" :disabled="loading" @click="fetchRanking(false)">
          <Icon name="refresh" size="md" class="mr-2" />
          {{ loading ? t('ranking.loading') : t('ranking.refresh') }}
        </button>
      </div>
    </div>

    <!-- 榜单 -->
    <div class="card">
      <div v-if="loading && !ranking" class="space-y-3 p-6">
        <div v-for="i in 5" :key="i" class="h-12 animate-pulse rounded-xl bg-gray-100 dark:bg-dark-800"></div>
      </div>

      <div v-else-if="loadFailed" class="flex flex-col items-center gap-3 py-12 text-center">
        <Icon name="exclamationCircle" size="xl" class="text-gray-400" />
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('ranking.loadFailed') }}</p>
        <button class="btn btn-secondary" @click="fetchRanking(false)">{{ t('ranking.retry') }}</button>
      </div>

      <div v-else-if="ranking && ranking.enabled === false" class="flex flex-col items-center gap-3 py-12 text-center">
        <Icon name="trophy" size="xl" class="text-gray-300 dark:text-dark-600" />
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('ranking.disabled') }}</p>
      </div>

      <div v-else-if="!ranking || ranking.list.length === 0" class="flex flex-col items-center gap-3 py-12 text-center">
        <Icon name="trophy" size="xl" class="text-gray-300 dark:text-dark-600" />
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('ranking.empty') }}</p>
      </div>

      <ul v-else class="divide-y divide-gray-100 dark:divide-dark-700">
        <li v-for="entry in ranking.list" :key="entry.rank" class="flex items-center gap-4 px-6 py-4">
          <span
            class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full text-sm font-bold ring-2"
            :class="rankMedal(entry.rank)"
          >
            {{ entry.rank }}
          </span>
          <span class="min-w-0 flex-1 truncate text-sm font-medium text-gray-800 dark:text-gray-100">
            {{ entry.masked_email }}
          </span>
          <span class="flex flex-shrink-0 items-center gap-4">
            <span class="font-semibold tabular-nums text-pink-500 dark:text-pink-400">
              {{ formatTokens(entry.total_tokens) }} {{ t('ranking.tokens') }}
            </span>
            <span
              class="font-semibold tabular-nums"
              :class="entry.rank <= 3 ? 'text-primary-600 dark:text-primary-400' : 'text-gray-700 dark:text-dark-200'"
            >
              US${{ entry.amount.toFixed(4) }}
            </span>
          </span>
        </li>
      </ul>
    </div>

    <!-- 底部说明 -->
    <div class="card">
      <div class="flex items-start gap-3 p-4">
        <Icon name="infoCircle" size="md" class="mt-0.5 flex-shrink-0 text-gray-400" />
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('ranking.footerNote') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { rankingAPI, type ConsumptionRanking } from '@/api'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

const ranking = ref<ConsumptionRanking | null>(null)
const loading = ref(true)
const loadFailed = ref(false)
let pollTimer: ReturnType<typeof setInterval> | null = null

const rankMedal = (rank: number) => {
  if (rank === 1) return 'bg-amber-400 text-amber-950 ring-amber-300/60'
  if (rank === 2) return 'bg-gray-300 text-gray-800 ring-gray-300/50'
  if (rank === 3) return 'bg-orange-300 text-orange-950 ring-orange-300/50'
  return 'bg-gray-100 text-gray-500 ring-transparent dark:bg-dark-800 dark:text-dark-400'
}

/** 大数字友好显示：1234 → 1.2K，1234567 → 1.2M，1234567890 → 1.2B（1000K=1M，1000M=1B） */
const formatTokens = (tokens: number) => {
  if (!tokens) return '0'
  if (tokens >= 1_000_000_000) return `${(tokens / 1_000_000_000).toFixed(1)}B`
  if (tokens >= 1_000_000) return `${(tokens / 1_000_000).toFixed(1)}M`
  if (tokens >= 1_000) return `${(tokens / 1_000).toFixed(1)}K`
  return String(tokens)
}

const fetchRanking = async (silent = false) => {
  if (!silent) loading.value = true
  try {
    ranking.value = await rankingAPI.getConsumptionRanking()
    loadFailed.value = false
  } catch (error) {
    console.error('Failed to load consumption ranking:', error)
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchRanking()
  // 准实时刷新：60s 轮询
  pollTimer = setInterval(() => fetchRanking(true), 60000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>
