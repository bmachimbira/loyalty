# Integration Agent

## Mission
Implement external service integrations including airtime providers, webhook delivery, and third-party connectors.

## Prerequisites
- Go 1.21+
- Understanding of HTTP clients and retry logic
- Knowledge of HMAC signatures

## Tasks

### 1. Connector Interface

**File**: `api/internal/connectors/interface.go`

```go
package connectors

import "context"

type Connector interface {
    Name() string
    IssueVoucher(ctx context.Context, params IssueParams) (*IssueResponse, error)
    CheckStatus(ctx context.Context, transactionID string) (*StatusResponse, error)
}

type IssueParams struct {
    ProductID   string
    CustomerID  string
    PhoneNumber string
    Amount      float64
    Currency    string
    Reference   string
}

type IssueResponse struct {
    VoucherCode   string
    TransactionID string
    Status        string
    Message       string
}

type StatusResponse struct {
    Status  string
    Message string
}
```

### 2. Airtime Connector (Example)

**File**: `api/internal/connectors/airtime/provider.go`

```go
package airtime

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
)

type Provider struct {
    client   *http.Client
    baseURL  string
    apiKey   string
    secret   string
}

func New(baseURL, apiKey, secret string) *Provider {
    return &Provider{
        client:  &http.Client{Timeout: 30 * time.Second},
        baseURL: baseURL,
        apiKey:  apiKey,
        secret:  secret,
    }
}

func (p *Provider) Name() string {
    return "airtime_provider"
}

func (p *Provider) IssueVoucher(ctx context.Context, params connectors.IssueParams) (*connectors.IssueResponse, error) {
    // Build request
    payload := map[string]interface{}{
        "product_id":   params.ProductID,
        "phone_number": params.PhoneNumber,
        "amount":       params.Amount,
        "reference":    params.Reference,
    }

    body, _ := json.Marshal(payload)

    req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/issue", bytes.NewReader(body))
    if err != nil {
        return nil, err
    }

    // Add auth headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-API-Key", p.apiKey)
    req.Header.Set("X-Signature", p.sign(body))

    // Send request with retry
    resp, err := p.sendWithRetry(req, 3)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Parse response
    var result struct {
        VoucherCode   string `json:"voucher_code"`
        TransactionID string `json:"transaction_id"`
        Status        string `json:"status"`
        Message       string `json:"message"`
    }

    json.NewDecoder(resp.Body).Decode(&result)

    return &connectors.IssueResponse{
        VoucherCode:   result.VoucherCode,
        TransactionID: result.TransactionID,
        Status:        result.Status,
        Message:       result.Message,
    }, nil
}

func (p *Provider) sign(body []byte) string {
    mac := hmac.New(sha256.New, []byte(p.secret))
    mac.Write(body)
    return hex.EncodeToString(mac.Sum(nil))
}

func (p *Provider) sendWithRetry(req *http.Request, maxRetries int) (*http.Response, error) {
    var resp *http.Response
    var err error

    backoff := time.Second

    for i := 0; i < maxRetries; i++ {
        resp, err = p.client.Do(req)

        if err == nil && resp.StatusCode < 500 {
            return resp, nil
        }

        // Exponential backoff
        time.Sleep(backoff)
        backoff *= 2

        if i < maxRetries-1 && resp != nil {
            resp.Body.Close()
        }
    }

    return resp, err
}
```

### 3. Connector Registry

**File**: `api/internal/connectors/registry.go`

```go
package connectors

type Registry struct {
    connectors map[string]Connector
}

func NewRegistry() *Registry {
    return &Registry{
        connectors: make(map[string]Connector),
    }
}

func (r *Registry) Register(connector Connector) {
    r.connectors[connector.Name()] = connector
}

func (r *Registry) Get(name string) (Connector, bool) {
    conn, ok := r.connectors[name]
    return conn, ok
}
```

### 4. Webhook Delivery System

**File**: `api/internal/webhooks/delivery.go`

```go
package webhooks

import (
    "bytes"
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "net/http"
    "time"
)

type DeliveryService struct {
    queries *db.Queries
    client  *http.Client
    queue   chan *Delivery
}

type Delivery struct {
    WebhookID string
    Event     string
    Payload   interface{}
}

func NewDeliveryService(queries *db.Queries) *DeliveryService {
    s := &DeliveryService{
        queries: queries,
        client:  &http.Client{Timeout: 10 * time.Second},
        queue:   make(chan *Delivery, 100),
    }

    // Start workers
    for i := 0; i < 5; i++ {
        go s.worker()
    }

    return s
}

func (s *DeliveryService) Send(ctx context.Context, webhookID, event string, payload interface{}) {
    s.queue <- &Delivery{
        WebhookID: webhookID,
        Event:     event,
        Payload:   payload,
    }
}

func (s *DeliveryService) worker() {
    for delivery := range s.queue {
        s.deliver(context.Background(), delivery)
    }
}

func (s *DeliveryService) deliver(ctx context.Context, delivery *Delivery) error {
    // Get webhook config
    webhook, err := s.queries.GetWebhookByID(ctx, delivery.WebhookID)
    if err != nil {
        return err
    }

    if !webhook.Active {
        return nil
    }

    // Check if event is subscribed
    if !contains(webhook.Events, delivery.Event) {
        return nil
    }

    // Build payload
    payload := map[string]interface{}{
        "event":     delivery.Event,
        "timestamp": time.Now().Unix(),
        "data":      delivery.Payload,
    }

    body, _ := json.Marshal(payload)

    // Create request
    req, _ := http.NewRequestWithContext(ctx, "POST", webhook.URL, bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Event", delivery.Event)
    req.Header.Set("X-Signature", s.sign(body, webhook.Secret))

    // Send with retry
    resp, err := s.sendWithRetry(req, 3)
    if err != nil {
        // Log failure
        return err
    }
    defer resp.Body.Close()

    // Log success
    return nil
}

func (s *DeliveryService) sign(body []byte, secret string) string {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    return hex.EncodeToString(mac.Sum(nil))
}

func (s *DeliveryService) sendWithRetry(req *http.Request, maxRetries int) (*http.Response, error) {
    var resp *http.Response
    var err error

    backoff := time.Second

    for i := 0; i < maxRetries; i++ {
        resp, err = s.client.Do(req)

        if err == nil && resp.StatusCode < 500 {
            return resp, nil
        }

        time.Sleep(backoff)
        backoff *= 2

        if i < maxRetries-1 && resp != nil {
            resp.Body.Close()
        }
    }

    return resp, err
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}
```

### 5. Webhook Events

**File**: `api/internal/webhooks/events.go`

```go
package webhooks

const (
    EventCustomerEnrolled  = "customer.enrolled"
    EventRewardIssued      = "reward.issued"
    EventRewardRedeemed    = "reward.redeemed"
    EventRewardExpired     = "reward.expired"
    EventBudgetThreshold   = "budget.threshold"
)

func (s *DeliveryService) NotifyRewardIssued(ctx context.Context, tenantID string, issuance *db.Issuance) {
    // Get all webhooks for tenant subscribed to reward.issued
    webhooks, _ := s.queries.GetWebhooksByEvent(ctx, tenantID, EventRewardIssued)

    for _, webhook := range webhooks {
        s.Send(ctx, webhook.ID, EventRewardIssued, issuance)
    }
}
```

### 6. Circuit Breaker

**File**: `api/internal/connectors/circuitbreaker.go`

```go
package connectors

import (
    "errors"
    "sync"
    "time"
)

type CircuitBreaker struct {
    mu            sync.Mutex
    failureCount  int
    threshold     int
    timeout       time.Duration
    state         string // "closed", "open", "half-open"
    lastFailTime  time.Time
}

func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
    return &CircuitBreaker{
        threshold: threshold,
        timeout:   timeout,
        state:     "closed",
    }
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    cb.mu.Lock()

    // Check if circuit is open
    if cb.state == "open" {
        if time.Since(cb.lastFailTime) > cb.timeout {
            cb.state = "half-open"
            cb.failureCount = 0
        } else {
            cb.mu.Unlock()
            return errors.New("circuit breaker open")
        }
    }

    cb.mu.Unlock()

    // Execute function
    err := fn()

    cb.mu.Lock()
    defer cb.mu.Unlock()

    if err != nil {
        cb.failureCount++
        cb.lastFailTime = time.Now()

        if cb.failureCount >= cb.threshold {
            cb.state = "open"
        }

        return err
    }

    // Success - reset
    if cb.state == "half-open" {
        cb.state = "closed"
    }
    cb.failureCount = 0

    return nil
}
```

### 7. Connector Configuration

**File**: `api/internal/connectors/config.go`

```go
package connectors

type Config struct {
    AirtimeProvider struct {
        Enabled bool   `json:"enabled"`
        BaseURL string `json:"base_url"`
        APIKey  string `json:"api_key"`
        Secret  string `json:"secret"`
    } `json:"airtime_provider"`

    GiftCardProvider struct {
        Enabled bool   `json:"enabled"`
        BaseURL string `json:"base_url"`
        APIKey  string `json:"api_key"`
    } `json:"gift_card_provider"`
}

func LoadConfig() (*Config, error) {
    // Load from environment or config file
}
```

### 8. Testing

**File**: `api/internal/connectors/airtime/provider_test.go`

```go
package airtime_test

func TestProvider_IssueVoucher(t *testing.T) {
    // Mock HTTP server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify signature
        // Return success response
        json.NewEncoder(w).Encode(map[string]string{
            "voucher_code":   "ABC123",
            "transaction_id": "TX123",
            "status":         "success",
        })
    }))
    defer server.Close()

    provider := airtime.New(server.URL, "key", "secret")

    resp, err := provider.IssueVoucher(ctx, connectors.IssueParams{
        ProductID:   "DATA_200MB",
        PhoneNumber: "+263771234567",
        Amount:      5.00,
    })

    assert.NoError(t, err)
    assert.Equal(t, "ABC123", resp.VoucherCode)
}

func TestProvider_Retry(t *testing.T) {
    // Test retry logic on failures
}
```

**File**: `api/internal/webhooks/delivery_test.go`

```go
package webhooks_test

func TestDeliveryService_Send(t *testing.T) {
    // Test webhook delivery
}

func TestDeliveryService_Retry(t *testing.T) {
    // Test retry on failure
}

func TestDeliveryService_Signature(t *testing.T) {
    // Test HMAC signature
}
```

## Completion Criteria

- [ ] Connector interface defined
- [ ] Airtime provider implemented
- [ ] Webhook delivery system working
- [ ] Retry logic implemented
- [ ] Circuit breaker implemented
- [ ] HMAC signatures correct
- [ ] Tests passing
- [ ] Error handling comprehensive
