package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/google/uuid"
)

const (
	MonitorCreditPolicyIgnore    = "ignore"
	MonitorCreditPolicyPropagate = "propagate"
	MonitorStatusObserving       = "observing"
	MonitorStatusCandidate       = "candidate"
	MonitorStatusWaiting         = "waiting"
	MonitorStatusSucceeded       = "succeeded"
	MonitorStatusIgnored         = "ignored"
	MonitorEventScheduled        = "scheduled_reset"
	MonitorEventCredit           = "credit_reset"
	MonitorEventOfficial         = "official_reset"
	MonitorEventUnknown          = "unknown"
	MonitorEventMixed            = "mixed"
)

var (
	ErrQuotaResetMonitorNotFound = infraerrors.NotFound("QUOTA_RESET_MONITOR_NOT_FOUND", "quota reset monitor not found")
	ErrQuotaResetMonitorInvalid  = infraerrors.BadRequest("QUOTA_RESET_MONITOR_INVALID", "invalid quota reset monitor")
)

type SubscriptionQuotaResetMonitor struct {
	ID                   int64      `json:"id"`
	Name                 string     `json:"name"`
	Enabled              bool       `json:"enabled"`
	ExecutionEnabled     bool       `json:"execution_enabled"`
	IntervalSeconds      int        `json:"interval_seconds"`
	DropThresholdPercent float64    `json:"drop_threshold_percent"`
	CreditPolicy         string     `json:"credit_policy"`
	ResetDaily           bool       `json:"reset_daily"`
	ResetWeekly          bool       `json:"reset_weekly"`
	ResetMonthly         bool       `json:"reset_monthly"`
	ResetFiveHour        bool       `json:"reset_five_hour"`
	AccountIDs           []int64    `json:"account_ids"`
	SubscriptionIDs      []int64    `json:"subscription_ids"`
	LastCheckedAt        *time.Time `json:"last_checked_at,omitempty"`
	NextCheckAt          *time.Time `json:"next_check_at,omitempty"`
	LastStatus           string     `json:"last_status"`
	LastError            string     `json:"last_error,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type SubscriptionQuotaResetMonitorState struct {
	MonitorID           int64
	AccountID           int64
	PreviousPercent     float64
	PreviousResetAt     *time.Time
	PreviousCreditHash  string
	PreviousCreditCount int
	SampledAt           *time.Time
	Candidate           *MonitorCandidate
	LastError           string
}

type MonitorCandidate struct {
	ObservedAt      time.Time `json:"observed_at"`
	Confirmed       bool      `json:"confirmed"`
	PreviousPercent float64   `json:"previous_percent"`
	CurrentPercent  float64   `json:"current_percent"`
	PreviousResetAt time.Time `json:"previous_reset_at"`
	CurrentResetAt  time.Time `json:"current_reset_at"`
	CreditHash      string    `json:"credit_hash"`
	CreditCount     int       `json:"credit_count"`
	CreditChanged   bool      `json:"credit_changed"`
}

type SubscriptionQuotaResetMonitorEvent struct {
	ID             uuid.UUID       `json:"id"`
	MonitorID      int64           `json:"monitor_id"`
	Fingerprint    string          `json:"fingerprint"`
	Classification string          `json:"classification"`
	Status         string          `json:"status"`
	DetectedAt     time.Time       `json:"detected_at"`
	ConfirmedAt    *time.Time      `json:"confirmed_at,omitempty"`
	SourceSnapshot json.RawMessage `json:"source_snapshot"`
	LastError      string          `json:"last_error,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type SubscriptionQuotaResetMonitorEventTarget struct {
	EventID        uuid.UUID  `json:"event_id"`
	SubscriptionID int64      `json:"subscription_id"`
	Status         string     `json:"status"`
	SkipReason     string     `json:"skip_reason,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
	AttemptedAt    *time.Time `json:"attempted_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

type SubscriptionQuotaResetMonitorRepository interface {
	List(ctx context.Context) ([]*SubscriptionQuotaResetMonitor, error)
	Get(ctx context.Context, id int64) (*SubscriptionQuotaResetMonitor, error)
	Create(ctx context.Context, monitor *SubscriptionQuotaResetMonitor, actorID *int64) error
	Update(ctx context.Context, monitor *SubscriptionQuotaResetMonitor) error
	GetState(ctx context.Context, monitorID, accountID int64) (*SubscriptionQuotaResetMonitorState, error)
	SaveState(ctx context.Context, state *SubscriptionQuotaResetMonitorState) error
	CreateEvent(ctx context.Context, event *SubscriptionQuotaResetMonitorEvent, subscriptionIDs []int64) (bool, error)
	ListEvents(ctx context.Context, monitorID int64, limit int) ([]*SubscriptionQuotaResetMonitorEvent, error)
	ListEventTargets(ctx context.Context, eventID uuid.UUID) ([]*SubscriptionQuotaResetMonitorEventTarget, error)
	UpdateEventTarget(ctx context.Context, target *SubscriptionQuotaResetMonitorEventTarget) error
	UpdateEventStatus(ctx context.Context, eventID uuid.UUID, status, lastError string) error
	ListActionableEvents(ctx context.Context, limit int) ([]*SubscriptionQuotaResetMonitorEvent, error)
	HasSubscriptionOverlap(ctx context.Context, monitorID int64, subscriptionIDs []int64) (bool, error)
}

type quotaResetMonitorQuota interface {
	QueryUsage(ctx context.Context, accountID int64) (*OpenAIQuotaUsage, error)
}

type SubscriptionQuotaResetMonitorService struct {
	repo         SubscriptionQuotaResetMonitorRepository
	accountRepo  AccountRepository
	quota        quotaResetMonitorQuota
	subscription *SubscriptionService
	audit        *AuditLogService
	leader       LeaderLockCache
	owner        string
	ctx          context.Context
	cancel       context.CancelFunc
	start        sync.Once
	stop         sync.Once
	wg           sync.WaitGroup
}

func NewSubscriptionQuotaResetMonitorService(repo SubscriptionQuotaResetMonitorRepository, accountRepo AccountRepository, quota quotaResetMonitorQuota, subscription *SubscriptionService, audit *AuditLogService, leader LeaderLockCache) *SubscriptionQuotaResetMonitorService {
	ctx, cancel := context.WithCancel(context.Background())
	return &SubscriptionQuotaResetMonitorService{repo: repo, accountRepo: accountRepo, quota: quota, subscription: subscription, audit: audit, leader: leader, owner: uuid.NewString(), ctx: ctx, cancel: cancel}
}

func (s *SubscriptionQuotaResetMonitorService) Start() {
	if s == nil || s.repo == nil || s.accountRepo == nil || s.quota == nil || s.subscription == nil {
		return
	}
	s.start.Do(func() { s.wg.Add(1); go s.run() })
}

func (s *SubscriptionQuotaResetMonitorService) Stop() {
	if s == nil {
		return
	}
	s.stop.Do(func() { s.cancel(); s.wg.Wait() })
}

func (s *SubscriptionQuotaResetMonitorService) run() {
	defer s.wg.Done()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	s.runDue(s.ctx)
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.runDue(s.ctx)
		}
	}
}

func (s *SubscriptionQuotaResetMonitorService) runDue(ctx context.Context) {
	if s.leader != nil {
		ok, err := s.leader.TryAcquireLeaderLock(ctx, "jobs:subscription-quota-reset-monitor", s.owner, 55*time.Second)
		if err != nil || !ok {
			return
		}
		defer func() {
			releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = s.leader.ReleaseLeaderLock(releaseCtx, "jobs:subscription-quota-reset-monitor", s.owner)
		}()
	}
	monitors, err := s.repo.List(ctx)
	if err != nil {
		slog.Warn("subscription_quota_reset_monitor_list_failed", "error", err)
		return
	}
	now := time.Now().UTC()
	for _, monitor := range monitors {
		if monitor != nil && monitor.Enabled && (monitor.NextCheckAt == nil || !now.Before(*monitor.NextCheckAt)) {
			if err := s.Check(ctx, monitor.ID, false); err != nil {
				slog.Warn("subscription_quota_reset_monitor_check_failed", "monitor_id", monitor.ID, "error", err)
			}
		}
	}
	events, err := s.repo.ListActionableEvents(ctx, 100)
	if err != nil {
		slog.Warn("subscription_quota_reset_monitor_event_list_failed", "error", err)
		return
	}
	for _, event := range events {
		if event == nil {
			continue
		}
		monitor, getErr := s.repo.Get(ctx, event.MonitorID)
		if getErr != nil || monitor == nil || !monitor.Enabled || !monitor.ExecutionEnabled {
			continue
		}
		if now.Sub(event.UpdatedAt) < time.Duration(monitor.IntervalSeconds)*time.Second {
			continue
		}
		if applyErr := s.applyEvent(ctx, monitor, event); applyErr != nil {
			slog.Warn("subscription_quota_reset_monitor_event_retry_failed", "event_id", event.ID, "error", applyErr)
		}
	}
}

func (s *SubscriptionQuotaResetMonitorService) List(ctx context.Context) ([]*SubscriptionQuotaResetMonitor, error) {
	return s.repo.List(ctx)
}
func (s *SubscriptionQuotaResetMonitorService) Get(ctx context.Context, id int64) (*SubscriptionQuotaResetMonitor, error) {
	return s.repo.Get(ctx, id)
}

func validateMonitor(m *SubscriptionQuotaResetMonitor) error {
	if m == nil {
		return ErrQuotaResetMonitorInvalid
	}
	m.Name = strings.TrimSpace(m.Name)
	m.AccountIDs = uniquePositiveIDs(m.AccountIDs)
	m.SubscriptionIDs = uniquePositiveIDs(m.SubscriptionIDs)
	if len(m.AccountIDs) == 0 || len(m.SubscriptionIDs) == 0 || m.Name == "" || len(m.Name) > 100 || len(m.AccountIDs) > 64 || len(m.SubscriptionIDs) > 10000 {
		return ErrQuotaResetMonitorInvalid
	}
	if m.IntervalSeconds < 60 || m.IntervalSeconds > 3600 || m.DropThresholdPercent < 1 || m.DropThresholdPercent > 100 {
		return ErrQuotaResetMonitorInvalid
	}
	if m.CreditPolicy != MonitorCreditPolicyIgnore && m.CreditPolicy != MonitorCreditPolicyPropagate {
		return ErrQuotaResetMonitorInvalid
	}
	if !m.ResetDaily && !m.ResetWeekly && !m.ResetMonthly && !m.ResetFiveHour {
		return ErrQuotaResetMonitorInvalid
	}
	return nil
}

func uniquePositiveIDs(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (s *SubscriptionQuotaResetMonitorService) Create(ctx context.Context, m *SubscriptionQuotaResetMonitor, actorID *int64) error {
	if err := validateMonitor(m); err != nil {
		return err
	}
	if err := s.validateSubscriptions(ctx, m.SubscriptionIDs); err != nil {
		return err
	}
	overlap, err := s.repo.HasSubscriptionOverlap(ctx, 0, m.SubscriptionIDs)
	if err != nil {
		return err
	}
	if overlap && m.Enabled {
		return infraerrors.Conflict("QUOTA_RESET_MONITOR_SUBSCRIPTION_CONFLICT", "a subscription is already bound to an enabled monitor")
	}
	return s.repo.Create(ctx, m, actorID)
}

func (s *SubscriptionQuotaResetMonitorService) Update(ctx context.Context, m *SubscriptionQuotaResetMonitor) error {
	if err := validateMonitor(m); err != nil {
		return err
	}
	if err := s.validateSubscriptions(ctx, m.SubscriptionIDs); err != nil {
		return err
	}
	overlap, err := s.repo.HasSubscriptionOverlap(ctx, m.ID, m.SubscriptionIDs)
	if err != nil {
		return err
	}
	if overlap && m.Enabled {
		return infraerrors.Conflict("QUOTA_RESET_MONITOR_SUBSCRIPTION_CONFLICT", "a subscription is already bound to an enabled monitor")
	}
	return s.repo.Update(ctx, m)
}

func (s *SubscriptionQuotaResetMonitorService) validateSubscriptions(ctx context.Context, ids []int64) error {
	for _, id := range ids {
		if subscription, err := s.subscription.GetByID(ctx, id); err != nil || subscription == nil {
			return infraerrors.BadRequest("QUOTA_RESET_MONITOR_SUBSCRIPTION_NOT_FOUND", fmt.Sprintf("subscription %d was not found", id))
		}
	}
	return nil
}

func (s *SubscriptionQuotaResetMonitorService) Events(ctx context.Context, id int64, limit int) ([]*SubscriptionQuotaResetMonitorEvent, error) {
	return s.repo.ListEvents(ctx, id, limit)
}

type monitorSample struct {
	AccountID      int64     `json:"account_id"`
	Utilization    float64   `json:"utilization_percent"`
	ResetAt        time.Time `json:"reset_at"`
	CreditHash     string    `json:"credit_hash"`
	CreditCount    int       `json:"credit_count"`
	Classification string    `json:"classification"`
}

func sevenDayWindow(usage *OpenAIQuotaUsage) (*OpenAIRateLimitWindow, bool) {
	if usage == nil || usage.RateLimit == nil {
		return nil, false
	}
	for _, w := range []*OpenAIRateLimitWindow{usage.RateLimit.PrimaryWindow, usage.RateLimit.SecondaryWindow} {
		if w != nil && w.LimitWindowSeconds >= int64(6*24*time.Hour/time.Second) {
			return w, true
		}
	}
	return nil, false
}

func creditFingerprint(credits *OpenAIRateLimitResetCredits) (string, int) {
	if credits == nil {
		return "", 0
	}
	expires := make([]string, 0, len(credits.Credits))
	for _, c := range credits.Credits {
		expires = append(expires, c.ExpiresAt)
	}
	sort.Strings(expires)
	h := sha256.Sum256([]byte(fmt.Sprintf("%d|%s", credits.AvailableCount, joinMonitorStrings(expires, "|"))))
	return hex.EncodeToString(h[:]), credits.AvailableCount
}

func joinMonitorStrings(values []string, sep string) string {
	out := ""
	for i, v := range values {
		if i > 0 {
			out += sep
		}
		out += v
	}
	return out
}

func classifyMonitorSample(previous *SubscriptionQuotaResetMonitorState, now time.Time, utilization, dropThreshold float64, resetAt time.Time, creditHash string, creditCount int) (*MonitorCandidate, string) {
	if previous == nil || previous.SampledAt == nil || previous.PreviousResetAt == nil || resetAt.IsZero() {
		return nil, MonitorStatusObserving
	}
	drop := previous.PreviousPercent - utilization
	if drop < dropThreshold {
		return nil, MonitorStatusObserving
	}
	if !now.Before(previous.PreviousResetAt.Add(-15 * time.Minute)) {
		return nil, MonitorEventScheduled
	}
	creditChanged := previous.PreviousCreditHash != "" && creditHash != "" && previous.PreviousCreditHash != creditHash && creditCount < previous.PreviousCreditCount
	classification := MonitorEventOfficial
	if creditChanged {
		classification = MonitorEventCredit
	}
	candidate := &MonitorCandidate{ObservedAt: now, PreviousPercent: previous.PreviousPercent, CurrentPercent: utilization, PreviousResetAt: *previous.PreviousResetAt, CurrentResetAt: resetAt, CreditHash: creditHash, CreditCount: creditCount, CreditChanged: creditChanged}
	return candidate, classification
}

func (s *SubscriptionQuotaResetMonitorService) Check(ctx context.Context, id int64, manual bool) error {
	m, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrQuotaResetMonitorNotFound
	}
	now := time.Now().UTC()
	eligible := true
	candidates := make(map[int64]*MonitorCandidate, len(m.AccountIDs))
	samples := make([]monitorSample, 0, len(m.AccountIDs))
	for _, accountID := range m.AccountIDs {
		account, getErr := s.accountRepo.GetByID(ctx, accountID)
		if getErr != nil || account == nil || account.Platform != PlatformOpenAI || account.Type != AccountTypeOAuth || account.IsShadow() {
			eligible = false
			continue
		}
		usage, queryErr := s.quota.QueryUsage(ctx, accountID)
		if queryErr != nil {
			eligible = false
			continue
		}
		window, ok := sevenDayWindow(usage)
		if !ok || window.ResetAt <= 0 {
			eligible = false
			continue
		}
		resetAt := time.Unix(window.ResetAt, 0).UTC()
		creditHash, creditCount := creditFingerprint(usage.RateLimitResetCredits)
		state, stateErr := s.loadState(ctx, m.ID, accountID)
		if stateErr != nil {
			return stateErr
		}
		previous := *state
		candidate, _ := classifyMonitorSample(&previous, now, window.UsedPercent, m.DropThresholdPercent, resetAt, creditHash, creditCount)
		if state.Candidate != nil {
			pending := state.Candidate
			withinWindow := now.Sub(pending.ObservedAt) <= 2*time.Duration(m.IntervalSeconds)*time.Second
			stableDrop := window.UsedPercent <= pending.PreviousPercent-m.DropThresholdPercent
			sameReset := pending.CurrentResetAt.Equal(resetAt)
			if withinWindow && stableDrop && sameReset {
				pending.Confirmed = true
				pending.CurrentPercent = window.UsedPercent
				pending.CurrentResetAt = resetAt
				pending.CreditChanged = pending.CreditChanged || (pending.CreditCount > creditCount && pending.CreditHash != "" && pending.CreditHash != creditHash)
				pending.CreditHash, pending.CreditCount = creditHash, creditCount
				candidates[accountID] = pending
			} else if candidate != nil {
				state.Candidate = candidate
			} else if !withinWindow {
				state.Candidate = nil
			}
		} else if candidate != nil {
			state.Candidate = candidate
		}
		if state.Candidate != nil && state.Candidate.Confirmed {
			candidates[accountID] = state.Candidate
		}
		state.MonitorID, state.AccountID, state.PreviousPercent, state.PreviousResetAt, state.PreviousCreditHash, state.PreviousCreditCount, state.SampledAt, state.LastError = m.ID, accountID, window.UsedPercent, &resetAt, creditHash, creditCount, &now, ""
		if err := s.repo.SaveState(ctx, state); err != nil {
			return err
		}
	}
	status := MonitorStatusObserving
	lastError := ""
	if !eligible {
		lastError = "one or more accounts could not be queried"
	} else if len(candidates) == len(m.AccountIDs) && len(candidates) > 0 {
		hasCredit := false
		for accountID, candidate := range candidates {
			if candidate.CreditChanged {
				hasCredit = true
			}
			samples = append(samples, monitorSample{AccountID: accountID, Utilization: candidate.CurrentPercent, ResetAt: candidate.CurrentResetAt, CreditHash: candidate.CreditHash, CreditCount: candidate.CreditCount, Classification: MonitorEventOfficial})
		}
		classification := MonitorEventOfficial
		if hasCredit {
			classification = MonitorEventCredit
		}
		if hasCredit && m.CreditPolicy == MonitorCreditPolicyIgnore {
			status = MonitorStatusIgnored
		} else {
			status = MonitorStatusCandidate
		}
		payload, _ := json.Marshal(samples)
		fingerprint := monitorEventFingerprint(m.ID, samples)
		if !manual {
			if m.ExecutionEnabled && status == MonitorStatusCandidate {
				status = MonitorStatusWaiting
			}
			event := &SubscriptionQuotaResetMonitorEvent{ID: uuid.New(), MonitorID: m.ID, Fingerprint: fingerprint, Classification: classification, Status: status, DetectedAt: now, ConfirmedAt: &now, SourceSnapshot: payload, LastError: lastError}
			created, createErr := s.repo.CreateEvent(ctx, event, m.SubscriptionIDs)
			if createErr != nil {
				return createErr
			}
			if created {
				for accountID := range candidates {
					state, stateErr := s.loadState(ctx, m.ID, accountID)
					if stateErr != nil {
						return stateErr
					}
					state.Candidate = nil
					if err := s.repo.SaveState(ctx, state); err != nil {
						return err
					}
				}
				if s.audit != nil {
					s.audit.Record(&AuditLog{ActorEmail: "system", ActorRole: "system", AuthMethod: "system", Action: "system.subscription_quota_reset_monitor.detected", Method: "SYSTEM", Path: fmt.Sprintf("/system/subscription-quota-reset-monitors/%d/check", m.ID), StatusCode: 200, Extra: map[string]any{"monitor_id": m.ID, "classification": classification, "status": status, "accounts": len(candidates), "subscriptions": len(m.SubscriptionIDs)}})
				}
				if status == MonitorStatusWaiting {
					if err := s.applyEvent(ctx, m, event); err != nil {
						_ = s.repo.UpdateEventStatus(ctx, event.ID, MonitorStatusFailed, err.Error())
						return err
					}
				}
			}
		}
	}
	next := now.Add(time.Duration(m.IntervalSeconds) * time.Second)
	m.LastCheckedAt, m.NextCheckAt, m.LastStatus, m.LastError = &now, &next, status, lastError
	if err := s.repo.Update(ctx, m); err != nil {
		return err
	}
	return nil
}

func monitorEventFingerprint(id int64, samples []monitorSample) string {
	sort.Slice(samples, func(i, j int) bool { return samples[i].AccountID < samples[j].AccountID })
	b, _ := json.Marshal(struct {
		ID      int64
		Samples []monitorSample
	}{id, samples})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func (s *SubscriptionQuotaResetMonitorService) loadState(ctx context.Context, monitorID, accountID int64) (*SubscriptionQuotaResetMonitorState, error) {
	return s.repo.GetState(ctx, monitorID, accountID)
}

func (s *SubscriptionQuotaResetMonitorService) applyEvent(ctx context.Context, monitor *SubscriptionQuotaResetMonitor, event *SubscriptionQuotaResetMonitorEvent) error {
	targets, err := s.repo.ListEventTargets(ctx, event.ID)
	if err != nil {
		return err
	}
	failed := false
	for _, target := range targets {
		if target == nil || target.Status == "succeeded" || target.Status == "skipped" {
			continue
		}
		now := time.Now().UTC()
		target.AttemptedAt = &now
		sub, getErr := s.subscription.GetByID(ctx, target.SubscriptionID)
		if getErr != nil {
			target.Status, target.LastError = "failed", getErr.Error()
			failed = true
		} else if !sub.IsActive() {
			target.Status, target.SkipReason = "skipped", "inactive_subscription"
			target.CompletedAt = &now
		} else if alreadyReset(sub, event.DetectedAt, monitor) {
			target.Status, target.SkipReason = "skipped", "already_reset"
			target.CompletedAt = &now
		} else if _, resetErr := s.subscription.AdminResetQuota(ctx, target.SubscriptionID, monitor.ResetDaily, monitor.ResetWeekly, monitor.ResetMonthly, monitor.ResetFiveHour); resetErr != nil {
			target.Status, target.LastError = "failed", resetErr.Error()
			failed = true
		} else {
			target.Status, target.CompletedAt = "succeeded", &now
		}
		if err := s.repo.UpdateEventTarget(ctx, target); err != nil {
			return err
		}
	}
	status := MonitorStatusSucceeded
	lastError := ""
	if failed {
		status, lastError = MonitorStatusFailed, "one or more subscriptions failed to reset"
	}
	if s.audit != nil {
		s.audit.Record(&AuditLog{ActorEmail: "system", ActorRole: "system", AuthMethod: "system", Action: "system.subscription_quota_reset_monitor.apply", Method: "SYSTEM", Path: fmt.Sprintf("/system/subscription-quota-reset-monitors/%d/apply", monitor.ID), StatusCode: 200, Extra: map[string]any{"monitor_id": monitor.ID, "event_id": event.ID.String(), "status": status}})
	}
	return s.repo.UpdateEventStatus(ctx, event.ID, status, lastError)
}

func alreadyReset(sub *UserSubscription, at time.Time, monitor *SubscriptionQuotaResetMonitor) bool {
	if sub == nil {
		return false
	}
	starts := []*time.Time{}
	if monitor.ResetDaily {
		starts = append(starts, sub.DailyWindowStart)
	}
	if monitor.ResetWeekly {
		starts = append(starts, sub.WeeklyWindowStart)
	}
	if monitor.ResetMonthly {
		starts = append(starts, sub.MonthlyWindowStart)
	}
	if monitor.ResetFiveHour {
		starts = append(starts, sub.FiveHourWindowStart)
	}
	for _, start := range starts {
		if start != nil && !start.Before(at) {
			return true
		}
	}
	return false
}
