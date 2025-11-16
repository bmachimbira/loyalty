# Build Agents

This document describes the AI agents available to help build and maintain the Zimbabwe White-Label Loyalty Platform.

## Available Agents

### 1. Backend API Agent (`agents/backend-api.md`)
**Purpose**: Implements Go API endpoints, middleware, and business logic.

**Responsibilities**:
- Create REST API endpoints following the spec
- Implement authentication and authorization (JWT, HMAC)
- Set up tenant context middleware (RLS)
- Create request validators and error handlers
- Implement idempotency handling

**Skills**: Go, gin framework, pgx/v5, PostgreSQL, REST API design

### 2. Database Agent (`agents/database.md`)
**Purpose**: Manages database schema, migrations, and sqlc queries.

**Responsibilities**:
- Create and modify database migrations
- Write sqlc queries for data access
- Generate sqlc code
- Optimize database indexes
- Implement database testing utilities

**Skills**: PostgreSQL, SQL, sqlc, database design, RLS

### 3. Frontend Agent (`agents/frontend.md`)
**Purpose**: Builds React components and pages for the merchant console.

**Responsibilities**:
- Implement shadcn/ui components
- Create pages (Dashboard, Customers, Rewards, Rules, Campaigns, Budgets)
- Build forms and validation
- Implement state management
- Create API client utilities

**Skills**: React, TypeScript, Tailwind CSS, shadcn/ui, Vite

### 4. Rules Engine Agent (`agents/rules-engine.md`)
**Purpose**: Implements the loyalty rules evaluation engine.

**Responsibilities**:
- Implement JsonLogic parser and evaluator
- Create rule evaluation functions
- Implement cap enforcement (per-user, global, cooldown)
- Build rule testing utilities
- Optimize rule matching performance

**Skills**: Go, JsonLogic, concurrency, testing

### 5. Rewards Agent (`agents/rewards.md`)
**Purpose**: Implements the reward issuance and state machine.

**Responsibilities**:
- Implement reward state machine (reserved → issued → redeemed)
- Create reward type handlers (discount, voucher, external, etc.)
- Integrate with external suppliers/connectors
- Implement expiry and cancellation logic
- Build ledger integration

**Skills**: Go, state machines, external APIs, PostgreSQL

### 6. Budget & Ledger Agent (`agents/budget-ledger.md`)
**Purpose**: Manages budget tracking and ledger entries.

**Responsibilities**:
- Implement budget reservation and charging
- Create ledger entry functions (reserve, charge, release, reverse)
- Build budget monitoring and alerts
- Implement currency handling (ZWG, USD)
- Create budget reporting utilities

**Skills**: Go, PostgreSQL, financial calculations, concurrency

### 7. Channels Agent (`agents/channels.md`)
**Purpose**: Implements communication channels (WhatsApp, USSD).

**Responsibilities**:
- Implement WhatsApp webhook handlers
- Create message template system
- Build USSD menu navigation
- Implement session management
- Create customer enrollment flows

**Skills**: Go, WhatsApp Business API, USSD protocols, state management

### 8. Testing Agent (`agents/testing.md`)
**Purpose**: Creates comprehensive tests for all components.

**Responsibilities**:
- Write unit tests for all packages
- Create integration tests
- Build test fixtures and helpers
- Implement RLS policy tests
- Create load/performance tests

**Skills**: Go testing, testify, database testing, HTTP testing

### 9. Integration Agent (`agents/integration.md`)
**Purpose**: Sets up external service integrations.

**Responsibilities**:
- Implement airtime/data provider connectors
- Create webhook delivery system
- Build retry and error handling
- Implement HMAC signing/verification
- Create integration tests

**Skills**: Go, HTTP clients, external APIs, error handling

### 10. DevOps Agent (`agents/devops.md`)
**Purpose**: Manages deployment, monitoring, and infrastructure.

**Responsibilities**:
- Optimize Docker builds
- Set up logging and monitoring
- Create backup/restore procedures
- Implement health checks
- Configure production deployments

**Skills**: Docker, Caddy, PostgreSQL administration, monitoring

## Agent Workflow

### Phase 1: Foundation (Weeks 1-2)
1. **Database Agent**: Complete schema and core queries
2. **Backend API Agent**: Set up auth, middleware, base endpoints
3. **Frontend Agent**: Create layout and navigation

### Phase 2: Core Features (Weeks 3-4)
4. **Rules Engine Agent**: Implement rule evaluation
5. **Rewards Agent**: Implement state machine and basic reward types
6. **Budget & Ledger Agent**: Implement budget tracking

### Phase 3: Channels (Weeks 5-6)
7. **Channels Agent**: Implement WhatsApp integration
8. **Frontend Agent**: Complete merchant console pages

### Phase 4: Quality & Integration (Weeks 7-8)
9. **Testing Agent**: Comprehensive test suite
10. **Integration Agent**: External connectors
11. **DevOps Agent**: Production readiness

## Using the Agents

Each agent has detailed instructions in their respective file. To use an agent:

```bash
# Example: Run the Backend API Agent
cat agents/backend-api.md

# Follow the instructions to implement specific features
```

## Agent Communication

Agents coordinate through:
- **Database schema**: Single source of truth
- **API contracts**: REST endpoints follow spec
- **Type definitions**: sqlc generates shared types
- **Documentation**: Each agent updates relevant docs

## Success Criteria

Each agent is considered complete when:
- ✅ All assigned features are implemented
- ✅ Unit tests pass with >80% coverage
- ✅ Integration tests pass
- ✅ Documentation is updated
- ✅ Code review is complete
- ✅ Performance targets are met
