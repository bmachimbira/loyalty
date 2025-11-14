package connectors

import (
	"errors"
	"sync"
	"time"
)

// CircuitState represents the state of the circuit breaker
type CircuitState string

const (
	StateClosed   CircuitState = "closed"
	StateOpen     CircuitState = "open"
	StateHalfOpen CircuitState = "half-open"
)

var (
	// ErrCircuitOpen is returned when the circuit breaker is open
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// CircuitBreaker implements the circuit breaker pattern to prevent
// cascading failures when external services are unavailable
type CircuitBreaker struct {
	mu            sync.Mutex
	failureCount  int
	successCount  int
	threshold     int           // Number of failures before opening
	timeout       time.Duration // Time before transitioning from open to half-open
	state         CircuitState
	lastFailTime  time.Time
	halfOpenMax   int // Max successful calls in half-open before closing
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold:   threshold,
		timeout:     timeout,
		state:       StateClosed,
		halfOpenMax: 3, // Allow 3 successful calls before closing
	}
}

// Execute wraps a function call with circuit breaker logic
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()

	// Check if circuit should transition from open to half-open
	if cb.state == StateOpen {
		if time.Since(cb.lastFailTime) > cb.timeout {
			cb.state = StateHalfOpen
			cb.failureCount = 0
			cb.successCount = 0
		} else {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
	}

	cb.mu.Unlock()

	// Execute the function
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

// recordFailure records a failed call and potentially opens the circuit
func (cb *CircuitBreaker) recordFailure() {
	cb.failureCount++
	cb.lastFailTime = time.Now()
	cb.successCount = 0

	if cb.failureCount >= cb.threshold {
		cb.state = StateOpen
	}

	// If in half-open state, one failure reopens the circuit
	if cb.state == StateHalfOpen {
		cb.state = StateOpen
	}
}

// recordSuccess records a successful call and potentially closes the circuit
func (cb *CircuitBreaker) recordSuccess() {
	cb.failureCount = 0

	if cb.state == StateHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.halfOpenMax {
			cb.state = StateClosed
			cb.successCount = 0
		}
	}
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// FailureCount returns the current failure count
func (cb *CircuitBreaker) FailureCount() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failureCount
}
