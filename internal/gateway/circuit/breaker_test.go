package circuit_test

import (
	"testing"
	"time"

	"github.com/widasinnacy/api-gateway/internal/gateway/circuit"
)

func TestBreaker_StartsClosedAndAllows(t *testing.T) {
	b := circuit.NewBreaker("test", 3, 100*time.Millisecond)

	if s := b.State(); s != circuit.Closed {
		t.Errorf("initial state = %v, want Closed", s)
	}
	if !b.Allow() {
		t.Error("should allow requests when closed")
	}
}

func TestBreaker_OpensAfterThreshold(t *testing.T) {
	b := circuit.NewBreaker("test", 3, 100*time.Millisecond)

	for i := 0; i < 3; i++ {
		b.RecordFailure()
	}

	if s := b.State(); s != circuit.Open {
		t.Errorf("state = %v, want Open after %d failures", s, 3)
	}
	if b.Allow() {
		t.Error("should not allow requests when open")
	}
}

func TestBreaker_TransitionsToHalfOpen(t *testing.T) {
	b := circuit.NewBreaker("test", 3, 50*time.Millisecond)

	for i := 0; i < 3; i++ {
		b.RecordFailure()
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	if s := b.State(); s != circuit.HalfOpen {
		t.Errorf("state = %v, want HalfOpen after timeout", s)
	}
	if !b.Allow() {
		t.Error("should allow one probe request in half-open")
	}
	// Second request should be blocked in half-open
	if b.Allow() {
		t.Error("should block second request in half-open")
	}
}

func TestBreaker_ClosesOnSuccessInHalfOpen(t *testing.T) {
	b := circuit.NewBreaker("test", 3, 50*time.Millisecond)

	for i := 0; i < 3; i++ {
		b.RecordFailure()
	}
	time.Sleep(60 * time.Millisecond)

	b.Allow() // probe request
	b.RecordSuccess()

	if s := b.State(); s != circuit.Closed {
		t.Errorf("state = %v, want Closed after success in half-open", s)
	}
	if !b.Allow() {
		t.Error("should allow requests after closing")
	}
}

func TestBreaker_ReopensOnFailureInHalfOpen(t *testing.T) {
	b := circuit.NewBreaker("test", 3, 50*time.Millisecond)

	for i := 0; i < 3; i++ {
		b.RecordFailure()
	}
	time.Sleep(60 * time.Millisecond)

	b.Allow() // probe
	b.RecordFailure()

	if s := b.State(); s != circuit.Open {
		t.Errorf("state = %v, want Open after failure in half-open", s)
	}
}

func TestBreaker_SuccessResetsFailureCount(t *testing.T) {
	b := circuit.NewBreaker("test", 3, 100*time.Millisecond)

	b.RecordFailure()
	b.RecordFailure()
	b.RecordSuccess() // reset

	b.RecordFailure()
	b.RecordFailure()

	if s := b.State(); s != circuit.Closed {
		t.Errorf("state = %v, want Closed (success should reset count)", s)
	}
}

func TestRegistry_ReturnsSameBreakerForService(t *testing.T) {
	reg := circuit.NewBreakerRegistry(5, 30*time.Second)

	b1 := reg.Get("url-extractor")
	b2 := reg.Get("url-extractor")
	b3 := reg.Get("data-parser")

	if b1 != b2 {
		t.Error("should return same breaker for same service")
	}
	if b1 == b3 {
		t.Error("should return different breaker for different service")
	}
}
