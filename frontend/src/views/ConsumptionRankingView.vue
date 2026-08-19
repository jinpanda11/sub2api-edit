<template>
  <!-- 登录态：套完整后台布局 -->
  <AppLayout v-if="isAuthenticated">
    <RankingContent />
  </AppLayout>

  <!-- 未登录：独立页面（自带导航） -->
  <div v-else class="min-h-screen bg-gray-50 dark:bg-dark-950">
    <header
      class="sticky top-0 z-30 border-b border-gray-200/50 bg-white/80 backdrop-blur dark:border-dark-700/50 dark:bg-dark-900/80"
    >
      <div class="mx-auto flex max-w-3xl items-center justify-between gap-4 px-4 py-3.5 sm:px-6">
        <div class="flex min-w-0 items-center gap-3">
          <span
            class="flex h-9 w-9 flex-shrink-0 items-center justify-center overflow-hidden rounded-xl bg-white shadow-sm ring-1 ring-gray-200 dark:bg-dark-800 dark:ring-dark-700"
          >
            <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-full w-full object-contain" />
          </span>
          <span class="truncate text-base font-semibold text-gray-950 dark:text-white">{{ siteName }}</span>
        </div>
        <RouterLink
          :to="{ path: '/login', query: { redirect: '/ranking' } }"
          class="inline-flex flex-shrink-0 items-center justify-center gap-1.5 rounded-xl bg-gradient-to-r from-primary-500 to-primary-600 px-4 py-2 text-sm font-semibold text-white shadow-md shadow-primary-500/25 transition-all duration-200 hover:from-primary-600 hover:to-primary-700 active:scale-[0.98] dark:shadow-primary-500/20"
        >
          {{ t('ranking.nav.login') }}
        </RouterLink>
      </div>
    </header>
    <main class="mx-auto max-w-3xl px-4 py-6 sm:px-6">
      <RankingContent />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import AppLayout from '@/components/layout/AppLayout.vue'
import RankingContent from '@/components/ranking/RankingContent.vue'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const isAuthenticated = computed(() => authStore.isAuthenticated)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || 'Sub2API')
const siteLogo = computed(() =>
  sanitizeUrl(appStore.cachedPublicSettings?.site_logo || '', { allowRelative: true, allowDataUrl: true })
)
</script>
