package testutil

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"github.com/bmachimbira/loyalty/api/internal/db"
)

// UUIDFromString converts a string UUID to pgtype.UUID
func UUIDFromString(t *testing.T, s string) pgtype.UUID {
	t.Helper()

	var pgUUID pgtype.UUID
	err := pgUUID.Scan(s)
	require.NoError(t, err, "Failed to create UUID from string %s", s)
	return pgUUID
}

// NewUUID generates a new random pgtype.UUID
func NewUUID(t *testing.T) pgtype.UUID {
	t.Helper()

	return UUIDFromString(t, uuid.New().String())
}

// NumericFromFloat converts a float to pgtype.Numeric
func NumericFromFloat(t *testing.T, value float64) pgtype.Numeric {
	t.Helper()

	var num pgtype.Numeric
	err := num.Scan(value)
	require.NoError(t, err, "Failed to create Numeric from float %f", value)
	return num
}

// TextFromString converts a string to pgtype.Text
func TextFromString(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

// TimestamptzNow returns current time as pgtype.Timestamptz
func TimestamptzNow() pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: time.Now(), Valid: true}
}

// TimestamptzFromTime converts time.Time to pgtype.Timestamptz
func TimestamptzFromTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// CreateTestTenant creates a test tenant with reasonable defaults
func CreateTestTenant(t *testing.T, queries *db.Queries, opts ...TenantOption) db.Tenant {
	t.Helper()

	params := db.CreateTenantParams{
		Name:       "Test Tenant",
		DefaultCcy: "USD",
	}

	// Apply options
	for _, opt := range opts {
		opt(&params)
	}

	ctx := context.Background()
	tenant, err := queries.CreateTenant(ctx, params)
	require.NoError(t, err, "Failed to create test tenant")

	return tenant
}

type TenantOption func(*db.CreateTenantParams)

func WithTenantName(name string) TenantOption {
	return func(p *db.CreateTenantParams) {
		p.Name = name
	}
}

func WithDefaultCurrency(ccy string) TenantOption {
	return func(p *db.CreateTenantParams) {
		p.DefaultCcy = ccy
	}
}

// CreateTestStaffUser creates a test staff user
func CreateTestStaffUser(t *testing.T, queries *db.Queries, tenantID pgtype.UUID, opts ...StaffUserOption) db.StaffUser {
	t.Helper()

	params := db.CreateStaffUserParams{
		TenantID:     tenantID,
		Email:        "test@example.com",
		PasswordHash: "$2a$10$abcdefghijklmnopqrstuv", // bcrypt hash
		Role:         "admin",
	}

	// Apply options
	for _, opt := range opts {
		opt(&params)
	}

	ctx := context.Background()
	user, err := queries.CreateStaffUser(ctx, params)
	require.NoError(t, err, "Failed to create test staff user")

	return user
}

type StaffUserOption func(*db.CreateStaffUserParams)

func WithEmail(email string) StaffUserOption {
	return func(p *db.CreateStaffUserParams) {
		p.Email = email
	}
}

func WithRole(role string) StaffUserOption {
	return func(p *db.CreateStaffUserParams) {
		p.Role = role
	}
}

// CreateTestCustomer creates a test customer
func CreateTestCustomer(t *testing.T, queries *db.Queries, tenantID pgtype.UUID, opts ...CustomerOption) db.Customer {
	t.Helper()

	params := db.CreateCustomerParams{
		TenantID:    tenantID,
		PhoneE164:   TextFromString("+263771234567"),
		ExternalRef: TextFromString("CUST001"),
		Status:      "active",
	}

	// Apply options
	for _, opt := range opts {
		opt(&params)
	}

	ctx := context.Background()
	customer, err := queries.CreateCustomer(ctx, params)
	require.NoError(t, err, "Failed to create test customer")

	return customer
}

type CustomerOption func(*db.CreateCustomerParams)

func WithPhone(phone string) CustomerOption {
	return func(p *db.CreateCustomerParams) {
		p.PhoneE164 = TextFromString(phone)
	}
}

func WithExternalRef(ref string) CustomerOption {
	return func(p *db.CreateCustomerParams) {
		p.ExternalRef = TextFromString(ref)
	}
}

func WithCustomerStatus(status string) CustomerOption {
	return func(p *db.CreateCustomerParams) {
		p.Status = status
	}
}

// CreateTestBudget creates a test budget
func CreateTestBudget(t *testing.T, queries *db.Queries, tenantID pgtype.UUID, opts ...BudgetOption) db.Budget {
	t.Helper()

	params := db.CreateBudgetParams{
		TenantID: tenantID,
		Name:     "Test Budget",
		Currency: "USD",
		SoftCap:  NumericFromFloat(t, 1000.0),
		HardCap:  NumericFromFloat(t, 1500.0),
		Balance:  NumericFromFloat(t, 1000.0),
		Period:   "monthly",
	}

	// Apply options
	for _, opt := range opts {
		opt(&params)
	}

	ctx := context.Background()
	budget, err := queries.CreateBudget(ctx, params)
	require.NoError(t, err, "Failed to create test budget")

	return budget
}

type BudgetOption func(*db.CreateBudgetParams)

func WithBudgetName(name string) BudgetOption {
	return func(p *db.CreateBudgetParams) {
		p.Name = name
	}
}

func WithBudgetBalance(balance float64) BudgetOption {
	return func(p *db.CreateBudgetParams) {
		p.Balance = NumericFromFloat(nil, balance)
	}
}

func WithBudgetCaps(soft, hard float64) BudgetOption {
	return func(p *db.CreateBudgetParams) {
		p.SoftCap = NumericFromFloat(nil, soft)
		p.HardCap = NumericFromFloat(nil, hard)
	}
}

func WithBudgetCurrency(currency string) BudgetOption {
	return func(p *db.CreateBudgetParams) {
		p.Currency = currency
	}
}

// CreateTestReward creates a test reward
func CreateTestReward(t *testing.T, queries *db.Queries, tenantID pgtype.UUID, opts ...RewardOption) db.Reward {
	t.Helper()

	params := db.CreateRewardParams{
		TenantID:    tenantID,
		Name:        "Test Reward",
		Type:        "discount",
		CostAmount:  NumericFromFloat(t, 10.0),
		FaceAmount:  NumericFromFloat(t, 10.0),
		Currency:    TextFromString("USD"),
		Status:      "active",
		Description: TextFromString("Test reward description"),
	}

	// Apply options
	for _, opt := range opts {
		opt(&params)
	}

	ctx := context.Background()
	reward, err := queries.CreateReward(ctx, params)
	require.NoError(t, err, "Failed to create test reward")

	return reward
}

type RewardOption func(*db.CreateRewardParams)

func WithRewardName(name string) RewardOption {
	return func(p *db.CreateRewardParams) {
		p.Name = name
	}
}

func WithRewardType(rewardType string) RewardOption {
	return func(p *db.CreateRewardParams) {
		p.Type = rewardType
	}
}

func WithRewardCost(cost float64) RewardOption {
	return func(p *db.CreateRewardParams) {
		p.CostAmount = NumericFromFloat(nil, cost)
	}
}

func WithRewardFace(face float64) RewardOption {
	return func(p *db.CreateRewardParams) {
		p.FaceAmount = NumericFromFloat(nil, face)
	}
}

// CreateTestRule creates a test rule
func CreateTestRule(t *testing.T, queries *db.Queries, tenantID, rewardID pgtype.UUID, opts ...RuleOption) db.Rule {
	t.Helper()

	condition := map[string]interface{}{
		">=": []interface{}{
			map[string]interface{}{"var": "amount"},
			20,
		},
	}
	conditionJSON, err := json.Marshal(condition)
	require.NoError(t, err)

	params := db.CreateRuleParams{
		TenantID:    tenantID,
		RewardID:    rewardID,
		Name:        "Test Rule",
		Description: TextFromString("Test rule description"),
		EventType:   "purchase",
		Conditions:  conditionJSON,
		Status:      "active",
	}

	// Apply options
	for _, opt := range opts {
		opt(&params)
	}

	ctx := context.Background()
	rule, err := queries.CreateRule(ctx, params)
	require.NoError(t, err, "Failed to create test rule")

	return rule
}

type RuleOption func(*db.CreateRuleParams)

func WithRuleName(name string) RuleOption {
	return func(p *db.CreateRuleParams) {
		p.Name = name
	}
}

func WithEventType(eventType string) RuleOption {
	return func(p *db.CreateRuleParams) {
		p.EventType = eventType
	}
}

func WithConditions(conditions map[string]interface{}) RuleOption {
	return func(p *db.CreateRuleParams) {
		conditionJSON, _ := json.Marshal(conditions)
		p.Conditions = conditionJSON
	}
}

func WithRuleStatus(status string) RuleOption {
	return func(p *db.CreateRuleParams) {
		p.Status = status
	}
}

func WithPerUserCap(cap int32) RuleOption {
	return func(p *db.CreateRuleParams) {
		p.CapPerUser = pgtype.Int4{Int32: cap, Valid: true}
	}
}

func WithGlobalCap(cap int32) RuleOption {
	return func(p *db.CreateRuleParams) {
		p.CapGlobal = pgtype.Int4{Int32: cap, Valid: true}
	}
}

func WithCooldownHours(hours int32) RuleOption {
	return func(p *db.CreateRuleParams) {
		p.CooldownHours = pgtype.Int4{Int32: hours, Valid: true}
	}
}

// CreateTestCampaign creates a test campaign
func CreateTestCampaign(t *testing.T, queries *db.Queries, tenantID, budgetID pgtype.UUID, opts ...CampaignOption) db.Campaign {
	t.Helper()

	params := db.CreateCampaignParams{
		TenantID: tenantID,
		Name:     "Test Campaign",
		StartAt:  TimestamptzFromTime(time.Now()),
		EndAt:    TimestamptzFromTime(time.Now().Add(30 * 24 * time.Hour)),
		BudgetID: budgetID,
		Status:   "active",
	}

	// Apply options
	for _, opt := range opts {
		opt(&params)
	}

	ctx := context.Background()
	campaign, err := queries.CreateCampaign(ctx, params)
	require.NoError(t, err, "Failed to create test campaign")

	return campaign
}

type CampaignOption func(*db.CreateCampaignParams)

func WithCampaignName(name string) CampaignOption {
	return func(p *db.CreateCampaignParams) {
		p.Name = name
	}
}

func WithCampaignDates(start, end time.Time) CampaignOption {
	return func(p *db.CreateCampaignParams) {
		p.StartAt = TimestamptzFromTime(start)
		p.EndAt = TimestamptzFromTime(end)
	}
}

// CreateTestEvent creates a test event
func CreateTestEvent(t *testing.T, queries *db.Queries, tenantID, customerID pgtype.UUID, opts ...EventOption) db.Event {
	t.Helper()

	properties := map[string]interface{}{
		"amount":   25.0,
		"currency": "USD",
	}
	propertiesJSON, err := json.Marshal(properties)
	require.NoError(t, err)

	params := db.CreateEventParams{
		TenantID:       tenantID,
		CustomerID:     customerID,
		EventType:      "purchase",
		Properties:     propertiesJSON,
		Source:         "api",
		IdempotencyKey: uuid.New().String(),
	}

	// Apply options
	for _, opt := range opts {
		opt(&params)
	}

	ctx := context.Background()
	event, err := queries.CreateEvent(ctx, params)
	require.NoError(t, err, "Failed to create test event")

	return event
}

type EventOption func(*db.CreateEventParams)

func WithEventType(eventType string) EventOption {
	return func(p *db.CreateEventParams) {
		p.EventType = eventType
	}
}

func WithProperties(properties map[string]interface{}) EventOption {
	return func(p *db.CreateEventParams) {
		propertiesJSON, _ := json.Marshal(properties)
		p.Properties = propertiesJSON
	}
}

func WithIdempotencyKey(key string) EventOption {
	return func(p *db.CreateEventParams) {
		p.IdempotencyKey = key
	}
}

// CreateTestIssuance creates a test issuance
func CreateTestIssuance(t *testing.T, queries *db.Queries, tenantID, customerID, campaignID, rewardID, eventID pgtype.UUID, opts ...IssuanceOption) db.Issuance {
	t.Helper()

	params := db.CreateIssuanceParams{
		TenantID:   tenantID,
		CustomerID: customerID,
		CampaignID: campaignID,
		RewardID:   rewardID,
		EventID:    eventID,
		Status:     "reserved",
		Currency:   TextFromString("USD"),
		CostAmount: NumericFromFloat(t, 10.0),
		FaceAmount: NumericFromFloat(t, 10.0),
		ExpiresAt:  TimestamptzFromTime(time.Now().Add(30 * 24 * time.Hour)),
	}

	// Apply options
	for _, opt := range opts {
		opt(&params)
	}

	ctx := context.Background()
	issuance, err := queries.CreateIssuance(ctx, params)
	require.NoError(t, err, "Failed to create test issuance")

	return issuance
}

type IssuanceOption func(*db.CreateIssuanceParams)

func WithIssuanceStatus(status string) IssuanceOption {
	return func(p *db.CreateIssuanceParams) {
		p.Status = status
	}
}

func WithIssuanceCode(code string) IssuanceOption {
	return func(p *db.CreateIssuanceParams) {
		p.Code = TextFromString(code)
	}
}

func WithExpiresAt(expiresAt time.Time) IssuanceOption {
	return func(p *db.CreateIssuanceParams) {
		p.ExpiresAt = TimestamptzFromTime(expiresAt)
	}
}
