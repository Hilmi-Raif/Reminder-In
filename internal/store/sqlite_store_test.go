package store

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newTestSQLiteStore(t *testing.T) *SQLiteStore {
	t.Helper()

	s, err := NewSQLiteStore(t.TempDir() + "/reminderin.db")
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestGetRemindersSkipsCorruptRows(t *testing.T) {
	s := newTestSQLiteStore(t)
	validID := uuid.Must(uuid.NewV7())
	validTime := time.Now().Add(time.Minute)

	_, err := s.db.Exec(
		"INSERT INTO reminders (id, message, target_wa, recurrence, scheduled_at, is_active) VALUES (?, ?, ?, ?, ?, ?)",
		validID.String(), "valid reminder", "6281234567890", "* * * * *", validTime, 1,
	)
	if err != nil {
		t.Fatalf("insert valid reminder: %v", err)
	}
	_, err = s.db.Exec(
		"INSERT INTO reminders (id, message, target_wa, recurrence, scheduled_at, is_active) VALUES (?, ?, ?, ?, ?, ?)",
		"not-a-uuid", "invalid uuid reminder", "6281234567890", "* * * * *", validTime, 1,
	)
	if err != nil {
		t.Fatalf("insert invalid uuid reminder: %v", err)
	}
	_, err = s.db.Exec(
		"INSERT INTO reminders (id, message, target_wa, recurrence, scheduled_at, is_active) VALUES (?, ?, ?, ?, ?, ?)",
		uuid.Must(uuid.NewV7()).String(), "invalid time reminder", "6281234567890", "* * * * *", "not-a-time", 1,
	)
	if err != nil {
		t.Fatalf("insert invalid time reminder: %v", err)
	}

	reminders, _, _ := s.GetReminders(nil, 10, "", "time", "asc")
	if len(reminders) != 1 {
		t.Fatalf("expected 1 valid reminder, got %d", len(reminders))
	}
	if reminders[0].ID != validID {
		t.Fatalf("expected valid reminder %s, got %s", validID, reminders[0].ID)
	}
}

func TestProcessDueRemindersSkipsCorruptRows(t *testing.T) {
	s := newTestSQLiteStore(t)
	dueTime := time.Now().Add(-time.Minute)

	_, err := s.db.Exec(
		"INSERT INTO reminders (id, message, target_wa, recurrence, scheduled_at, is_active) VALUES (?, ?, ?, ?, ?, ?)",
		"not-a-uuid", "invalid uuid reminder", "6281234567890", "* * * * *", dueTime, 1,
	)
	if err != nil {
		t.Fatalf("insert invalid uuid reminder: %v", err)
	}
	_, err = s.db.Exec(
		"INSERT INTO reminders (id, message, target_wa, recurrence, scheduled_at, is_active) VALUES (?, ?, ?, ?, ?, ?)",
		uuid.Must(uuid.NewV7()).String(), "invalid time reminder", "6281234567890", "* * * * *", "not-a-time", 1,
	)
	if err != nil {
		t.Fatalf("insert invalid time reminder: %v", err)
	}

	called := false
	s.ProcessDueReminders(func(rem Reminder) error {
		called = true
		return nil
	})

	if called {
		t.Fatal("expected corrupt reminders to be skipped")
	}
}

func TestToggleReminderActiveDisablesActiveReminder(t *testing.T) {
	s := newTestSQLiteStore(t)
	id := uuid.Must(uuid.NewV7())
	scheduledAt := time.Now().Add(time.Hour).Truncate(time.Second)

	if err := s.CreateReminder(Reminder{
		ID:          id,
		Message:     "test recurring reminder",
		TargetWa:    "6281234567890",
		Recurrence:  "* * * * *",
		ScheduledAt: scheduledAt,
		IsActive:    true,
	}); err != nil {
		t.Fatalf("create reminder: %v", err)
	}

	if _, err := s.ToggleReminderActive(id); err != nil {
		t.Fatalf("toggle reminder: %v", err)
	}

	reminders, _, _ := s.GetReminders(nil, 10, "", "time", "asc")
	if len(reminders) != 1 {
		t.Fatalf("expected 1 reminder, got %d", len(reminders))
	}
	if reminders[0].IsActive {
		t.Fatal("expected reminder to be inactive")
	}
	if !reminders[0].ScheduledAt.Equal(scheduledAt) {
		t.Fatalf("expected scheduled_at to stay %s, got %s", scheduledAt, reminders[0].ScheduledAt)
	}
}

func TestToggleReminderActiveRecalculatesWhenEnablingRecurring(t *testing.T) {
	s := newTestSQLiteStore(t)
	id := uuid.Must(uuid.NewV7())
	staleTime := time.Date(2027, 7, 6, 0, 0, 0, 0, time.UTC)

	if err := s.CreateReminder(Reminder{
		ID:          id,
		Message:     "test recurring reminder",
		TargetWa:    "6281234567890",
		Recurrence:  "* * * * *",
		ScheduledAt: staleTime,
		IsActive:    false,
	}); err != nil {
		t.Fatalf("create reminder: %v", err)
	}

	before := time.Now()
	if _, err := s.ToggleReminderActive(id); err != nil {
		t.Fatalf("toggle reminder: %v", err)
	}

	reminders, _, _ := s.GetReminders(nil, 10, "", "time", "asc")
	if len(reminders) != 1 {
		t.Fatalf("expected 1 reminder, got %d", len(reminders))
	}
	if !reminders[0].IsActive {
		t.Fatal("expected reminder to be active")
	}
	if !reminders[0].ScheduledAt.After(before) {
		t.Fatalf("expected recalculated scheduled_at after test start, got %s", reminders[0].ScheduledAt)
	}
	if reminders[0].ScheduledAt.After(before.Add(2 * time.Minute)) {
		t.Fatalf("expected scheduled_at to be recalculated from now, got %s", reminders[0].ScheduledAt)
	}
}

func TestToggleReminderActiveRejectsInvalidRecurrenceWhenEnabling(t *testing.T) {
	s := newTestSQLiteStore(t)
	id := uuid.Must(uuid.NewV7())
	staleTime := time.Date(2027, 7, 6, 0, 0, 0, 0, time.UTC)

	if err := s.CreateReminder(Reminder{
		ID:          id,
		Message:     "test recurring reminder",
		TargetWa:    "6281234567890",
		Recurrence:  "not a cron",
		ScheduledAt: staleTime,
		IsActive:    false,
	}); err != nil {
		t.Fatalf("create reminder: %v", err)
	}

	if _, err := s.ToggleReminderActive(id); !errors.Is(err, ErrInvalidRecurrence) {
		t.Fatalf("expected ErrInvalidRecurrence, got %v", err)
	}

	reminders, _, _ := s.GetReminders(nil, 10, "", "time", "asc")
	if len(reminders) != 1 {
		t.Fatalf("expected 1 reminder, got %d", len(reminders))
	}
	if reminders[0].IsActive {
		t.Fatal("expected reminder to remain inactive")
	}
	if !reminders[0].ScheduledAt.Equal(staleTime) {
		t.Fatalf("expected scheduled_at to stay %s, got %s", staleTime, reminders[0].ScheduledAt)
	}
}
