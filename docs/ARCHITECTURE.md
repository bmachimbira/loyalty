# System Architecture
# Zimbabwe Loyalty Platform

This document provides an overview of the system architecture, components, and technical design decisions.

## Table of Contents

- [System Overview](#system-overview)
- [Architecture Diagram](#architecture-diagram)
- [Components](#components)
- [Database Schema](#database-schema)
- [API Endpoints](#api-endpoints)
- [Security Model](#security-model)
- [Data Flow](#data-flow)
- [Integration Points](#integration-points)
- [Technology Stack](#technology-stack)
- [Design Decisions](#design-decisions)

## System Overview

The Zimbabwe Loyalty Platform is a multi-tenant loyalty management system designed for businesses in Zimbabwe. It supports:

- Multi-channel customer engagement (WhatsApp, USSD, Web)
- Flexible rules engine for reward issuance
- Real-time event processing
- Budget management and tracking
- Multiple reward types
- Webhook integrations

### Key Features

- **Multi-tenancy**: Isolated data per merchant
- **Rule-based rewards**: JsonLogic conditions with custom operators
- **Multiple channels**: WhatsApp, USSD, Web dashboard
- **Flexible rewards**: Discounts, vouchers, points, physical items
- **Budget controls**: Hard/soft caps, period-based budgets
- **Real-time processing**: < 150ms event processing
- **High performance**: 100+ RPS sustained
- **Security**: JWT auth, HMAC webhooks, RLS policies

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        Client Layer                              │
├─────────────┬──────────────┬─────────────┬─────────────────────┤
│  WhatsApp   │    USSD      │  Web App    │  External Systems   │
│  Messages   │   Sessions   │  (React)    │   (Webhooks)        │
└─────┬───────┴──────┬───────┴──────┬──────┴─────────┬───────────┘
      │              │              │                │
      │              │              │                │
┌─────▼──────────────▼──────────────▼────────────────▼───────────┐
│                     Caddy Reverse Proxy                         │
│  - HTTPS/TLS                                                    │
│  - Rate Limiting                                                │
│  - Security Headers                                             │
│  - Load Balancing                                               │
└──────────────────────────┬──────────────────────────────────────┘
                           │
      ┌────────────────────┼────────────────────┐
      │                    │                    │
┌─────▼────────┐  ┌────────▼────────┐  ┌───────▼────────┐
│  API Server  │  │  API Server     │  │  Web Server    │
│  (Instance 1)│  │  (Instance 2)   │  │  (nginx)       │
│              │  │                 │  │                │
│  - Go/Gin    │  │  - Go/Gin       │  │  - React SPA   │
│  - JWT Auth  │  │  - JWT Auth     │  │  - Static      │
│  - Logging   │  │  - Logging      │  │    Assets      │
└──────┬───────┘  └────────┬────────┘  └────────────────┘
       │                   │
       └───────┬───────────┘
               │
       ┌───────▼──────────┐
       │   PostgreSQL     │
       │   Database       │
       │                  │
       │  - Row-Level     │
       │    Security      │
       │  - Connection    │
       │    Pooling       │
       └──────────────────┘
```

## Components

### 1. Reverse Proxy (Caddy)

**Purpose**: Entry point for all HTTP traffic

**Responsibilities**:
- Automatic HTTPS via Let's Encrypt
- SSL/TLS termination
- Load balancing across API instances
- Rate limiting
- Security headers
- Access logging
- Gzip compression

**Configuration**: `Caddyfile.prod`

### 2. API Server (Go + Gin)

**Purpose**: Core application logic and API endpoints

**Responsibilities**:
- REST API endpoints
- Authentication/Authorization
- Request validation
- Business logic execution
- Rules engine
- Reward issuance
- Budget management
- Event processing
- Webhook delivery

**Technology**:
- Language: Go 1.21
- Framework: Gin
- Database: pgx v5
- Logging: slog (structured)

**Configuration**: Environment variables in `.env.prod`

### 3. Web Application (React)

**Purpose**: Merchant dashboard

**Responsibilities**:
- Merchant console UI
- Customer management
- Campaign configuration
- Rule builder
- Reward catalog
- Budget monitoring
- Analytics dashboards

**Technology**:
- React 18
- TypeScript
- Vite (build tool)
- Tailwind CSS
- shadcn/ui components

**Deployment**: Static files served by nginx

### 4. Database (PostgreSQL 16)

**Purpose**: Persistent data storage

**Responsibilities**:
- Multi-tenant data storage
- Row-Level Security (RLS)
- Transaction management
- Query optimization
- Data integrity

**Key Tables**:
- `tenants` - Merchant accounts
- `customers` - Enrolled customers
- `events` - Customer activity events
- `rules` - Reward rules
- `rewards` - Reward catalog
- `issuances` - Issued rewards
- `budgets` - Budget allocations
- `ledger` - Budget transactions

**Configuration**: Optimized for performance in `docker-compose.prod.yml`

### 5. Redis (Optional)

**Purpose**: Caching and rate limiting

**Responsibilities**:
- Session storage
- Rate limit counters
- Cache frequently accessed data
- Idempotency tracking

**Technology**: Redis 7 Alpine

## Database Schema

### Core Tables

#### Tenants
```sql
tenants (
  id UUID PRIMARY KEY,
  name VARCHAR NOT NULL,
  settings JSONB,
  created_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ
)
```

#### Customers
```sql
customers (
  id UUID PRIMARY KEY,
  tenant_id UUID REFERENCES tenants,
  phone VARCHAR NOT NULL,
  external_ref VARCHAR,
  status VARCHAR,
  created_at TIMESTAMPTZ,
  UNIQUE(tenant_id, phone)
)
```

#### Events
```sql
events (
  id UUID PRIMARY KEY,
  tenant_id UUID REFERENCES tenants,
  customer_id UUID REFERENCES customers,
  event_type VARCHAR NOT NULL,
  event_data JSONB,
  idempotency_key VARCHAR UNIQUE,
  created_at TIMESTAMPTZ
)
```

#### Rules
```sql
rules (
  id UUID PRIMARY KEY,
  tenant_id UUID REFERENCES tenants,
  name VARCHAR NOT NULL,
  description TEXT,
  conditions JSONB NOT NULL,
  reward_id UUID REFERENCES rewards,
  caps JSONB,
  priority INTEGER,
  active BOOLEAN,
  created_at TIMESTAMPTZ
)
```

#### Rewards
```sql
rewards (
  id UUID PRIMARY KEY,
  tenant_id UUID REFERENCES tenants,
  name VARCHAR NOT NULL,
  reward_type VARCHAR NOT NULL,
  config JSONB NOT NULL,
  active BOOLEAN,
  created_at TIMESTAMPTZ
)
```

#### Issuances
```sql
issuances (
  id UUID PRIMARY KEY,
  tenant_id UUID REFERENCES tenants,
  customer_id UUID REFERENCES customers,
  reward_id UUID REFERENCES rewards,
  state VARCHAR NOT NULL,
  delivery_method VARCHAR,
  delivery_data JSONB,
  expires_at TIMESTAMPTZ,
  redeemed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ
)
```

#### Budgets
```sql
budgets (
  id UUID PRIMARY KEY,
  tenant_id UUID REFERENCES tenants,
  name VARCHAR NOT NULL,
  total_amount DECIMAL NOT NULL,
  utilized_amount DECIMAL DEFAULT 0,
  currency VARCHAR DEFAULT 'USD',
  period VARCHAR,
  hard_cap BOOLEAN,
  soft_cap_threshold DECIMAL,
  created_at TIMESTAMPTZ
)
```

### Row-Level Security (RLS)

All tables use RLS policies to enforce tenant isolation:

```sql
-- Example RLS policy
CREATE POLICY tenant_isolation ON events
  USING (tenant_id::TEXT = current_setting('app.tenant_id', true));
```

The `app.tenant_id` session variable is set by middleware on each request.

## API Endpoints

### Authentication

```
POST /public/auth/login          - Login (get JWT)
POST /public/auth/refresh        - Refresh JWT token
```

### Customers

```
POST   /v1/tenants/:tid/customers           - Create customer
GET    /v1/tenants/:tid/customers/:id       - Get customer
GET    /v1/tenants/:tid/customers           - List customers
PATCH  /v1/tenants/:tid/customers/:id/status - Update status
```

### Events

```
POST   /v1/tenants/:tid/events              - Create event
GET    /v1/tenants/:tid/events/:id          - Get event
GET    /v1/tenants/:tid/events              - List events
```

### Rules

```
POST   /v1/tenants/:tid/rules               - Create rule
GET    /v1/tenants/:tid/rules/:id           - Get rule
GET    /v1/tenants/:tid/rules               - List rules
PATCH  /v1/tenants/:tid/rules/:id           - Update rule
DELETE /v1/tenants/:tid/rules/:id           - Delete rule
```

### Rewards

```
POST   /v1/tenants/:tid/reward-catalog      - Create reward
GET    /v1/tenants/:tid/reward-catalog/:id  - Get reward
GET    /v1/tenants/:tid/reward-catalog      - List rewards
PATCH  /v1/tenants/:tid/reward-catalog/:id  - Update reward
POST   /v1/tenants/:tid/reward-catalog/:id/upload-codes - Upload codes
```

### Issuances

```
GET    /v1/tenants/:tid/issuances           - List issuances
GET    /v1/tenants/:tid/issuances/:id       - Get issuance
POST   /v1/tenants/:tid/issuances/:id/redeem - Redeem reward
POST   /v1/tenants/:tid/issuances/:id/cancel - Cancel issuance
```

### Budgets

```
POST   /v1/tenants/:tid/budgets             - Create budget
GET    /v1/tenants/:tid/budgets/:id         - Get budget
GET    /v1/tenants/:tid/budgets             - List budgets
POST   /v1/tenants/:tid/budgets/:id/topup   - Top up budget
GET    /v1/tenants/:tid/ledger              - Query ledger
```

### Campaigns

```
POST   /v1/tenants/:tid/campaigns           - Create campaign
GET    /v1/tenants/:tid/campaigns           - List campaigns
PATCH  /v1/tenants/:tid/campaigns/:id       - Update campaign
```

### Channels

```
GET    /public/wa/webhook                   - WhatsApp verification
POST   /public/wa/webhook                   - WhatsApp messages
POST   /public/ussd/callback                - USSD sessions
```

### Health

```
GET    /health                              - Health check
GET    /ready                               - Readiness check
```

## Security Model

### Authentication

**JWT Tokens**:
- Access tokens: 15-minute expiry
- Refresh tokens: 7-day expiry
- Claims: `user_id`, `tenant_id`, `role`
- Algorithm: HS256

**HMAC Signatures**:
- Used for webhook verification
- Server-to-server authentication
- Key rotation supported via `HMAC_KEYS_JSON`

### Authorization

**Role-Based Access Control (RBAC)**:
- `admin` - Full access
- `manager` - Manage campaigns, rewards
- `viewer` - Read-only access

**Tenant Isolation**:
- Enforced via Row-Level Security (RLS)
- Session variable `app.tenant_id` set per request
- Automatic filtering of all queries

### Security Headers

Set by Caddy:
- `Strict-Transport-Security`
- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`
- `X-XSS-Protection`
- `Content-Security-Policy`
- `Referrer-Policy`

### Rate Limiting

Implemented at two levels:
1. **Caddy**: IP-based rate limiting (100 req/min)
2. **Application**: Token bucket algorithm for fine-grained control

## Data Flow

### Event Processing Flow

```
1. Customer performs action (purchase, visit, etc.)
   ↓
2. POS/System sends event to API
   POST /v1/tenants/:tid/events
   ↓
3. API validates request (auth, tenant, idempotency)
   ↓
4. Event saved to database
   ↓
5. Rules engine triggered
   ↓
6. Matching rules evaluated against event
   ↓
7. Caps and cooldowns checked
   ↓
8. Budget reservation attempted
   ↓
9. Reward issuance created (state: reserved)
   ↓
10. Async reward processing
   ↓
11. State transition: reserved → issued
   ↓
12. Delivery via channel (WhatsApp, SMS, email)
   ↓
13. Budget charged (reserved → utilized)
   ↓
14. Webhooks sent to external systems
```

### Reward Redemption Flow

```
1. Customer requests redemption (USSD, WhatsApp, Web)
   ↓
2. API validates redemption request
   ↓
3. Checks:
   - Reward not expired
   - Reward not already redeemed
   - OTP/code valid (if required)
   ↓
4. State transition: issued → redeemed
   ↓
5. Budget operation: utilized → spent
   ↓
6. Notification sent to customer
   ↓
7. Webhook sent to merchant
```

## Integration Points

### WhatsApp Business API

**Provider**: Meta (Facebook)

**Integration**:
- Webhook for incoming messages
- Send API for outgoing messages
- Template messages for notifications
- Signature verification (HMAC)

**Flow**:
1. Customer sends WhatsApp message
2. Meta forwards to `/public/wa/webhook`
3. API processes message (enrollment, balance, redeem)
4. API sends response via Meta Send API

### USSD Gateway

**Provider**: Econet, NetOne, Telecel

**Integration**:
- Session-based interaction
- Menu navigation
- Short code dial (*123#)

**Flow**:
1. Customer dials USSD code
2. Gateway posts to `/public/ussd/callback`
3. API returns menu response (CON/END)
4. Session state maintained in database

### External Systems (Webhooks)

**Outbound Webhooks**:
- `customer.enrolled` - New customer enrolled
- `reward.issued` - Reward issued
- `reward.redeemed` - Reward redeemed
- `reward.expired` - Reward expired
- `budget.threshold` - Budget threshold reached

**Webhook Security**:
- HMAC signature verification
- Retry logic (exponential backoff)
- Timeout handling
- Circuit breaker pattern

## Technology Stack

### Backend

- **Language**: Go 1.21
- **Framework**: Gin (HTTP router)
- **Database Driver**: pgx v5
- **Logging**: slog (structured)
- **Authentication**: JWT (golang-jwt)
- **Validation**: validator v10

### Frontend

- **Framework**: React 18
- **Language**: TypeScript
- **Build Tool**: Vite
- **Styling**: Tailwind CSS
- **UI Components**: shadcn/ui
- **State Management**: React Context
- **HTTP Client**: Fetch API
- **Routing**: React Router

### Database

- **RDBMS**: PostgreSQL 16
- **Connection Pooling**: pgxpool
- **Migrations**: SQL files
- **ORM**: None (raw SQL for performance)

### Infrastructure

- **Containerization**: Docker
- **Orchestration**: Docker Compose
- **Reverse Proxy**: Caddy 2
- **SSL/TLS**: Let's Encrypt (automatic)
- **Caching**: Redis 7
- **Monitoring**: Structured logs (JSON)

## Design Decisions

### Why Go?

- High performance (compiled, fast)
- Excellent concurrency model
- Strong typing
- Great HTTP libraries
- Low memory footprint
- Fast compilation
- Single binary deployment

### Why PostgreSQL?

- ACID compliance
- Row-Level Security (RLS) for multi-tenancy
- JSONB for flexible data structures
- Excellent performance
- Mature ecosystem
- Strong data integrity
- Full-text search capabilities

### Why Multi-Tenant Database?

**Pros**:
- Easier maintenance
- Cross-tenant analytics
- Cost-effective
- Simpler deployment

**Cons**:
- Careful isolation required
- Shared resource contention

**Mitigation**:
- Row-Level Security (RLS)
- Connection pooling
- Query optimization
- Tenant-specific indexes

### Why Docker?

- Consistent environments
- Easy deployment
- Resource isolation
- Scalability
- Portability
- Version control

### Why Caddy?

- Automatic HTTPS
- Simple configuration
- HTTP/2 and HTTP/3 support
- Built-in security features
- Excellent performance
- No complex setup

## Performance Targets

- **Event Processing**: p95 < 150ms
- **Rule Evaluation**: < 25ms per event
- **API Throughput**: 100+ RPS
- **Database Queries**: < 50ms p95
- **Uptime**: 99.9%

## Scalability Considerations

### Horizontal Scaling

- **API Servers**: Stateless, can scale to N instances
- **Database**: Read replicas, connection pooling
- **Caching**: Redis cluster for distributed cache

### Vertical Scaling

- **Database**: Increase CPU, RAM for complex queries
- **API Servers**: More memory for caching

### Database Optimization

- **Indexes**: On frequently queried columns
- **Partitioning**: Large tables by tenant_id or date
- **Connection Pooling**: pgxpool configuration
- **Query Optimization**: EXPLAIN ANALYZE, slow query log

### Future Considerations

- **Message Queue**: For async processing (RabbitMQ, Kafka)
- **Service Mesh**: For microservices (Istio, Linkerd)
- **Distributed Tracing**: For observability (Jaeger, Zipkin)
- **Time-Series DB**: For metrics (InfluxDB, TimescaleDB)

## Monitoring and Observability

### Structured Logging

All logs in JSON format with:
- `timestamp` - ISO8601 format
- `level` - debug, info, warn, error
- `message` - Log message
- `request_id` - Request correlation
- `tenant_id` - Tenant context
- `duration_ms` - Request duration

### Metrics

Key metrics tracked:
- HTTP request count/duration
- Database query count/duration
- Events processed
- Rules evaluated
- Rewards issued
- Budget utilization
- Error rates
- Cache hit rates

### Health Checks

- `/health` - Basic health (database ping)
- `/ready` - Readiness (all dependencies)

## Disaster Recovery

### Backups

- **Frequency**: Daily automated backups
- **Retention**: 30 days local, 90 days S3
- **Type**: Full database dump
- **Verification**: Automated integrity checks
- **Testing**: Monthly restore tests

### High Availability

- **Database**: Streaming replication (future)
- **API**: Multiple instances behind load balancer
- **Reverse Proxy**: Caddy with failover
- **Monitoring**: Automated health checks

### Recovery Time Objectives (RTO/RPO)

- **RTO**: < 1 hour
- **RPO**: < 24 hours (daily backups)

## Security Considerations

### Data at Rest

- Database encryption (PostgreSQL encryption)
- Backup encryption (GPG or S3 server-side)

### Data in Transit

- TLS 1.3 for all HTTP traffic
- Database connections with SSL

### Secrets Management

- Environment variables
- Docker secrets (production)
- Secrets rotation every 90 days

### Compliance

- GDPR considerations
- Data export/deletion functionality
- Audit logging
- Consent management
