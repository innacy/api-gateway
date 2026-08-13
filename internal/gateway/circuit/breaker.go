package circuit

import (
	"sync"
	"time"
)

type State int

const (
	Closed   State = 0
	Open     State = 1
	HalfOpen State = 2
)

func (s State) String() string {
	switch s {
	case Closed:
		return "closed"
	case Open:
		return "open"
	case HalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

type Breaker struct {
	mu sync.Mutex

	name      string
	threshold int
	timeout   time.Duration

	state        State
	failures     int
	lastFailure  time.Time
	halfOpenUsed bool
}

func NewBreaker(name string, threshold int, timeout time.Duration) *Breaker {
	return &Breaker{
		name:      name,
		threshold: threshold,
		timeout:   timeout,
		state:     Closed,
	}
}

func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.currentState()
}

func (b *Breaker) currentState() State {
	if b.state == Open && time.Since(b.lastFailure) > b.timeout {
		b.state = HalfOpen
		b.halfOpenUsed = false
	}
	return b.state
}

func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.currentState() {
	case Closed:
		return true
	case Open:
		return false
	case HalfOpen:
		if !b.halfOpenUsed {
			b.halfOpenUsed = true
			return true
		}
		return false
	default:
		return true
	}
}

func (b *Breaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case HalfOpen:
		b.state = Closed
		b.failures = 0
		b.halfOpenUsed = false
	case Closed:
		b.failures = 0
	}
}

func (b *Breaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.currentState() {
	case HalfOpen:
		b.state = Open
		b.lastFailure = time.Now()
		b.halfOpenUsed = false
	case Closed:
		b.failures++
		b.lastFailure = time.Now()
		if b.failures >= b.threshold {
			b.state = Open
		}
	}
}

type Registry struct {
	mu        sync.RWMutex
	breakers  map[string]*Breaker
	threshold int
	timeout   time.Duration
}

func NewBreakerRegistry(threshold int, timeout time.Duration) *Registry {
	return &Registry{
		breakers:  make(map[string]*Breaker),
		threshold: threshold,
		timeout:   timeout,
	}
}

func (r *Registry) Get(service string) *Breaker {
	r.mu.RLock()
	if b, ok := r.breakers[service]; ok {
		r.mu.RUnlock()
		return b
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if b, ok := r.breakers[service]; ok {
		return b
	}

	b := NewBreaker(service, r.threshold, r.timeout)
	r.breakers[service] = b
	return b
}
