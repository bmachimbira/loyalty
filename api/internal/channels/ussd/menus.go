package ussd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// MenuSystem manages all USSD menus
type MenuSystem struct {
	queries *db.Queries
	menus   map[string]Menu
}

// NewMenuSystem creates a new menu system
func NewMenuSystem(queries *db.Queries) *MenuSystem {
	ms := &MenuSystem{
		queries: queries,
		menus:   make(map[string]Menu),
	}

	// Register menus
	ms.menus["main"] = &MainMenu{queries: queries}
	ms.menus["balance"] = &BalanceMenu{queries: queries}
	ms.menus["rewards"] = &RewardsMenu{queries: queries}
	ms.menus["myrewards"] = &MyRewardsMenu{queries: queries}
	ms.menus["redeem"] = &RedeemMenu{queries: queries}
	ms.menus["help"] = &HelpMenu{queries: queries}

	return ms
}

// GetMenu retrieves a menu by name
func (ms *MenuSystem) GetMenu(name string) Menu {
	menu, ok := ms.menus[name]
	if !ok {
		return ms.menus["main"] // Default to main menu
	}
	return menu
}

// MainMenu is the entry point menu
type MainMenu struct {
	queries *db.Queries
}

func (m *MainMenu) Render(session *SessionData) USSDResponse {
	return FormatMenu("Welcome to Loyalty", []MenuOption{
		{Key: "1", Label: "My Rewards"},
		{Key: "2", Label: "Check Balance"},
		{Key: "3", Label: "Redeem Reward"},
		{Key: "4", Label: "Help"},
	})
}

func (m *MainMenu) Handle(input string, session *SessionData) (string, USSDResponse) {
	choice, ok := ParseMenuChoice(input, 4)
	if !ok {
		return "main", FormatContinue("Invalid option. Please try again.\n\n" + m.Render(session).Message)
	}

	switch choice {
	case 1:
		return "myrewards", USSDResponse{}
	case 2:
		return "balance", USSDResponse{}
	case 3:
		return "redeem", USSDResponse{}
	case 4:
		return "help", USSDResponse{}
	default:
		return "main", FormatContinue("Invalid option. Please try again.\n\n" + m.Render(session).Message)
	}
}

// BalanceMenu shows the customer's points balance
type BalanceMenu struct {
	queries *db.Queries
}

func (m *BalanceMenu) Render(session *SessionData) USSDResponse {
	// For Phase 3, we don't have a points system yet
	rb := NewResponseBuilder()
	rb.AddLine("Your Balance")
	rb.AddBlankLine()
	rb.AddLine("Points: Coming soon!")
	rb.AddBlankLine()
	rb.AddLine("You can check your rewards")
	rb.AddLine("with option 1 from main menu.")

	return rb.End()
}

func (m *BalanceMenu) Handle(input string, session *SessionData) (string, USSDResponse) {
	return "main", m.Render(session)
}

// RewardsMenu lists available rewards from catalog
type RewardsMenu struct {
	queries *db.Queries
}

func (m *RewardsMenu) Render(session *SessionData) USSDResponse {
	// This would need context to query database
	// For now, return a message
	return FormatEnd("Available rewards:\n\nVisit our website or WhatsApp\nfor the full catalog.")
}

func (m *RewardsMenu) Handle(input string, session *SessionData) (string, USSDResponse) {
	return "main", m.Render(session)
}

// MyRewardsMenu shows customer's active rewards
type MyRewardsMenu struct {
	queries *db.Queries
}

func (m *MyRewardsMenu) Render(session *SessionData) USSDResponse {
	// Check if customer is linked
	if session.CustomerID == "" {
		return FormatEnd("Please register first.\n\nContact customer support\nor use WhatsApp to enroll.")
	}

	return FormatEnd("Your rewards will be\nshown here.\n\nUse WhatsApp for\ndetailed reward info.")
}

func (m *MyRewardsMenu) Handle(input string, session *SessionData) (string, USSDResponse) {
	return "main", m.Render(session)
}

// RedeemMenu handles reward redemption
type RedeemMenu struct {
	queries *db.Queries
}

func (m *RedeemMenu) Render(session *SessionData) USSDResponse {
	return FormatContinue("Enter your reward code:")
}

func (m *RedeemMenu) Handle(input string, session *SessionData) (string, USSDResponse) {
	// Check if customer is linked
	if session.CustomerID == "" {
		return "main", FormatEnd("Please register first.\n\nContact customer support.")
	}

	// Validate code format
	code := strings.TrimSpace(strings.ToUpper(input))
	if code == "" || code == "0" {
		return "main", FormatEnd("Redemption cancelled.")
	}

	if len(code) < 3 {
		return "redeem", FormatContinue("Invalid code format.\nPlease enter a valid code:")
	}

	// Store the code for processing
	session.SetData("redeem_code", code)

	return "redeem_confirm", FormatContinue(fmt.Sprintf("Redeem code: %s\n\n1. Confirm\n2. Cancel", code))
}

// RedeemConfirmMenu confirms redemption
type RedeemConfirmMenu struct {
	queries *db.Queries
}

func (m *RedeemConfirmMenu) Render(session *SessionData) USSDResponse {
	code, _ := session.GetDataString("redeem_code")
	return FormatConfirmation(fmt.Sprintf("Redeem code: %s", code))
}

func (m *RedeemConfirmMenu) Handle(input string, session *SessionData) (string, USSDResponse) {
	choice, ok := ParseMenuChoice(input, 2)
	if !ok {
		return "redeem_confirm", FormatContinue("Invalid option.\n\n" + m.Render(session).Message)
	}

	if choice == 2 {
		return "main", FormatEnd("Redemption cancelled.")
	}

	// Get the code
	code, ok := session.GetDataString("redeem_code")
	if !ok {
		return "main", FormatEnd("Error: Code not found.\nPlease try again.")
	}

	// In production, this would process the redemption
	// For Phase 3, we return a success message
	return "main", FormatEnd(fmt.Sprintf("Success!\n\nCode %s redeemed.\n\nThank you!", code))
}

// HelpMenu shows help information
type HelpMenu struct {
	queries *db.Queries
}

func (m *HelpMenu) Render(session *SessionData) USSDResponse {
	rb := NewResponseBuilder()
	rb.AddLine("Loyalty Program Help")
	rb.AddBlankLine()
	rb.AddLine("1. My Rewards - View your")
	rb.AddLine("   active rewards")
	rb.AddBlankLine()
	rb.AddLine("2. Check Balance - View")
	rb.AddLine("   your points")
	rb.AddBlankLine()
	rb.AddLine("3. Redeem - Use a reward")
	rb.AddLine("   code")
	rb.AddBlankLine()
	rb.AddLine("For more info, contact")
	rb.AddLine("customer support.")

	return rb.End()
}

func (m *HelpMenu) Handle(input string, session *SessionData) (string, USSDResponse) {
	return "main", m.Render(session)
}

// MenuWithContext is a menu that needs database access
type MenuWithContext struct {
	ctx     context.Context
	queries *db.Queries
	session *db.UssdSession
}

// NewMenuWithContext creates a menu with context
func NewMenuWithContext(ctx context.Context, queries *db.Queries, session *db.UssdSession) *MenuWithContext {
	return &MenuWithContext{
		ctx:     ctx,
		queries: queries,
		session: session,
	}
}

// RenderMyRewards renders the my rewards menu with database data
func (m *MenuWithContext) RenderMyRewards() USSDResponse {
	if !m.session.CustomerID.Valid {
		return FormatEnd("Please register first.\n\nContact customer support\nor use WhatsApp to enroll.")
	}

	// Get customer's active issuances
	issuances, err := m.queries.ListActiveIssuances(m.ctx, db.ListActiveIssuancesParams{
		TenantID:   m.session.TenantID,
		CustomerID: m.session.CustomerID,
	})
	if err != nil {
		return FormatError("Failed to load rewards")
	}

	if len(issuances) == 0 {
		return FormatEnd("You have no active rewards.\n\nKeep shopping to earn rewards!")
	}

	rb := NewResponseBuilder()
	rb.AddLine("Your Active Rewards")
	rb.AddBlankLine()

	// Show up to 3 rewards on USSD (character limit)
	count := len(issuances)
	if count > 3 {
		count = 3
	}

	for i := 0; i < count; i++ {
		iss := issuances[i]

		// Get reward details
		reward, err := m.queries.GetRewardByID(m.ctx, db.GetRewardByIDParams{
			ID:       iss.RewardID,
			TenantID: m.session.TenantID,
		})
		if err != nil {
			continue
		}

		rb.AddLine(fmt.Sprintf("%d. %s", i+1, reward.Name))
		if iss.Code.Valid {
			rb.AddLine(fmt.Sprintf("   Code: %s", iss.Code.String))
		}

		if iss.ExpiresAt.Valid {
			daysLeft := int(time.Until(iss.ExpiresAt.Time).Hours() / 24)
			if daysLeft > 0 {
				rb.AddLine(fmt.Sprintf("   Exp: %dd", daysLeft))
			}
		}
		rb.AddBlankLine()
	}

	if len(issuances) > 3 {
		rb.AddLine(fmt.Sprintf("+ %d more rewards", len(issuances)-3))
		rb.AddBlankLine()
	}

	rb.AddLine("Use WhatsApp for full")
	rb.AddLine("details and redemption.")

	return rb.End()
}

// RenderRedeemWithCode handles redemption with database access
func (m *MenuWithContext) RenderRedeemWithCode(code string) USSDResponse {
	if !m.session.CustomerID.Valid {
		return FormatEnd("Please register first.\n\nContact customer support.")
	}

	// Find issuance by code
	issuances, err := m.queries.ListActiveIssuances(m.ctx, db.ListActiveIssuancesParams{
		TenantID:   m.session.TenantID,
		CustomerID: m.session.CustomerID,
	})
	if err != nil {
		return FormatError("Failed to process redemption")
	}

	var targetIssuance *db.Issuance
	for _, iss := range issuances {
		if iss.Code.Valid && strings.EqualFold(iss.Code.String, code) {
			targetIssuance = &iss
			break
		}
	}

	if targetIssuance == nil {
		return FormatEnd("Invalid or expired code.\n\nPlease check and try again.")
	}

	// Check if already redeemed
	if targetIssuance.Status == "redeemed" {
		return FormatEnd("This reward has already\nbeen redeemed.")
	}

	// Check expiry
	if targetIssuance.ExpiresAt.Valid && time.Now().After(targetIssuance.ExpiresAt.Time) {
		return FormatEnd("This reward has expired.")
	}

	// Update status to redeemed
	err = m.queries.UpdateIssuanceStatus(m.ctx, db.UpdateIssuanceStatusParams{
		ID:       targetIssuance.ID,
		TenantID: m.session.TenantID,
		Status:   "issued", // Old status
		Status_2: "redeemed",   // New status
	})
	if err != nil {
		return FormatError("Redemption failed")
	}

	// Get reward details
	reward, err := m.queries.GetRewardByID(m.ctx, db.GetRewardByIDParams{
		ID:       targetIssuance.RewardID,
		TenantID: m.session.TenantID,
	})

	rewardName := "Your reward"
	if err == nil {
		rewardName = reward.Name
	}

	return FormatEnd(fmt.Sprintf("Success!\n\n%s redeemed.\n\nThank you for your loyalty!", rewardName))
}

// GetCustomerByPhone finds a customer by phone number
func (m *MenuWithContext) GetCustomerByPhone(phoneE164 string) (uuid.UUID, error) {
	customer, err := m.queries.GetCustomerByPhone(m.ctx, db.GetCustomerByPhoneParams{
		TenantID: m.session.TenantID,
		PhoneE164: pgtype.Text{
			String: phoneE164,
			Valid:  true,
		},
	})
	if err != nil {
		return uuid.UUID{}, err
	}

	return uuid.UUID(customer.ID.Bytes), nil
}
