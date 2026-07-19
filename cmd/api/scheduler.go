package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"reminderin/internal/store"
	"reminderin/internal/whatsapp"
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"
	"go.mau.fi/whatsmeow/types"
)

type Scheduler struct {
	cron              *cron.Cron
	store             *store.SQLiteStore
	waMgr             *whatsapp.ClientManager
	running           atomic.Bool
	keepaliveInterval time.Duration
	keepaliveStop     chan struct{}
	healthStop        chan struct{}
}

func NewScheduler(store *store.SQLiteStore, waMgr *whatsapp.ClientManager, keepaliveInterval time.Duration) *Scheduler {
	c := cron.New(cron.WithSeconds())
	return &Scheduler{
		cron:              c,
		store:             store,
		waMgr:             waMgr,
		keepaliveInterval: keepaliveInterval,
		keepaliveStop:     make(chan struct{}),
		healthStop:        make(chan struct{}),
	}
}

func keepaliveIntervalFromEnv() time.Duration {
	minutes := nonNegativeIntFromEnv("WA_KEEPALIVE_MINUTES", 5)
	if minutes <= 0 {
		minutes = 5
	}
	return time.Duration(minutes) * time.Minute
}

func (s *Scheduler) Start() {
	_, err := s.cron.AddFunc("0 * * * * *", s.processReminders)
	if err != nil {
		log.Printf("Error scheduling reminders: %v", err)
		return
	}

	s.cron.Start()
	go s.healthCheckLoop()
	go s.smartKeepaliveLoop()

	log.Printf("Scheduler started (reminders: 1m, health: 5m, smart-keepalive: active)")
}

func (s *Scheduler) Stop() {
	close(s.healthStop)
	close(s.keepaliveStop)
	s.cron.Stop()
}

func (s *Scheduler) processReminders() {
	if !s.running.CompareAndSwap(false, true) {
		return
	}
	defer s.running.Store(false)

	// Add Jitter (0-15 seconds) so it doesn't trigger exactly on the minute
	jitter := time.Duration(rand.Intn(16)) * time.Second
	time.Sleep(jitter)

	waNumber := s.store.GetWANumber()
	if waNumber == "" {
		return
	}

	client, err := s.waMgr.GetClient(waNumber)
	if err != nil || client == nil || !client.IsConnected() {
		if ensureErr := s.waMgr.EnsureClient(waNumber); ensureErr != nil {
			log.Printf("Reminder WA recovery: failed to reload client %s: %v", waNumber, ensureErr)
		}
		return
	}

	s.store.ProcessDueReminders(func(rem store.Reminder) error {
		targets := store.ParseTargets(rem.TargetWa)
		if len(targets) == 0 {
			targets = []string{waNumber}
		}

		var lastErr error
		failed := 0
		dispatchAt := time.Now()
		for i, target := range targets {
			sent, err := s.store.HasTargetDispatchMark(rem.ID, rem.ScheduledAt, target)
			if err != nil {
				log.Printf("Failed to read dispatch mark for reminder %s target %s: %v", rem.ID, target, err)
				return err
			}
			if sent {
				continue
			}

			err = s.waMgr.SendMessage(waNumber, target, rem.Message)
			if err != nil {
				log.Printf("Failed to send WA reminder %s to %s: %v", rem.ID, target, err)
				lastErr = err
				failed++
				continue
			}

			if err := s.store.PutTargetDispatchMark(rem.ID, rem.ScheduledAt, target, dispatchAt); err != nil {
				log.Printf("Failed to save dispatch mark for reminder %s target %s: %v", rem.ID, target, err)
				return err
			}

			log.Printf("WA Reminder %s sent successfully to %s", rem.ID, target)

			// Larger more human-like delay between messages (base + 3-8 seconds)
			if delay := randomSendDelay(s.waMgr.SendDelay()); delay > 0 && i < len(targets)-1 {
				extraDelay := time.Duration(rand.Intn(6)+3) * time.Second
				time.Sleep(delay + extraDelay)
			}
		}

		if failed > 0 {
			log.Printf("WA Reminder %s had partial delivery failure: %d/%d failed (error: %v)", rem.ID, failed, len(targets), lastErr)
			return reminderDeliveryError(failed, lastErr)
		}

		if rem.Recurrence != "" {
			log.Printf("WA Reminder %s will be rescheduled", rem.ID)
		}
		return nil
	})
}

func reminderDeliveryError(failed int, lastErr error) error {
	if failed <= 0 {
		return nil
	}
	if lastErr == nil {
		return fmt.Errorf("%d reminder target deliveries failed", failed)
	}
	return fmt.Errorf("%d reminder target deliveries failed: %w", failed, lastErr)
}

func (s *Scheduler) smartKeepaliveLoop() {
	// Tunggu 5-15 menit pertama kali saat startup agar tidak langsung ping
	startupJitter := time.Duration(rand.Intn(10)+5) * time.Minute

	startupTimer := time.NewTimer(startupJitter)
	select {
	case <-startupTimer.C:
	case <-s.keepaliveStop:
		startupTimer.Stop()
		return
	}

	for {
		s.checkAndSendSmartKeepalive()

		// Sleep untuk 20 - 28 jam (harian acak)
		dailyJitter := time.Duration(rand.Intn(8)+20) * time.Hour
		timer := time.NewTimer(dailyJitter)
		select {
		case <-timer.C:
		case <-s.keepaliveStop:
			timer.Stop()
			return
		}
	}
}

func (s *Scheduler) checkAndSendSmartKeepalive() {
	waNumber := s.store.GetWANumber()
	if waNumber == "" {
		return
	}

	lastPingStr := s.store.GetSetting("wa_last_keepalive_time")
	var lastPing time.Time

	if lastPingStr != "" {
		parsed, err := time.Parse(time.RFC3339, lastPingStr)
		if err == nil {
			lastPing = parsed
		}
	}

	// Tentukan durasi target ping berikutnya (antara 5-9 hari)
	// Kita ambil 7 hari sebagai base (168 jam) dan beri jitter -2 s/d +2 hari
	targetDuration := time.Duration(120+rand.Intn(96)) * time.Hour

	// Jika ini ping pertama (lastPing kosong) ATAU sudah melewati targetDuration
	if lastPing.IsZero() || time.Since(lastPing) >= targetDuration {
		client, err := s.waMgr.GetClient(waNumber)
		if err == nil && client != nil && client.IsConnected() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			log.Printf("Smart Keepalive: Mengirim status PresenceAvailable (Online) sementara")

			// Send Online
			if err := client.SendPresence(ctx, types.PresenceAvailable); err != nil {
				log.Printf("Smart Keepalive failed to send presence: %v", err)
				return // Gagal, coba lagi di cycle berikutnya tanpa update waktu
			}

			// Tahan online sebentar (15 - 45 detik) layaknya orang cek WA
			time.Sleep(time.Duration(rand.Intn(30)+15) * time.Second)

			// Send Offline
			_ = client.SendPresence(context.Background(), types.PresenceUnavailable)

			// Simpan waktu berhasil
			if err := s.store.SetSetting("wa_last_keepalive_time", time.Now().Format(time.RFC3339)); err != nil {
				log.Printf("Smart Keepalive failed to save state: %v", err)
			} else {
				log.Printf("Smart Keepalive berhasil. Menyimpan state untuk siklus berikutnya.")
			}
		}
	} else {
		log.Printf("Smart Keepalive: Belum waktunya. Waktu tersisa: %v", targetDuration-time.Since(lastPing))
	}
}

func (s *Scheduler) healthCheckLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkWAHealth()
		case <-s.healthStop:
			return
		}
	}
}

func (s *Scheduler) checkWAHealth() {
	waNumber := s.store.GetWANumber()
	if waNumber == "" {
		return
	}
	if s.waMgr.IsConnected(waNumber) {
		return
	}
	if err := s.waMgr.EnsureClient(waNumber); err != nil {
		log.Printf("WA health: client %s is disconnected and reload failed: %v", waNumber, err)
		return
	}
	log.Printf("WA health: client %s is disconnected, recovery checked", waNumber)
}

func randomSendDelay(base time.Duration) time.Duration {
	if base <= 0 {
		return 0
	}
	return base + time.Duration(rand.Int63n(int64(base)+1))
}
