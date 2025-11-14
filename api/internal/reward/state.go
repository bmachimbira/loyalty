package reward

import "fmt"

// State represents the lifecycle state of a reward issuance
type State string

const (
	StateReserved  State = "reserved"  // Budget reserved, awaiting processing
	StateIssued    State = "issued"    // Successfully issued to customer
	StateRedeemed  State = "redeemed"  // Customer has redeemed the reward
	StateExpired   State = "expired"   // Reward expired before redemption
	StateCancelled State = "cancelled" // Manually cancelled
	StateFailed    State = "failed"    // Processing failed
)

// Valid state transitions
// reserved -> issued, cancelled, failed
// issued -> redeemed, expired, cancelled
var transitions = map[State][]State{
	StateReserved: {StateIssued, StateCancelled, StateFailed},
	StateIssued:   {StateRedeemed, StateExpired, StateCancelled},
	// Terminal states have no valid transitions
	StateRedeemed:  {},
	StateExpired:   {},
	StateCancelled: {},
	StateFailed:    {},
}

// CanTransitionTo checks if a state transition is valid
func (s State) CanTransitionTo(target State) bool {
	validTargets, ok := transitions[s]
	if !ok {
		return false
	}
	for _, t := range validTargets {
		if t == target {
			return true
		}
	}
	return false
}

// IsValid checks if the state is a valid state
func (s State) IsValid() bool {
	switch s {
	case StateReserved, StateIssued, StateRedeemed, StateExpired, StateCancelled, StateFailed:
		return true
	default:
		return false
	}
}

// IsTerminal returns true if this is a terminal state (no further transitions possible)
func (s State) IsTerminal() bool {
	return s == StateRedeemed || s == StateExpired || s == StateCancelled || s == StateFailed
}

// String returns the string representation of the state
func (s State) String() string {
	return string(s)
}

// ValidateTransition returns an error if the transition is invalid
func (s State) ValidateTransition(target State) error {
	if !s.IsValid() {
		return fmt.Errorf("invalid current state: %s", s)
	}
	if !target.IsValid() {
		return fmt.Errorf("invalid target state: %s", target)
	}
	if !s.CanTransitionTo(target) {
		return fmt.Errorf("invalid state transition: %s -> %s", s, target)
	}
	return nil
}
