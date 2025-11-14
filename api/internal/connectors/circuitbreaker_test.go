package connectors_test

import (
	"errors"
	"testing"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/connectors"
	"github.com/stretchr/testify/assert"
)

func TestCircuitBreaker_InitialState(t *testing.T) {
	cb := connectors.NewCircuitBreaker(3, 1*time.Second)
	assert.Equal(t, connectors.StateClosed, cb.State())
	assert.Equal(t, 0, cb.FailureCount())
}

func TestCircuitBreaker_SuccessfulCall(t *testing.T) {
	cb := connectors.NewCircuitBreaker(3, 1*time.Second)

	err := cb.Execute(func() error {
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, connectors.StateClosed, cb.State())
	assert.Equal(t, 0, cb.FailureCount())
}

func TestCircuitBreaker_FailedCall(t *testing.T) {
	cb := connectors.NewCircuitBreaker(3, 1*time.Second)

	err := cb.Execute(func() error {
		return errors.New("test error")
	})

	assert.Error(t, err)
	assert.Equal(t, connectors.StateClosed, cb.State())
	assert.Equal(t, 1, cb.FailureCount())
}

func TestCircuitBreaker_OpensAfterThresholdFailures(t *testing.T) {
	cb := connectors.NewCircuitBreaker(3, 1*time.Second)

	// Fail 3 times
	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return errors.New("test error")
		})
	}

	assert.Equal(t, connectors.StateOpen, cb.State())
	assert.Equal(t, 3, cb.FailureCount())

	// Next call should fail immediately without executing
	callCount := 0
	err := cb.Execute(func() error {
		callCount++
		return nil
	})

	assert.Error(t, err)
	assert.Equal(t, connectors.ErrCircuitOpen, err)
	assert.Equal(t, 0, callCount, "Function should not have been called")
}

func TestCircuitBreaker_TransitionToHalfOpen(t *testing.T) {
	cb := connectors.NewCircuitBreaker(3, 100*time.Millisecond)

	// Open the circuit
	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return errors.New("test error")
		})
	}

	assert.Equal(t, connectors.StateOpen, cb.State())

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Multiple successful calls should transition to closed (requires 3 successes)
	callCount := 0
	for i := 0; i < 3; i++ {
		err := cb.Execute(func() error {
			callCount++
			return nil
		})
		assert.NoError(t, err)
	}

	assert.Equal(t, 3, callCount, "Function should have been called 3 times")
	assert.Equal(t, connectors.StateClosed, cb.State(), "Should transition to closed after 3 successful calls in half-open")
}

func TestCircuitBreaker_HalfOpenReturnsToOpen(t *testing.T) {
	cb := connectors.NewCircuitBreaker(3, 100*time.Millisecond)

	// Open the circuit
	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return errors.New("test error")
		})
	}

	assert.Equal(t, connectors.StateOpen, cb.State())

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Fail in half-open state
	err := cb.Execute(func() error {
		return errors.New("test error")
	})

	assert.Error(t, err)
	assert.Equal(t, connectors.StateOpen, cb.State(), "Should return to open after failure in half-open")
}

func TestCircuitBreaker_HalfOpenToClosedAfterSuccesses(t *testing.T) {
	cb := connectors.NewCircuitBreaker(3, 100*time.Millisecond)

	// Open the circuit
	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return errors.New("test error")
		})
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Multiple successful calls should close the circuit
	for i := 0; i < 3; i++ {
		err := cb.Execute(func() error {
			return nil
		})
		assert.NoError(t, err)
	}

	assert.Equal(t, connectors.StateClosed, cb.State())
	assert.Equal(t, 0, cb.FailureCount())
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := connectors.NewCircuitBreaker(3, 1*time.Second)

	// Open the circuit
	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return errors.New("test error")
		})
	}

	assert.Equal(t, connectors.StateOpen, cb.State())

	// Reset
	cb.Reset()

	assert.Equal(t, connectors.StateClosed, cb.State())
	assert.Equal(t, 0, cb.FailureCount())

	// Should work normally after reset
	err := cb.Execute(func() error {
		return nil
	})

	assert.NoError(t, err)
}

func TestCircuitBreaker_SuccessResetsFailureCount(t *testing.T) {
	cb := connectors.NewCircuitBreaker(3, 1*time.Second)

	// Fail twice
	cb.Execute(func() error {
		return errors.New("test error")
	})
	cb.Execute(func() error {
		return errors.New("test error")
	})

	assert.Equal(t, 2, cb.FailureCount())
	assert.Equal(t, connectors.StateClosed, cb.State())

	// Succeed
	cb.Execute(func() error {
		return nil
	})

	assert.Equal(t, 0, cb.FailureCount())
	assert.Equal(t, connectors.StateClosed, cb.State())
}

func TestCircuitBreaker_ConcurrentCalls(t *testing.T) {
	cb := connectors.NewCircuitBreaker(10, 1*time.Second)

	// Execute concurrent calls
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func(n int) {
			if n%3 == 0 {
				cb.Execute(func() error {
					return errors.New("test error")
				})
			} else {
				cb.Execute(func() error {
					return nil
				})
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Circuit should still be in a valid state
	state := cb.State()
	assert.Contains(t, []connectors.CircuitState{
		connectors.StateClosed,
		connectors.StateOpen,
		connectors.StateHalfOpen,
	}, state)
}
