package webhooks

import (
	"time"

	"github.com/google/uuid"
)

// Event types
const (
	EventCustomerEnrolled = "customer.enrolled"
	EventRewardIssued     = "reward.issued"
	EventRewardRedeemed   = "reward.redeemed"
	EventRewardExpired    = "reward.expired"
	EventBudgetThreshold  = "budget.threshold"
)

// EventPayload is the base structure for all webhook events
type EventPayload struct {
	Event     string      `json:"event"`
	Timestamp int64       `json:"timestamp"`
	TenantID  string      `json:"tenant_id"`
	Data      interface{} `json:"data"`
}

// CustomerEnrolledData contains data for customer.enrolled event
type CustomerEnrolledData struct {
	CustomerID  string `json:"customer_id"`
	PhoneE164   string `json:"phone_e164,omitempty"`
	ExternalRef string `json:"external_ref,omitempty"`
	Status      string `json:"status"`
	EnrolledAt  string `json:"enrolled_at"`
}

// RewardIssuedData contains data for reward.issued event
type RewardIssuedData struct {
	IssuanceID   string  `json:"issuance_id"`
	CustomerID   string  `json:"customer_id"`
	RewardID     string  `json:"reward_id"`
	RewardName   string  `json:"reward_name"`
	RewardType   string  `json:"reward_type"`
	Status       string  `json:"status"`
	Code         string  `json:"code,omitempty"`
	ExternalRef  string  `json:"external_ref,omitempty"`
	FaceAmount   float64 `json:"face_amount,omitempty"`
	Currency     string  `json:"currency,omitempty"`
	IssuedAt     string  `json:"issued_at"`
	ExpiresAt    string  `json:"expires_at,omitempty"`
	CampaignID   string  `json:"campaign_id,omitempty"`
	CampaignName string  `json:"campaign_name,omitempty"`
}

// RewardRedeemedData contains data for reward.redeemed event
type RewardRedeemedData struct {
	IssuanceID  string  `json:"issuance_id"`
	CustomerID  string  `json:"customer_id"`
	RewardID    string  `json:"reward_id"`
	RewardName  string  `json:"reward_name"`
	RewardType  string  `json:"reward_type"`
	Code        string  `json:"code,omitempty"`
	FaceAmount  float64 `json:"face_amount,omitempty"`
	Currency    string  `json:"currency,omitempty"`
	RedeemedAt  string  `json:"redeemed_at"`
	RedeemedBy  string  `json:"redeemed_by,omitempty"`
	LocationID  string  `json:"location_id,omitempty"`
}

// RewardExpiredData contains data for reward.expired event
type RewardExpiredData struct {
	IssuanceID string  `json:"issuance_id"`
	CustomerID string  `json:"customer_id"`
	RewardID   string  `json:"reward_id"`
	RewardName string  `json:"reward_name"`
	RewardType string  `json:"reward_type"`
	FaceAmount float64 `json:"face_amount,omitempty"`
	Currency   string  `json:"currency,omitempty"`
	ExpiredAt  string  `json:"expired_at"`
	IssuedAt   string  `json:"issued_at"`
}

// BudgetThresholdData contains data for budget.threshold event
type BudgetThresholdData struct {
	BudgetID   string  `json:"budget_id"`
	BudgetName string  `json:"budget_name"`
	Threshold  string  `json:"threshold"` // "soft_cap" or "hard_cap"
	Balance    float64 `json:"balance"`
	SoftCap    float64 `json:"soft_cap"`
	HardCap    float64 `json:"hard_cap"`
	Currency   string  `json:"currency"`
	Utilization float64 `json:"utilization"` // Percentage
}

// NewEventPayload creates a new event payload
func NewEventPayload(eventType string, tenantID uuid.UUID, data interface{}) EventPayload {
	return EventPayload{
		Event:     eventType,
		Timestamp: time.Now().Unix(),
		TenantID:  tenantID.String(),
		Data:      data,
	}
}

// NewCustomerEnrolledEvent creates a customer.enrolled event
func NewCustomerEnrolledEvent(tenantID uuid.UUID, data CustomerEnrolledData) EventPayload {
	return NewEventPayload(EventCustomerEnrolled, tenantID, data)
}

// NewRewardIssuedEvent creates a reward.issued event
func NewRewardIssuedEvent(tenantID uuid.UUID, data RewardIssuedData) EventPayload {
	return NewEventPayload(EventRewardIssued, tenantID, data)
}

// NewRewardRedeemedEvent creates a reward.redeemed event
func NewRewardRedeemedEvent(tenantID uuid.UUID, data RewardRedeemedData) EventPayload {
	return NewEventPayload(EventRewardRedeemed, tenantID, data)
}

// NewRewardExpiredEvent creates a reward.expired event
func NewRewardExpiredEvent(tenantID uuid.UUID, data RewardExpiredData) EventPayload {
	return NewEventPayload(EventRewardExpired, tenantID, data)
}

// NewBudgetThresholdEvent creates a budget.threshold event
func NewBudgetThresholdEvent(tenantID uuid.UUID, data BudgetThresholdData) EventPayload {
	return NewEventPayload(EventBudgetThreshold, tenantID, data)
}
