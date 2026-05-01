package handlers

import (
	"testing"
	"time"
)

func TestApplyRandomDelayStaysWithinBounds(t *testing.T) {
	scheduledAt := time.Date(2026, time.May, 1, 12, 0, 0, 0, time.UTC)
	const maxDelay = 15

	for i := 0; i < 200; i++ {
		actual := applyRandomDelay(scheduledAt, maxDelay)
		diff := actual.Sub(scheduledAt)
		if diff < -15*time.Minute || diff > 15*time.Minute {
			t.Fatalf("random delay out of bounds: got %v", diff)
		}
	}
}

func TestApplyRandomDelayWithZeroDelayReturnsScheduledTime(t *testing.T) {
	scheduledAt := time.Date(2026, time.May, 1, 12, 0, 0, 0, time.UTC)

	actual := applyRandomDelay(scheduledAt, 0)
	if !actual.Equal(scheduledAt) {
		t.Fatalf("expected unchanged time, got %s want %s", actual, scheduledAt)
	}
}
