package webhooks_test

import (
	"encoding/json"
	"testing"

	"github.com/bmachimbira/loyalty/api/internal/webhooks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventPayload(t *testing.T) {
	tenantID := uuid.New()
	data := map[string]string{"test": "value"}

	payload := webhooks.NewEventPayload("test.event", tenantID, data)

	assert.Equal(t, "test.event", payload.Event)
	assert.Equal(t, tenantID.String(), payload.TenantID)
	assert.NotZero(t, payload.Timestamp)
	assert.Equal(t, data, payload.Data)
}

func TestNewCustomerEnrolledEvent(t *testing.T) {
	tenantID := uuid.New()
	data := webhooks.CustomerEnrolledData{
		CustomerID:  "customer-123",
		PhoneE164:   "+263771234567",
		ExternalRef: "ext-ref-123",
		Status:      "active",
		EnrolledAt:  "2025-11-14T10:00:00Z",
	}

	event := webhooks.NewCustomerEnrolledEvent(tenantID, data)

	assert.Equal(t, webhooks.EventCustomerEnrolled, event.Event)
	assert.Equal(t, tenantID.String(), event.TenantID)
	assert.Equal(t, data, event.Data)
}

func TestNewRewardIssuedEvent(t *testing.T) {
	tenantID := uuid.New()
	data := webhooks.RewardIssuedData{
		IssuanceID:   "issuance-123",
		CustomerID:   "customer-123",
		RewardID:     "reward-123",
		RewardName:   "5% Discount",
		RewardType:   "discount",
		Status:       "issued",
		FaceAmount:   5.00,
		Currency:     "USD",
		IssuedAt:     "2025-11-14T10:00:00Z",
		ExpiresAt:    "2025-11-21T10:00:00Z",
		CampaignID:   "campaign-123",
		CampaignName: "Black Friday Sale",
	}

	event := webhooks.NewRewardIssuedEvent(tenantID, data)

	assert.Equal(t, webhooks.EventRewardIssued, event.Event)
	assert.Equal(t, tenantID.String(), event.TenantID)
	assert.Equal(t, data, event.Data)
}

func TestNewRewardRedeemedEvent(t *testing.T) {
	tenantID := uuid.New()
	data := webhooks.RewardRedeemedData{
		IssuanceID: "issuance-123",
		CustomerID: "customer-123",
		RewardID:   "reward-123",
		RewardName: "5% Discount",
		RewardType: "discount",
		FaceAmount: 5.00,
		Currency:   "USD",
		RedeemedAt: "2025-11-14T10:00:00Z",
		RedeemedBy: "staff-user-123",
		LocationID: "location-123",
	}

	event := webhooks.NewRewardRedeemedEvent(tenantID, data)

	assert.Equal(t, webhooks.EventRewardRedeemed, event.Event)
	assert.Equal(t, tenantID.String(), event.TenantID)
	assert.Equal(t, data, event.Data)
}

func TestNewRewardExpiredEvent(t *testing.T) {
	tenantID := uuid.New()
	data := webhooks.RewardExpiredData{
		IssuanceID: "issuance-123",
		CustomerID: "customer-123",
		RewardID:   "reward-123",
		RewardName: "5% Discount",
		RewardType: "discount",
		FaceAmount: 5.00,
		Currency:   "USD",
		ExpiredAt:  "2025-11-21T10:00:00Z",
		IssuedAt:   "2025-11-14T10:00:00Z",
	}

	event := webhooks.NewRewardExpiredEvent(tenantID, data)

	assert.Equal(t, webhooks.EventRewardExpired, event.Event)
	assert.Equal(t, tenantID.String(), event.TenantID)
	assert.Equal(t, data, event.Data)
}

func TestNewBudgetThresholdEvent(t *testing.T) {
	tenantID := uuid.New()
	data := webhooks.BudgetThresholdData{
		BudgetID:   "budget-123",
		BudgetName: "Q4 Budget",
		Threshold:  "soft_cap",
		Balance:    7500.00,
		SoftCap:    8000.00,
		HardCap:    10000.00,
		Currency:   "USD",
		Utilization: 93.75,
	}

	event := webhooks.NewBudgetThresholdEvent(tenantID, data)

	assert.Equal(t, webhooks.EventBudgetThreshold, event.Event)
	assert.Equal(t, tenantID.String(), event.TenantID)
	assert.Equal(t, data, event.Data)
}

func TestEventPayload_JSONSerialization(t *testing.T) {
	tenantID := uuid.New()
	data := webhooks.RewardIssuedData{
		IssuanceID: "issuance-123",
		CustomerID: "customer-123",
		RewardID:   "reward-123",
		RewardName: "5% Discount",
		RewardType: "discount",
		Status:     "issued",
		FaceAmount: 5.00,
		Currency:   "USD",
		IssuedAt:   "2025-11-14T10:00:00Z",
	}

	event := webhooks.NewRewardIssuedEvent(tenantID, data)

	// Marshal to JSON
	jsonBytes, err := json.Marshal(event)
	require.NoError(t, err)

	// Unmarshal back
	var decoded webhooks.EventPayload
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, event.Event, decoded.Event)
	assert.Equal(t, event.TenantID, decoded.TenantID)
	assert.Equal(t, event.Timestamp, decoded.Timestamp)

	// Data will be map[string]interface{} after unmarshaling
	decodedData, ok := decoded.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "issuance-123", decodedData["issuance_id"])
	assert.Equal(t, "customer-123", decodedData["customer_id"])
}

func TestEventTypes_Constants(t *testing.T) {
	assert.Equal(t, "customer.enrolled", webhooks.EventCustomerEnrolled)
	assert.Equal(t, "reward.issued", webhooks.EventRewardIssued)
	assert.Equal(t, "reward.redeemed", webhooks.EventRewardRedeemed)
	assert.Equal(t, "reward.expired", webhooks.EventRewardExpired)
	assert.Equal(t, "budget.threshold", webhooks.EventBudgetThreshold)
}
