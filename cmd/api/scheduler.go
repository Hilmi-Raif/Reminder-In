package main

import (
	"log"
	"math/rand"
	"reminderin/internal/store"
	"reminderin/internal/whatsapp"
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron    *cron.Cron
	store   *store.SQLiteStore
	waMgr   *whatsapp.ClientManager
	running atomic.Bool
}

func NewScheduler(store *store.SQLiteStore, waMgr *whatsapp.ClientManager) *Scheduler {
	c := cron.New(cron.WithSeconds())
	return &Scheduler{
		cron:  c,
		store: store,
		waMgr: waMgr,
	}
}

func (s *Scheduler) Start() {
	_, err := s.cron.AddFunc("*/30 * * * * *", s.processReminders)
	if err != nil {
		log.Printf("Error scheduling reminders: %v", err)
		return
	}

	_, err = s.cron.AddFunc("0 */30 * * * *", s.keepAliveClients)
	if err != nil {
		log.Printf("Error scheduling keep-alive: %v", err)
	}

	_, err = s.cron.AddFunc("0 */5 * * * *", s.checkWAHealth)
	if err != nil {
		log.Printf("Error scheduling WA health check: %v", err)
	}

	s.cron.Start()

	log.Println("Scheduler Service started (checking every 30s)...")
}

func (s *Scheduler) Stop() {
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

func (s *Scheduler) keepAliveClients() {
	waNumber := s.store.GetWANumber()
	if waNumber == "" {
		return
	}

	client, err := s.waMgr.GetClient(waNumber)
	if err != nil || client == nil || !client.IsConnected() {
		if err := s.waMgr.LoadClient(waNumber); err != nil {
			log.Printf("Failed to reconnect WA client %s before keep-alive: %v", waNumber, err)
			return
		}
	}

	err = s.waMgr.SendPresence(waNumber)
	if err != nil {
		log.Printf("Failed to send presence keep-alive for %s: %v", waNumber, err)
	} else {
		log.Printf("Presence keep-alive sent for %s", waNumber)
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
	if err := s.waMgr.LoadClient(waNumber); err != nil {
		log.Printf("WA health reconnect failed for %s: %v", waNumber, err)
	}
}

func randomSendDelay(base time.Duration) time.Duration {
	if base <= 0 {
		return 0
	}
	return base + time.Duration(rand.Int63n(int64(base)+1))
}
