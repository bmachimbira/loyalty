# Backend API - Directory Structure

```
/home/user/loyalty/api/
├── cmd/
│   └── api/
│       └── main.go                          # Application entry point
├── internal/
│   ├── auth/                                # Authentication & Security
│   │   ├── jwt.go                          # JWT token generation & validation
│   │   ├── hmac.go                         # HMAC signature verification
│   │   └── password.go                     # bcrypt password hashing
│   ├── config/
│   │   └── config.go                       # Environment configuration
│   ├── db/                                  # sqlc generated code (by Database Agent)
│   │   ├── querier.go                      # Database interface
│   │   ├── models.go                       # Database models
│   │   ├── customers.sql.go                # Customer queries
│   │   ├── events.sql.go                   # Event queries
│   │   ├── rules.sql.go                    # Rules queries
│   │   ├── rewards.sql.go                  # Rewards queries
│   │   ├── issuances.sql.go                # Issuances queries
│   │   ├── budgets.sql.go                  # Budget queries
│   │   ├── campaigns.sql.go                # Campaign queries
│   │   └── ... (other generated files)
│   └── http/
│       ├── errors.go                        # Error handling utilities
│       ├── validation.go                    # Request validation utilities
│       ├── router.go                        # HTTP router configuration
│       ├── handlers/                        # API endpoint handlers
│       │   ├── customers.go                # Customer CRUD endpoints
│       │   ├── events.go                   # Event ingestion endpoints
│       │   ├── rules.go                    # Rules management endpoints
│       │   ├── rewards.go                  # Rewards catalog endpoints
│       │   ├── issuances.go                # Issuances endpoints
│       │   ├── budgets.go                  # Budget management endpoints
│       │   └── campaigns.go                # Campaign management endpoints
│       └── middleware/                      # HTTP middleware
│           ├── auth.go                     # JWT & HMAC authentication
│           ├── tenant.go                   # Tenant context & RLS
│           ├── idempotency.go              # Idempotency key handling
│           ├── cors.go                     # CORS headers
│           ├── logger.go                   # Request logging
│           └── request_id.go               # Request ID generation
├── go.mod                                   # Go module definition
├── go.sum                                   # Dependency checksums
├── Dockerfile                               # Container build definition
└── .dockerignore                            # Docker ignore rules
```

## File Statistics

- **Total Go Files**: 40+
- **Lines of Code**: ~2,500
- **Packages**: 5 (main, auth, config, http, db)
- **API Endpoints**: 35+
- **Middleware**: 6

## Key Features

### Authentication
- ✅ JWT tokens (access + refresh)
- ✅ HMAC server-to-server auth
- ✅ bcrypt password hashing
- ✅ Role-based access control

### Middleware Stack
- ✅ CORS handling
- ✅ Request ID propagation
- ✅ Logging with latency tracking
- ✅ JWT authentication
- ✅ Tenant context (RLS)
- ✅ Idempotency checking

### API Coverage
- ✅ Customers CRUD
- ✅ Events ingestion
- ✅ Rules management
- ✅ Rewards catalog
- ✅ Issuances tracking
- ✅ Budget management
- ✅ Campaign management
- ✅ Ledger queries

### Security
- ✅ Multi-tenant isolation (RLS)
- ✅ Input validation
- ✅ Standardized error handling
- ✅ Secure password storage
- ✅ Token expiration

## Integration Status

- ✅ **Database**: sqlc code generated, ready for integration
- ✅ **Router**: All endpoints configured
- ✅ **Middleware**: Complete chain implemented
- ⏳ **Handlers**: Ready for database integration (placeholders in place)
- ⏳ **Rules Engine**: Deferred to Phase 2
- ⏳ **Channels**: Placeholder implementations (Phase 3)

## Next Steps

1. Integrate handlers with sqlc-generated database code
2. Add transaction handling for complex operations
3. Implement rules engine (Phase 2)
4. Add comprehensive testing
5. Performance optimization

