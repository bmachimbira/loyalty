package ussd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// SessionManager manages USSD sessions
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
func (sm *SessionManager) GetOrCreateSession(ctx context.Context, sessionID, phoneNumber string, tenantID uuid.UUID) (*db.UssdSession, *SessionData, error) {
	// Try to get existing session
	session, err := sm.queries.GetUSSDSessionByID(ctx, sessionID)
	if err == nil {
		// Parse session data
		data, err := sm.parseSessionData(session.State)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse session data: %w", err)
		}
		return &session, data, nil
	}

	if err != pgx.ErrNoRows {
		return nil, nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Create new session
	data := NewSessionData()
	data.TenantID = tenantID.String()

	stateJSON, err := json.Marshal(data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal session data: %w", err)
	}

	session, err = sm.queries.CreateUSSDSession(ctx, db.CreateUSSDSessionParams{
		TenantID: pgtype.UUID{
			Bytes: tenantID,
			Valid: true,
		},
		CustomerID: pgtype.UUID{Valid: false}, // Will be set when customer is identified
		SessionID:  sessionID,
		PhoneE164:  phoneNumber,
		State:      stateJSON,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &session, data, nil
}

// UpdateSession updates the session state
func (sm *SessionManager) UpdateSession(ctx context.Context, sessionID string, data *SessionData) error {
	stateJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	err = sm.queries.UpdateUSSDSessionState(ctx, db.UpdateUSSDSessionStateParams{
		SessionID: sessionID,
		State:     stateJSON,
	})
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// LinkCustomer links a customer to a session
func (sm *SessionManager) LinkCustomer(ctx context.Context, sessionID string, customerID uuid.UUID) error {
	err := sm.queries.UpdateUSSDSessionCustomer(ctx, db.UpdateUSSDSessionCustomerParams{
		SessionID: sessionID,
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

// DeleteSession deletes a session
func (sm *SessionManager) DeleteSession(ctx context.Context, sessionID string) error {
	err := sm.queries.DeleteUSSDSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// GetCustomerSession retrieves the most recent session for a customer
func (sm *SessionManager) GetCustomerSession(ctx context.Context, tenantID, customerID uuid.UUID) (*db.UssdSession, error) {
	session, err := sm.queries.GetUSSDSessionByCustomer(ctx, db.GetUSSDSessionByCustomerParams{
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
		return nil, fmt.Errorf("failed to get customer session: %w", err)
	}

	return &session, nil
}

// parseSessionData parses session state JSON
func (sm *SessionManager) parseSessionData(stateJSON []byte) (*SessionData, error) {
	var data SessionData
	if err := json.Unmarshal(stateJSON, &data); err != nil {
		return nil, err
	}

	// Ensure data map is initialized
	if data.Data == nil {
		data.Data = make(map[string]interface{})
	}

	// Ensure menu stack is initialized
	if data.MenuStack == nil {
		data.MenuStack = make([]string, 0)
	}

	// Set default current menu if empty
	if data.CurrentMenu == "" {
		data.CurrentMenu = "main"
	}

	return &data, nil
}

// GetCustomerIDFromSession retrieves customer ID from session
func (sm *SessionManager) GetCustomerIDFromSession(session *db.UssdSession) (uuid.UUID, bool) {
	if !session.CustomerID.Valid {
		return uuid.UUID{}, false
	}
	return uuid.UUID(session.CustomerID.Bytes), true
}

// GetTenantIDFromSession retrieves tenant ID from session
func (sm *SessionManager) GetTenantIDFromSession(session *db.UssdSession) uuid.UUID {
	return uuid.UUID(session.TenantID.Bytes)
}
