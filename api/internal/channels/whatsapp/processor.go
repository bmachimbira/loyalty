package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MessageProcessor processes incoming WhatsApp messages
type MessageProcessor struct {
	pool           *pgxpool.Pool
	queries        *db.Queries
	sender         *MessageSender
	sessionManager *SessionManager
}

// NewMessageProcessor creates a new message processor
func NewMessageProcessor(pool *pgxpool.Pool, queries *db.Queries, sender *MessageSender) *MessageProcessor {
	return &MessageProcessor{
		pool:           pool,
		queries:        queries,
		sender:         sender,
		sessionManager: NewSessionManager(queries),
	}
}

// ProcessMessage processes an incoming message
func (p *MessageProcessor) ProcessMessage(ctx context.Context, msg Message) error {
	// For now, use a default tenant ID (in production, this would be determined by phone number routing)
	// TODO: Implement tenant resolution based on phone number
	tenantID := p.getTenantIDFromPhoneNumber(msg.From)

	// Set tenant context for RLS
	if _, err := p.pool.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID); err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	// Get or create session
	session, err := p.sessionManager.GetOrCreateSession(ctx, msg.From, msg.From, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Parse message text
	text := p.getMessageText(msg)
	if text == "" {
		// Non-text messages not supported yet
		return p.sender.SendText(ctx, msg.From, "Sorry, I can only process text messages right now.")
	}

	// Trim whitespace
	text = strings.TrimSpace(text)

	// Mark message as read
	if err := p.sender.MarkAsRead(ctx, msg.ID); err != nil {
		slog.Warn("Failed to mark message as read", "error", err)
	}

	// Check if it's a command
	if strings.HasPrefix(text, "/") {
		return p.handleCommand(ctx, session, text)
	}

	// Get session state
	state, err := p.sessionManager.GetSessionState(session)
	if err != nil {
		return fmt.Errorf("failed to get session state: %w", err)
	}

	// Handle based on current flow
	if state.IsIdle() {
		// No active flow, show help
		return p.sender.SendText(ctx, msg.From, "I didn't understand that. "+InvalidCommandMessage)
	}

	// Handle active flows
	switch state.CurrentFlow {
	case "enrollment":
		return p.handleEnrollmentFlow(ctx, session, state, text)
	case "redemption":
		return p.handleRedemptionFlow(ctx, session, state, text)
	default:
		// Unknown flow, reset
		p.sessionManager.ResetSessionState(ctx, session.WaID)
		return p.sender.SendText(ctx, msg.From, InvalidCommandMessage)
	}
}

// handleCommand routes commands to appropriate handlers
func (p *MessageProcessor) handleCommand(ctx context.Context, session *db.WaSession, text string) error {
	// Parse command and args
	parts := strings.Fields(text)
	command := strings.ToLower(parts[0])
	args := parts[1:]

	slog.Info("Processing WhatsApp command",
		"command", command,
		"from", session.WaID,
	)

	switch command {
	case "/start", "/enroll":
		return p.handleEnroll(ctx, session)
	case "/balance":
		return p.handleBalance(ctx, session)
	case "/rewards":
		return p.handleRewards(ctx, session)
	case "/myrewards":
		return p.handleMyRewards(ctx, session)
	case "/redeem":
		return p.handleRedeem(ctx, session, args)
	case "/refer":
		return p.handleReferral(ctx, session)
	case "/help":
		return p.handleHelp(ctx, session)
	default:
		return p.sender.SendText(ctx, session.WaID, InvalidCommandMessage)
	}
}

// handleEnroll handles customer enrollment
func (p *MessageProcessor) handleEnroll(ctx context.Context, session *db.WaSession) error {
	// Check if customer already exists
	customer, err := p.queries.GetCustomerByPhone(ctx, db.GetCustomerByPhoneParams{
		TenantID: session.TenantID,
		PhoneE164: pgtype.Text{
			String: session.PhoneE164,
			Valid:  true,
		},
	})

	if err == nil {
		// Customer already exists
		if session.CustomerID.Valid && session.CustomerID.Bytes == customer.ID.Bytes {
			return p.sender.SendText(ctx, session.WaID, "You're already enrolled in our loyalty program! Send /help to see what you can do.")
		}

		// Link existing customer to session
		if err := p.sessionManager.LinkCustomer(ctx, session.WaID, uuid.UUID(customer.ID.Bytes)); err != nil {
			slog.Error("Failed to link customer to session", "error", err)
		}

		return p.sender.SendText(ctx, session.WaID, "Welcome back! You're now connected via WhatsApp. Send /help to see available commands.")
	}

	if err != pgx.ErrNoRows {
		return fmt.Errorf("failed to check customer: %w", err)
	}

	// Create new customer
	customer, err = p.queries.CreateCustomer(ctx, db.CreateCustomerParams{
		TenantID: session.TenantID,
		PhoneE164: pgtype.Text{
			String: session.PhoneE164,
			Valid:  true,
		},
		ExternalRef: pgtype.Text{Valid: false},
	})
	if err != nil {
		return fmt.Errorf("failed to create customer: %w", err)
	}

	// Link customer to session
	if err := p.sessionManager.LinkCustomer(ctx, session.WaID, uuid.UUID(customer.ID.Bytes)); err != nil {
		slog.Error("Failed to link customer to session", "error", err)
	}

	// Record consent
	_, err = p.queries.RecordConsent(ctx, db.RecordConsentParams{
		TenantID:   session.TenantID,
		CustomerID: customer.ID,
		Channel:    "whatsapp",
		Purpose:    "loyalty",
		Granted:    true,
		OccurredAt: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
	})
	if err != nil {
		slog.Error("Failed to record consent", "error", err)
	}

	// Send welcome message
	return p.sender.SendText(ctx, session.WaID, WelcomeMessage)
}

// handleBalance shows the customer's points balance
func (p *MessageProcessor) handleBalance(ctx context.Context, session *db.WaSession) error {
	if !session.CustomerID.Valid {
		return p.sender.SendText(ctx, session.WaID, "Please enroll first using /enroll")
	}

	// For Phase 3, we don't have a points system yet
	// This would be implemented in the future
	return p.sender.SendText(ctx, session.WaID, "Points balance feature coming soon!\n\nFor now, you can:\n• Check your rewards with /myrewards\n• View available rewards with /rewards")
}

// handleRewards lists available rewards
func (p *MessageProcessor) handleRewards(ctx context.Context, session *db.WaSession) error {
	if !session.CustomerID.Valid {
		return p.sender.SendText(ctx, session.WaID, "Please enroll first using /enroll")
	}

	// Get active rewards from catalog
	rewards, err := p.queries.ListRewards(ctx, db.ListRewardsParams{
		TenantID: session.TenantID,
		Limit:    10,
		Offset:   0,
	})
	if err != nil {
		return fmt.Errorf("failed to list rewards: %w", err)
	}

	if len(rewards) == 0 {
		return p.sender.SendText(ctx, session.WaID, "No rewards available right now. Check back soon!")
	}

	// Format rewards list
	var msg strings.Builder
	msg.WriteString("*Available Rewards:*\n\n")

	for i, reward := range rewards {
		msg.WriteString(fmt.Sprintf("%d. *%s*\n", i+1, reward.Name))
		msg.WriteString(fmt.Sprintf("   Type: %s\n", reward.Type))
		if reward.Currency.Valid && reward.FaceValue.Valid {
			msg.WriteString(fmt.Sprintf("   Value: %s %.2f\n", reward.Currency.String, reward.FaceValue.Float64))
		}
		msg.WriteString("\n")
	}

	msg.WriteString("Earn these rewards by shopping with us!")

	return p.sender.SendText(ctx, session.WaID, msg.String())
}

// handleMyRewards shows customer's active rewards
func (p *MessageProcessor) handleMyRewards(ctx context.Context, session *db.WaSession) error {
	if !session.CustomerID.Valid {
		return p.sender.SendText(ctx, session.WaID, "Please enroll first using /enroll")
	}

	// Get customer's active issuances
	issuances, err := p.queries.ListActiveIssuances(ctx, db.ListActiveIssuancesParams{
		TenantID:   session.TenantID,
		CustomerID: session.CustomerID,
	})
	if err != nil {
		return fmt.Errorf("failed to list issuances: %w", err)
	}

	if len(issuances) == 0 {
		return p.sender.SendText(ctx, session.WaID, NoRewardsMessage)
	}

	// Format rewards list
	var msg strings.Builder
	msg.WriteString("*Your Active Rewards:*\n\n")

	for i, issuance := range issuances {
		// Get reward details
		reward, err := p.queries.GetRewardByID(ctx, db.GetRewardByIDParams{
			ID:       issuance.RewardID,
			TenantID: session.TenantID,
		})
		if err != nil {
			slog.Error("Failed to get reward details", "error", err, "reward_id", issuance.RewardID)
			continue
		}

		msg.WriteString(fmt.Sprintf("%d. *%s*\n", i+1, reward.Name))
		msg.WriteString(fmt.Sprintf("   Status: %s\n", issuance.Status))

		if issuance.Code.Valid {
			msg.WriteString(fmt.Sprintf("   Code: %s\n", issuance.Code.String))
		}

		if issuance.ExpiresAt.Valid {
			expiry := issuance.ExpiresAt.Time
			daysLeft := int(time.Until(expiry).Hours() / 24)
			if daysLeft > 0 {
				msg.WriteString(fmt.Sprintf("   Expires in: %d days\n", daysLeft))
			} else {
				msg.WriteString("   Expires: Soon\n")
			}
		}

		msg.WriteString("\n")
	}

	msg.WriteString("Use /redeem [code] to redeem a reward.")

	return p.sender.SendText(ctx, session.WaID, msg.String())
}

// handleRedeem handles reward redemption
func (p *MessageProcessor) handleRedeem(ctx context.Context, session *db.WaSession, args []string) error {
	if !session.CustomerID.Valid {
		return p.sender.SendText(ctx, session.WaID, "Please enroll first using /enroll")
	}

	if len(args) == 0 {
		return p.sender.SendText(ctx, session.WaID, "Please provide a redemption code.\n\nUsage: /redeem [code]\nExample: /redeem ABC123")
	}

	code := strings.ToUpper(args[0])

	// Find issuance by code
	issuances, err := p.queries.ListActiveIssuances(ctx, db.ListActiveIssuancesParams{
		TenantID:   session.TenantID,
		CustomerID: session.CustomerID,
	})
	if err != nil {
		return fmt.Errorf("failed to list issuances: %w", err)
	}

	var targetIssuance *db.Issuance
	for _, iss := range issuances {
		if iss.Code.Valid && strings.ToUpper(iss.Code.String) == code {
			targetIssuance = &iss
			break
		}
	}

	if targetIssuance == nil {
		return p.sender.SendText(ctx, session.WaID, "Invalid or expired redemption code. Use /myrewards to see your active rewards.")
	}

	// Check if already redeemed
	if targetIssuance.Status == "redeemed" {
		return p.sender.SendText(ctx, session.WaID, "This reward has already been redeemed.")
	}

	// Check expiry
	if targetIssuance.ExpiresAt.Valid && time.Now().After(targetIssuance.ExpiresAt.Time) {
		return p.sender.SendText(ctx, session.WaID, "This reward has expired.")
	}

	// Update status to redeemed
	err = p.queries.UpdateIssuanceStatus(ctx, db.UpdateIssuanceStatusParams{
		ID:       targetIssuance.ID,
		TenantID: session.TenantID,
		Status:   "reserved", // Old status
		Status_2: "redeemed", // New status
	})
	if err != nil {
		return fmt.Errorf("failed to redeem reward: %w", err)
	}

	// Get reward details
	reward, err := p.queries.GetRewardByID(ctx, db.GetRewardByIDParams{
		ID:       targetIssuance.RewardID,
		TenantID: session.TenantID,
	})
	if err != nil {
		slog.Error("Failed to get reward details", "error", err)
	}

	rewardName := "Your reward"
	if err == nil {
		rewardName = reward.Name
	}

	return p.sender.SendText(ctx, session.WaID, fmt.Sprintf("✅ Success!\n\n*%s* has been redeemed.\n\nThank you for being a loyal customer!", rewardName))
}

// handleReferral provides referral information
func (p *MessageProcessor) handleReferral(ctx context.Context, session *db.WaSession) error {
	if !session.CustomerID.Valid {
		return p.sender.SendText(ctx, session.WaID, "Please enroll first using /enroll")
	}

	// For Phase 3, referral system is not implemented yet
	return p.sender.SendText(ctx, session.WaID, "Referral program coming soon!\n\nShare our loyalty program with friends and earn bonus rewards.")
}

// handleHelp shows help message
func (p *MessageProcessor) handleHelp(ctx context.Context, session *db.WaSession) error {
	return p.sender.SendText(ctx, session.WaID, HelpMessage)
}

// handleEnrollmentFlow handles multi-step enrollment flow
func (p *MessageProcessor) handleEnrollmentFlow(ctx context.Context, session *db.WaSession, state *SessionState, text string) error {
	// For Phase 3, we use simple enrollment
	// This could be extended for multi-step flows in the future
	return p.sender.SendText(ctx, session.WaID, "Please use /enroll to start enrollment.")
}

// handleRedemptionFlow handles multi-step redemption flow
func (p *MessageProcessor) handleRedemptionFlow(ctx context.Context, session *db.WaSession, state *SessionState, text string) error {
	// For Phase 3, we use command-based redemption
	// This could be extended for multi-step flows in the future
	return p.sender.SendText(ctx, session.WaID, "Please use /redeem [code] to redeem a reward.")
}

// getMessageText extracts text from a message
func (p *MessageProcessor) getMessageText(msg Message) string {
	if msg.Text != nil {
		return msg.Text.Body
	}
	if msg.Button != nil {
		return msg.Button.Text
	}
	return ""
}

// getTenantIDFromPhoneNumber resolves tenant from phone number
// In production, this would look up routing configuration
func (p *MessageProcessor) getTenantIDFromPhoneNumber(phoneNumber string) uuid.UUID {
	// For Phase 3, return a default tenant ID
	// TODO: Implement proper tenant routing
	// This could be done by:
	// 1. Looking up tenant by WhatsApp Business Phone Number
	// 2. Checking a phone number prefix routing table
	// 3. Using a dedicated tenant endpoint

	// For now, we'll use a hardcoded tenant ID that matches the seed data
	// In production, this MUST be implemented properly
	tenantID, _ := uuid.Parse("00000000-0000-0000-0000-000000000000")
	return tenantID
}
