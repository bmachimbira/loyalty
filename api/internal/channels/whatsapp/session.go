package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// SessionState represents the state of a WhatsApp conversation
type SessionState struct {
	CurrentFlow string                 `json:"current_flow"`
	StepIndex   int                    `json:"step_index"`
	Data        map[string]interface{} `json:"data"`
}

// SessionManager handles WhatsApp session state
type SessionManager struct {
	queries *db.Queries
}

// NewSessionManager creates a new session manager
func NewSessionManager(queries *db.Queries) *SessionManager {
	return &SessionManager{
		queries: queries,
	}
}

// GetOrCreateSession gets an existing session or creates a new one
func (sm *SessionManager) GetOrCreateSession(ctx context.Context, waID, phoneE164 string, tenantID uuid.UUID) (*db.WaSession, error) {
	// Try to get existing session
	session, err := sm.queries.GetWASessionByWAID(ctx, waID)
	if err == nil {
		return &session, nil
	}

	if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Create new session
	initialState := SessionState{
		CurrentFlow: "idle",
		StepIndex:   0,
		Data:        make(map[string]interface{}),
	}

	stateJSON, err := json.Marshal(initialState)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal initial state: %w", err)
	}

	session, err = sm.queries.UpsertWASession(ctx, db.UpsertWASessionParams{
		TenantID: pgtype.UUID{
			Bytes: tenantID,
			Valid: true,
		},
		CustomerID: pgtype.UUID{Valid: false}, // Will be set during enrollment
		WaID:       waID,
		PhoneE164:  phoneE164,
		State:      stateJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &session, nil
}

// GetSessionState retrieves and parses the session state
func (sm *SessionManager) GetSessionState(session *db.WaSession) (*SessionState, error) {
	var state SessionState
	if err := json.Unmarshal(session.State, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session state: %w", err)
	}

	if state.Data == nil {
		state.Data = make(map[string]interface{})
	}

	return &state, nil
}

// UpdateSessionState updates the session state
func (sm *SessionManager) UpdateSessionState(ctx context.Context, waID string, state *SessionState) error {
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	err = sm.queries.UpdateWASessionState(ctx, db.UpdateWASessionStateParams{
		WaID:  waID,
		State: stateJSON,
	})
	if err != nil {
		return fmt.Errorf("failed to update session state: %w", err)
	}

	return nil
}

// LinkCustomer links a customer to a session
func (sm *SessionManager) LinkCustomer(ctx context.Context, waID string, customerID uuid.UUID) error {
	err := sm.queries.UpdateWASessionCustomer(ctx, db.UpdateWASessionCustomerParams{
		WaID: waID,
		CustomerID: pgtype.UUID{
			Bytes: customerID,
			Valid: true,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to link customer: %w", err)
	}

	return nil
}

// GetSessionByCustomer retrieves the most recent session for a customer
func (sm *SessionManager) GetSessionByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (*db.WaSession, error) {
	session, err := sm.queries.GetWASessionByCustomer(ctx, db.GetWASessionByCustomerParams{
		TenantID: pgtype.UUID{
			Bytes: tenantID,
			Valid: true,
		},
		CustomerID: pgtype.UUID{
			Bytes: customerID,
			Valid: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get session by customer: %w", err)
	}

	return &session, nil
}

// ResetSessionState resets the session to idle state
func (sm *SessionManager) ResetSessionState(ctx context.Context, waID string) error {
	state := &SessionState{
		CurrentFlow: "idle",
		StepIndex:   0,
		Data:        make(map[string]interface{}),
	}

	return sm.UpdateSessionState(ctx, waID, state)
}

// SetFlowData sets a value in the session flow data
func (state *SessionState) SetFlowData(key string, value interface{}) {
	if state.Data == nil {
		state.Data = make(map[string]interface{})
	}
	state.Data[key] = value
}

// GetFlowData gets a value from the session flow data
func (state *SessionState) GetFlowData(key string) (interface{}, bool) {
	if state.Data == nil {
		return nil, false
	}
	value, ok := state.Data[key]
	return value, ok
}

// GetFlowDataString gets a string value from the session flow data
func (state *SessionState) GetFlowDataString(key string) (string, bool) {
	value, ok := state.GetFlowData(key)
	if !ok {
		return "", false
	}
	str, ok := value.(string)
	return str, ok
}

// IsInFlow checks if the session is in a specific flow
func (state *SessionState) IsInFlow(flowName string) bool {
	return state.CurrentFlow == flowName
}

// IsIdle checks if the session is idle
func (state *SessionState) IsIdle() bool {
	return state.CurrentFlow == "idle" || state.CurrentFlow == ""
}

// StartFlow starts a new flow
func (state *SessionState) StartFlow(flowName string) {
	state.CurrentFlow = flowName
	state.StepIndex = 0
	state.Data = make(map[string]interface{})
}

// NextStep moves to the next step in the flow
func (state *SessionState) NextStep() {
	state.StepIndex++
}

// EndFlow ends the current flow and resets to idle
func (state *SessionState) EndFlow() {
	state.CurrentFlow = "idle"
	state.StepIndex = 0
	state.Data = make(map[string]interface{})
}
