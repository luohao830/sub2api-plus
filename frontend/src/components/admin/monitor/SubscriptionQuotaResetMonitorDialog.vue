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
          <div class="relative">
            <button type="button" class="input flex w-full items-center justify-between text-left" @click="openDropdown = openDropdown === 'accounts' ? null : 'accounts'">
              <span :class="form.account_ids.length ? 'text-gray-900 dark:text-white' : 'text-gray-400'">{{ selectedAccountLabel }}</span>
              <span aria-hidden="true">⌄</span>
            </button>
            <div v-if="openDropdown === 'accounts'" class="absolute z-30 mt-1 max-h-56 w-full overflow-y-auto rounded-lg border border-gray-200 bg-white p-2 shadow-lg dark:border-dark-700 dark:bg-dark-800">
              <p v-if="optionsLoading" class="px-2 py-2 text-xs text-gray-400">{{ t('common.loading') }}</p>
              <label v-for="account in accountOptions" :key="account.id" class="flex cursor-pointer items-center gap-2 rounded px-2 py-2 text-sm hover:bg-gray-50 dark:hover:bg-dark-700">
                <input v-model="form.account_ids" type="checkbox" :value="account.id" />
                <span>{{ account.id }} · {{ account.name }}</span>
              </label>
              <p v-if="!optionsLoading && !accountOptions.length" class="px-2 py-2 text-xs text-gray-400">{{ t('common.noOptionsFound') }}</p>
            </div>
          </div>
          <p class="mt-1 text-xs text-gray-400">{{ t('admin.channelMonitorReset.accountsHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.channelMonitorReset.subscriptions') }}</label>
          <div class="relative">
            <button type="button" class="input flex w-full items-center justify-between text-left" @click="openDropdown = openDropdown === 'subscriptions' ? null : 'subscriptions'">
              <span :class="form.subscription_ids.length ? 'text-gray-900 dark:text-white' : 'text-gray-400'">{{ selectedSubscriptionLabel }}</span>
              <span aria-hidden="true">⌄</span>
            </button>
            <div v-if="openDropdown === 'subscriptions'" class="absolute z-30 mt-1 max-h-56 w-full overflow-y-auto rounded-lg border border-gray-200 bg-white p-2 shadow-lg dark:border-dark-700 dark:bg-dark-800">
              <p v-if="optionsLoading" class="px-2 py-2 text-xs text-gray-400">{{ t('common.loading') }}</p>
              <label v-for="subscription in subscriptionOptions" :key="subscription.id" class="flex cursor-pointer items-center gap-2 rounded px-2 py-2 text-sm hover:bg-gray-50 dark:hover:bg-dark-700">
                <input v-model="form.subscription_ids" type="checkbox" :value="subscription.id" />
                <span>{{ subscription.id }} · {{ subscription.user?.email || `User #${subscription.user_id}` }}</span>
              </label>
              <p v-if="!optionsLoading && !subscriptionOptions.length" class="px-2 py-2 text-xs text-gray-400">{{ t('common.noOptionsFound') }}</p>
            </div>
          </div>
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
        <label class="flex items-center gap-2"><input v-model="form.reset_weekly" type="checkbox" @change="normalizeResetWindows" /> {{ t('admin.channelMonitorReset.weekly') }}</label>
        <label class="flex items-center gap-2"><input v-model="form.reset_daily" type="checkbox" @change="normalizeResetWindows" /> {{ t('admin.channelMonitorReset.daily') }}</label>
        <label class="flex items-center gap-2"><input v-model="form.reset_monthly" type="checkbox" @change="normalizeResetWindows" /> {{ t('admin.channelMonitorReset.monthly') }}</label>
        <label class="flex items-center gap-2"><input v-model="form.reset_five_hour" type="checkbox" @change="normalizeResetWindows" /> {{ t('admin.channelMonitorReset.fiveHour') }}</label>
      </div>
      <p class="-mt-2 text-xs text-gray-400">{{ t('admin.channelMonitorReset.windowCascadeHint') }}</p>
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
import type { Account, UserSubscription } from '@/types'

const props = defineProps<{ show: boolean; monitor?: SubscriptionQuotaResetMonitor | null }>()
const emit = defineEmits<{ (event: 'close'): void; (event: 'saved'): void }>()
const { t } = useI18n()
const appStore = useAppStore()
const saving = ref(false)
const optionsLoading = ref(false)
const openDropdown = ref<'accounts' | 'subscriptions' | null>(null)
const accountOptions = ref<Account[]>([])
const subscriptionOptions = ref<UserSubscription[]>([])
const form = reactive<MonitorParams>({ name: '', enabled: false, execution_enabled: false, interval_seconds: 600, drop_threshold_percent: 1, credit_policy: 'ignore', reset_daily: false, reset_weekly: true, reset_monthly: false, reset_five_hour: false, account_ids: [], subscription_ids: [] })
const selectedAccountLabel = computed(() => form.account_ids.length ? form.account_ids.join(', ') : t('admin.channelMonitorReset.selectAccounts'))
const selectedSubscriptionLabel = computed(() => form.subscription_ids.length ? form.subscription_ids.join(', ') : t('admin.channelMonitorReset.selectSubscriptions'))
function normalizeResetWindows() {
  if (form.reset_monthly) form.reset_weekly = true
  if (form.reset_weekly) form.reset_daily = true
  if (form.reset_daily) form.reset_five_hour = true
}
async function loadOptions() {
  optionsLoading.value = true
  try {
    const [accounts, subscriptions] = await Promise.all([
      adminAPI.accounts.list(1, 100, { platform: 'openai', type: 'oauth', status: 'active', lite: 'true' }),
      adminAPI.subscriptions.list(1, 100, { status: 'active', platform: 'openai' })
    ])
    accountOptions.value = accounts.items.filter(account => account.parent_account_id == null)
    subscriptionOptions.value = subscriptions.items
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    optionsLoading.value = false
  }
}
watch(() => props.monitor, monitor => { if (monitor) Object.assign(form, { ...monitor }); else Object.assign(form, { name: '', enabled: false, execution_enabled: false, interval_seconds: 600, drop_threshold_percent: 1, credit_policy: 'ignore', reset_daily: false, reset_weekly: true, reset_monthly: false, reset_five_hour: false, account_ids: [], subscription_ids: [] }); normalizeResetWindows() }, { immediate: true })
watch(() => props.show, show => { if (show) { openDropdown.value = null; void loadOptions() } })
async function submit() { if (!form.account_ids.length || !form.subscription_ids.length || (!form.reset_daily && !form.reset_weekly && !form.reset_monthly && !form.reset_five_hour)) return; saving.value = true; try { if (props.monitor) await adminAPI.subscriptionQuotaResetMonitor.update(props.monitor.id, form); else await adminAPI.subscriptionQuotaResetMonitor.create(form); appStore.showSuccess(t(props.monitor ? 'admin.channelMonitorReset.updateSuccess' : 'admin.channelMonitorReset.createSuccess')); emit('saved'); emit('close') } catch (err: unknown) { appStore.showError(extractApiErrorMessage(err, t('common.error'))) } finally { saving.value = false } }
</script>
