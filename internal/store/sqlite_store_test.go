package store

import (
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
