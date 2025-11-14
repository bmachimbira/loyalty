package reward

import (
	"testing"
)

func TestStateTransitions(t *testing.T) {
	tests := []struct {
		name      string
		from      State
		to        State
		shouldErr bool
	}{
		// Valid transitions from reserved
		{"reserved to issued", StateReserved, StateIssued, false},
		{"reserved to cancelled", StateReserved, StateCancelled, false},
		{"reserved to failed", StateReserved, StateFailed, false},

		// Valid transitions from issued
		{"issued to redeemed", StateIssued, StateRedeemed, false},
		{"issued to expired", StateIssued, StateExpired, false},
		{"issued to cancelled", StateIssued, StateCancelled, false},

		// Invalid transitions from reserved
		{"reserved to redeemed", StateReserved, StateRedeemed, true},
		{"reserved to expired", StateReserved, StateExpired, true},

		// Invalid transitions from issued
		{"issued to reserved", StateIssued, StateReserved, true},
		{"issued to failed", StateIssued, StateFailed, true},

		// Terminal states (no transitions)
		{"redeemed to issued", StateRedeemed, StateIssued, true},
		{"expired to issued", StateExpired, StateIssued, true},
		{"cancelled to issued", StateCancelled, StateIssued, true},
		{"failed to issued", StateFailed, StateIssued, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canTransition := tt.from.CanTransitionTo(tt.to)
			if tt.shouldErr && canTransition {
				t.Errorf("Expected transition %s -> %s to be invalid, but it was valid", tt.from, tt.to)
			}
			if !tt.shouldErr && !canTransition {
				t.Errorf("Expected transition %s -> %s to be valid, but it was invalid", tt.from, tt.to)
			}

			// Also test ValidateTransition
			err := tt.from.ValidateTransition(tt.to)
			if tt.shouldErr && err == nil {
				t.Errorf("Expected ValidateTransition to return error for %s -> %s", tt.from, tt.to)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected ValidateTransition to succeed for %s -> %s, got error: %v", tt.from, tt.to, err)
			}
		})
	}
}

func TestStateIsValid(t *testing.T) {
	tests := []struct {
		state State
		valid bool
	}{
		{StateReserved, true},
		{StateIssued, true},
		{StateRedeemed, true},
		{StateExpired, true},
		{StateCancelled, true},
		{StateFailed, true},
		{State("invalid"), false},
		{State(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if tt.state.IsValid() != tt.valid {
				t.Errorf("Expected IsValid() for %s to be %v", tt.state, tt.valid)
			}
		})
	}
}

func TestStateIsTerminal(t *testing.T) {
	tests := []struct {
		state    State
		terminal bool
	}{
		{StateReserved, false},
		{StateIssued, false},
		{StateRedeemed, true},
		{StateExpired, true},
		{StateCancelled, true},
		{StateFailed, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if tt.state.IsTerminal() != tt.terminal {
				t.Errorf("Expected IsTerminal() for %s to be %v", tt.state, tt.terminal)
			}
		})
	}
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateReserved, "reserved"},
		{StateIssued, "issued"},
		{StateRedeemed, "redeemed"},
		{StateExpired, "expired"},
		{StateCancelled, "cancelled"},
		{StateFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.state.String() != tt.expected {
				t.Errorf("Expected String() to return %s, got %s", tt.expected, tt.state.String())
			}
		})
	}
}
