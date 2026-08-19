<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-6">
      <!-- ===================== 顶部 Banner（渐变蓝绿） ===================== -->
      <div class="card overflow-hidden">
        <div class="bg-gradient-to-br from-primary-500 via-primary-600 to-accent-600 px-6 py-6">
          <div class="flex flex-col gap-6 md:flex-row md:items-center">
            <!-- 左侧：礼物盒 + 文案 -->
            <div class="flex flex-1 items-center gap-4">
              <div
                class="flex h-16 w-16 flex-shrink-0 items-center justify-center rounded-2xl bg-white/20 backdrop-blur-sm"
              >
                <Icon name="gift" size="xl" class="text-white" />
              </div>
              <div>
                <p class="text-lg font-bold text-white">{{ t('lottery.bannerTitle') }}</p>
                <p class="mt-1 text-sm text-primary-100">
                  {{ t('lottery.alwaysWin') }} · {{ t('lottery.fromAmount') }}
                </p>
                <p class="mt-1 text-xs text-primary-200">{{ t('lottery.chancesSource') }}</p>
              </div>
            </div>
            <!-- 右侧：可用次数 + 今日充值进度 -->
            <div class="flex flex-col gap-3 rounded-2xl bg-white/10 p-4 backdrop-blur-sm">
              <div class="flex items-baseline gap-2">
                <span class="text-sm text-primary-100">{{ t('lottery.availableChances') }}</span>
                <span class="text-4xl font-bold text-white">{{ status?.available_count ?? 0 }}</span>
                <span class="text-sm text-primary-200">{{ t('lottery.chances') }}</span>
              </div>
              <div>
                <div class="mb-1 flex justify-between text-xs text-primary-100">
                  <span>{{ t('lottery.todayRechargeProgress') }}</span>
                  <span>{{ status?.today_recharge_count ?? 0 }}/{{ status?.today_recharge_max ?? 5 }}</span>
                </div>
                <div class="h-1.5 w-48 overflow-hidden rounded-full bg-white/20">
                  <div
                    class="h-full rounded-full bg-white transition-all duration-500"
                    :style="{ width: rechargeProgressPercent }"
                  ></div>
                </div>
              </div>
              <p class="text-xs text-primary-200">
                {{ t('lottery.todayDate') }}: {{ status?.today ?? '' }} ({{ status?.timezone ?? 'Asia/Shanghai' }})
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- ===================== 中奖结果条（浅黄） ===================== -->
      <transition name="pop">
        <div
          v-if="lastResult"
          class="card border-amber-200 bg-amber-50 dark:border-amber-800/50 dark:bg-amber-900/20"
        >
          <div class="flex flex-col items-center gap-4 p-4 sm:flex-row sm:justify-between">
            <div class="flex items-center gap-3">
              <div class="flex h-10 w-10 items-center justify-center rounded-full bg-amber-100 dark:bg-amber-900/40">
                <Icon name="checkCircle" size="lg" class="text-amber-500" />
              </div>
              <div>
                <p class="font-semibold text-amber-800 dark:text-amber-200">
                  {{ t('lottery.resultTitle') }} {{ t('lottery.wonAmount') }}
                  <span class="text-xl font-bold">US${{ lastResult.prize_amount.toFixed(2) }}</span>
                </p>
                <p class="text-sm text-amber-700/80 dark:text-amber-300/80">
                  {{ t('lottery.currentBalance') }}: US${{ lastResult.new_balance.toFixed(2) }}
                </p>
              </div>
            </div>
            <button
              class="btn btn-primary"
              :disabled="drawing || !status?.available_count"
              @click="handleDraw"
            >
              {{ t('lottery.drawAgain') }}
            </button>
          </div>
        </div>
      </transition>

      <!-- ===================== 主区域 ===================== -->
      <div class="grid gap-6 md:grid-cols-3">
        <!-- 左侧：开启盲盒 -->
        <div class="card md:col-span-2">
          <div class="flex flex-col items-center p-8 text-center">
            <div
              class="relative mb-6 flex h-36 w-36 items-center justify-center rounded-full bg-gradient-to-br from-primary-100 to-accent-100 dark:from-dark-700 dark:to-dark-800"
            >
              <!-- 礼物盒插画（抽奖时抖动） -->
              <div :class="['gift-box', drawing && 'box-shake']">
                <div class="gift-lid"></div>
                <div class="gift-body">
                  <div class="gift-ribbon"></div>
                </div>
              </div>
            </div>

            <h2 class="text-xl font-bold text-gray-800 dark:text-gray-100">
              {{ t('lottery.mainTitle') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
              {{ status?.available_count ? `${t('lottery.availableChances')} ${status.available_count} ${t('lottery.chances')}` : t('lottery.noChancesToday') }}
            </p>

            <button
              class="btn btn-primary mt-6 w-full max-w-xs py-3 text-base"
              :disabled="drawing || !status?.available_count || !status?.enabled"
              @click="handleDraw"
            >
              <span v-if="drawing" class="flex items-center justify-center gap-2">
                <svg class="h-5 w-5 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
                </svg>
                {{ t('lottery.drawing') }}
              </span>
              <span v-else-if="!status?.enabled">{{ t('lottery.drawingDisabled') }}</span>
              <span v-else-if="!status?.available_count">{{ t('lottery.noChances') }}</span>
              <span v-else>{{ t('lottery.openBox') }}</span>
            </button>

            <router-link
              to="/purchase"
              class="mt-3 inline-flex items-center text-sm font-medium text-primary-600 hover:underline dark:text-primary-400"
            >
              {{ t('lottery.goRecharge') }} →
            </router-link>

            <!-- 三步说明 -->
            <div class="mt-8 grid w-full gap-3 sm:grid-cols-3">
              <div class="rounded-xl border border-gray-100 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-800">
                <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('lottery.stepLogin') }}</p>
                <p class="mt-1 text-lg font-bold text-emerald-500">{{ t('lottery.stepLoginValue') }}</p>
              </div>
              <div class="rounded-xl border border-gray-100 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-800">
                <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('lottery.stepRecharge') }}</p>
                <p class="mt-1 text-lg font-bold text-emerald-500">{{ t('lottery.stepRechargeValue') }}</p>
              </div>
              <div class="rounded-xl border border-gray-100 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-800">
                <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('lottery.stepLimit') }}</p>
                <p class="mt-1 text-lg font-bold text-emerald-500">{{ t('lottery.stepLimitValue') }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- 右侧：参与资格 -->
        <div class="card">
          <div class="p-6">
            <h3 class="mb-4 font-semibold text-gray-800 dark:text-gray-100">
              {{ t('lottery.qualifyTitle') }}
            </h3>

            <div class="space-y-3">
              <div class="flex items-center justify-between rounded-xl bg-gray-50 px-4 py-3 dark:bg-dark-800">
                <div class="flex items-center gap-2">
                  <Icon name="checkCircle" size="md" class="text-emerald-500" />
                  <span class="text-sm text-gray-600 dark:text-dark-300">{{ t('lottery.loginReward') }}</span>
                </div>
                <div class="text-right">
                  <span class="text-sm font-bold text-emerald-500">{{ t('lottery.stepLoginValue') }}</span>
                  <span class="ml-2 text-xs text-gray-400">
                    {{ status?.login_rewarded_today ? t('lottery.loginRewarded') : t('lottery.loginNotRewarded') }}
                  </span>
                </div>
              </div>

              <div class="flex items-center justify-between rounded-xl bg-gray-50 px-4 py-3 dark:bg-dark-800">
                <span class="text-sm text-gray-600 dark:text-dark-300">{{ t('lottery.rechargeReward') }}</span>
                <span class="text-sm font-bold text-gray-700 dark:text-dark-200">
                  {{ status?.today_recharge_count ?? 0 }}/{{ status?.today_recharge_max ?? 5 }}
                </span>
              </div>

              <div class="flex items-center justify-between rounded-xl bg-gray-50 px-4 py-3 dark:bg-dark-800">
                <span class="text-sm text-gray-600 dark:text-dark-300">{{ t('lottery.dailyRule') }}</span>
                <span class="text-sm font-bold text-gray-700 dark:text-dark-200">
                  {{ status?.today_recharge_max ?? 5 }}
                </span>
              </div>
            </div>

            <!-- 底部三个小卡片 -->
            <div class="mt-4 grid grid-cols-3 gap-2">
              <div class="rounded-lg bg-emerald-50 py-2 text-center dark:bg-emerald-900/20">
                <p class="text-sm font-bold text-emerald-600">+1</p>
                <p class="text-[10px] text-emerald-500/80">{{ t('lottery.stepLogin') }}</p>
              </div>
              <div class="rounded-lg bg-emerald-50 py-2 text-center dark:bg-emerald-900/20">
                <p class="text-sm font-bold text-emerald-600">10:1</p>
                <p class="text-[10px] text-emerald-500/80">{{ t('lottery.rechargeReward') }}</p>
              </div>
              <div class="rounded-lg bg-emerald-50 py-2 text-center dark:bg-emerald-900/20">
                <p class="text-sm font-bold text-emerald-600">{{ status?.today_recharge_max ?? 5 }}</p>
                <p class="text-[10px] text-emerald-500/80">{{ t('lottery.dailyLimit') }}</p>
              </div>
            </div>

            <!-- 蓝色提示条 -->
            <div class="mt-4 rounded-xl bg-primary-50 px-4 py-3 text-xs leading-relaxed text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">
              {{ t('lottery.note') }}
            </div>
          </div>
        </div>
      </div>

      <!-- ===================== 时区说明 ===================== -->
      <div class="card">
        <div class="flex items-start gap-3 p-4">
          <Icon name="clock" size="md" class="mt-0.5 flex-shrink-0 text-gray-400" />
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('lottery.timezoneNote') }}</p>
        </div>
      </div>

      <!-- ===================== 历史记录 ===================== -->
      <div class="card">
        <div class="p-6">
          <h3 class="mb-4 font-semibold text-gray-800 dark:text-gray-100">
            {{ t('lottery.historyTitle') }}
          </h3>
          <div v-if="records.length" class="overflow-x-auto">
            <table class="table">
              <thead>
                <tr>
                  <th>{{ t('lottery.recordTime') }}</th>
                  <th>{{ t('lottery.recordAmount') }}</th>
                  <th>{{ t('lottery.recordBalanceAfter') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="record in records" :key="record.id">
                  <td>{{ formatDateTime(record.created_at) }}</td>
                  <td class="font-semibold text-emerald-600">+US${{ (record.amount ?? 0).toFixed(2) }}</td>
                  <td>US${{ (record.balance_after ?? 0).toFixed(2) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-else class="flex flex-col items-center gap-3 py-8 text-center">
            <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-gray-100 dark:bg-dark-800">
              <Icon name="gift" size="lg" class="text-gray-400 dark:text-dark-500" />
            </div>
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('lottery.empty') }}</p>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { lotteryAPI, type LotteryStatus, type LotteryDrawResult, type LotteryRecordItem } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const status = ref<LotteryStatus | null>(null)
const lastResult = ref<LotteryDrawResult | null>(null)
const drawing = ref(false)
const records = ref<LotteryRecordItem[]>([])

const rechargeProgressPercent = computed(() => {
  const current = status.value?.today_recharge_count ?? 0
  const max = status.value?.today_recharge_max ?? 5
  return `${Math.min(100, Math.round((current / max) * 100))}%`
})

const fetchStatus = async () => {
  try {
    status.value = await lotteryAPI.getStatus()
    // 同步全局余额展示（顶部 Header）
    if (status.value.current_balance !== undefined) {
      await authStore.refreshUser().catch(() => {})
    }
  } catch (error) {
    console.error('Failed to load lottery status:', error)
    appStore.showError(t('lottery.loadingFailed'))
  }
}

const fetchRecords = async () => {
  try {
    const page = await lotteryAPI.getRecords(1, 10)
    records.value = page.items
  } catch (error) {
    console.error('Failed to load lottery records:', error)
  }
}

const handleDraw = async () => {
  if (drawing.value) return
  if (!status.value?.available_count) return

  drawing.value = true
  try {
    const result = await lotteryAPI.draw()
    lastResult.value = result
    await fetchStatus()
    await fetchRecords()
    appStore.showSuccess(
      `${t('lottery.resultTitle')} +US$${result.prize_amount.toFixed(2)}`
    )
  } catch (error: any) {
    const reason: string = error?.reason || error?.message || ''
    if (reason.includes('NO_CHANCES')) {
      appStore.showError(t('lottery.errorNoChances'))
    } else if (reason.includes('RATE_LIMITED')) {
      appStore.showError(t('lottery.errorRateLimit'))
    } else if (reason.includes('DISABLED')) {
      appStore.showError(t('lottery.drawingDisabled'))
    } else {
      appStore.showError(t('lottery.errorNoChances'))
    }
    await fetchStatus()
  } finally {
    drawing.value = false
  }
}

onMounted(() => {
  fetchStatus()
  fetchRecords()
})
</script>

<style scoped>
/* 礼物盒插画 */
.gift-box {
  position: relative;
  width: 96px;
  height: 96px;
}
.gift-lid {
  position: absolute;
  top: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 104px;
  height: 26px;
  border-radius: 8px;
  background: linear-gradient(180deg, #34d399, #10b981);
  z-index: 2;
}
.gift-body {
  position: absolute;
  top: 26px;
  left: 50%;
  transform: translateX(-50%);
  width: 88px;
  height: 66px;
  border-radius: 0 0 12px 12px;
  background: linear-gradient(180deg, #5eead4, #2dd4bf);
}
.gift-ribbon {
  position: absolute;
  top: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 22px;
  height: 100%;
  background: linear-gradient(180deg, #fbbf24, #f59e0b);
  border-radius: 4px;
}

/* 抽奖时抖动动画 */
@keyframes box-shake {
  0%, 100% { transform: rotate(0deg); }
  20% { transform: rotate(-12deg) translateY(-4px); }
  40% { transform: rotate(10deg) translateY(-8px); }
  60% { transform: rotate(-8deg) translateY(-4px); }
  80% { transform: rotate(6deg) translateY(-2px); }
}
.box-shake {
  animation: box-shake 0.6s ease-in-out;
}

/* 结果条弹出动画 */
.pop-enter-active {
  transition: all 0.35s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.pop-leave-active {
  transition: all 0.2s ease;
}
.pop-enter-from,
.pop-leave-to {
  opacity: 0;
  transform: scale(0.92) translateY(-8px);
}
</style>
