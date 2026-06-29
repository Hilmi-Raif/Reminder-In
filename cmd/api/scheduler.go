package main

import (
	"context"
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
	_, err := s.cron.AddFunc("*/30 * * * * *", s.processReminders)
	if err != nil {
		log.Printf("Error scheduling reminders: %v", err)
		return
	}

	s.cron.Start()
	go s.keepaliveLoop()
	go s.healthCheckLoop()

	log.Printf("Scheduler started (reminders: 30s, keepalive: %v, health: 5m)", s.keepaliveInterval)
}

func (s *Scheduler) Stop() {
	close(s.keepaliveStop)
	close(s.healthStop)
	s.cron.Stop()
}

func (s *Scheduler) processReminders() {
	if !s.running.CompareAndSwap(false, true) {
		return
	}
	defer s.running.Store(false)

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
			if delay := randomSendDelay(s.waMgr.SendDelay()); delay > 0 && i < len(targets)-1 {
				time.Sleep(delay)
			}
		}

		if failed > 0 {
			log.Printf("WA Reminder %s had partial delivery failure: %d/%d failed (error: %v)", rem.ID, failed, len(targets), lastErr)

		}

		if rem.Recurrence != "" {
			log.Printf("WA Reminder %s will be rescheduled", rem.ID)
		}
		return nil
	})
}

func (s *Scheduler) keepaliveLoop() {
	if s.keepaliveInterval <= 0 {
		return
	}
	ticker := time.NewTicker(s.keepaliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.sendKeepalive()
		case <-s.keepaliveStop:
			return
		}
	}
}

func (s *Scheduler) sendKeepalive() {
	waNumber := s.store.GetWANumber()
	if waNumber == "" {
		return
	}

	client, err := s.waMgr.GetClient(waNumber)
	if err != nil || client == nil || !client.IsConnected() {
		if ensureErr := s.waMgr.EnsureClient(waNumber); ensureErr != nil {
			log.Printf("Keepalive: WA client %s not connected and reload failed: %v", waNumber, ensureErr)
		} else {
			log.Printf("Keepalive: WA client %s not connected, recovery checked", waNumber)
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.SendPresence(ctx, types.PresenceAvailable); err != nil {
		log.Printf("Keepalive: failed to send presence for %s: %v", waNumber, err)
	} else {
		log.Printf("Keepalive: presence sent for %s", waNumber)
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
