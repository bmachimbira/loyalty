# Zimbabwe‑First White‑Label Loyalty Platform  
**Technical Specification (Go + Postgres + sqlc + gin • React + Vite • TypeScript • shadcn/ui • Caddy + Docker Compose)**  
**Version:** 1.0 • **Date:** 14 Nov 2025

---

## Table of Contents
1. [Purpose & Scope](#purpose--scope)  
2. [Zimbabwe‑Specific Anchors](#zimbabwe-specific-anchors)  
3. [Personas & Roles](#personas--roles)  
4. [High‑Level Architecture](#high-level-architecture)  
5. [Non‑Functional Requirements](#non-functional-requirements)  
6. [Data Model](#data-model)  
   - [ER Overview](#er-overview)  
   - [PostgreSQL DDL (excerpts)](#postgresql-ddl-excerpts)  
   - [Row‑Level Security (RLS)](#row-level-security-rls)  
7. [Rules Engine](#rules-engine)  
8. [Rewards, Budgets & Ledger](#rewards-budgets--ledger)  
9. [Channels](#channels)  
   - [WhatsApp (MVP)](#whatsapp-mvp)  
   - [USSD (Phase 2)](#ussd-phase-2)  
10. [API Design (gin)](#api-design-gin)  
11. [sqlc Setup & Queries](#sqlc-setup--queries)  
12. [Merchant Console (React + shadcn/ui)](#merchant-console-react--shadcnui)  
13. [Security, Privacy & Compliance](#security-privacy--compliance)  
14. [Fraud & Abuse Controls](#fraud--abuse-controls)  
15. [DevOps: Caddy + Docker Compose](#devops-caddy--docker-compose)  
16. [External Reward Connectors](#external-reward-connectors)  
17. [Example Flows](#example-flows)  
18. [Testing & Analytics](#testing--analytics)  
19. [Roadmap (Post‑MVP)](#roadmap-post-mvp)  
20. [Implementation Checklists](#implementation-checklists)  
21. [Appendices](#appendices)

---

## Purpose & Scope
**Goal:** Launch a **multi‑tenant (white‑label)** loyalty platform in **Zimbabwe** where client companies configure **actions/thresholds** and issue **their own rewards** (not limited to airtime/data).  
**Channels:** WhatsApp first; Merchant web console; USSD later.  
**Non‑Goals (MVP):** Cash‑out, P2P transfer, holding consumer funds, cross‑merchant earn‑and‑burn.

---

## Zimbabwe‑Specific Anchors
- **Data protection:** Align to Zimbabwe’s Cyber & Data Protection regime: purpose limitation, consent logging, breach notification workflows.  
- **Currency:** Support **ZWG (ZiG)** and **USD** from day one. Tenants set a default currency; rewards may be denominated per‑item.  
- **Channels:** High WhatsApp usage; USSD remains important for reach. Design for WhatsApp‑first, USSD optional.  
- **Payments perimeter:** Keep rewards **non‑cashlike** and **merchant‑funded** (no cash‑out or stored value) to avoid e‑money/payment service licensing scope.

---

## Personas & Roles
- **Tenant Admin (owner/ops):** Configure catalog, rules, budgets; view analytics.  
- **Store Staff (cashier):** Look up customers, trigger manual events, redeem.  
- **Customer (end‑user):** Enrol via WhatsApp; view and redeem rewards.  
- **Platform Ops (you):** Manage connectors/suppliers, monitor abuse, support.

---

## High‑Level Architecture
**Pattern:** Modular monolith in Go.

```
[Caddy] ──► [gin API]
             ├─ Auth & RBAC
             ├─ Event Ingestion + Idempotency
             ├─ Rules Engine (Go lib)
             ├─ Reward Service (state machine)
             ├─ Budget/Ledger
             ├─ Channels: WhatsApp (webhooks), USSD (phase 2)
             ├─ Webhooks (outbound to tenants)
             └─ Worker jobs (Postgres-based queue)

[Postgres 15/16]  ← sqlc generated code (pgx/v5)
[React + Vite + TS + shadcn/ui]  (merchant console)
```

**Key choices**
- Single DB with **RLS** per tenant.  
- **Idempotent** event ingestion.  
- Pure SQL (via **sqlc**) for data access; no ORM.  
- Background jobs via Postgres (no Redis in MVP).  

---

## Non‑Functional Requirements
- **Performance targets:**  
  - p95 event ingestion < **150 ms**  
  - rules evaluation per event < **25 ms**  
  - sustain **100 RPS** on a single API node for event ingestion
- **Availability:** Single region; daily backups; point‑in‑time recovery.  
- **Observability:** Structured logs, request IDs, basic metrics; trace IDs propagated.  
- **Security:** JWT for staff; HMAC for server‑to‑server; least‑privilege DB roles.  

---

## Data Model

### ER Overview
- **Tenancy & Identity:** `tenants`, `staff_users`, `customers`, `consents`  
- **Catalog & Rules:** `reward_catalog`, `campaigns`, `rules`, `events`, `issuances`  
- **Money Control:** `budgets`, `ledger_entries` (per currency)  
- **Channels & I/O:** `wa_sessions`, `ussd_sessions`, `webhooks`  
- **Audit:** `audit_logs`

### PostgreSQL DDL (excerpts)
```sql
-- Tenancy
CREATE TABLE tenants (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name          text NOT NULL,
  country_code  text NOT NULL DEFAULT 'ZW',
  default_ccy   text NOT NULL CHECK (default_ccy IN ('ZWG','USD')),
  theme         jsonb NOT NULL DEFAULT '{}',
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE staff_users (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL REFERENCES tenants(id),
  email        citext NOT NULL,
  full_name    text NOT NULL,
  role         text NOT NULL CHECK (role IN ('owner','admin','staff','viewer')),
  pwd_hash     text NOT NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, email)
);

-- Customers & consent
CREATE TABLE customers (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id),
  phone_e164    text,
  external_ref  text,
  status        text NOT NULL DEFAULT 'active',
  created_at    timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, phone_e164),
  UNIQUE (tenant_id, external_ref)
);

CREATE TABLE consents (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL REFERENCES tenants(id),
  customer_id  uuid NOT NULL REFERENCES customers(id),
  channel      text NOT NULL CHECK (channel IN ('whatsapp','sms','email','web')),
  purpose      text NOT NULL, -- 'marketing','service','loyalty'
  granted      boolean NOT NULL,
  occurred_at  timestamptz NOT NULL DEFAULT now()
);

-- Budgets & ledger
CREATE TABLE budgets (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL REFERENCES tenants(id),
  name         text NOT NULL,
  currency     text NOT NULL CHECK (currency IN ('ZWG','USD')),
  soft_cap     numeric(18,2) NOT NULL DEFAULT 0,
  hard_cap     numeric(18,2) NOT NULL DEFAULT 0,
  balance      numeric(18,2) NOT NULL DEFAULT 0,
  period       text NOT NULL DEFAULT 'rolling',   -- 'monthly','rolling'
  created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE ledger_entries (
  id           bigserial PRIMARY KEY,
  tenant_id    uuid NOT NULL REFERENCES tenants(id),
  budget_id    uuid NOT NULL REFERENCES budgets(id),
  entry_type   text NOT NULL CHECK (entry_type IN
                 ('fund','reserve','release','charge','expire','reverse')),
  currency     text NOT NULL CHECK (currency IN ('ZWG','USD')),
  amount       numeric(18,2) NOT NULL,
  ref_type     text,     -- 'issuance','topup'
  ref_id       uuid,
  created_at   timestamptz NOT NULL DEFAULT now()
);

-- Catalog & rules
CREATE TABLE reward_catalog (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id),
  name          text NOT NULL,
  type          text NOT NULL CHECK (type IN
    ('discount','voucher_code','points_credit','external_voucher','physical_item','webhook_custom')),
  face_value    numeric(18,2),
  currency      text CHECK (currency IN ('ZWG','USD')),
  inventory     text NOT NULL CHECK (inventory IN ('none','pool','jit_external')),
  supplier_id   uuid,
  metadata      jsonb NOT NULL DEFAULT '{}',
  active        boolean NOT NULL DEFAULT true,
  UNIQUE (tenant_id, name)
);

CREATE TABLE campaigns (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id),
  name          text NOT NULL,
  start_at      timestamptz,
  end_at        timestamptz,
  budget_id     uuid REFERENCES budgets(id),
  status        text NOT NULL DEFAULT 'active'
);

CREATE TABLE rules (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL REFERENCES tenants(id),
  campaign_id   uuid REFERENCES campaigns(id),
  name          text NOT NULL,
  event_type    text NOT NULL,   -- 'purchase','visit','referral','survey','manual'
  conditions    jsonb NOT NULL,  -- restricted JsonLogic/DSL
  reward_id     uuid NOT NULL REFERENCES reward_catalog(id),
  per_user_cap  int NOT NULL DEFAULT 1,
  global_cap    int,
  cool_down_sec int NOT NULL DEFAULT 0,
  active        boolean NOT NULL DEFAULT true
);

-- Events & issuances
CREATE TABLE events (
  id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id        uuid NOT NULL REFERENCES tenants(id),
  customer_id      uuid REFERENCES customers(id),
  event_type       text NOT NULL,
  properties       jsonb NOT NULL DEFAULT '{}',
  occurred_at      timestamptz NOT NULL,
  source           text NOT NULL,        -- 'pos','api','whatsapp','ussd','console'
  location_id      uuid,
  idempotency_key  text NOT NULL,
  created_at       timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, idempotency_key)
);

CREATE TABLE issuances (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id      uuid NOT NULL REFERENCES tenants(id),
  customer_id    uuid NOT NULL REFERENCES customers(id),
  campaign_id    uuid REFERENCES campaigns(id),
  reward_id      uuid NOT NULL REFERENCES reward_catalog(id),
  status         text NOT NULL CHECK (status IN ('reserved','issued','redeemed','expired','cancelled','failed')),
  code           text,                 -- voucher/otp
  external_ref   text,                 -- supplier reference
  currency       text CHECK (currency IN ('ZWG','USD')),
  cost_amount    numeric(18,2),
  face_amount    numeric(18,2),
  issued_at      timestamptz,
  expires_at     timestamptz,
  redeemed_at    timestamptz,
  event_id       uuid REFERENCES events(id)
);
```

**Indexes (suggested)**
```sql
CREATE INDEX ON events (tenant_id, event_type, occurred_at DESC);
CREATE INDEX ON rules (tenant_id, event_type) WHERE active = true;
CREATE INDEX ON issuances (tenant_id, customer_id, status);
CREATE INDEX ON ledger_entries (tenant_id, budget_id, created_at DESC);
```

### Row‑Level Security (RLS)
- Every tenant‑scoped table has **RLS enabled** and **FORCE RLS** (where appropriate).  
- App sets connection context: `SET app.tenant_id = '<uuid>'` per request.

```sql
ALTER TABLE customers ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_customers
  ON customers
  USING (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);
ALTER TABLE customers FORCE ROW LEVEL SECURITY;
-- Repeat for tenant-scoped tables
```

---

## Rules Engine
**Flow:** Event → Match Conditions → Enforce Caps → Reserve Budget → Issue Reward

**Condition model (restricted JsonLogic):**
Supported ops: `==`, `>=`, `<=`, `in`, `all`/`any`, `within_days`, `nth_event_in_period`, `distinct_visit_days`.

```json
{
  "all": [
    {"==": [{"var":"event_type"}, "purchase"]},
    {">=": [{"var":"properties.amount"}, 20.00]},
    {"==": [{"var":"properties.currency"}, "ZWG"]},
    {"within_days": [{"var":"occurred_at"}, 30]}
  ]
}
```

**Concurrency & safety**
- Pre‑filter by SQL (cheap predicates) then evaluate JSON in Go (pure functions).  
- Use **advisory locks** on `(tenant_id, rule_id, customer_id)` during issuance.  
- **Caps & cooldowns:** `per_user_cap`, `global_cap`, `cool_down_sec`.

**Performance**
- Cache active rules per `(tenant_id, event_type)` in‑process with short TTL.  
- Keep evaluators allocation‑free hot paths.

---

## Rewards, Budgets & Ledger
**Reward Types**
- `discount` (single‑use code; % or amount; POS validation)  
- `voucher_code` (from preloaded code pool)  
- `external_voucher` (JIT via supplier; store `external_ref`)  
- `points_credit` (points ledger; redeem via catalog later)  
- `physical_item` (manual fulfilment; claim token)  
- `webhook_custom` (POST to tenant endpoint for bespoke fulfilment)

**State Machine**
```
reserved -> issued -> redeemed
     \-> cancelled
     \-> expired
     \-> failed
```

**Ledger Semantics**
- **reserve**: create `issuances(reserved)` and `ledger_entries(reserve)`; ensure `balance` and `hard_cap` allow.  
- **issued**: operational state change; no ledger movement (cost reserved).  
- **redeemed**: `ledger charge` consumes reservation permanently.  
- **expired/cancelled**: `ledger release` returns funds to budget.  
- **reverse**: admin remediation entry (audited).

**Budget Controls**
- Per‑campaign budget via `campaigns.budget_id`.  
- Per‑currency budgets (ZWG/USD).  
- Soft cap (alerts) + hard cap (enforced).

---

## Channels

### WhatsApp (MVP)
- Use WhatsApp Business Platform (Cloud API).  
- Endpoints:
  - `GET /public/wa/webhook` — validation challenge  
  - `POST /public/wa/webhook` — inbound messages  
- Flows:
  - **Enroll** (capture consent)  
  - **My rewards** (list/CTA)  
  - **Referral** (unique link/code)  
- Message templates (examples): `LOYALTY_WELCOME`, `REWARD_ISSUED`, `REWARD_REMINDER`.  
- Policy: keep messaging in **service/loyalty** context.

### USSD (Phase 2)
- Endpoint: `POST /public/ussd/callback`  
- Stateless menus; session state persists in `ussd_sessions`.  
- Procure short code via local integrator or MNO agreements.

---

## API Design (gin)

### Authentication
- **Staff/Admin:** JWT (short‑lived) + refresh; roles: `owner|admin|staff|viewer`.  
- **Server‑to‑server (POS/ERP):** HMAC API keys with `X-Key`, `X-Timestamp`, `X-Signature`.  
- Middleware sets `app.tenant_id` on DB connection per request.

### Idempotency & Signatures
- `Idempotency-Key` header on mutating endpoints (e.g., `/events`).  
- Verify HMAC for external system posts.

### Core Endpoints (selected)
```
POST   /v1/tenants/:tid/customers
GET    /v1/tenants/:tid/customers/:id

POST   /v1/tenants/:tid/events                  # ingests events (Idempotency-Key)
POST   /v1/tenants/:tid/rules
GET    /v1/tenants/:tid/rules
PATCH  /v1/tenants/:tid/rules/:id

POST   /v1/tenants/:tid/reward-catalog
GET    /v1/tenants/:tid/reward-catalog

GET    /v1/tenants/:tid/issuances
POST   /v1/tenants/:tid/issuances/:id/redeem    # OTP or staff PIN

POST   /v1/tenants/:tid/budgets
POST   /v1/tenants/:tid/budgets/:id/topup
GET    /v1/tenants/:tid/ledger?from=..&to=..

POST   /v1/tenants/:tid/webhooks                # register outbound webhooks
```

### Example Payloads
**Event ingestion**
```json
{
  "event_type": "purchase",
  "customer": {"phone_e164":"+26377XXXXXXX"},
  "occurred_at": "2025-11-14T11:25:11Z",
  "properties": {
    "amount": 28.50,
    "currency": "ZWG",
    "receipt_id": "R-9231",
    "items": [{"sku":"MILK","qty":2}]
  },
  "source": "pos",
  "location_id": "..."
}
```

**Rule: amount threshold → external voucher**
```json
{
  "name": "Spend >= ZWG 20 → 200MB bundle",
  "event_type": "purchase",
  "conditions": {
    "all": [
      {">=":[{"var":"properties.amount"},20]},
      {"==":[{"var":"properties.currency"},"ZWG"]}
    ]
  },
  "reward_id": "<uuid>",
  "per_user_cap": 1,
  "global_cap": 5000,
  "cool_down_sec": 86400,
  "campaign_id": "<uuid>"
}
```

---

## sqlc Setup & Queries

**Project layout**
```
/api
  /cmd/api
  /internal
    /db              # sqlc generated (pgx/v5)
    /rules
    /reward
    /budget
    /auth
    /channels
    /http
/migrations
/queries
/web
```

**sqlc.yaml**
```yaml
version: "2"
sql:
  - schema: "./migrations"
    queries: "./queries"
    engine: "postgresql"
    gen:
      go:
        package: "db"
        out: "./api/internal/db"
        sql_package: "pgx/v5"
```

**Query examples**

`queries/events.sql`
```sql
-- name: InsertEvent :one
INSERT INTO events (tenant_id, customer_id, event_type, properties, occurred_at, source, idempotency_key)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetEventByIdemKey :one
SELECT * FROM events WHERE tenant_id = $1 AND idempotency_key = $2;

-- name: GetActiveRulesForEvent :many
SELECT * FROM rules
WHERE tenant_id = $1 AND event_type = $2 AND active = true;
```

`queries/issuance.sql`
```sql
-- name: ReserveIssuance :one
INSERT INTO issuances (tenant_id, customer_id, campaign_id, reward_id, status, currency, face_amount, cost_amount, issued_at)
VALUES ($1,$2,$3,$4,'reserved',$5,$6,$7,now())
RETURNING *;

-- name: UpdateIssuanceStatus :exec
UPDATE issuances
SET status=$4,
    redeemed_at = CASE WHEN $4='redeemed' THEN now() END
WHERE id=$1 AND tenant_id=$2 AND status=$3;
```

---

## Merchant Console (React + shadcn/ui)

**Pages**
- **Dashboard:** active customers, events/day, issued vs redeemed, budget burn.  
- **Customers:** search (phone/external_ref); detail with history & consents.  
- **Rewards:** catalog CRUD; voucher pool upload (CSV of codes).  
- **Rules:** visual builder for thresholds and windows; **simulator** (dry‑run JSON event).  
- **Campaigns & Budgets:** create/assign budgets; set soft/hard caps.  
- **Audit & Consents:** filter by actor/time; export CSV.

**Theming & White‑label**
- Tenant theme JSON (logo/colors); applied via CSS variables.  
- Optional custom domains per tenant in Phase 2 (Caddy + TLS).

---

## Security, Privacy & Compliance
- **Data minimisation:** phone number (E.164) + optional name/external_ref only.  
- **Consent:** channel‑specific opt‑in captured and stored in `consents`.  
- **Rights:** export & delete endpoints per tenant; audit every admin action.  
- **Cross‑border:** if hosting outside ZW, provide DPAs and document processors.  
- **Scope control:** no P2P, no cash‑out, no holding customer funds in MVP.

---

## Fraud & Abuse Controls
- **Velocity limits:** per‑customer issuance/day (global + per rule).  
- **Receipt replay:** `idempotency_key = hash(tenant_id, receipt_id)`.  
- **Referral abuse:** same‑device heuristic, blacklist shared identifiers, enforce first‑purchase threshold for referee.  
- **Staff actions:** staff PIN for manual issuance; capture IP/location; strict audit.

---

## DevOps: Caddy + Docker Compose

**Caddyfile (example)**
```caddy
:80 {
  respond /health 200

  @api path /v1/* /public/*
  reverse_proxy @api api:8080

  handle_path / {
    root * /srv/web
    file_server
    try_files {path} /index.html
  }
}
```

**docker-compose.yml (excerpt)**

```yaml
services:
  db:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: postgres
    volumes:
      - dbdata:/var/lib/postgresql/data

  api:
    build: ../api
    environment:
      DATABASE_URL: postgres://postgres:postgres@db:5432/postgres?sslmode=disable
      JWT_SECRET: ${JWT_SECRET}
    depends_on: [ db ]
    ports: [ "8080:8080" ]

  web:
    build: ../web
    depends_on: [ api ]

  caddy:
    image: caddy:2
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - webdist:/srv/web
    depends_on: [ web, api ]
    ports: [ "80:80" ]

volumes:
  dbdata:
  webdist:
```

**Environment variables (suggested)**
```
DATABASE_URL=postgres://...
JWT_SECRET=change-me
HMAC_KEYS_JSON={"key_id":"base64secret"}
WHATSAPP_VERIFY_TOKEN=...
WHATSAPP_APP_ID=...
WHATSAPP_APP_SECRET=...
WHATSAPP_PHONE_ID=...
```

**Backups & PITR**
- Daily full backup; WAL archiving; restore runbook.

---

## External Reward Connectors
- **Airtime/Data:** local aggregator or MNO integrations (JIT issuance; store `external_ref`).  
- **Gift cards:** phase 2 via `jit_external`.  
- **Custom:** `webhook_custom` reward posts to tenant URL (with HMAC signature).

---

## Example Flows

### A) Purchase → Threshold → Airtime Voucher (JIT)
1. POS posts `purchase` event (amount=ZWG 28.50).  
2. Rules engine matches `>= ZWG 20` and passes caps.  
3. Budget **reserve**; call connector for bundle; issuance moves to **issued** with code/ref.  
4. WhatsApp template **REWARD_ISSUED** delivered.  
5. Voucher auto‑redeems (if airtime) or user enters code at POS (if discount).

### B) Visit Streak (3 in 30 days) → Discount Code
- Event type `visit` (no amount).  
- Rule `nth_event_in_period = 3`.  
- Issue `discount` code valid 7 days (`metadata.min_basket`, `valid_days`).

---

## Testing & Analytics
- **Unit tests:** rule evaluation; issuance lifecycle; ledger math; RLS policy tests.  
- **Integration tests:** WhatsApp webhook validation; connector sandbox; idempotency.  
- **Seed data:** one demo tenant; 100 customers; 3 rules; 2 rewards; 1 campaign.  
- **KPIs:** enrolment rate; WAU; issuance→redemption rate; cost per incremental visit; budget burn; fraud triggers.

---

## Roadmap (Post‑MVP)
- **USSD entry** (short code via local integrator).  
- **Supplier marketplace** shared across tenants (curated rewards).  
- **Points store** & cross‑merchant redemption (requires compliance review).  
- **Tenant custom domains** (Caddy + wildcard certs).  
- **WhatsApp Flows** for surveys/referrals.

---

## Implementation Checklists

**Backend**
- [ ] Schema + migrations  
- [ ] RLS policies + `SET app.tenant_id` middleware  
- [ ] sqlc codegen + repositories  
- [ ] Rules evaluator (restricted JsonLogic) + tests  
- [ ] Reward service (state machine) + ledger  
- [ ] WhatsApp webhooks + outbound send  
- [ ] HMAC API keys + idempotency store  
- [ ] Audit logging

**Frontend**
- [ ] shadcn/ui scaffolding, auth, RBAC  
- [ ] Reward catalog CRUD + voucher pool upload (CSV)  
- [ ] Rule builder + simulator  
- [ ] Campaign & budget management  
- [ ] Customers & consents  
- [ ] Analytics dashboard

**Ops**
- [ ] Caddy + Compose deployment  
- [ ] Backups & PITR  
- [ ] Logs/metrics/alerts  
- [ ] DPIA & privacy docs per tenant

---

## Appendices

### A. Minimal State Transitions
```
reserved -> issued -> redeemed
     \-> cancelled
     \-> expired
     \-> failed
```

### B. Discount Reward Metadata (example)
```json
{
  "discount_type": "amount",
  "amount": 5.00,
  "currency": "ZWG",
  "min_basket": 20.00,
  "valid_days": 7,
  "pos_validation": "code"
}
```

### C. Ledger Entry Sequence (example)
- `reserve` (ZWG 20.00) → temporary hold  
- `charge` at redemption → consume hold  
- `release` at expiry/cancel → return to balance

### D. Sample Event Payloads
See **API Design** section; include `Idempotency-Key` and (for server‑to‑server) HMAC headers.

---

**End of Document**
