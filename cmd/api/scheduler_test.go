package main

import (
	"errors"
	"testing"
	"time"
)

func TestKeepaliveIntervalFromEnvDefault(t *testing.T) {
	t.Setenv("WA_KEEPALIVE_MINUTES", "")

	got := keepaliveIntervalFromEnv()
	want := 5 * time.Minute

	if got != want {
		t.Fatalf("keepaliveIntervalFromEnv() = %v, want %v", got, want)
	}
}

func TestKeepaliveIntervalFromEnvUsesPositiveMinutes(t *testing.T) {
	t.Setenv("WA_KEEPALIVE_MINUTES", "7")

	got := keepaliveIntervalFromEnv()
	want := 7 * time.Minute

	if got != want {
		t.Fatalf("keepaliveIntervalFromEnv() = %v, want %v", got, want)
	}
}

func TestKeepaliveIntervalFromEnvFallsBackForInvalidValues(t *testing.T) {
	for _, value := range []string{"0", "-1", "abc"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("WA_KEEPALIVE_MINUTES", value)

			got := keepaliveIntervalFromEnv()
			want := 5 * time.Minute

			if got != want {
				t.Fatalf("keepaliveIntervalFromEnv() = %v, want %v", got, want)
			}
		})
	}
}

func TestNewSchedulerStoresKeepaliveInterval(t *testing.T) {
	interval := 3 * time.Minute

	scheduler := NewScheduler(nil, nil, interval)

	if scheduler.keepaliveInterval != interval {
		t.Fatalf("keepaliveInterval = %v, want %v", scheduler.keepaliveInterval, interval)
	}
	if scheduler.keepaliveStop == nil {
		t.Fatal("keepaliveStop channel is nil")
	}
	if scheduler.healthStop == nil {
		t.Fatal("healthStop channel is nil")
	}
}

func TestSchedulerStopIsIdempotent(t *testing.T) {
	scheduler := NewScheduler(nil, nil, time.Minute)

	scheduler.Stop()
	scheduler.Stop()
}

func TestReminderDeliveryErrorRequiresRetryWhenAnyTargetFails(t *testing.T) {
	rootErr := errors.New("whatsapp disconnected")

	err := reminderDeliveryError(1, rootErr)

	if err == nil {
		t.Fatal("reminderDeliveryError() = nil, want retry error")
	}
	if !errors.Is(err, rootErr) {
		t.Fatalf("reminderDeliveryError() does not wrap root error: %v", err)
	}
}

func TestReminderDeliveryErrorAllowsCompletionWhenNoTargetFails(t *testing.T) {
	if err := reminderDeliveryError(0, errors.New("ignored")); err != nil {
		t.Fatalf("reminderDeliveryError() = %v, want nil", err)
	}
}
