package service

import (
	"testing"
	"time"
)

func TestNormalizeMonitorResetWindows(t *testing.T) {
	tests := []struct {
		name    string
		monitor SubscriptionQuotaResetMonitor
		want    [4]bool
	}{
		{name: "monthly cascades", monitor: SubscriptionQuotaResetMonitor{ResetMonthly: true}, want: [4]bool{true, true, true, true}},
		{name: "weekly cascades", monitor: SubscriptionQuotaResetMonitor{ResetWeekly: true}, want: [4]bool{true, true, false, true}},
		{name: "daily cascades", monitor: SubscriptionQuotaResetMonitor{ResetDaily: true}, want: [4]bool{true, false, false, true}},
		{name: "five hour stays independent", monitor: SubscriptionQuotaResetMonitor{ResetFiveHour: true}, want: [4]bool{false, false, false, true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalizeMonitorResetWindows(&tt.monitor)
			got := [4]bool{tt.monitor.ResetDaily, tt.monitor.ResetWeekly, tt.monitor.ResetMonthly, tt.monitor.ResetFiveHour}
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClassifyMonitorSampleRequiresAnExistingBaseline(t *testing.T) {
	now := time.Date(2026, time.January, 2, 12, 0, 0, 0, time.UTC)
	candidate, classification := classifyMonitorSample(nil, now, 20, 1, now.Add(24*time.Hour), "credit", 2)
	if candidate != nil || classification != MonitorStatusObserving {
		t.Fatalf("first sample must establish a baseline, got candidate=%v classification=%q", candidate, classification)
	}
}

func TestClassifyMonitorSampleRejectsScheduledReset(t *testing.T) {
	now := time.Date(2026, time.January, 2, 12, 0, 0, 0, time.UTC)
	previousReset := now.Add(-10 * time.Minute)
	previous := &SubscriptionQuotaResetMonitorState{
		PreviousPercent:     80,
		PreviousResetAt:     &previousReset,
		PreviousCreditHash:  "credit",
		PreviousCreditCount: 2,
		SampledAt:           &previousReset,
	}
	candidate, classification := classifyMonitorSample(previous, now, 10, 1, now.Add(7*24*time.Hour), "credit", 2)
	if candidate != nil || classification != MonitorEventScheduled {
		t.Fatalf("near-due reset must be classified as scheduled, got candidate=%v classification=%q", candidate, classification)
	}
}

func TestClassifyMonitorSampleMarksCreditConsumption(t *testing.T) {
	now := time.Date(2026, time.January, 2, 12, 0, 0, 0, time.UTC)
	previousReset := now.Add(24 * time.Hour)
	previous := &SubscriptionQuotaResetMonitorState{
		PreviousPercent:     80,
		PreviousResetAt:     &previousReset,
		PreviousCreditHash:  "before",
		PreviousCreditCount: 2,
		SampledAt:           &previousReset,
	}
	candidate, classification := classifyMonitorSample(previous, now, 10, 1, now.Add(7*24*time.Hour), "after", 1)
	if candidate == nil || classification != MonitorEventCredit || !candidate.CreditChanged {
		t.Fatalf("credit consumption must be classified separately, got candidate=%v classification=%q", candidate, classification)
	}
}
