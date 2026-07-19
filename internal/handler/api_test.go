package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"reminderin/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func newTestAPIHandler(t *testing.T) *APIHandler {
	t.Helper()

	db, err := store.NewSQLiteStore(t.TempDir() + "/reminderin.db")
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return &APIHandler{Store: db}
}

func decodeReminderResponse(t *testing.T, rr *httptest.ResponseRecorder) store.Reminder {
	t.Helper()

	var rem store.Reminder
	if err := json.NewDecoder(rr.Body).Decode(&rem); err != nil {
		t.Fatalf("decode reminder response: %v", err)
	}
	return rem
}

func TestCreateRecurringReminderRejectsClientScheduledAt(t *testing.T) {
	h := newTestAPIHandler(t)
	future := time.Date(2027, 7, 6, 0, 0, 0, 0, time.FixedZone("WIB", 7*60*60))
	payload := map[string]string{
		"message":      "test recurring reminder",
		"target_wa":    "6281234567890",
		"recurrence":   "* * * * *",
		"scheduled_at": future.Format(time.RFC3339),
	}
	body, _ := json.Marshal(payload)

	r := httptest.NewRequest(http.MethodPost, "/api/reminders", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateReminder(rr, r)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestCreateRecurringReminderDoesNotRequireScheduledAt(t *testing.T) {
	h := newTestAPIHandler(t)
	payload := map[string]string{
		"message":    "test recurring reminder",
		"target_wa":  "6281234567890",
		"recurrence": "* * * * *",
	}
	body, _ := json.Marshal(payload)

	before := time.Now()
	r := httptest.NewRequest(http.MethodPost, "/api/reminders", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateReminder(rr, r)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}
	rem := decodeReminderResponse(t, rr)
	if !rem.ScheduledAt.After(before) {
		t.Fatalf("expected next run after test start, got %s", rem.ScheduledAt)
	}
	if rem.ScheduledAt.After(before.Add(2 * time.Minute)) {
		t.Fatalf("expected next run to be computed from now, got %s", rem.ScheduledAt)
	}
}

func TestCreateReminderRequiresRecurrence(t *testing.T) {
	h := newTestAPIHandler(t)
	payload := map[string]string{
		"message":   "test recurring reminder",
		"target_wa": "6281234567890",
	}
	body, _ := json.Marshal(payload)

	r := httptest.NewRequest(http.MethodPost, "/api/reminders", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateReminder(rr, r)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestUpdateRecurringReminderRecomputesNextRunFromNow(t *testing.T) {
	h := newTestAPIHandler(t)
	id := uuid.Must(uuid.NewV7())
	future := time.Date(2027, 7, 6, 0, 0, 0, 0, time.FixedZone("WIB", 7*60*60))
	if err := h.Store.CreateReminder(store.Reminder{
		ID:          id,
		Message:     "test recurring reminder",
		TargetWa:    "6281234567890",
		Recurrence:  "0 0 6 7 *",
		ScheduledAt: future,
		IsActive:    true,
	}); err != nil {
		t.Fatalf("create reminder: %v", err)
	}

	payload := map[string]string{
		"message":    "test recurring reminder edited",
		"target_wa":  "6281234567890",
		"recurrence": "* * * * *",
	}
	body, _ := json.Marshal(payload)

	before := time.Now()
	r := httptest.NewRequest(http.MethodPut, "/api/reminders/"+id.String(), bytes.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateReminder(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	reminders, _, _ := h.Store.GetReminders(nil, 10, "", "time", "asc")
	if len(reminders) != 1 {
		t.Fatalf("expected 1 reminder, got %d", len(reminders))
	}
	got := reminders[0].ScheduledAt
	if !got.After(before) {
		t.Fatalf("expected next run after test start, got %s", got)
	}
	if got.After(before.Add(2 * time.Minute)) {
		t.Fatalf("expected next run to be recomputed from now, got %s", got)
	}
}

func TestUpdateRecurringReminderDoesNotRequireScheduledAt(t *testing.T) {
	h := newTestAPIHandler(t)
	id := uuid.Must(uuid.NewV7())
	future := time.Date(2027, 7, 6, 0, 0, 0, 0, time.FixedZone("WIB", 7*60*60))
	if err := h.Store.CreateReminder(store.Reminder{
		ID:          id,
		Message:     "test recurring reminder",
		TargetWa:    "6281234567890",
		Recurrence:  "0 0 6 7 *",
		ScheduledAt: future,
		IsActive:    true,
	}); err != nil {
		t.Fatalf("create reminder: %v", err)
	}

	payload := map[string]string{
		"message":    "test recurring reminder edited",
		"target_wa":  "6281234567890",
		"recurrence": "* * * * *",
	}
	body, _ := json.Marshal(payload)

	before := time.Now()
	r := httptest.NewRequest(http.MethodPut, "/api/reminders/"+id.String(), bytes.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateReminder(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	reminders, _, _ := h.Store.GetReminders(nil, 10, "", "time", "asc")
	if len(reminders) != 1 {
		t.Fatalf("expected 1 reminder, got %d", len(reminders))
	}
	got := reminders[0].ScheduledAt
	if !got.After(before) {
		t.Fatalf("expected next run after test start, got %s", got)
	}
	if got.After(before.Add(2 * time.Minute)) {
		t.Fatalf("expected next run to be recomputed from now, got %s", got)
	}
}

func TestUpdateReminderRequiresRecurrence(t *testing.T) {
	h := newTestAPIHandler(t)
	id := uuid.Must(uuid.NewV7())
	if err := h.Store.CreateReminder(store.Reminder{
		ID:          id,
		Message:     "test recurring reminder",
		TargetWa:    "6281234567890",
		Recurrence:  "* * * * *",
		ScheduledAt: time.Now().Add(time.Minute),
		IsActive:    true,
	}); err != nil {
		t.Fatalf("create reminder: %v", err)
	}

	payload := map[string]string{
		"message":   "test recurring reminder edited",
		"target_wa": "6281234567890",
	}
	body, _ := json.Marshal(payload)

	r := httptest.NewRequest(http.MethodPut, "/api/reminders/"+id.String(), bytes.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateReminder(rr, r)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}
