package handler

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"reminderin/internal/store"
	"reminderin/internal/whatsapp"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/skip2/go-qrcode"
)

const (
	maxJSONBodyBytes int64 = 1 << 20
	maxMessageChars        = 4000
)

var reminderCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

var (
	waDirectNumberPattern = regexp.MustCompile(`^\d{6,15}$`)
	waGroupPattern        = regexp.MustCompile(`^\d+-\d+(@g\.us)?$`)
	waJIDPattern          = regexp.MustCompile(`^\d+@(s\.whatsapp\.net|g\.us|broadcast)$`)
)

type APIHandler struct {
	Store       *store.SQLiteStore
	WaMgr       *whatsapp.ClientManager
	LinkLimiter chan struct{}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *APIHandler) CreateReminder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message    string `json:"message"`
		TargetWa   string `json:"target_wa"`
		Recurrence string `json:"recurrence"`
	}
	if !decodeJSONBody(w, r, &req) {
		return
	}

	req.Message = strings.TrimSpace(req.Message)
	req.TargetWa = strings.TrimSpace(req.TargetWa)
	req.Recurrence = strings.TrimSpace(req.Recurrence)

	normalizedTargets, err := normalizeReminderTargets(req.TargetWa)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid target format", map[string]string{"field": "target_wa", "detail": err.Error()})
		return
	}
	req.TargetWa = normalizedTargets

	if req.Message == "" {
		WriteError(w, http.StatusBadRequest, "Message is required", map[string]string{"field": "message"})
		return
	}
	if len([]rune(req.Message)) > maxMessageChars {
		WriteError(w, http.StatusBadRequest, "Message is too long", map[string]string{"field": "message"})
		return
	}

	target := req.TargetWa
	if target == "" {
		target = h.Store.GetWANumber()
	}

	rem := store.Reminder{
		ID:         uuid.Must(uuid.NewV7()),
		Message:    req.Message,
		TargetWa:   target,
		Recurrence: req.Recurrence,
		IsActive:   true,
	}

	now := time.Now()
	if req.Recurrence == "" {
		WriteError(w, http.StatusBadRequest, "Recurrence is required", map[string]string{"field": "recurrence"})
		return
	} else if strings.HasPrefix(req.Recurrence, "plugin:") {
		WriteError(w, http.StatusBadRequest, "Plugin recurrence is not supported", map[string]string{"field": "recurrence"})
		return
	} else {
		nextRun, err := nextScheduledTime(req.Recurrence, now)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid cron expression", map[string]string{"field": "recurrence"})
			return
		}
		rem.ScheduledAt = nextRun
	}

	if err := h.Store.CreateReminder(rem); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to create reminder", nil)
		return
	}

	writeJSON(w, http.StatusCreated, rem)
}

func (h *APIHandler) ListReminders(w http.ResponseWriter, r *http.Request) {
	cursorStr := r.URL.Query().Get("cursor")
	limitStr := r.URL.Query().Get("limit")
	sortBy := r.URL.Query().Get("sortBy")
	sortOrder := r.URL.Query().Get("order")
	search := r.URL.Query().Get("search")

	limit := 50
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	var cursor *uuid.UUID
	if cursorStr != "" {
		parsed, err := uuid.Parse(cursorStr)
		if err == nil {
			cursor = &parsed
		}
	}

	etag := remindersListETag(h.Store.Version(), cursorStr, limit, search, sortBy, sortOrder)
	if match := r.Header.Get("If-None-Match"); match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	rems, nextCursor, total := h.Store.GetReminders(cursor, limit, search, sortBy, sortOrder)

	w.Header().Set("ETag", etag)

	response := map[string]interface{}{
		"data":        rems,
		"next_cursor": nextCursor,
		"total":       total,
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *APIHandler) DeleteReminder(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid ID", map[string]string{"field": "id"})
		return
	}

	err = h.Store.DeleteReminder(id)
	if err != nil {
		if errors.Is(err, store.ErrReminderNotFound) {
			WriteError(w, http.StatusNotFound, "Reminder not found", nil)
			return
		}
		WriteError(w, http.StatusInternalServerError, "Failed to delete reminder", nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *APIHandler) DeleteAllReminders(w http.ResponseWriter, r *http.Request) {
	err := h.Store.DeleteAllReminders()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to delete reminders", nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *APIHandler) UpdateReminder(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid ID", map[string]string{"field": "id"})
		return
	}

	var req struct {
		Message    string `json:"message"`
		TargetWa   string `json:"target_wa"`
		Recurrence string `json:"recurrence"`
	}
	if !decodeJSONBody(w, r, &req) {
		return
	}

	req.Message = strings.TrimSpace(req.Message)
	req.TargetWa = strings.TrimSpace(req.TargetWa)
	req.Recurrence = strings.TrimSpace(req.Recurrence)

	normalizedTargets, targetErr := normalizeReminderTargets(req.TargetWa)
	if targetErr != nil {
		WriteError(w, http.StatusBadRequest, "Invalid target format", map[string]string{"field": "target_wa", "detail": targetErr.Error()})
		return
	}
	req.TargetWa = normalizedTargets

	if req.Message == "" {
		WriteError(w, http.StatusBadRequest, "Message is required", map[string]string{"field": "message"})
		return
	}
	if len([]rune(req.Message)) > maxMessageChars {
		WriteError(w, http.StatusBadRequest, "Message is too long", map[string]string{"field": "message"})
		return
	}

	now := time.Now()
	if req.Recurrence == "" {
		WriteError(w, http.StatusBadRequest, "Recurrence is required", map[string]string{"field": "recurrence"})
		return
	} else if strings.HasPrefix(req.Recurrence, "plugin:") {
		WriteError(w, http.StatusBadRequest, "Plugin recurrence is not supported", map[string]string{"field": "recurrence"})
		return
	} else {
		nextRun, err := nextScheduledTime(req.Recurrence, now)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid cron expression", map[string]string{"field": "recurrence"})
			return
		}
		updated := store.Reminder{
			Message:     req.Message,
			TargetWa:    req.TargetWa,
			Recurrence:  req.Recurrence,
			ScheduledAt: nextRun,
			IsActive:    true,
		}

		err = h.Store.UpdateReminder(id, updated)
		if err != nil {
			if errors.Is(err, store.ErrReminderNotFound) {
				WriteError(w, http.StatusNotFound, "Reminder not found", nil)
				return
			}
			WriteError(w, http.StatusInternalServerError, "Failed to update reminder", nil)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
		return
	}
}

func (h *APIHandler) ToggleReminder(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid ID", map[string]string{"field": "id"})
		return
	}

	rem, err := h.Store.ToggleReminderActive(id)
	if err != nil {
		if errors.Is(err, store.ErrReminderNotFound) {
			WriteError(w, http.StatusNotFound, "Reminder not found", nil)
			return
		}
		if errors.Is(err, store.ErrInvalidRecurrence) {
			WriteError(w, http.StatusBadRequest, "Invalid cron expression", map[string]string{"field": "recurrence"})
			return
		}
		WriteError(w, http.StatusInternalServerError, "Failed to toggle reminder", nil)
		return
	}
	writeJSON(w, http.StatusOK, rem)
}

func (h *APIHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	waNumber := h.Store.GetWANumber()
	if waNumber == "" {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}

	groups, err := h.WaMgr.GetJoinedGroups(waNumber)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	writeJSON(w, http.StatusOK, groups)
}

func (h *APIHandler) ListContacts(w http.ResponseWriter, r *http.Request) {
	waNumber := h.Store.GetWANumber()
	if waNumber == "" {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}

	contacts, err := h.WaMgr.GetContacts(waNumber)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	writeJSON(w, http.StatusOK, contacts)
}

func (h *APIHandler) GetWAStatus(w http.ResponseWriter, r *http.Request) {
	waNumber := h.Store.GetWANumber()
	if waNumber == "" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "not_linked"})
		return
	}

	client, err := h.WaMgr.GetClient(waNumber)
	if err != nil || client == nil || !client.IsConnected() {
		writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected", "number": waNumber})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "connected", "number": waNumber})
}

func (h *APIHandler) GetWAHealth(w http.ResponseWriter, r *http.Request) {
	waNumber := h.Store.GetWANumber()
	logoutReason := h.Store.GetWALogoutReason()
	connected := false
	if waNumber != "" {
		connected = h.WaMgr.IsConnected(waNumber)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"number":        waNumber,
		"connected":     connected,
		"logout_reason": logoutReason,
	})
}

func (h *APIHandler) UnlinkWA(w http.ResponseWriter, r *http.Request) {
	waNumber := h.Store.GetWANumber()
	if waNumber == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "not linked"})
		return
	}

	_ = h.WaMgr.Logout(waNumber)
	if err := h.Store.UpdateWANumber(""); err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to unlink whatsapp", nil)
		return
	}
	_ = h.Store.ClearWALogoutReason()
	w.WriteHeader(http.StatusNoContent)
}

func (h *APIHandler) replaceLinkedWANumber(newNumber string) error {
	oldNumber := h.Store.GetWANumber()
	if err := h.Store.UpdateWANumber(newNumber); err != nil {
		return err
	}
	_ = h.Store.ClearWALogoutReason()
	if oldNumber != "" && oldNumber != newNumber {
		_ = h.WaMgr.Logout(oldNumber)
	}
	return nil
}

func (h *APIHandler) GetQR(w http.ResponseWriter, r *http.Request) {
	if !h.acquireLinkSlot() {
		WriteError(w, http.StatusTooManyRequests, "Too many active link sessions", nil)
		return
	}
	defer h.releaseLinkSlot()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteError(w, http.StatusInternalServerError, "streaming unsupported", nil)
		return
	}

	client := h.WaMgr.GetNewAuthClient()
	linked := false
	defer func() {
		if !linked {
			client.Disconnect()
		}
	}()

	qrChan, err := h.WaMgr.GetLinkQR(client)
	if err != nil {
		sendSSE(w, flusher, map[string]string{"type": "error", "message": err.Error()})
		return
	}

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-qrChan:
			if !ok {
				return
			}
			switch evt.Event {
			case "code":
				qrImage, err := qrCodeDataURI(evt.Code)
				if err != nil {
					sendSSE(w, flusher, map[string]string{"type": "error", "message": "failed to render qr"})
					return
				}
				sendSSE(w, flusher, map[string]string{"type": "qr", "image": qrImage, "code": evt.Code})
			case "success":
				if client.Store.ID == nil {
					sendSSE(w, flusher, map[string]string{"type": "error", "message": "missing linked account"})
					return
				}
				waNumber := client.Store.ID.User
				if err := h.replaceLinkedWANumber(waNumber); err != nil {
					sendSSE(w, flusher, map[string]string{"type": "error", "message": "failed to save linked number"})
					return
				}
				h.WaMgr.AddClient(client)
				linked = true
				sendSSE(w, flusher, map[string]string{"type": "success", "number": waNumber})
				return
			case "error", "timeout":
				sendSSE(w, flusher, map[string]string{"type": "error", "message": evt.Event})
				return
			}
		}
	}
}

func (h *APIHandler) GetPairCode(w http.ResponseWriter, r *http.Request) {
	if !h.acquireLinkSlot() {
		WriteError(w, http.StatusTooManyRequests, "Too many active link sessions", nil)
		return
	}
	defer h.releaseLinkSlot()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteError(w, http.StatusInternalServerError, "streaming unsupported", nil)
		return
	}

	phone := normalizePhone(r.URL.Query().Get("phone"))
	if len(phone) < 8 || len(phone) > 20 {
		sendSSE(w, flusher, map[string]string{"type": "error", "message": "Phone number required"})
		return
	}

	client := h.WaMgr.GetNewAuthClient()
	linked := false
	defer func() {
		if !linked {
			client.Disconnect()
		}
	}()

	ch, err := client.GetQRChannel(r.Context())
	if err != nil {
		sendSSE(w, flusher, map[string]string{"type": "error", "message": err.Error()})
		return
	}

	code, err := h.WaMgr.GetLinkCode(client, phone)
	if err != nil {
		sendSSE(w, flusher, map[string]string{"type": "error", "message": err.Error()})
		return
	}

	sendSSE(w, flusher, map[string]string{"type": "code", "code": code})

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			if evt.Event == "success" {
				if client.Store.ID == nil {
					sendSSE(w, flusher, map[string]string{"type": "error", "message": "missing linked account"})
					return
				}
				waNumber := client.Store.ID.User
				if err := h.replaceLinkedWANumber(waNumber); err != nil {
					sendSSE(w, flusher, map[string]string{"type": "error", "message": "failed to save linked number"})
					return
				}
				h.WaMgr.AddClient(client)
				linked = true
				sendSSE(w, flusher, map[string]string{"type": "success", "number": waNumber})
				return
			}
			if evt.Event == "error" || evt.Event == "timeout" {
				sendSSE(w, flusher, map[string]string{"type": "error", "message": evt.Event})
				return
			}
		}
	}
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			WriteError(w, http.StatusRequestEntityTooLarge, "Request body too large", nil)
			return false
		}
		WriteError(w, http.StatusBadRequest, "Invalid request body", nil)
		return false
	}

	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		WriteError(w, http.StatusBadRequest, "Invalid request body", nil)
		return false
	}
	return true
}

func nextScheduledTime(recurrence string, requested time.Time) (time.Time, error) {
	sched, err := reminderCronParser.Parse(recurrence)
	if err != nil {
		return time.Time{}, err
	}
	return sched.Next(requested), nil
}

func normalizePhone(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(trimmed))
	for _, r := range trimmed {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func normalizeReminderTargets(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", nil
	}

	targets := store.ParseTargets(input)
	if len(targets) == 0 {
		return "", nil
	}
	for _, target := range targets {
		if !isValidReminderTarget(target) {
			return "", fmt.Errorf("invalid target %q", target)
		}
	}

	return strings.Join(targets, ","), nil
}

func isValidReminderTarget(target string) bool {
	if waDirectNumberPattern.MatchString(target) {
		return true
	}
	if waGroupPattern.MatchString(target) {
		return true
	}
	if waJIDPattern.MatchString(target) {
		return true
	}
	return false
}

func qrCodeDataURI(code string) (string, error) {
	pngBytes, err := qrcode.Encode(code, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes), nil
}

func sendSSE(w http.ResponseWriter, flusher http.Flusher, payload interface{}) {
	body, err := json.Marshal(payload)
	if err != nil {
		body = []byte(`{"type":"error","message":"internal error"}`)
	}
	_, _ = fmt.Fprintf(w, "data: %s\n\n", body)
	flusher.Flush()
}

func remindersListETag(version uint64, cursor string, limit int, search, sortBy, sortOrder string) string {
	raw := fmt.Sprintf(
		"v=%d|c=%s|l=%d|q=%s|s=%s|o=%s",
		version,
		cursor,
		limit,
		search,
		sortBy,
		sortOrder,
	)
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf(`"r%x"`, sum[:8])
}

func (h *APIHandler) acquireLinkSlot() bool {
	if h == nil || h.LinkLimiter == nil {
		return true
	}
	select {
	case h.LinkLimiter <- struct{}{}:
		return true
	default:
		return false
	}
}

func (h *APIHandler) releaseLinkSlot() {
	if h == nil || h.LinkLimiter == nil {
		return
	}
	select {
	case <-h.LinkLimiter:
	default:
	}
}
