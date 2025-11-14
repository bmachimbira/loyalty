# Rewards Agent

## Mission
Implement the reward state machine, reward type handlers, and integration with external suppliers.

## Prerequisites
- Go 1.21+
- Understanding of state machines
- Knowledge of external API integration

## Tasks

### 1. Reward State Machine

**File**: `api/internal/reward/state.go`

```go
package reward

type State string

const (
    StateReserved  State = "reserved"
    StateIssued    State = "issued"
    StateRedeemed  State = "redeemed"
    StateExpired   State = "expired"
    StateCancelled State = "cancelled"
    StateFailed    State = "failed"
)

// Valid state transitions
var transitions = map[State][]State{
    StateReserved: {StateIssued, StateCancelled, StateFailed},
    StateIssued:   {StateRedeemed, StateExpired, StateCancelled},
}

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
```

### 2. Reward Service

**File**: `api/internal/reward/service.go`

```go
package reward

import (
    "context"
    "errors"
)

type Service struct {
    queries   *db.Queries
    handlers  map[string]RewardHandler
    budget    *budget.Service
}

func NewService(queries *db.Queries, budgetService *budget.Service) *Service {
    s := &Service{
        queries:  queries,
        budget:   budgetService,
        handlers: make(map[string]RewardHandler),
    }

    // Register handlers
    s.RegisterHandler("discount", &DiscountHandler{})
    s.RegisterHandler("voucher_code", &VoucherCodeHandler{queries})
    s.RegisterHandler("external_voucher", &ExternalVoucherHandler{})
    s.RegisterHandler("points_credit", &PointsCreditHandler{})
    s.RegisterHandler("physical_item", &PhysicalItemHandler{})
    s.RegisterHandler("webhook_custom", &WebhookHandler{})

    return s
}

// ProcessIssuance moves from reserved → issued
func (s *Service) ProcessIssuance(ctx context.Context, issuanceID string) error {
    issuance, err := s.queries.GetIssuanceByID(ctx, ...)
    if err != nil {
        return err
    }

    if issuance.Status != string(StateReserved) {
        return errors.New("invalid state")
    }

    reward, err := s.queries.GetRewardByID(ctx, ...)
    if err != nil {
        return err
    }

    // Get handler for reward type
    handler, ok := s.handlers[reward.Type]
    if !ok {
        return errors.New("unknown reward type")
    }

    // Process reward
    result, err := handler.Process(ctx, issuance, reward)
    if err != nil {
        // Mark as failed
        s.updateState(ctx, issuance.ID, StateFailed)
        return err
    }

    // Update issuance to issued
    err = s.updateState(ctx, issuance.ID, StateIssued)
    if err != nil {
        return err
    }

    // Update code/external_ref
    if result.Code != "" {
        s.queries.UpdateIssuanceCode(ctx, ...)
    }

    return nil
}

func (s *Service) updateState(ctx context.Context, issuanceID string, state State) error {
    // Validate state transition
    // Update database
    // Update ledger if needed
}
```

### 3. Reward Handlers

#### Discount Handler
**File**: `api/internal/reward/handlers/discount.go`

```go
package handlers

type DiscountHandler struct{}

type ProcessResult struct {
    Code        string
    ExternalRef string
}

func (h *DiscountHandler) Process(ctx context.Context, issuance *db.Issuance, reward *db.Reward) (*ProcessResult, error) {
    // Generate unique discount code
    code := generateDiscountCode()

    // Parse metadata
    var meta struct {
        DiscountType string  `json:"discount_type"` // "amount" or "percent"
        Amount       float64 `json:"amount"`
        MinBasket    float64 `json:"min_basket"`
        ValidDays    int     `json:"valid_days"`
    }
    json.Unmarshal(reward.Metadata, &meta)

    // Set expiry based on valid_days
    expiresAt := time.Now().AddDate(0, 0, meta.ValidDays)

    return &ProcessResult{
        Code: code,
    }, nil
}

func generateDiscountCode() string {
    // Generate 8-char alphanumeric code
    const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
    b := make([]byte, 8)
    for i := range b {
        b[i] = chars[rand.Intn(len(chars))]
    }
    return string(b)
}
```

#### Voucher Code Handler
**File**: `api/internal/reward/handlers/voucher_code.go`

```go
package handlers

type VoucherCodeHandler struct {
    queries *db.Queries
}

func (h *VoucherCodeHandler) Process(ctx context.Context, issuance *db.Issuance, reward *db.Reward) (*ProcessResult, error) {
    // Reserve a code from voucher_codes table
    code, err := h.queries.ReserveVoucherCode(ctx, db.ReserveVoucherCodeParams{
        TenantID: issuance.TenantID,
        RewardID: reward.ID,
        IssuanceID: issuance.ID,
    })
    if err != nil {
        return nil, errors.New("no codes available")
    }

    // Mark as issued
    h.queries.MarkVoucherCodeIssued(ctx, db.MarkVoucherCodeIssuedParams{
        ID:       code.ID,
        TenantID: issuance.TenantID,
    })

    return &ProcessResult{
        Code: code.Code,
    }, nil
}
```

#### External Voucher Handler
**File**: `api/internal/reward/handlers/external_voucher.go`

```go
package handlers

type ExternalVoucherHandler struct {
    connectors map[string]Connector
}

type Connector interface {
    IssueVoucher(ctx context.Context, params IssueParams) (*IssueResponse, error)
}

func (h *ExternalVoucherHandler) Process(ctx context.Context, issuance *db.Issuance, reward *db.Reward) (*ProcessResult, error) {
    // Get supplier connector
    var meta struct {
        SupplierID string `json:"supplier_id"`
        ProductID  string `json:"product_id"`
    }
    json.Unmarshal(reward.Metadata, &meta)

    connector, ok := h.connectors[meta.SupplierID]
    if !ok {
        return nil, errors.New("supplier not configured")
    }

    // Call external API
    resp, err := connector.IssueVoucher(ctx, IssueParams{
        ProductID:  meta.ProductID,
        CustomerID: issuance.CustomerID,
        Amount:     reward.FaceValue,
    })
    if err != nil {
        return nil, err
    }

    return &ProcessResult{
        Code:        resp.VoucherCode,
        ExternalRef: resp.TransactionID,
    }, nil
}
```

#### Points Credit Handler
**File**: `api/internal/reward/handlers/points.go`

```go
package handlers

type PointsCreditHandler struct{}

func (h *PointsCreditHandler) Process(ctx context.Context, issuance *db.Issuance, reward *db.Reward) (*ProcessResult, error) {
    // Points are just recorded in issuance
    // No external action needed
    // Customer can redeem later via points catalog
    return &ProcessResult{}, nil
}
```

#### Physical Item Handler
**File**: `api/internal/reward/handlers/physical.go`

```go
package handlers

type PhysicalItemHandler struct{}

func (h *PhysicalItemHandler) Process(ctx context.Context, issuance *db.Issuance, reward *db.Reward) (*ProcessResult, error) {
    // Generate claim token
    token := generateClaimToken()

    // Physical items require manual fulfilment
    // Store manager validates token when customer collects item

    return &ProcessResult{
        Code: token,
    }, nil
}
```

#### Webhook Custom Handler
**File**: `api/internal/reward/handlers/webhook.go`

```go
package handlers

type WebhookHandler struct {
    client *http.Client
}

func (h *WebhookHandler) Process(ctx context.Context, issuance *db.Issuance, reward *db.Reward) (*ProcessResult, error) {
    var meta struct {
        WebhookURL string `json:"webhook_url"`
        Secret     string `json:"secret"`
    }
    json.Unmarshal(reward.Metadata, &meta)

    // Prepare payload
    payload := map[string]interface{}{
        "issuance_id": issuance.ID,
        "customer_id": issuance.CustomerID,
        "reward":      reward,
        "timestamp":   time.Now().Unix(),
    }

    payloadBytes, _ := json.Marshal(payload)

    // Sign with HMAC
    signature := computeHMAC(meta.Secret, payloadBytes)

    // POST to webhook URL
    req, _ := http.NewRequestWithContext(ctx, "POST", meta.WebhookURL, bytes.NewReader(payloadBytes))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Signature", signature)

    resp, err := h.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, errors.New("webhook failed")
    }

    return &ProcessResult{}, nil
}
```

### 4. Redemption

**File**: `api/internal/reward/redemption.go`

```go
package reward

func (s *Service) RedeemIssuance(ctx context.Context, issuanceID string, otp string) error {
    issuance, err := s.queries.GetIssuanceByID(ctx, ...)
    if err != nil {
        return err
    }

    if issuance.Status != string(StateIssued) {
        return errors.New("cannot redeem")
    }

    // Verify OTP/code
    if issuance.Code != otp {
        return errors.New("invalid code")
    }

    // Check expiry
    if issuance.ExpiresAt.Valid && time.Now().After(issuance.ExpiresAt.Time) {
        return errors.New("expired")
    }

    // Update state to redeemed
    err = s.updateState(ctx, issuanceID, StateRedeemed)
    if err != nil {
        return err
    }

    // Charge budget (ledger: reserve → charge)
    s.budget.ChargeReservation(ctx, issuance.ID)

    return nil
}
```

### 5. Expiry Worker

**File**: `api/internal/reward/expiry.go`

```go
package reward

func (s *Service) ExpireOldIssuances(ctx context.Context) error {
    // Query issuances where expires_at < NOW() and status = 'issued'
    expired, err := s.queries.GetExpiredIssuances(ctx)
    if err != nil {
        return err
    }

    for _, issuance := range expired {
        // Update to expired
        s.updateState(ctx, issuance.ID, StateExpired)

        // Release budget
        s.budget.ReleaseReservation(ctx, issuance.ID)
    }

    return nil
}

// Run this as a background job every hour
```

### 6. Testing

**File**: `api/internal/reward/service_test.go`

Test cases:
- [ ] State transitions (valid/invalid)
- [ ] Discount code generation
- [ ] Voucher code reservation
- [ ] External voucher issuance
- [ ] Webhook delivery
- [ ] Redemption with valid/invalid code
- [ ] Expiry processing
- [ ] Budget integration

## Completion Criteria

- [ ] All reward types implemented
- [ ] State machine enforced
- [ ] Redemption working
- [ ] Expiry worker running
- [ ] External connectors integrated
- [ ] Tests passing (>80% coverage)
- [ ] Error handling comprehensive
