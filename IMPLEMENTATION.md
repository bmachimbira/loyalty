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
- [ ] Go project initialized (go.mod)
- [ ] Main application entry point (cmd/api/main.go)
- [ ] Database connection pool setup
- [ ] Environment variable configuration
- [ ] Graceful shutdown implemented
- [ ] Project structure organized

### Backend - Authentication & Authorization

**JWT Authentication**:
- [ ] JWT token generation (api/internal/auth/jwt.go)
- [ ] JWT token validation
- [ ] Claims extraction (user_id, tenant_id, role)
- [ ] Access token (15 min expiry)
- [ ] Refresh token (7 day expiry)
- [ ] Refresh token flow

**HMAC Authentication**:
- [ ] HMAC signature generation (api/internal/auth/hmac.go)
- [ ] HMAC signature verification
- [ ] Timestamp validation (5 min window)
- [ ] Key management from HMAC_KEYS_JSON

**Password Handling**:
- [ ] bcrypt password hashing (api/internal/auth/password.go)
- [ ] Password comparison

### Backend - Middleware

**Tenant Context**:
- [ ] Extract tenant_id from JWT claims (api/internal/http/middleware/tenant.go)
- [ ] Set PostgreSQL app.tenant_id session variable
- [ ] Handle missing tenant_id errors
- [ ] Add tenant_id to request context

**Authentication**:
- [ ] RequireAuth middleware (api/internal/http/middleware/auth.go)
- [ ] RequireRole middleware
- [ ] RequireHMAC middleware for server-to-server

**Idempotency**:
- [ ] Idempotency-Key header handling (api/internal/http/middleware/idempotency.go)
- [ ] Check for duplicate keys in database
- [ ] Return cached responses for duplicates

**Other Middleware**:
- [ ] CORS middleware
- [ ] Request ID middleware
- [ ] Logging middleware
- [ ] Metrics middleware

### Backend - API Handlers

**Customers API** (api/internal/http/handlers/customers.go):
- [ ] POST /v1/tenants/:tid/customers (create)
- [ ] GET /v1/tenants/:tid/customers/:id (get)
- [ ] GET /v1/tenants/:tid/customers (list with pagination)
- [ ] PATCH /v1/tenants/:tid/customers/:id/status (update status)
- [ ] E.164 phone validation

**Events API** (api/internal/http/handlers/events.go):
- [ ] POST /v1/tenants/:tid/events (create with idempotency)
- [ ] GET /v1/tenants/:tid/events/:id (get)
- [ ] GET /v1/tenants/:tid/events (list)
- [ ] Trigger rules engine on event creation

**Rules API** (api/internal/http/handlers/rules.go):
- [ ] POST /v1/tenants/:tid/rules (create)
- [ ] GET /v1/tenants/:tid/rules (list)
- [ ] GET /v1/tenants/:tid/rules/:id (get)
- [ ] PATCH /v1/tenants/:tid/rules/:id (update)
- [ ] DELETE /v1/tenants/:tid/rules/:id (soft delete)
- [ ] JsonLogic condition validation

**Rewards API** (api/internal/http/handlers/rewards.go):
- [ ] POST /v1/tenants/:tid/reward-catalog (create)
- [ ] GET /v1/tenants/:tid/reward-catalog (list)
- [ ] GET /v1/tenants/:tid/reward-catalog/:id (get)
- [ ] PATCH /v1/tenants/:tid/reward-catalog/:id (update)
- [ ] POST /v1/tenants/:tid/reward-catalog/:id/upload-codes (CSV upload)

**Issuances API** (api/internal/http/handlers/issuances.go):
- [ ] GET /v1/tenants/:tid/issuances (list)
- [ ] GET /v1/tenants/:tid/issuances/:id (get)
- [ ] POST /v1/tenants/:tid/issuances/:id/redeem (redeem)
- [ ] POST /v1/tenants/:tid/issuances/:id/cancel (cancel)

**Budgets API** (api/internal/http/handlers/budgets.go):
- [ ] POST /v1/tenants/:tid/budgets (create)
- [ ] GET /v1/tenants/:tid/budgets (list)
- [ ] GET /v1/tenants/:tid/budgets/:id (get)
- [ ] POST /v1/tenants/:tid/budgets/:id/topup (topup)
- [ ] GET /v1/tenants/:tid/ledger (query ledger)

**Campaigns API** (api/internal/http/handlers/campaigns.go):
- [ ] POST /v1/tenants/:tid/campaigns (create)
- [ ] GET /v1/tenants/:tid/campaigns (list)
- [ ] PATCH /v1/tenants/:tid/campaigns/:id (update)

**Error Handling**:
- [ ] ErrorResponse type defined (api/internal/http/errors.go)
- [ ] Error codes implemented
- [ ] Consistent error responses

**Request Validation**:
- [ ] ValidateE164Phone (api/internal/http/validation.go)
- [ ] ValidateUUID
- [ ] ValidateCurrency
- [ ] ValidateEventType
- [ ] ValidateRewardType

**Router Setup**:
- [ ] SetupRouter function (api/internal/http/router.go)
- [ ] Health check endpoint
- [ ] Public routes group
- [ ] V1 API routes group with auth
- [ ] Update main.go to use router

### Frontend - Core Setup
- [ ] Vite + React + TypeScript project initialized
- [ ] Tailwind CSS configured
- [ ] shadcn/ui initialized
- [ ] All required shadcn/ui components installed
- [ ] TypeScript types defined (web/src/lib/types.ts)
- [ ] API client implemented (web/src/lib/api.ts)

### Frontend - Authentication
- [ ] AuthContext created (web/src/contexts/AuthContext.tsx)
- [ ] Login page
- [ ] Token management (localStorage)
- [ ] Protected routes
- [ ] Logout functionality

### Frontend - Layout
- [ ] Layout component (web/src/components/Layout.tsx)
- [ ] Navigation bar
- [ ] Routing setup (React Router)

### Frontend - Pages (Basic Structure)
- [ ] Dashboard page (web/src/pages/Dashboard.tsx)
- [ ] Customers page (web/src/pages/Customers.tsx)
- [ ] Rewards page (web/src/pages/Rewards.tsx)
- [ ] Rules page (web/src/pages/Rules.tsx)
- [ ] Campaigns page (web/src/pages/Campaigns.tsx)
- [ ] Budgets page (web/src/pages/Budgets.tsx)

---

## Phase 2: Core Features (Weeks 3-4)

### Rules Engine

**JsonLogic Evaluator** (api/internal/rules/jsonlogic.go):
- [ ] Parse JsonLogic conditions
- [ ] Comparison operators (==, !=, >, >=, <, <=)
- [ ] Logical operators (all, any, none)
- [ ] Array operators (in)
- [ ] Variable access (var)

**Custom Operators** (api/internal/rules/custom_operators.go):
- [ ] within_days operator
- [ ] nth_event_in_period operator
- [ ] distinct_visit_days operator

**Rules Engine** (api/internal/rules/engine.go):
- [ ] ProcessEvent function
- [ ] Get matching rules (with cache)
- [ ] Evaluate rule conditions
- [ ] Check caps and cooldowns
- [ ] Issue rewards
- [ ] Advisory locks for concurrency

**Cap Enforcement** (api/internal/rules/caps.go):
- [ ] Per-user cap checking
- [ ] Global cap checking
- [ ] Cooldown checking
- [ ] Query optimization for cap checks

**Reward Issuance** (api/internal/rules/issuance.go):
- [ ] Transaction-based issuance
- [ ] Advisory lock implementation
- [ ] Budget reservation
- [ ] State creation (reserved)
- [ ] Async processing trigger

**Rule Cache** (api/internal/rules/cache.go):
- [ ] Cache implementation
- [ ] Get/Set with TTL
- [ ] Cleanup expired entries

**Tests**:
- [ ] Simple condition tests
- [ ] Complex condition tests
- [ ] Custom operator tests
- [ ] Per-user cap tests
- [ ] Global cap tests
- [ ] Cooldown tests
- [ ] Concurrent event tests
- [ ] Performance benchmark (<25ms target)

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
- [ ] ReserveBudget function
- [ ] ChargeReservation function
- [ ] ReleaseReservation function
- [ ] Hard cap enforcement
- [ ] Soft cap alerting
- [ ] Currency validation

**Budget Topup** (api/internal/budget/topup.go):
- [ ] TopupBudget function
- [ ] Ledger entry creation

**Reconciliation** (api/internal/budget/reconciliation.go):
- [ ] ReconcileBudget function
- [ ] Balance verification
- [ ] Discrepancy detection

**Alerts** (api/internal/budget/alerts.go):
- [ ] Soft cap exceeded alert
- [ ] Hard cap approaching alert

**Period Budgets** (api/internal/budget/period.go):
- [ ] Monthly budget reset
- [ ] Cron job for reset

**Reports** (api/internal/budget/reports.go):
- [ ] GenerateReport function
- [ ] Aggregation logic

**Tests**:
- [ ] Reserve budget tests
- [ ] Hard cap tests
- [ ] Soft cap tests
- [ ] Charge/release tests
- [ ] Reconciliation tests
- [ ] Currency validation tests
- [ ] Concurrent reservation tests

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
