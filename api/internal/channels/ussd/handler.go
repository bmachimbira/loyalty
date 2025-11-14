package ussd

import (
	"context"
	"log/slog"
	"strings"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler handles USSD callback requests
type Handler struct {
	pool           *pgxpool.Pool
	queries        *db.Queries
	sessionManager *SessionManager
	menuSystem     *MenuSystem
}

// NewHandler creates a new USSD handler
func NewHandler(pool *pgxpool.Pool) *Handler {
	queries := db.New(pool)
	return &Handler{
		pool:           pool,
		queries:        queries,
		sessionManager: NewSessionManager(queries),
		menuSystem:     NewMenuSystem(queries),
	}
}

// HandleCallback handles the USSD callback request
func (h *Handler) HandleCallback(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse USSD request (supports both form and JSON)
	var req USSDRequest
	if err := c.ShouldBind(&req); err != nil {
		slog.Error("Failed to parse USSD request", "error", err)
		c.String(200, "END Error processing request")
		return
	}

	slog.Info("USSD request received",
		"session_id", req.SessionID,
		"phone", req.PhoneNumber,
		"text", req.Text,
		"service_code", req.ServiceCode,
	)

	// Determine tenant from service code or default
	tenantID := h.getTenantIDFromServiceCode(req.ServiceCode)

	// Set tenant context for RLS
	if _, err := h.pool.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID); err != nil {
		slog.Error("Failed to set tenant context", "error", err)
		c.String(200, "END System error. Please try again.")
		return
	}

	// Get or create session
	session, sessionData, err := h.sessionManager.GetOrCreateSession(ctx, req.SessionID, req.PhoneNumber, tenantID)
	if err != nil {
		slog.Error("Failed to get session", "error", err)
		c.String(200, "END System error. Please try again.")
		return
	}

	// Try to link customer if not already linked
	if !session.CustomerID.Valid {
		h.tryLinkCustomer(ctx, session, req.PhoneNumber, sessionData)
	}

	// Process the request
	response := h.processRequest(ctx, session, sessionData, req)

	// Update session state if continuing
	if response.Type == Continue {
		if err := h.sessionManager.UpdateSession(ctx, req.SessionID, sessionData); err != nil {
			slog.Error("Failed to update session", "error", err)
		}
	} else {
		// Session ended, delete it
		if err := h.sessionManager.DeleteSession(ctx, req.SessionID); err != nil {
			slog.Warn("Failed to delete session", "error", err)
		}
	}

	// Return response
	responseText := response.String()
	slog.Info("USSD response",
		"session_id", req.SessionID,
		"type", response.Type,
		"length", len(responseText),
	)

	c.String(200, responseText)
}

// processRequest processes the USSD request and returns a response
func (h *Handler) processRequest(ctx context.Context, session *db.UssdSession, data *SessionData, req USSDRequest) USSDResponse {
	// Parse input
	input := strings.TrimSpace(req.Text)

	// Check if this is the first message (empty text)
	if input == "" {
		// Show main menu
		data.CurrentMenu = "main"
		menu := h.menuSystem.GetMenu("main")
		return menu.Render(data)
	}

	// Get the last input (after the last *)
	parts := strings.Split(input, "*")
	lastInput := parts[len(parts)-1]

	// Special handling for back option (0)
	if lastInput == "0" {
		if len(data.MenuStack) > 0 {
			// Go back to previous menu
			previousMenu := data.PopMenu()
			menu := h.menuSystem.GetMenu(previousMenu)
			return menu.Render(data)
		} else {
			// Already at main menu
			menu := h.menuSystem.GetMenu("main")
			return menu.Render(data)
		}
	}

	// Get current menu
	currentMenu := h.menuSystem.GetMenu(data.CurrentMenu)

	// Handle input with current menu
	nextMenuName, response := currentMenu.Handle(lastInput, data)

	// If response is empty, we need to render the next menu
	if response.Message == "" {
		// Save current menu to stack if changing menus
		if nextMenuName != data.CurrentMenu && nextMenuName != "main" {
			data.PushMenu(nextMenuName)
		}

		data.CurrentMenu = nextMenuName
		nextMenu := h.menuSystem.GetMenu(nextMenuName)
		response = nextMenu.Render(data)
	}

	// Special handling for menus that need database context
	if response.Message == "" {
		response = h.handleContextualMenu(ctx, session, data, lastInput)
	}

	return response
}

// handleContextualMenu handles menus that need database access
func (h *Handler) handleContextualMenu(ctx context.Context, session *db.UssdSession, data *SessionData, input string) USSDResponse {
	menuCtx := NewMenuWithContext(ctx, h.queries, session)

	switch data.CurrentMenu {
	case "myrewards":
		return menuCtx.RenderMyRewards()

	case "redeem_confirm":
		choice, ok := ParseMenuChoice(input, 2)
		if !ok {
			return FormatContinue("Invalid option.\n\n1. Confirm\n2. Cancel")
		}

		if choice == 2 {
			data.CurrentMenu = "main"
			return FormatEnd("Redemption cancelled.")
		}

		// Get the code from session data
		code, ok := data.GetDataString("redeem_code")
		if !ok {
			data.CurrentMenu = "main"
			return FormatEnd("Error: Code not found.")
		}

		// Process redemption
		response := menuCtx.RenderRedeemWithCode(code)
		data.CurrentMenu = "main"
		return response

	default:
		// Return to main menu if unknown
		data.CurrentMenu = "main"
		menu := h.menuSystem.GetMenu("main")
		return menu.Render(data)
	}
}

// tryLinkCustomer attempts to link a customer to the session
func (h *Handler) tryLinkCustomer(ctx context.Context, session *db.UssdSession, phoneNumber string, data *SessionData) {
	// Normalize phone number to E.164 format
	phoneE164 := h.normalizePhoneNumber(phoneNumber)

	// Try to find customer
	menuCtx := NewMenuWithContext(ctx, h.queries, session)
	customerID, err := menuCtx.GetCustomerByPhone(phoneE164)
	if err != nil {
		// Customer not found, that's okay
		slog.Info("Customer not found for USSD session",
			"phone", phoneE164,
			"session_id", session.SessionID,
		)
		return
	}

	// Link customer to session
	if err := h.sessionManager.LinkCustomer(ctx, session.SessionID, customerID); err != nil {
		slog.Error("Failed to link customer to USSD session", "error", err)
		return
	}

	// Update session data
	data.CustomerID = customerID.String()

	slog.Info("Customer linked to USSD session",
		"customer_id", customerID,
		"session_id", session.SessionID,
	)
}

// normalizePhoneNumber normalizes a phone number to E.164 format
func (h *Handler) normalizePhoneNumber(phone string) string {
	// Remove whitespace and special characters
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")

	// Add country code if missing (Zimbabwe = +263)
	if !strings.HasPrefix(phone, "+") {
		if strings.HasPrefix(phone, "0") {
			phone = "+263" + phone[1:] // Remove leading 0 and add +263
		} else if strings.HasPrefix(phone, "263") {
			phone = "+" + phone
		} else {
			phone = "+263" + phone
		}
	}

	return phone
}

// getTenantIDFromServiceCode determines tenant from USSD service code
func (h *Handler) getTenantIDFromServiceCode(serviceCode string) uuid.UUID {
	// For Phase 3, return a default tenant ID
	// TODO: Implement proper tenant routing based on service code
	// In production, this would:
	// 1. Look up tenant by USSD short code
	// 2. Query a routing table
	// 3. Use a configuration mapping

	// For now, use a default tenant ID
	tenantID, _ := uuid.Parse("00000000-0000-0000-0000-000000000000")
	return tenantID
}
