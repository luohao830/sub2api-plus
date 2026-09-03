package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type subscriptionQuotaResetMonitorRepository struct{ db *sql.DB }

func NewSubscriptionQuotaResetMonitorRepository(db *sql.DB) service.SubscriptionQuotaResetMonitorRepository {
	return &subscriptionQuotaResetMonitorRepository{db: db}
}

func scanMonitor(scanner interface{ Scan(...any) error }) (*service.SubscriptionQuotaResetMonitor, error) {
	var m service.SubscriptionQuotaResetMonitor
	var accounts, subscriptions pq.Int64Array
	var lastChecked, nextCheck sql.NullTime
	if err := scanner.Scan(&m.ID, &m.Name, &m.Enabled, &m.ExecutionEnabled, &m.IntervalSeconds, &m.DropThresholdPercent, &m.CreditPolicy, &m.ResetDaily, &m.ResetWeekly, &m.ResetMonthly, &m.ResetFiveHour, &accounts, &subscriptions, &lastChecked, &nextCheck, &m.LastStatus, &m.LastError, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return nil, err
	}
	m.AccountIDs, m.SubscriptionIDs = []int64(accounts), []int64(subscriptions)
	if lastChecked.Valid {
		m.LastCheckedAt = &lastChecked.Time
	}
	if nextCheck.Valid {
		m.NextCheckAt = &nextCheck.Time
	}
	return &m, nil
}

const monitorSelect = `SELECT id,name,enabled,execution_enabled,interval_seconds,drop_threshold_percent,credit_policy,reset_daily,reset_weekly,reset_monthly,reset_five_hour,account_ids,subscription_ids,last_checked_at,next_check_at,last_status,last_error,created_at,updated_at FROM subscription_quota_reset_monitors`

func (r *subscriptionQuotaResetMonitorRepository) List(ctx context.Context) ([]*service.SubscriptionQuotaResetMonitor, error) {
	rows, err := r.db.QueryContext(ctx, monitorSelect+" ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]*service.SubscriptionQuotaResetMonitor, 0)
	for rows.Next() {
		item, err := scanMonitor(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *subscriptionQuotaResetMonitorRepository) Get(ctx context.Context, id int64) (*service.SubscriptionQuotaResetMonitor, error) {
	item, err := scanMonitor(r.db.QueryRowContext(ctx, monitorSelect+" WHERE id=$1", id))
	if err == sql.ErrNoRows {
		return nil, service.ErrQuotaResetMonitorNotFound
	}
	return item, err
}

func (r *subscriptionQuotaResetMonitorRepository) Create(ctx context.Context, m *service.SubscriptionQuotaResetMonitor, actorID *int64) error {
	return r.db.QueryRowContext(ctx, `INSERT INTO subscription_quota_reset_monitors (name,enabled,execution_enabled,interval_seconds,drop_threshold_percent,credit_policy,reset_daily,reset_weekly,reset_monthly,reset_five_hour,account_ids,subscription_ids,created_by,next_check_at) VALUES ($1,$2,$3,$4::int,$5,$6,$7,$8,$9,$10,$11,$12,$13,NOW()+($4::int * INTERVAL '1 second')) RETURNING id,created_at,updated_at`, m.Name, m.Enabled, m.ExecutionEnabled, m.IntervalSeconds, m.DropThresholdPercent, m.CreditPolicy, m.ResetDaily, m.ResetWeekly, m.ResetMonthly, m.ResetFiveHour, pq.Array(m.AccountIDs), pq.Array(m.SubscriptionIDs), actorID).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

func (r *subscriptionQuotaResetMonitorRepository) Update(ctx context.Context, m *service.SubscriptionQuotaResetMonitor) error {
	result, err := r.db.ExecContext(ctx, `UPDATE subscription_quota_reset_monitors SET name=$1,enabled=$2,execution_enabled=$3,interval_seconds=$4,drop_threshold_percent=$5,credit_policy=$6,reset_daily=$7,reset_weekly=$8,reset_monthly=$9,reset_five_hour=$10,account_ids=$11,subscription_ids=$12,last_checked_at=$13,next_check_at=$14,last_status=$15,last_error=$16,updated_at=NOW() WHERE id=$17`, m.Name, m.Enabled, m.ExecutionEnabled, m.IntervalSeconds, m.DropThresholdPercent, m.CreditPolicy, m.ResetDaily, m.ResetWeekly, m.ResetMonthly, m.ResetFiveHour, pq.Array(m.AccountIDs), pq.Array(m.SubscriptionIDs), m.LastCheckedAt, m.NextCheckAt, m.LastStatus, m.LastError, m.ID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return service.ErrQuotaResetMonitorNotFound
	}
	return nil
}

func (r *subscriptionQuotaResetMonitorRepository) GetState(ctx context.Context, monitorID, accountID int64) (*service.SubscriptionQuotaResetMonitorState, error) {
	state := &service.SubscriptionQuotaResetMonitorState{MonitorID: monitorID, AccountID: accountID}
	var previousReset, sampled sql.NullTime
	var candidate []byte
	err := r.db.QueryRowContext(ctx, `SELECT previous_utilization_percent,previous_reset_at,previous_credit_hash,previous_credit_count,sampled_at,candidate,last_error FROM subscription_quota_reset_monitor_states WHERE monitor_id=$1 AND account_id=$2`, monitorID, accountID).Scan(&state.PreviousPercent, &previousReset, &state.PreviousCreditHash, &state.PreviousCreditCount, &sampled, &candidate, &state.LastError)
	if err == sql.ErrNoRows {
		return state, nil
	}
	if err != nil {
		return nil, err
	}
	if previousReset.Valid {
		state.PreviousResetAt = &previousReset.Time
	}
	if sampled.Valid {
		state.SampledAt = &sampled.Time
	}
	if len(candidate) > 0 {
		state.Candidate = &service.MonitorCandidate{}
		if err := json.Unmarshal(candidate, state.Candidate); err != nil {
			state.Candidate = nil
		}
	}
	return state, nil
}

func (r *subscriptionQuotaResetMonitorRepository) SaveState(ctx context.Context, state *service.SubscriptionQuotaResetMonitorState) error {
	var candidate any
	if state.Candidate != nil {
		data, err := json.Marshal(state.Candidate)
		if err != nil {
			return err
		}
		candidate = data
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO subscription_quota_reset_monitor_states (monitor_id,account_id,previous_utilization_percent,previous_reset_at,previous_credit_hash,previous_credit_count,sampled_at,candidate,last_error) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (monitor_id,account_id) DO UPDATE SET previous_utilization_percent=EXCLUDED.previous_utilization_percent,previous_reset_at=EXCLUDED.previous_reset_at,previous_credit_hash=EXCLUDED.previous_credit_hash,previous_credit_count=EXCLUDED.previous_credit_count,sampled_at=EXCLUDED.sampled_at,candidate=EXCLUDED.candidate,last_error=EXCLUDED.last_error`, state.MonitorID, state.AccountID, state.PreviousPercent, state.PreviousResetAt, state.PreviousCreditHash, state.PreviousCreditCount, state.SampledAt, candidate, state.LastError)
	return err
}

func (r *subscriptionQuotaResetMonitorRepository) CreateEvent(ctx context.Context, event *service.SubscriptionQuotaResetMonitorEvent, subscriptionIDs []int64) (bool, error) {
	data := event.SourceSnapshot
	if len(data) == 0 {
		data = []byte("[]")
	}
	result, err := r.db.ExecContext(ctx, `INSERT INTO subscription_quota_reset_monitor_events (id,monitor_id,fingerprint,classification,status,detected_at,confirmed_at,source_snapshot,last_error) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (fingerprint) DO NOTHING`, event.ID, event.MonitorID, event.Fingerprint, event.Classification, event.Status, event.DetectedAt, event.ConfirmedAt, data, event.LastError)
	if err != nil {
		return false, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return false, nil
	}
	for _, subscriptionID := range subscriptionIDs {
		if _, err := r.db.ExecContext(ctx, `INSERT INTO subscription_quota_reset_monitor_event_targets (event_id,subscription_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, event.ID, subscriptionID); err != nil {
			return false, err
		}
	}
	return true, nil
}

func scanEvent(scanner interface{ Scan(...any) error }) (*service.SubscriptionQuotaResetMonitorEvent, error) {
	var e service.SubscriptionQuotaResetMonitorEvent
	var id string
	var confirmed sql.NullTime
	var snapshot []byte
	if err := scanner.Scan(&id, &e.MonitorID, &e.Fingerprint, &e.Classification, &e.Status, &e.DetectedAt, &confirmed, &snapshot, &e.LastError, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return nil, err
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	e.ID = parsed
	e.SourceSnapshot = snapshot
	if confirmed.Valid {
		e.ConfirmedAt = &confirmed.Time
	}
	return &e, nil
}

func (r *subscriptionQuotaResetMonitorRepository) ListEvents(ctx context.Context, monitorID int64, limit int) ([]*service.SubscriptionQuotaResetMonitorEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,monitor_id,fingerprint,classification,status,detected_at,confirmed_at,source_snapshot,last_error,created_at,updated_at FROM subscription_quota_reset_monitor_events WHERE monitor_id=$1 ORDER BY created_at DESC LIMIT $2`, monitorID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]*service.SubscriptionQuotaResetMonitorEvent, 0)
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *subscriptionQuotaResetMonitorRepository) ListEventTargets(ctx context.Context, eventID uuid.UUID) ([]*service.SubscriptionQuotaResetMonitorEventTarget, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT event_id,subscription_id,status,skip_reason,last_error,attempted_at,completed_at FROM subscription_quota_reset_monitor_event_targets WHERE event_id=$1 ORDER BY subscription_id`, eventID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]*service.SubscriptionQuotaResetMonitorEventTarget, 0)
	for rows.Next() {
		var t service.SubscriptionQuotaResetMonitorEventTarget
		var eid string
		var attempted, completed sql.NullTime
		if err := rows.Scan(&eid, &t.SubscriptionID, &t.Status, &t.SkipReason, &t.LastError, &attempted, &completed); err != nil {
			return nil, err
		}
		t.EventID, _ = uuid.Parse(eid)
		if attempted.Valid {
			t.AttemptedAt = &attempted.Time
		}
		if completed.Valid {
			t.CompletedAt = &completed.Time
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}

func (r *subscriptionQuotaResetMonitorRepository) UpdateEventTarget(ctx context.Context, target *service.SubscriptionQuotaResetMonitorEventTarget) error {
	result, err := r.db.ExecContext(ctx, `UPDATE subscription_quota_reset_monitor_event_targets SET status=$1,skip_reason=$2,last_error=$3,attempted_at=$4,completed_at=$5 WHERE event_id=$6 AND subscription_id=$7`, target.Status, target.SkipReason, target.LastError, target.AttemptedAt, target.CompletedAt, target.EventID, target.SubscriptionID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("event target not found")
	}
	return nil
}
func (r *subscriptionQuotaResetMonitorRepository) UpdateEventStatus(ctx context.Context, eventID uuid.UUID, status, lastError string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE subscription_quota_reset_monitor_events SET status=$1,last_error=$2,updated_at=NOW() WHERE id=$3`, status, lastError, eventID)
	return err
}
func (r *subscriptionQuotaResetMonitorRepository) ListActionableEvents(ctx context.Context, limit int) ([]*service.SubscriptionQuotaResetMonitorEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,monitor_id,fingerprint,classification,status,detected_at,confirmed_at,source_snapshot,last_error,created_at,updated_at FROM subscription_quota_reset_monitor_events WHERE status IN ('waiting','failed') ORDER BY updated_at ASC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	result := make([]*service.SubscriptionQuotaResetMonitorEvent, 0)
	for rows.Next() {
		event, scanErr := scanEvent(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, event)
	}
	return result, rows.Err()
}
func (r *subscriptionQuotaResetMonitorRepository) HasSubscriptionOverlap(ctx context.Context, monitorID int64, subscriptionIDs []int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM subscription_quota_reset_monitors WHERE enabled=TRUE AND id<>$1 AND subscription_ids && $2::bigint[])`, monitorID, pq.Array(subscriptionIDs)).Scan(&exists)
	return exists, err
}
