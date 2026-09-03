<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div><h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.channelMonitorReset.title') }}</h2><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.channelMonitorReset.description') }}</p></div>
      <button type="button" class="btn btn-primary" @click="openCreate">{{ t('admin.channelMonitorReset.create') }}</button>
    </div>
    <div class="overflow-x-auto rounded-xl border border-gray-200 dark:border-dark-700">
      <table class="w-full text-left text-sm"><thead class="bg-gray-50 text-xs uppercase text-gray-500 dark:bg-dark-800 dark:text-gray-400"><tr><th class="px-4 py-3">{{ t('admin.channelMonitorReset.name') }}</th><th class="px-4 py-3">{{ t('admin.channelMonitorReset.accounts') }}</th><th class="px-4 py-3">{{ t('admin.channelMonitorReset.subscriptions') }}</th><th class="px-4 py-3">{{ t('admin.channelMonitorReset.status') }}</th><th class="px-4 py-3">{{ t('admin.channelMonitorReset.lastCheck') }}</th><th class="px-4 py-3">{{ t('admin.channelMonitorReset.actions') }}</th></tr></thead><tbody><tr v-for="row in monitors" :key="row.id" class="border-t border-gray-100 dark:border-dark-700"><td class="px-4 py-3 font-medium text-gray-900 dark:text-white">{{ row.name }}</td><td class="px-4 py-3">{{ row.account_ids.length }}</td><td class="px-4 py-3">{{ row.subscription_ids.length }}</td><td class="px-4 py-3"><span :class="row.execution_enabled ? 'text-emerald-500' : 'text-amber-500'">{{ row.execution_enabled ? t('admin.channelMonitorReset.automatic') : t('admin.channelMonitorReset.observing') }}</span></td><td class="px-4 py-3 text-xs text-gray-500">{{ formatTime(row.last_checked_at) }}</td><td class="px-4 py-3"><div class="flex gap-2"><button type="button" class="btn btn-secondary btn-sm" :disabled="checking === row.id" @click="check(row)">{{ checking === row.id ? t('common.loading') : t('admin.channelMonitorReset.check') }}</button><button type="button" class="btn btn-secondary btn-sm" @click="openEdit(row)">{{ t('common.edit') }}</button><button type="button" class="btn btn-secondary btn-sm" @click="showEvents(row)">{{ t('admin.channelMonitorReset.history') }}</button></div><p v-if="row.last_error" class="mt-1 max-w-xs text-xs text-red-500">{{ row.last_error }}</p></td></tr><tr v-if="!loading && monitors.length === 0"><td colspan="6" class="px-4 py-8 text-center text-gray-500">{{ t('admin.channelMonitorReset.empty') }}</td></tr></tbody></table>
    </div>
    <SubscriptionQuotaResetMonitorDialog :show="dialog" :monitor="editing" @close="closeDialog" @saved="load" />
    <BaseDialog :show="eventsDialog" :title="t('admin.channelMonitorReset.history')" width="wide" @close="eventsDialog = false"><div class="space-y-3"><div v-for="event in events" :key="event.id" class="rounded-lg border border-gray-200 p-3 text-sm dark:border-dark-700"><div class="flex justify-between gap-3"><span class="font-medium">{{ event.classification }}</span><span class="text-xs text-gray-500">{{ formatTime(event.detected_at) }}</span></div><div class="mt-1 text-xs text-gray-500">{{ event.status }}<span v-if="event.last_error"> · {{ event.last_error }}</span></div></div><p v-if="events.length === 0" class="py-6 text-center text-gray-500">{{ t('admin.channelMonitorReset.noEvents') }}</p></div></BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useAppStore } from '@/stores/app'
import type { MonitorEvent, SubscriptionQuotaResetMonitor } from '@/api/admin/subscriptionQuotaResetMonitor'
import BaseDialog from '@/components/common/BaseDialog.vue'
import SubscriptionQuotaResetMonitorDialog from './SubscriptionQuotaResetMonitorDialog.vue'

const { t } = useI18n()
const appStore = useAppStore()
const monitors = ref<SubscriptionQuotaResetMonitor[]>([])
const events = ref<MonitorEvent[]>([])
const loading = ref(false)
const checking = ref<number | null>(null)
const dialog = ref(false)
const editing = ref<SubscriptionQuotaResetMonitor | null>(null)
const eventsDialog = ref(false)
function formatTime(value?: string | null) { return value ? new Date(value).toLocaleString() : '-' }
async function load() { loading.value = true; try { monitors.value = await adminAPI.subscriptionQuotaResetMonitor.list() } catch (err: unknown) { appStore.showError(extractApiErrorMessage(err, t('common.error'))) } finally { loading.value = false } }
function openCreate() { editing.value = null; dialog.value = true }
function openEdit(row: SubscriptionQuotaResetMonitor) { editing.value = row; dialog.value = true }
function closeDialog() { dialog.value = false }
async function check(row: SubscriptionQuotaResetMonitor) { checking.value = row.id; try { const updated = await adminAPI.subscriptionQuotaResetMonitor.check(row.id); const index = monitors.value.findIndex(item => item.id === row.id); if (index >= 0) monitors.value[index] = updated } catch (err: unknown) { appStore.showError(extractApiErrorMessage(err, t('common.error'))) } finally { checking.value = null } }
async function showEvents(row: SubscriptionQuotaResetMonitor) { try { events.value = await adminAPI.subscriptionQuotaResetMonitor.events(row.id); eventsDialog.value = true } catch (err: unknown) { appStore.showError(extractApiErrorMessage(err, t('common.error'))) } }
onMounted(load)
</script>
