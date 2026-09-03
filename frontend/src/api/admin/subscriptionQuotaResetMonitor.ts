import { apiClient } from '../client'

export interface SubscriptionQuotaResetMonitor {
  id: number
  name: string
  enabled: boolean
  execution_enabled: boolean
  interval_seconds: number
  drop_threshold_percent: number
  credit_policy: 'ignore' | 'propagate'
  reset_daily: boolean
  reset_weekly: boolean
  reset_monthly: boolean
  reset_five_hour: boolean
  account_ids: number[]
  subscription_ids: number[]
  last_checked_at?: string | null
  next_check_at?: string | null
  last_status: string
  last_error?: string
  created_at: string
  updated_at: string
}

export interface MonitorEvent {
  id: string
  monitor_id: number
  fingerprint: string
  classification: string
  status: string
  detected_at: string
  confirmed_at?: string | null
  source_snapshot: unknown
  last_error?: string
  created_at: string
  updated_at: string
}

export interface MonitorParams {
  name: string
  enabled: boolean
  execution_enabled: boolean
  interval_seconds: number
  drop_threshold_percent: number
  credit_policy: 'ignore' | 'propagate'
  reset_daily: boolean
  reset_weekly: boolean
  reset_monthly: boolean
  reset_five_hour: boolean
  account_ids: number[]
  subscription_ids: number[]
}

const subscriptionQuotaResetMonitorAPI = {
  async list(): Promise<SubscriptionQuotaResetMonitor[]> {
    const { data } = await apiClient.get<SubscriptionQuotaResetMonitor[]>('/admin/subscription-quota-reset-monitors')
    return data
  },
  async get(id: number): Promise<SubscriptionQuotaResetMonitor> {
    const { data } = await apiClient.get<SubscriptionQuotaResetMonitor>(`/admin/subscription-quota-reset-monitors/${id}`)
    return data
  },
  async create(params: MonitorParams): Promise<SubscriptionQuotaResetMonitor> {
    const { data } = await apiClient.post<SubscriptionQuotaResetMonitor>('/admin/subscription-quota-reset-monitors', params)
    return data
  },
  async update(id: number, params: MonitorParams): Promise<SubscriptionQuotaResetMonitor> {
    const { data } = await apiClient.put<SubscriptionQuotaResetMonitor>(`/admin/subscription-quota-reset-monitors/${id}`, params)
    return data
  },
  async check(id: number): Promise<SubscriptionQuotaResetMonitor> {
    const { data } = await apiClient.post<SubscriptionQuotaResetMonitor>(`/admin/subscription-quota-reset-monitors/${id}/check`)
    return data
  },
  async events(id: number): Promise<MonitorEvent[]> {
    const { data } = await apiClient.get<MonitorEvent[]>(`/admin/subscription-quota-reset-monitors/${id}/events`)
    return data
  }
}

export default subscriptionQuotaResetMonitorAPI
