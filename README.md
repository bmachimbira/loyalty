# Zimbabwe White-Label Loyalty Platform

A multi-tenant loyalty platform designed for Zimbabwe, supporting ZWG and USD currencies with WhatsApp and USSD channels.

## Tech Stack

- **Backend**: Go + PostgreSQL + sqlc + gin
- **Frontend**: React + Vite + TypeScript + shadcn/ui
- **Deployment**: Caddy + Docker Compose

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for local development)
- Node.js 18+ (for local development)
- sqlc (for code generation)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd loyalty
```

2. Copy environment file:
```bash
cp .env.example .env
```

3. Start the services:
```bash
make up
```

The application will be available at:
- Web UI: http://localhost
- API: http://localhost/v1
- Database: localhost:5432

### Development

For local development without Docker:

1. Start the database:
```bash
make dev
```

2. In one terminal, start the API:
```bash
cd api
go run cmd/api/main.go
```

3. In another terminal, start the web app:
```bash
cd web
npm run dev
```

## Available Commands

Run `make help` to see all available commands:

- `make install` - Install dependencies
- `make dev` - Start development environment
- `make build` - Build all Docker images
- `make up` - Start all services
- `make down` - Stop all services
- `make logs` - Show logs from all services
- `make clean` - Clean up Docker volumes and build artifacts
- `make sqlc` - Generate sqlc code
- `make migrate` - Run database migrations
- `make test` - Run tests
- `make reset-db` - Reset database (WARNING: deletes all data)

## Project Structure

```
.
├── api/                    # Go backend
│   ├── cmd/
│   │   └── api/           # Main application
│   ├── internal/
│   │   ├── db/            # Generated sqlc code
│   │   ├── rules/         # Rules engine
│   │   ├── reward/        # Reward service
│   │   ├── budget/        # Budget & ledger
│   │   ├── auth/          # Authentication
│   │   ├── channels/      # WhatsApp, USSD
│   │   └── http/          # HTTP handlers
│   ├── Dockerfile
│   └── go.mod
├── web/                   # React frontend
│   ├── src/
│   │   ├── pages/        # Page components
│   │   └── components/   # Reusable components
│   ├── Dockerfile
│   └── package.json
├── migrations/            # Database migrations
├── queries/              # SQL queries for sqlc
├── docker-compose.yml
├── Caddyfile
├── Makefile
└── README.md
```

## Architecture

The platform follows a modular monolith architecture:

- **Multi-tenancy**: Row-level security (RLS) for tenant isolation
- **Idempotency**: Event ingestion with idempotency keys
- **Rules Engine**: JsonLogic-based conditions with caps and cooldowns
- **Reward Types**: Discounts, vouchers, points, external vouchers, physical items
- **Channels**: WhatsApp (MVP), USSD (Phase 2)

## Database

The application uses PostgreSQL 16 with:
- Row-Level Security (RLS) for tenant isolation
- UUID primary keys
- JSONB for flexible metadata
- Support for ZWG and USD currencies

### Running Migrations

Migrations are automatically applied when the database container starts. To manually run migrations:

```bash
make migrate
```

### Resetting the Database

⚠️ WARNING: This will delete all data!

```bash
make reset-db
```

## Code Generation

The project uses sqlc for type-safe SQL queries. After modifying queries in the `queries/` directory:

```bash
make sqlc
```

## Configuration

Environment variables are configured in `.env`:

- `DATABASE_URL`: PostgreSQL connection string
- `JWT_SECRET`: Secret for JWT token signing
- `PORT`: API server port (default: 8080)
- `WHATSAPP_*`: WhatsApp Business API credentials
- `HMAC_KEYS_JSON`: API authentication keys

## Testing

Run all tests:

```bash
make test
```

## Deployment

For production deployment:

1. Update the Caddyfile with your domain
2. Set strong JWT_SECRET in .env
3. Configure WhatsApp credentials
4. Use proper database credentials
5. Enable HTTPS in Caddy

```bash
make build
make up
```

## License

See LICENSE file for details.