# Zimbabwe Loyalty Platform - Implementation Tracker

**Project Start Date**: 2025-11-14
**Current Sprint**: Foundation (Week 1-2)
**Overall Progress**: 0% Complete

This document tracks the implementation progress of all features in the loyalty platform. Update checkboxes as tasks are completed.

---

## Phase 1: Foundation (Weeks 1-2)

### Database Schema & Migrations
- [ ] Initial schema migration (001_initial_schema.sql)
- [ ] Seed data migration (002_seed_data.sql)
- [ ] Index optimization migration (003_indexes_optimization.sql)
- [ ] Voucher pool table migration (004_voucher_pool.sql)
- [ ] Database functions migration (005_functions.sql)
- [ ] RLS policies verified and tested
- [ ] All migrations executable without errors

### sqlc Code Generation
- [ ] Configure sqlc.yaml
- [ ] Tenants queries implemented
- [ ] Staff users queries implemented
- [ ] Customers queries implemented
- [ ] Consents queries implemented
- [ ] Budgets queries implemented
- [ ] Ledger queries implemented
- [ ] Rewards queries implemented
- [ ] Rules queries implemented
- [ ] Campaigns queries implemented
- [ ] Events queries implemented
- [ ] Issuances queries implemented
- [ ] WhatsApp sessions queries implemented
- [ ] USSD sessions queries implemented
- [ ] Audit logs queries implemented
- [ ] Voucher codes queries implemented
- [ ] Analytics queries implemented
- [ ] Run `sqlc generate` successfully
- [ ] All generated Go code compiles

### Backend - Core Setup
- [x] Go project initialized (go.mod)
- [x] Main application entry point (cmd/api/main.go)
- [x] Database connection pool setup
- [x] Environment variable configuration
- [x] Graceful shutdown implemented
- [x] Project structure organized

### Backend - Authentication & Authorization

**JWT Authentication**:
- [x] JWT token generation (api/internal/auth/jwt.go)
- [x] JWT token validation
- [x] Claims extraction (user_id, tenant_id, role)
- [x] Access token (15 min expiry)
- [x] Refresh token (7 day expiry)
- [x] Refresh token flow

**HMAC Authentication**:
- [x] HMAC signature generation (api/internal/auth/hmac.go)
- [x] HMAC signature verification
- [x] Timestamp validation (5 min window)
- [x] Key management from HMAC_KEYS_JSON

**Password Handling**:
- [x] bcrypt password hashing (api/internal/auth/password.go)
- [x] Password comparison

### Backend - Middleware

**Tenant Context**:
- [x] Extract tenant_id from JWT claims (api/internal/http/middleware/tenant.go)
- [x] Set PostgreSQL app.tenant_id session variable
- [x] Handle missing tenant_id errors
- [x] Add tenant_id to request context

**Authentication**:
- [x] RequireAuth middleware (api/internal/http/middleware/auth.go)
- [x] RequireRole middleware
- [x] RequireHMAC middleware for server-to-server

**Idempotency**:
- [x] Idempotency-Key header handling (api/internal/http/middleware/idempotency.go)
- [x] Check for duplicate keys (in-memory cache for Phase 1)
- [x] Return cached responses for duplicates

**Other Middleware**:
- [x] CORS middleware
- [x] Request ID middleware
- [x] Logging middleware
- [ ] Metrics middleware (deferred to Phase 4)

### Backend - API Handlers

**Customers API** (api/internal/http/handlers/customers.go):
- [x] POST /v1/tenants/:tid/customers (create) - ready for sqlc integration
- [x] GET /v1/tenants/:tid/customers/:id (get) - ready for sqlc integration
- [x] GET /v1/tenants/:tid/customers (list with pagination) - ready for sqlc integration
- [x] PATCH /v1/tenants/:tid/customers/:id/status (update status) - ready for sqlc integration
- [x] E.164 phone validation

**Events API** (api/internal/http/handlers/events.go):
- [x] POST /v1/tenants/:tid/events (create with idempotency) - ready for sqlc integration
- [x] GET /v1/tenants/:tid/events/:id (get) - ready for sqlc integration
- [x] GET /v1/tenants/:tid/events (list) - ready for sqlc integration
- [ ] Trigger rules engine on event creation (Phase 2)

**Rules API** (api/internal/http/handlers/rules.go):
- [x] POST /v1/tenants/:tid/rules (create) - ready for sqlc integration
- [x] GET /v1/tenants/:tid/rules (list) - ready for sqlc integration
- [x] GET /v1/tenants/:tid/rules/:id (get) - ready for sqlc integration
- [x] PATCH /v1/tenants/:tid/rules/:id (update) - ready for sqlc integration
- [x] DELETE /v1/tenants/:tid/rules/:id (soft delete) - ready for sqlc integration
- [ ] JsonLogic condition validation (Phase 2)

**Rewards API** (api/internal/http/handlers/rewards.go):
- [x] POST /v1/tenants/:tid/reward-catalog (create) - ready for sqlc integration
- [x] GET /v1/tenants/:tid/reward-catalog (list) - ready for sqlc integration
- [x] GET /v1/tenants/:tid/reward-catalog/:id (get) - ready for sqlc integration
- [x] PATCH /v1/tenants/:tid/reward-catalog/:id (update) - ready for sqlc integration
- [x] POST /v1/tenants/:tid/reward-catalog/:id/upload-codes (CSV upload) - ready for sqlc integration

**Issuances API** (api/internal/http/handlers/issuances.go):
- [x] GET /v1/tenants/:tid/issuances (list) - ready for sqlc integration
- [x] GET /v1/tenants/:tid/issuances/:id (get) - ready for sqlc integration
- [x] POST /v1/tenants/:tid/issuances/:id/redeem (redeem) - ready for sqlc integration
- [x] POST /v1/tenants/:tid/issuances/:id/cancel (cancel) - ready for sqlc integration

**Budgets API** (api/internal/http/handlers/budgets.go):
- [x] POST /v1/tenants/:tid/budgets (create) - ready for sqlc integration
- [x] GET /v1/tenants/:tid/budgets (list) - ready for sqlc integration
- [x] GET /v1/tenants/:tid/budgets/:id (get) - ready for sqlc integration
- [x] POST /v1/tenants/:tid/budgets/:id/topup (topup) - ready for sqlc integration
- [x] GET /v1/tenants/:tid/ledger (query ledger) - ready for sqlc integration

**Campaigns API** (api/internal/http/handlers/campaigns.go):
- [x] POST /v1/tenants/:tid/campaigns (create) - ready for sqlc integration
- [x] GET /v1/tenants/:tid/campaigns (list) - ready for sqlc integration
- [x] PATCH /v1/tenants/:tid/campaigns/:id (update) - ready for sqlc integration

**Error Handling**:
- [x] ErrorResponse type defined (api/internal/http/errors.go)
- [x] Error codes implemented
- [x] Consistent error responses

**Request Validation**:
- [x] ValidateE164Phone (api/internal/http/validation.go)
- [x] ValidateUUID
- [x] ValidateCurrency
- [x] ValidateEventType
- [x] ValidateRewardType

**Router Setup**:
- [x] SetupRouter function (api/internal/http/router.go)
- [x] Health check endpoint
- [x] Public routes group
- [x] V1 API routes group with auth
- [x] Update main.go to use router

### Frontend - Core Setup
- [x] Vite + React + TypeScript project initialized
- [x] Tailwind CSS configured
- [x] shadcn/ui initialized
- [x] All required shadcn/ui components installed
- [x] TypeScript types defined (web/src/lib/types.ts)
- [x] API client implemented (web/src/lib/api.ts)

### Frontend - Authentication
- [x] AuthContext created (web/src/contexts/AuthContext.tsx)
- [x] Login page
- [x] Token management (localStorage)
- [x] Protected routes
- [x] Logout functionality

### Frontend - Layout
- [x] Layout component (web/src/components/Layout.tsx)
- [x] Navigation bar
- [x] Routing setup (React Router)

### Frontend - Pages (Basic Structure)
- [x] Dashboard page (web/src/pages/Dashboard.tsx)
- [x] Customers page (web/src/pages/Customers.tsx)
- [x] Rewards page (web/src/pages/Rewards.tsx)
- [x] Rules page (web/src/pages/Rules.tsx)
- [x] Campaigns page (web/src/pages/Campaigns.tsx)
- [x] Budgets page (web/src/pages/Budgets.tsx)

---

## Phase 2: Core Features (Weeks 3-4)

### Rules Engine

**JsonLogic Evaluator** (api/internal/rules/jsonlogic.go):
- [x] Parse JsonLogic conditions
- [x] Comparison operators (==, !=, >, >=, <, <=)
- [x] Logical operators (all, any, none)
- [x] Array operators (in)
- [x] Variable access (var)

**Custom Operators** (api/internal/rules/custom_operators.go):
- [x] within_days operator
- [x] nth_event_in_period operator
- [x] distinct_visit_days operator

**Rules Engine** (api/internal/rules/engine.go):
- [x] ProcessEvent function
- [x] Get matching rules (with cache)
- [x] Evaluate rule conditions
- [x] Check caps and cooldowns
- [x] Issue rewards
- [x] Advisory locks for concurrency

**Cap Enforcement** (api/internal/rules/caps.go):
- [x] Per-user cap checking
- [x] Global cap checking
- [x] Cooldown checking
- [x] Query optimization for cap checks

**Reward Issuance** (api/internal/rules/issuance.go):
- [x] Transaction-based issuance
- [x] Advisory lock implementation
- [x] Budget reservation
- [x] State creation (reserved)
- [ ] Async processing trigger (deferred to Reward Service)

**Rule Cache** (api/internal/rules/cache.go):
- [x] Cache implementation
- [x] Get/Set with TTL
- [x] Cleanup expired entries

**Tests**:
- [x] Simple condition tests
- [x] Complex condition tests
- [x] Custom operator tests (implementation ready)
- [ ] Per-user cap tests (requires test database)
- [ ] Global cap tests (requires test database)
- [ ] Cooldown tests (requires test database)
- [ ] Concurrent event tests (requires test database)
- [x] Performance benchmark (<25ms target)

**Integration**:
- [x] Updated events handler to call rules engine
- [x] Event creation with idempotency
- [x] Rules engine invocation on event creation
- [x] Return issuances in event response

### Reward Service

**State Machine** (api/internal/reward/state.go):
- [ ] State type defined
- [ ] Valid transitions map
- [ ] CanTransitionTo function

**Reward Service** (api/internal/reward/service.go):
- [ ] Service struct with handler registry
- [ ] ProcessIssuance function (reserved → issued)
- [ ] updateState function
- [ ] Handler registration

**Reward Handlers**:
- [ ] Discount handler (api/internal/reward/handlers/discount.go)
- [ ] Voucher code handler (api/internal/reward/handlers/voucher_code.go)
- [ ] External voucher handler (api/internal/reward/handlers/external_voucher.go)
- [ ] Points credit handler (api/internal/reward/handlers/points.go)
- [ ] Physical item handler (api/internal/reward/handlers/physical.go)
- [ ] Webhook custom handler (api/internal/reward/handlers/webhook.go)

**Redemption** (api/internal/reward/redemption.go):
- [ ] RedeemIssuance function
- [ ] OTP/code verification
- [ ] Expiry checking
- [ ] State transition to redeemed
- [ ] Budget charging

**Expiry Worker** (api/internal/reward/expiry.go):
- [ ] ExpireOldIssuances function
- [ ] Background job scheduling
- [ ] Budget release on expiry

**Tests**:
- [ ] State transition tests
- [ ] Discount handler tests
- [ ] Voucher code handler tests
- [ ] External voucher tests
- [ ] Redemption tests
- [ ] Expiry tests

### Budget & Ledger Service

**Budget Service** (api/internal/budget/service.go):
- [x] ReserveBudget function
- [x] ChargeReservation function
- [x] ReleaseReservation function
- [x] Hard cap enforcement
- [x] Soft cap alerting
- [x] Currency validation

**Budget Topup** (api/internal/budget/topup.go):
- [x] TopupBudget function
- [x] Ledger entry creation

**Reconciliation** (api/internal/budget/reconciliation.go):
- [x] ReconcileBudget function
- [x] Balance verification
- [x] Discrepancy detection

**Alerts** (api/internal/budget/alerts.go):
- [x] Soft cap exceeded alert
- [x] Hard cap approaching alert

**Period Budgets** (api/internal/budget/period.go):
- [x] Monthly budget reset
- [x] Cron job for reset

**Reports** (api/internal/budget/reports.go):
- [x] GenerateReport function
- [x] Aggregation logic

**Tests**:
- [x] Reserve budget tests
- [x] Hard cap tests
- [x] Soft cap tests
- [x] Charge/release tests
- [x] Reconciliation tests
- [x] Currency validation tests
- [x] Concurrent reservation tests

---

## Phase 3: Channels (Weeks 5-6)

### WhatsApp Integration

**Webhook Handler** (api/internal/channels/whatsapp/webhook.go):
- [ ] GET /public/wa/webhook (verification)
- [ ] POST /public/wa/webhook (messages)
- [ ] Signature verification

**Message Types** (api/internal/channels/whatsapp/types.go):
- [ ] WebhookPayload struct
- [ ] Message types defined

**Message Processor** (api/internal/channels/whatsapp/processor.go):
- [ ] ProcessMessage function
- [ ] Command parsing
- [ ] Enrollment flow
- [ ] Rewards listing
- [ ] Balance checking
- [ ] Referral handling
- [ ] Help command

**Message Sender** (api/internal/channels/whatsapp/sender.go):
- [ ] SendText function
- [ ] SendTemplate function
- [ ] HTTP client setup

**Templates** (api/internal/channels/whatsapp/templates.go):
- [ ] LOYALTY_WELCOME template
- [ ] REWARD_ISSUED template
- [ ] REWARD_REMINDER template
- [ ] REWARD_REDEEMED template

**Tests**:
- [ ] Webhook verification test
- [ ] Signature verification test
- [ ] Message parsing test
- [ ] Enrollment flow test
- [ ] Template sending test

### USSD Integration

**USSD Handler** (api/internal/channels/ussd/handler.go):
- [ ] POST /public/ussd/callback endpoint
- [ ] Session management
- [ ] Input parsing
- [ ] Response formatting

**Menu System** (api/internal/channels/ussd/menus.go):
- [ ] MainMenu implementation
- [ ] RewardsMenu implementation
- [ ] BalanceMenu implementation
- [ ] RedeemMenu implementation
- [ ] Menu navigation

**Tests**:
- [ ] Menu navigation tests
- [ ] Session management tests
- [ ] Input validation tests

### Frontend - Complete Features

**Dashboard**:
- [ ] Stats cards (customers, events, rewards, redemption rate)
- [ ] Charts (issuances over time, top rewards)
- [ ] Recent activity list
- [ ] API integration

**Customers Page**:
- [ ] Customer list table with pagination
- [ ] Search functionality (phone, external_ref)
- [ ] Create customer form
- [ ] Customer detail view
- [ ] Issuance history
- [ ] Consent management

**Rewards Page**:
- [ ] Reward catalog table
- [ ] Create/edit reward form
- [ ] Reward type selector
- [ ] CSV voucher code upload
- [ ] Activate/deactivate rewards
- [ ] Inventory management

**Rules Page**:
- [ ] Rules list table
- [ ] Create/edit rule form
- [ ] Rule builder component (visual JsonLogic)
- [ ] Rule simulator component
- [ ] Cap configuration (per-user, global, cooldown)
- [ ] Activate/deactivate rules

**Campaigns Page**:
- [ ] Campaign list table
- [ ] Create/edit campaign form
- [ ] Date range picker
- [ ] Budget assignment
- [ ] Campaign performance metrics

**Budgets Page**:
- [ ] Budget list table
- [ ] Create budget form
- [ ] Topup form
- [ ] Ledger entries table
- [ ] Utilization chart
- [ ] Balance display

**Forms & Validation**:
- [ ] React Hook Form integration
- [ ] Zod schema validation
- [ ] Error handling
- [ ] Success notifications (toast)

**Theming**:
- [ ] Tenant theme application
- [ ] CSS variable system
- [ ] Logo/color customization

---

## Phase 4: Quality & Integration (Weeks 7-8)

### Testing

**Test Infrastructure**:
- [ ] Test database setup (api/internal/testutil/db.go)
- [ ] Test fixtures (api/internal/testutil/fixtures.go)
- [ ] CI pipeline configuration (.github/workflows/test.yml)

**Unit Tests**:
- [ ] Rules engine tests (>80% coverage)
- [ ] Budget service tests (>80% coverage)
- [ ] Reward service tests (>80% coverage)
- [ ] Auth tests
- [ ] Middleware tests
- [ ] Handler tests

**Integration Tests**:
- [ ] Event ingestion end-to-end
- [ ] Idempotency tests
- [ ] Tenant isolation tests
- [ ] RLS policy tests

**Performance Tests**:
- [ ] Event ingestion benchmark (p95 <150ms)
- [ ] Rule evaluation benchmark (<25ms)
- [ ] Concurrent load test (100 RPS)

**API Tests**:
- [ ] Customers API tests
- [ ] Events API tests
- [ ] Rules API tests
- [ ] Rewards API tests
- [ ] Issuances API tests
- [ ] Budgets API tests
- [ ] Auth tests (401, 403)
- [ ] Cross-tenant access tests

**Frontend Tests**:
- [ ] Component unit tests
- [ ] Form validation tests
- [ ] API client tests
- [ ] E2E tests (Playwright)

**Test Coverage**:
- [ ] Backend coverage >80%
- [ ] Frontend coverage >70%
- [ ] Coverage reports generated

### External Integrations

**Connector Interface** (api/internal/connectors/interface.go):
- [ ] Connector interface defined
- [ ] IssueParams struct
- [ ] IssueResponse struct
- [ ] StatusResponse struct

**Airtime Provider** (api/internal/connectors/airtime/provider.go):
- [ ] Provider implementation
- [ ] IssueVoucher function
- [ ] CheckStatus function
- [ ] HMAC signing
- [ ] Retry logic
- [ ] Tests

**Connector Registry** (api/internal/connectors/registry.go):
- [ ] Registry implementation
- [ ] Register function
- [ ] Get function

**Webhook Delivery** (api/internal/webhooks/delivery.go):
- [ ] DeliveryService implementation
- [ ] Worker pool
- [ ] HMAC signature generation
- [ ] Retry logic
- [ ] Event subscription filtering

**Webhook Events** (api/internal/webhooks/events.go):
- [ ] customer.enrolled event
- [ ] reward.issued event
- [ ] reward.redeemed event
- [ ] reward.expired event
- [ ] budget.threshold event

**Circuit Breaker** (api/internal/connectors/circuitbreaker.go):
- [ ] CircuitBreaker implementation
- [ ] State management (closed, open, half-open)
- [ ] Failure counting
- [ ] Timeout handling

**Configuration** (api/internal/connectors/config.go):
- [ ] Config struct
- [ ] Load from environment

**Tests**:
- [ ] Airtime provider tests
- [ ] Webhook delivery tests
- [ ] Circuit breaker tests
- [ ] Retry logic tests

### DevOps & Production

**Docker Optimization**:
- [ ] Multi-stage API Dockerfile
- [ ] Multi-stage Web Dockerfile
- [ ] .dockerignore files
- [ ] Image size optimization

**Production Setup**:
- [ ] docker-compose.prod.yml
- [ ] Caddyfile.prod (with HTTPS)
- [ ] Environment configuration (.env.prod)
- [ ] Health checks configured
- [ ] Restart policies

**Logging**:
- [ ] Structured logging (slog)
- [ ] Log levels (debug, info, warn, error)
- [ ] Request logging middleware
- [ ] JSON log format

**Monitoring**:
- [ ] Metrics middleware
- [ ] Health check endpoint
- [ ] Ready check endpoint
- [ ] Prometheus metrics (optional)
- [ ] Grafana dashboard (optional)

**Backup & Restore**:
- [ ] Backup script (scripts/backup.sh)
- [ ] Restore script (scripts/restore.sh)
- [ ] Automated daily backups
- [ ] S3 upload (optional)
- [ ] 30-day retention

**Deployment**:
- [ ] Deployment script (scripts/deploy.sh)
- [ ] Rolling restart strategy
- [ ] Migration execution
- [ ] Zero-downtime deployment

**Database Maintenance**:
- [ ] Maintenance script (scripts/db-maintenance.sh)
- [ ] VACUUM ANALYZE
- [ ] REINDEX
- [ ] Bloat checking

**Cron Jobs**:
- [ ] Daily backups (2 AM)
- [ ] Weekly DB maintenance (Sunday 3 AM)
- [ ] Hourly reward expiry
- [ ] Monthly budget reset (1st at midnight)

**Security**:
- [ ] Security headers (Caddy)
- [ ] Rate limiting
- [ ] HTTPS enforced
- [ ] Non-root container users
- [ ] Secret management

---

## Documentation

- [ ] API documentation (OpenAPI/Swagger)
- [ ] Database schema documentation
- [ ] Deployment guide
- [ ] Development setup guide
- [ ] Agent implementation guides (complete)
- [ ] User guide for merchant console
- [ ] WhatsApp setup guide
- [ ] USSD setup guide

---

## Performance Targets

- [ ] Event ingestion p95 < 150ms
- [ ] Rule evaluation < 25ms per event
- [ ] API sustains 100 RPS
- [ ] Database query optimization
- [ ] Connection pool tuning

---

## Security & Compliance

- [ ] JWT tokens secure (short expiry)
- [ ] HMAC signatures verified
- [ ] Passwords bcrypt hashed
- [ ] RLS policies enforced
- [ ] HTTPS enabled in production
- [ ] Rate limiting configured
- [ ] Security headers set
- [ ] Consent recording implemented
- [ ] Audit logging complete
- [ ] DPIA documentation (per tenant)
- [ ] Data export functionality
- [ ] Data deletion functionality

---

## Completion Milestones

### Alpha Release (End of Phase 2)
- [ ] All core features working
- [ ] Basic testing complete
- [ ] Can process events and issue rewards
- [ ] Merchant console functional

### Beta Release (End of Phase 3)
- [ ] WhatsApp integration working
- [ ] All pages complete
- [ ] Integration tests passing
- [ ] Ready for pilot testing

### Production Release (End of Phase 4)
- [ ] All tests passing (>80% coverage)
- [ ] Performance targets met
- [ ] Documentation complete
- [ ] Production deployment successful
- [ ] Monitoring and alerting active
- [ ] Backup/restore verified

---

## Notes

- Update this document daily during active development
- Mark items complete with [x] as they are finished
- Add new items as requirements emerge
- Track blockers and dependencies in comments
- Review progress weekly in team meetings

**Last Updated**: 2025-11-14
**Updated By**: Initial Setup
