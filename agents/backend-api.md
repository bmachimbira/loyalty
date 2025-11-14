# Backend API Agent

## Mission
Implement the Go backend API with gin framework, including authentication, authorization, and all REST endpoints defined in the specification.

## Prerequisites
- Go 1.21+
- PostgreSQL connection established
- sqlc code generated
- Understand the spec: `Zimbabwe-White-Label-Loyalty-Spec-v1.0.md`

## Tasks

### 1. Authentication & Authorization

#### JWT Authentication for Staff Users
**File**: `api/internal/auth/jwt.go`

```go
package auth

import (
    "time"
    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    UserID    string `json:"user_id"`
    TenantID  string `json:"tenant_id"`
    Email     string `json:"email"`
    Role      string `json:"role"`
    jwt.RegisteredClaims
}

// GenerateToken creates a JWT token for a staff user
// CreateRefreshToken creates a long-lived refresh token
// ValidateToken verifies and parses a JWT token
// RefreshAccessToken exchanges refresh token for new access token
```

**Implementation checklist**:
- [ ] Generate JWT access tokens (15 min expiry)
- [ ] Generate refresh tokens (7 day expiry)
- [ ] Validate and parse tokens
- [ ] Extract claims (user_id, tenant_id, role)
- [ ] Handle token expiration

#### HMAC Authentication for Server-to-Server
**File**: `api/internal/auth/hmac.go`

```go
package auth

// ValidateHMAC verifies X-Signature header
// Expected headers: X-Key, X-Timestamp, X-Signature
// Signature = HMAC-SHA256(key_secret, X-Timestamp + request_body)
```

**Implementation checklist**:
- [ ] Parse HMAC headers (X-Key, X-Timestamp, X-Signature)
- [ ] Verify timestamp (within 5 minutes)
- [ ] Compute HMAC signature
- [ ] Compare signatures securely
- [ ] Load keys from HMAC_KEYS_JSON env var

#### Password Handling
**File**: `api/internal/auth/password.go`

```go
package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword using bcrypt
// ComparePassword verifies password against hash
```

### 2. Middleware

#### Tenant Context Middleware
**File**: `api/internal/http/middleware/tenant.go`

```go
package middleware

// TenantContext extracts tenant_id from JWT claims
// Sets PostgreSQL session variable: SET app.tenant_id = 'uuid'
// This enables Row-Level Security (RLS)
```

**Implementation checklist**:
- [ ] Extract tenant_id from JWT claims
- [ ] Execute `SET app.tenant_id = $1` on connection
- [ ] Handle errors (missing tenant_id, db errors)
- [ ] Add tenant_id to request context

#### Authentication Middleware
**File**: `api/internal/http/middleware/auth.go`

```go
package middleware

// RequireAuth validates JWT token
// RequireRole checks user has required role (owner|admin|staff|viewer)
// RequireHMAC validates HMAC signature for server-to-server
```

#### Idempotency Middleware
**File**: `api/internal/http/middleware/idempotency.go`

```go
package middleware

// IdempotencyCheck for POST requests with Idempotency-Key header
// Check if key exists in events/issuances table
// Return cached response if duplicate
```

### 3. Core API Endpoints

#### Customers API
**File**: `api/internal/http/handlers/customers.go`

```go
package handlers

type CustomersHandler struct {
    queries *db.Queries
}

// POST   /v1/tenants/:tid/customers
// GET    /v1/tenants/:tid/customers/:id
// GET    /v1/tenants/:tid/customers (list with pagination)
// PATCH  /v1/tenants/:tid/customers/:id/status
```

**Implementation checklist**:
- [ ] Create customer (phone_e164 or external_ref required)
- [ ] Get customer by ID
- [ ] Search customers (by phone or external_ref)
- [ ] List customers with pagination
- [ ] Update customer status
- [ ] Validate E.164 phone format

#### Events API
**File**: `api/internal/http/handlers/events.go`

```go
package handlers

type EventsHandler struct {
    queries *db.Queries
    rulesEngine *rules.Engine
}

// POST   /v1/tenants/:tid/events (with Idempotency-Key)
// GET    /v1/tenants/:tid/events/:id
// GET    /v1/tenants/:tid/events (list by customer)
```

**Implementation checklist**:
- [ ] Validate event payload
- [ ] Check idempotency key
- [ ] Insert event record
- [ ] Trigger rules engine evaluation
- [ ] Return event + any triggered issuances
- [ ] Handle concurrent requests

#### Rules API
**File**: `api/internal/http/handlers/rules.go`

```go
package handlers

// POST   /v1/tenants/:tid/rules
// GET    /v1/tenants/:tid/rules
// GET    /v1/tenants/:tid/rules/:id
// PATCH  /v1/tenants/:tid/rules/:id
// DELETE /v1/tenants/:tid/rules/:id (soft delete - set active=false)
```

**Implementation checklist**:
- [ ] Create rule with JSON conditions
- [ ] Validate JsonLogic conditions
- [ ] List active rules
- [ ] Get rule by ID
- [ ] Update rule
- [ ] Activate/deactivate rule

#### Rewards Catalog API
**File**: `api/internal/http/handlers/rewards.go`

```go
package handlers

// POST   /v1/tenants/:tid/reward-catalog
// GET    /v1/tenants/:tid/reward-catalog
// GET    /v1/tenants/:tid/reward-catalog/:id
// PATCH  /v1/tenants/:tid/reward-catalog/:id
// POST   /v1/tenants/:tid/reward-catalog/:id/upload-codes (CSV for voucher_code type)
```

**Implementation checklist**:
- [ ] Create reward item
- [ ] List rewards (active only or all)
- [ ] Get reward by ID
- [ ] Update reward
- [ ] Upload voucher codes (CSV)
- [ ] Validate reward types

#### Issuances API
**File**: `api/internal/http/handlers/issuances.go`

```go
package handlers

// GET    /v1/tenants/:tid/issuances
// GET    /v1/tenants/:tid/issuances/:id
// POST   /v1/tenants/:tid/issuances/:id/redeem (with OTP or staff PIN)
// POST   /v1/tenants/:tid/issuances/:id/cancel
```

**Implementation checklist**:
- [ ] List issuances (filter by customer, status)
- [ ] Get issuance details
- [ ] Redeem issuance (state: issued → redeemed)
- [ ] Cancel issuance (state: issued → cancelled)
- [ ] Validate state transitions

#### Budgets & Ledger API
**File**: `api/internal/http/handlers/budgets.go`

```go
package handlers

// POST   /v1/tenants/:tid/budgets
// GET    /v1/tenants/:tid/budgets
// GET    /v1/tenants/:tid/budgets/:id
// POST   /v1/tenants/:tid/budgets/:id/topup
// GET    /v1/tenants/:tid/ledger (query with from/to dates)
```

**Implementation checklist**:
- [ ] Create budget
- [ ] List budgets
- [ ] Get budget by ID
- [ ] Topup budget (ledger entry: fund)
- [ ] Query ledger entries with date range
- [ ] Calculate budget balance

#### Campaigns API
**File**: `api/internal/http/handlers/campaigns.go`

```go
package handlers

// POST   /v1/tenants/:tid/campaigns
// GET    /v1/tenants/:tid/campaigns
// PATCH  /v1/tenants/:tid/campaigns/:id
```

### 4. Public Endpoints (No Auth)

#### WhatsApp Webhooks
**File**: `api/internal/http/handlers/whatsapp.go`

```go
package handlers

// GET    /public/wa/webhook (verification challenge)
// POST   /public/wa/webhook (incoming messages)
```

**Implementation checklist**:
- [ ] Handle webhook verification (hub.challenge)
- [ ] Verify webhook signature
- [ ] Parse incoming messages
- [ ] Route to channel handler

#### USSD Callback
**File**: `api/internal/http/handlers/ussd.go`

```go
package handlers

// POST   /public/ussd/callback
```

### 5. Error Handling

**File**: `api/internal/http/errors.go`

```go
package http

type ErrorResponse struct {
    Error   string `json:"error"`
    Code    string `json:"code"`
    Details any    `json:"details,omitempty"`
}

// Error codes:
// - invalid_request
// - unauthorized
// - forbidden
// - not_found
// - conflict (duplicate idempotency key)
// - budget_exceeded
// - rate_limited
// - internal_error
```

### 6. Request Validation

**File**: `api/internal/http/validation.go`

```go
package http

// ValidateE164Phone validates phone number format
// ValidateUUID validates UUID format
// ValidateCurrency checks ZWG or USD
// ValidateEventType checks allowed event types
// ValidateRewardType checks allowed reward types
```

### 7. Router Setup

**File**: `api/internal/http/router.go`

```go
package http

func SetupRouter(queries *db.Queries, jwtSecret string) *gin.Engine {
    r := gin.Default()

    // Middleware
    r.Use(middleware.CORS())
    r.Use(middleware.RequestID())
    r.Use(middleware.Logger())

    // Health check
    r.GET("/health", HealthCheck)

    // Public routes
    public := r.Group("/public")
    public.GET("/wa/webhook", whatsappHandler.Verify)
    public.POST("/wa/webhook", whatsappHandler.Webhook)

    // V1 API (authenticated)
    v1 := r.Group("/v1")
    v1.Use(middleware.RequireAuth(jwtSecret))
    v1.Use(middleware.TenantContext())

    // Register handlers...

    return r
}
```

Update `api/cmd/api/main.go` to use this router.

## Testing

Create tests for each handler:
- `api/internal/http/handlers/*_test.go`

Test cases:
- [ ] Valid requests return 200/201
- [ ] Invalid auth returns 401
- [ ] Missing tenant context returns 400
- [ ] Invalid payload returns 400
- [ ] Duplicate idempotency key returns cached response
- [ ] RLS prevents cross-tenant access

## Performance Targets

- Event ingestion: p95 < 150ms
- Rule evaluation: < 25ms per event
- Sustain 100 RPS on single node

## Documentation

Update API documentation with example requests/responses.

## Completion Criteria

- [ ] All endpoints implemented and tested
- [ ] Authentication working (JWT + HMAC)
- [ ] Tenant isolation via RLS verified
- [ ] Idempotency working
- [ ] Error handling comprehensive
- [ ] Request validation complete
- [ ] Performance targets met
