# saftaja | سفتجة

A self-hosted payment checkout system. Accept card payments via MPGS (Mastercard Payment Gateway Services) with 3DS authentication. Think of it as your own mini-Stripe on top of MPGS.

## Features

- Hosted checkout pages with custom branding
- 3DS2 authentication flow (frictionless + challenge)
- Multi-language support (English, Arabic)
- Custom domain support per merchant
- Apple Pay (coming soon)

## Quick Start

```bash
# Copy environment file
cp .env.example .env
# Edit .env with your database credentials

# Start PostgreSQL and the app
docker compose up

# App runs at http://localhost:8080
```

## Development

### Prerequisites

- Go 1.24+
- Docker & Docker Compose
- Bun (for e2e tests)
- buf cli: go install github.com/bufbuild/buf/cmd/buf@v1.67.0

### Project Structure

```
internal/
├── domain/           # Core types, i18n, constants
├── connectors/       # Payment gateway abstraction
│   └── mpgs/         # MPGS connector
├── payment/          # Business logic & state machine
├── clients/mpgs/     # MPGS HTTP client
├── store/            # Database layer (sqlc-generated)
│   ├── queries/      # SQL query files
│   └── migrations/   # Database migrations
├── web/              # HTTP handlers
│   └── templfiles/   # HTML templates (templ)
├── config/           # Configuration
└── utils/            # Utilities
```

### Code Generation

```bash
# After editing SQL queries (internal/store/queries/*.sql)
go tool sqlc generate

# After editing templates (internal/web/templfiles/*.templ)
go tool templ generate

# After editing interfaces (for mocks)
go tool mockery
```

### Running Tests

```bash
# Unit tests
go test ./internal/...

# E2E tests
cd tests/e2e
bun playwright test -j 1 --headed
```

### Database Migrations

Migrations run automatically on startup. For manual operations:

```bash
# Apply all pending migrations
make migrate-up

# Rollback one migration
make migrate-down

# Create new migration
make migrate-create name=add_user_session
```

## Payment Flow

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐     ┌──────────────┐
│  Checkout   │────►│   Initiate   │────►│  3DS Auth   │────►│   Finalize   │
│    Page     │     │   Session    │     │   (if req)  │     │   Payment    │
└─────────────┘     └──────────────┘     └─────────────┘     └──────────────┘
       │                   │                    │                    │
       ▼                   ▼                    ▼                    ▼
   GET /checkout     POST /initiate      POST /process-auth    POST /finalize
       │                   │                    │                    │
       ▼                   ▼                    ▼                    ▼
   Invoice info      Session created     Authentication       Payment executed
   + gateway config  + MPGS session      completed            Invoice marked paid
```

### State Machine

```
PaymentSession: created → authenticating → authenticated → paying → completed
                    ↓           ↓                            ↓
                  failed      failed                       failed

Invoice: pending → processing → paid
             ↓          ↓
           failed     failed
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/checkout/{invoice_id}` | Render checkout page |
| POST | `/checkout/{invoice_id}/initiate` | Create payment session |
| POST | `/checkout/{invoice_id}/pay/card/{session_id}/initiate-auth` | Start 3DS auth |
| POST | `/checkout/{invoice_id}/pay/card/{session_id}/process-auth` | Process 3DS challenge |
| POST | `/checkout/{invoice_id}/pay/card/{session_id}/finalize` | Execute payment |

## Configuration

Environment variables (set in `.env`):

| Variable | Description | Required |
|----------|-------------|----------|
| `DB_HOST` | PostgreSQL host | Yes |
| `DB_PORT` | PostgreSQL port | Yes |
| `DB_DATABASE` | Database name | Yes |
| `DB_USER` | Database user | Yes |
| `DB_PASSWORD` | Database password | Yes |
| `PORT` | HTTP server port | Yes |
| `ENCRYPTION_KEY` | 32-byte key, base64 encoded | Yes |
| `VERIFY_DOMAIN_SECRET` | Secret for Caddy domain verification | Yes |
| `ADMIN_API_KEY` | API key for admin endpoints | Yes |
| `ACME_EMAIL` | Email for Let's Encrypt (prod only) | Prod |
| `BASE_DOMAIN` | Your main domain (prod only) | Prod |

## Generating Secrets

```bash
# Generate ENCRYPTION_KEY (32 bytes, base64)
openssl rand -base64 32

# Generate ADMIN_API_KEY
openssl rand -hex 32

# Generate VERIFY_DOMAIN_SECRET
openssl rand -hex 16
```

## Admin API

Protected by `X-Admin-Key` header. Use these to create orgs, projects, gateway accounts, and invoices.

| Method | Path | Description |
|--------|------|-------------|
| POST | `/admin/organizations` | Create organization |
| POST | `/admin/projects` | Create project |
| POST | `/admin/gateway-accounts` | Create gateway account |
| POST | `/admin/invoices` | Create invoice |

Example:
```bash
# Create org
curl -X POST http://localhost:8080/admin/organizations \
  -H "X-Admin-Key: $ADMIN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name": "My Company"}'

# Create project
curl -X POST http://localhost:8080/admin/projects \
  -H "X-Admin-Key: $ADMIN_API_KEY" \
  -d '{"organization_id": "uuid", "name": "Production", "environment": "production", "custom_domain": "pay.example.com"}'

# Create gateway account
curl -X POST http://localhost:8080/admin/gateway-accounts \
  -H "X-Admin-Key: $ADMIN_API_KEY" \
  -d '{"project_id": "uuid", "connector_type": "mpgs", "account_name": "MPGS Prod", "credentials": {"merchant_id": "...", "api_password": "...", "base_url": "https://..."}}'

# Create invoice
curl -X POST http://localhost:8080/admin/invoices \
  -H "X-Admin-Key: $ADMIN_API_KEY" \
  -d '{"project_id": "uuid", "amount": "15.000", "currency": "BHD", "customer_email": "test@example.com"}'
```


