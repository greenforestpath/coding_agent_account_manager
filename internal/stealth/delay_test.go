package stealth

import (
	"context"
	"math/rand"
	"testing"
	"time"
)

func TestComputeDelay(t *testing.T) {
	t.Run("valid range", func(t *testing.T) {
		rng := rand.New(rand.NewSource(1))
		got, err := ComputeDelay(5, 30, rng)
		if err != nil {
			t.Fatalf("ComputeDelay() error = %v", err)
		}
		if got < 5*time.Second || got > 30*time.Second {
			t.Fatalf("ComputeDelay() = %v, want within [5s, 30s]", got)
		}
	})

	t.Run("equal bounds", func(t *testing.T) {
		got, err := ComputeDelay(7, 7, rand.New(rand.NewSource(1)))
		if err != nil {
			t.Fatalf("ComputeDelay() error = %v", err)
		}
		if got != 7*time.Second {
			t.Fatalf("ComputeDelay() = %v, want %v", got, 7*time.Second)
		}
	})

	t.Run("invalid bounds", func(t *testing.T) {
		if _, err := ComputeDelay(10, 5, rand.New(rand.NewSource(1))); err == nil {
			t.Fatal("ComputeDelay() should error when min > max")
		}
	})
}

func TestWait(t *testing.T) {
	t.Run("skips when Skip channel fires", func(t *testing.T) {
		skip := make(chan struct{})
		close(skip)

		skipped, err := Wait(context.Background(), 10*time.Second, WaitOptions{
			Skip: skip,
		})
		if err != nil {
			t.Fatalf("Wait() error = %v", err)
		}
		if !skipped {
			t.Fatal("Wait() skipped = false, want true")
		}
	})

	t.Run("returns ctx error when canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		skipped, err := Wait(ctx, 10*time.Second, WaitOptions{})
		if err == nil {
			t.Fatal("Wait() error = nil, want context cancellation")
		}
		if skipped {
			t.Fatal("Wait() skipped = true, want false")
		}
	})

	t.Run("no-op on zero duration", func(t *testing.T) {
		skipped, err := Wait(context.Background(), 0, WaitOptions{})
		if err != nil {
			t.Fatalf("Wait() error = %v", err)
		}
		if skipped {
			t.Fatal("Wait() skipped = true, want false")
		}
	})
}
