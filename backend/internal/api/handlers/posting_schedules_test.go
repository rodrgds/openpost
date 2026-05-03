package handlers

import (
	"testing"
	"time"

	"github.com/openpost/backend/internal/models"
)

func TestFindNextAvailableSlotTimeReturnsFirstFreeScheduledSlot(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, time.May, 4, 8, 0, 0, 0, loc)
	schedules := []models.PostingSchedule{
		{ID: "slot-1", DayOfWeek: int(time.Monday), UTCHour: 9, UTCMinute: 0},
		{ID: "slot-2", DayOfWeek: int(time.Monday), UTCHour: 17, UTCMinute: 0},
	}

	slot, when := findNextAvailableSlotTime(now, loc, schedules, nil, 60)
	if slot == nil {
		t.Fatal("expected a slot")
	}
	if slot.ID != "slot-1" {
		t.Fatalf("expected first slot, got %q", slot.ID)
	}
	expected := time.Date(2026, time.May, 4, 9, 0, 0, 0, loc)
	if !when.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected, when)
	}
}

func TestFindNextAvailableSlotTimeFallsBackToDraftGapWhenDayIsFull(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, time.May, 4, 8, 0, 0, 0, loc)
	schedules := []models.PostingSchedule{
		{ID: "slot-1", DayOfWeek: int(time.Monday), UTCHour: 9, UTCMinute: 0},
	}
	scheduledPosts := []models.Post{
		{ScheduledAt: time.Date(2026, time.May, 4, 9, 0, 0, 0, time.UTC)},
		{ScheduledAt: time.Date(2026, time.May, 4, 11, 0, 0, 0, time.UTC)},
	}

	slot, when := findNextAvailableSlotTime(now, loc, schedules, scheduledPosts, 90)
	if slot != nil {
		t.Fatalf("expected gap fallback without a matching schedule slot, got %q", slot.ID)
	}
	expected := time.Date(2026, time.May, 4, 12, 30, 0, 0, loc)
	if !when.Equal(expected) {
		t.Fatalf("expected fallback %s, got %s", expected, when)
	}
}

func TestFindNextAvailableSlotTimePreservesLocalTimeAcrossDST(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	now := time.Date(2026, time.October, 24, 8, 30, 0, 0, loc)
	schedules := []models.PostingSchedule{
		{ID: "slot-1", DayOfWeek: int(time.Sunday), UTCHour: 9, UTCMinute: 0},
	}

	slot, when := findNextAvailableSlotTime(now, loc, schedules, nil, 60)
	if slot == nil {
		t.Fatal("expected a slot")
	}

	expected := time.Date(2026, time.October, 25, 9, 0, 0, 0, loc)
	if !when.Equal(expected) {
		t.Fatalf("expected local slot %s, got %s", expected, when)
	}
	if when.UTC().Hour() != 9 {
		t.Fatalf("expected DST-adjusted UTC hour 9 after fallback, got %d", when.UTC().Hour())
	}
}

func TestPostingScheduleResponseForWorkspaceReturnsStoredLocalFields(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	schedule := models.PostingSchedule{
		ID:          "slot-1",
		WorkspaceID: "workspace-1",
		DayOfWeek:   int(time.Monday),
		UTCHour:     9,
		UTCMinute:   15,
	}

	resp := postingScheduleResponseForWorkspace(time.Date(2026, time.January, 5, 0, 0, 0, 0, loc), loc, schedule)
	if resp.LocalDayOfWeek != int(time.Monday) || resp.LocalHour != 9 || resp.LocalMinute != 15 {
		t.Fatalf("expected local fields to mirror stored wall-clock schedule, got %+v", resp)
	}
}
