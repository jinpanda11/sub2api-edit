<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Date range filter -->
      <div class="card p-4 sm:p-5">
        <div class="flex flex-wrap items-end gap-4">
          <div class="w-full sm:w-auto sm:min-w-[180px]">
            <label class="input-label">{{ t('admin.lottery.startDate') }}</label>
            <input v-model="startDate" type="date" class="input" />
          </div>
          <div class="w-full sm:w-auto sm:min-w-[180px]">
            <label class="input-label">{{ t('admin.lottery.endDate') }}</label>
            <input v-model="endDate" type="date" class="input" />
          </div>
          <div class="flex w-full flex-wrap items-center gap-3 sm:w-auto">
            <button type="button" class="btn btn-secondary" :disabled="loading" @click="setLast30Days">
              {{ t('admin.lottery.days30') }}
            </button>
            <button type="button" class="btn btn-primary" :disabled="loading" @click="loadStats">
              <Icon name="refresh" size="sm" class="mr-1.5" />
              {{ t('common.search') }}
            </button>
          </div>
        </div>
      </div>

      <!-- Summary -->
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <div class="card p-4">
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-500/10 dark:text-primary-400">
              <Icon name="gift" size="md" />
            </div>
            <div class="min-w-0">
              <p class="truncate text-sm text-gray-500 dark:text-gray-400">{{ t('admin.lottery.totalDraws') }}</p>
              <p class="text-xl font-semibold text-gray-900 dark:text-white">{{ formatNumber(summary?.total_draws ?? 0) }}</p>
            </div>
          </div>
        </div>
        <div class="card p-4">
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-sky-50 text-sky-600 dark:bg-sky-500/10 dark:text-sky-400">
              <Icon name="users" size="md" />
            </div>
            <div class="min-w-0">
              <p class="truncate text-sm text-gray-500 dark:text-gray-400">{{ t('admin.lottery.totalParticipants') }}</p>
              <p class="text-xl font-semibold text-gray-900 dark:text-white">{{ formatNumber(summary?.total_participants ?? 0) }}</p>
            </div>
          </div>
        </div>
        <div class="card p-4">
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400">
              <Icon name="dollar" size="md" />
            </div>
            <div class="min-w-0">
              <p class="truncate text-sm text-gray-500 dark:text-gray-400">{{ t('admin.lottery.totalAmount') }}</p>
              <p class="text-xl font-semibold text-gray-900 dark:text-white">{{ formatCurrency(summary?.total_amount ?? 0) }}</p>
            </div>
          </div>
        </div>
        <div class="card p-4">
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-400">
              <Icon name="trendingUp" size="md" />
            </div>
            <div class="min-w-0">
              <p class="truncate text-sm text-gray-500 dark:text-gray-400">{{ t('admin.lottery.avgAmount') }}</p>
              <p class="text-xl font-semibold text-gray-900 dark:text-white">{{ formatCurrency(summary?.avg_amount ?? 0) }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Daily table -->
      <div class="card overflow-hidden">
        <div class="flex flex-wrap items-center justify-between gap-2 border-b border-gray-200 px-4 py-3 dark:border-dark-700">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.lottery.summary') }}</h3>
          <span class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.lottery.timezone') }}: {{ t('admin.lottery.timezoneValue') }}
          </span>
        </div>
        <DataTable :columns="columns" :data="dailyRows" :loading="loading" row-key="date">
          <template #cell-draws="{ value }">
            <span class="text-gray-900 dark:text-white">{{ formatNumber(value) }}</span>
          </template>
          <template #cell-participants="{ value }">
            <span class="text-gray-900 dark:text-white">{{ formatNumber(value) }}</span>
          </template>
          <template #cell-total_amount="{ value }">
            <span class="text-gray-900 dark:text-white">{{ formatCurrency(value) }}</span>
          </template>
          <template #cell-average_amount="{ value }">
            <span class="text-gray-500 dark:text-gray-400">{{ formatCurrency(value) }}</span>
          </template>
        </DataTable>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI, type LotteryDailyStats, type LotteryDailyAggregate } from '@/api/admin'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatCurrency, formatNumber, formatDateLocalInput } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const stats = ref<LotteryDailyStats | null>(null)
const startDate = ref('')
const endDate = ref('')

function setToday() {
  const today = formatDateLocalInput(new Date())
  startDate.value = today
  endDate.value = today
  loadStats()
}

function setLast30Days() {
  const now = new Date()
  startDate.value = formatDateLocalInput(new Date(now.getTime() - 29 * 24 * 60 * 60 * 1000))
  endDate.value = formatDateLocalInput(now)
  loadStats()
}

async function loadStats() {
  loading.value = true
  try {
    stats.value = await adminAPI.lottery.getDailyStats(startDate.value || undefined, endDate.value || undefined)
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.lottery.loadFailed')))
  } finally {
    loading.value = false
  }
}

interface DailyRow extends LotteryDailyAggregate {
  average_amount: number
}

const dailyRows = computed<DailyRow[]>(() => {
  return (stats.value?.daily ?? []).map((d) => ({
    ...d,
    average_amount: d.draws > 0 ? d.total_amount / d.draws : 0
  }))
})

const summary = computed(() => stats.value?.summary ?? null)

const columns = computed<Column[]>(() => [
  { key: 'date', label: t('admin.lottery.date') },
  { key: 'draws', label: t('admin.lottery.draws') },
  { key: 'participants', label: t('admin.lottery.participants') },
  { key: 'total_amount', label: t('admin.lottery.amount') },
  { key: 'average_amount', label: t('admin.lottery.averageAmount') }
])

onMounted(() => {
  setToday()
})
</script>
