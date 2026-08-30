package service

import (
	"testing"
	"time"
)

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
