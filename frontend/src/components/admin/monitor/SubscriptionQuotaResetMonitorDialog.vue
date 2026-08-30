<template>
  <BaseDialog :show="show" :title="monitor ? t('admin.channelMonitorReset.editTitle') : t('admin.channelMonitorReset.createTitle')" width="wide" @close="$emit('close')">
    <form id="subscription-quota-reset-monitor-form" class="space-y-4" @submit.prevent="submit">
      <div>
        <label class="input-label">{{ t('admin.channelMonitorReset.name') }}</label>
        <input v-model="form.name" class="input" required maxlength="100" />
      </div>
      <div class="grid gap-4 sm:grid-cols-2">
        <div>
          <label class="input-label">{{ t('admin.channelMonitorReset.accounts') }}</label>
          <input v-model="accountIDs" class="input" placeholder="25,31" required />
          <p class="mt-1 text-xs text-gray-400">{{ t('admin.channelMonitorReset.accountsHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.channelMonitorReset.subscriptions') }}</label>
          <input v-model="subscriptionIDs" class="input" placeholder="101,102" required />
          <p class="mt-1 text-xs text-gray-400">{{ t('admin.channelMonitorReset.subscriptionsHint') }}</p>
        </div>
      </div>
      <div class="grid gap-4 sm:grid-cols-2">
        <div>
          <label class="input-label">{{ t('admin.channelMonitorReset.interval') }}</label>
          <input v-model.number="form.interval_seconds" type="number" min="60" max="3600" class="input" required />
        </div>
        <div>
          <label class="input-label">{{ t('admin.channelMonitorReset.threshold') }}</label>
          <input v-model.number="form.drop_threshold_percent" type="number" min="1" max="100" step="0.1" class="input" required />
        </div>
      </div>
      <div class="grid gap-3 sm:grid-cols-2">
        <label class="flex items-center gap-2"><input v-model="form.reset_weekly" type="checkbox" /> {{ t('admin.channelMonitorReset.weekly') }}</label>
        <label class="flex items-center gap-2"><input v-model="form.reset_daily" type="checkbox" /> {{ t('admin.channelMonitorReset.daily') }}</label>
        <label class="flex items-center gap-2"><input v-model="form.reset_monthly" type="checkbox" /> {{ t('admin.channelMonitorReset.monthly') }}</label>
        <label class="flex items-center gap-2"><input v-model="form.reset_five_hour" type="checkbox" /> {{ t('admin.channelMonitorReset.fiveHour') }}</label>
      </div>
      <label class="flex items-center gap-2"><input v-model="form.credit_policy" type="checkbox" true-value="propagate" false-value="ignore" /> {{ t('admin.channelMonitorReset.creditPolicy') }}</label>
      <div class="border-t border-gray-100 pt-3 dark:border-dark-700">
        <label class="flex items-center justify-between"><span class="input-label mb-0">{{ t('admin.channelMonitorReset.enabled') }}</span><Toggle v-model="form.enabled" /></label>
        <label class="mt-3 flex items-center justify-between"><span class="input-label mb-0">{{ t('admin.channelMonitorReset.executionEnabled') }}</span><Toggle v-model="form.execution_enabled" /></label>
        <p class="mt-2 text-xs text-amber-600 dark:text-amber-400">{{ t('admin.channelMonitorReset.safetyHint') }}</p>
      </div>
    </form>
    <template #footer>
      <div class="flex justify-end gap-3"><button type="button" class="btn btn-secondary" @click="$emit('close')">{{ t('common.cancel') }}</button><button type="submit" form="subscription-quota-reset-monitor-form" class="btn btn-primary" :disabled="saving">{{ saving ? t('common.submitting') : t('common.save') }}</button></div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Toggle from '@/components/common/Toggle.vue'
import { adminAPI } from '@/api/admin'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useAppStore } from '@/stores/app'
import type { MonitorParams, SubscriptionQuotaResetMonitor } from '@/api/admin/subscriptionQuotaResetMonitor'

const props = defineProps<{ show: boolean; monitor?: SubscriptionQuotaResetMonitor | null }>()
const emit = defineEmits<{ (event: 'close'): void; (event: 'saved'): void }>()
const { t } = useI18n()
const appStore = useAppStore()
const saving = ref(false)
const form = reactive<MonitorParams>({ name: '', enabled: false, execution_enabled: false, interval_seconds: 600, drop_threshold_percent: 1, credit_policy: 'ignore', reset_daily: false, reset_weekly: true, reset_monthly: false, reset_five_hour: false, account_ids: [], subscription_ids: [] })
const accountIDs = computed({ get: () => form.account_ids.join(','), set: (value: string) => { form.account_ids = parseIDs(value) } })
const subscriptionIDs = computed({ get: () => form.subscription_ids.join(','), set: (value: string) => { form.subscription_ids = parseIDs(value) } })
function parseIDs(value: string): number[] { return value.split(',').map(item => Number(item.trim())).filter(item => Number.isInteger(item) && item > 0) }
watch(() => props.monitor, monitor => { if (monitor) Object.assign(form, { ...monitor }); else Object.assign(form, { name: '', enabled: false, execution_enabled: false, interval_seconds: 600, drop_threshold_percent: 1, credit_policy: 'ignore', reset_daily: false, reset_weekly: true, reset_monthly: false, reset_five_hour: false, account_ids: [], subscription_ids: [] }) }, { immediate: true })
async function submit() { if (!form.account_ids.length || !form.subscription_ids.length || (!form.reset_daily && !form.reset_weekly && !form.reset_monthly && !form.reset_five_hour)) return; saving.value = true; try { if (props.monitor) await adminAPI.subscriptionQuotaResetMonitor.update(props.monitor.id, form); else await adminAPI.subscriptionQuotaResetMonitor.create(form); appStore.showSuccess(t(props.monitor ? 'admin.channelMonitorReset.updateSuccess' : 'admin.channelMonitorReset.createSuccess')); emit('saved'); emit('close') } catch (err: unknown) { appStore.showError(extractApiErrorMessage(err, t('common.error'))) } finally { saving.value = false } }
</script>
