# Go Rate Limiter Gateway

A high-performance API gateway with rate limiting, JWT authentication, and embedded backend services built in Go.

## Features

- **Reverse Proxy**: Routes requests to backend services based on path prefixes
- **Embedded Backends**: Run API and Auth services in the same binary or separately
- **Rate Limiting**: Two algorithms supported:
  - Token Bucket: Smooth rate limiting with burst capacity
  - Sliding Window: Fixed window request counting
- **JWT Authentication**: Secure endpoints with JSON Web Tokens
- **Redis Backend**: Distributed rate limiting across multiple gateway instances
- **Request Logging**: Structured JSON logging with slog

## Project Structure

```
go-rate-limiter-gateway/
├── cmd/gateway/              # Application entrypoint
├── internal/
│   ├── backends/
│   │   ├── api/              # API service (users, products, orders)
│   │   └── auth/             # Auth service (login, register, JWT)
│   ├── gateway/              # Reverse proxy implementation
│   ├── ratelimiter/          # Rate limiting algorithms
│   ├── middleware/           # HTTP middlewares
│   └── redis/                # Redis client wrapper
├── config/                   # Configuration files
├── docker/                   # Docker and compose files
└── scripts/                  # Load testing scripts
```

## Quick Start

### Prerequisites

- Go 1.21+
- Redis 7+
- Docker & Docker Compose (optional)

### Run Locally (Single Binary with Embedded Backends)

1. Start Redis:
```bash
redis-server
```

2. Run the gateway with embedded backends:
```bash
cd go-rate-limiter-gateway
go mod tidy
go run ./cmd/gateway
```

This starts:
- Gateway on port 8080
- API service on port 3001 (embedded)
- Auth service on port 3002 (embedded)

### Run with Docker Compose

```bash
cd go-rate-limiter-gateway/docker
docker-compose up -d
```

## Configuration

Edit `config/config.yaml`:

```yaml
server:
  port: 8080

redis:
  host: localhost
  port: 6379

rate_limiter:
  algorithm: token_bucket
  token_bucket:
    rate: 100
    capacity: 200

jwt:
  secret: "your-secret-key"
  expiry: 24h

backends:
  - name: api-service
    prefix: /api
    target: http://localhost:3001
    embedded: true        # Run in same binary
    port: 3001
  - name: auth-service
    prefix: /auth
    target: http://localhost:3002
    embedded: true
    port: 3002
```

Set `embedded: false` to proxy to external services instead.

## API Endpoints

### Gateway

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/health` | GET | No | Health check |
| `/gateway/token` | POST | No | Generate gateway JWT token |

### API Service (`/api/*`)

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/api/health` | GET | Yes | API service health |
| `/api/users` | GET/POST | Yes | List/create users |
| `/api/users/{id}` | GET/PUT/DELETE | Yes | User CRUD |
| `/api/products` | GET/POST | Yes | List/create products |
| `/api/products/{id}` | GET/PUT/DELETE | Yes | Product CRUD |
| `/api/orders` | GET/POST | Yes | List/create orders |
| `/api/orders/{id}` | GET | Yes | Get order |
| `/api/orders/{id}/status` | PUT | Yes | Update order status |

### Auth Service (`/auth/*`)

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/auth/health` | GET | No | Auth service health |
| `/auth/register` | POST | No | Register new user |
| `/auth/login` | POST | No | Login (returns JWT) |
| `/auth/logout` | POST | No | Logout (invalidate refresh token) |
| `/auth/refresh` | POST | No | Refresh access token |
| `/auth/me` | GET/PUT | Yes | Get/update profile |
| `/auth/change-password` | POST | Yes | Change password |
| `/auth/validate` | GET | No | Validate JWT token |

## Usage Examples

### Register a User

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123","name":"Test User"}'
```

### Login

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"password123"}'
```

### Access Protected API

```bash
# Get token from login response
TOKEN="<access_token from login>"

# List products
curl http://localhost:8080/api/products \
  -H "Authorization: Bearer $TOKEN"

# Create order
curl -X POST http://localhost:8080/api/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_id":1,"product_id":1,"quantity":2}'
```

### Default Test Users

| Email | Password | Role |
|-------|----------|------|
| admin@example.com | password123 | admin |
| user@example.com | password123 | user |

## Load Testing

Install [k6](https://k6.io/docs/getting-started/installation/) and run:

```bash
k6 run scripts/load-test.js
```

## Rate Limiting Algorithms

### Token Bucket

Tokens are added at a fixed rate. Each request consumes one token. Allows bursts up to bucket capacity.

### Sliding Window

Counts requests in a sliding time window. Rejects requests when the count exceeds the limit.

## License

MIT
