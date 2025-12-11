# Temporal Order Processing System

A production-ready order processing system built with Temporal demonstrating workflows, signals, queries, encryption, child workflows, and versioning.

## What This Is

An e-commerce order processing system that showcases Temporal's core features:
- **Workflows & Activities**: Orchestrated order validation, payment, and fulfillment
- **Signals & Queries**: Real-time order status updates and cancellations
- **Child Workflows**: Independent payment processing workflow
- **Versioning**: Backward-compatible workflow evolution with `GetVersion`
- **Encryption**: AES-256-GCM payload codec for sensitive data
- **Comprehensive Testing**: Unit tests with mocked dependencies

## Quick Start

### Prerequisites
- Go 1.21+
- Docker & Docker Compose

### 1. Start Infrastructure

```bash
# Clone the repository
git clone https://github.com/aswathylr-builds/temporal-order-processing.git
cd temporal-order-processing

# Start Temporal Server, PostgreSQL, UI, and WireMock
docker-compose up -d

# Wait 30 seconds for services to initialize
```

**Services:**
- Temporal UI: http://localhost:8080
- Temporal Server: localhost:7233
- WireMock: http://localhost:8081

### 2. Run the Worker

```bash
# Terminal 1 - Start worker
go run worker/main.go
```

### 3. Start a Workflow

```bash
# Terminal 2 - Create an order
go run starter/main.go -order-id=ORDER-001 -amount=500.00 -items="laptop,mouse"
```

## Usage Examples

### Query Order Status
```bash
go run starter/main.go -action=query -workflow-id=order-workflow-ORDER-001
```

### Expedite an Order
```bash
go run starter/main.go -action=expedite -workflow-id=order-workflow-ORDER-001
```

### Cancel an Order
```bash
go run starter/main.go -action=cancel -workflow-id=order-workflow-ORDER-001
```

### Trigger Validation Failure
```bash
# Orders over $10,000 fail validation
go run starter/main.go -order-id=FAIL-001 -amount=15000.00
```

### Run with Encryption
```bash
# Terminal 1
ENCRYPTION_ENABLED=true go run worker/main.go

# Terminal 2
ENCRYPTION_ENABLED=true go run starter/main.go -order-id=SECURE-001 -amount=100.00
```

## Architecture

```
┌─────────────┐
│ Starter CLI │
└──────┬──────┘
       │
       ▼
┌──────────────────────────────┐
│    Temporal Server           │
│  ┌────────────────────────┐  │
│  │ Order Workflow         │  │
│  │  1. Validate Order     │  │
│  │  2. Payment (Child WF) │  │
│  │  3. Process Order      │  │
│  │  4. Notify             │  │
│  └────────────────────────┘  │
└──────────────────────────────┘
       │              │
       ▼              ▼
  ┌────────┐    ┌──────────┐
  │ Worker │    │ WireMock │
  └────────┘    └──────────┘
```

## Project Structure

```
.
├── activities/          # Activity implementations
├── codec/              # Encryption codec
├── health/             # Health check endpoints
├── models/             # Data models
├── workflows/          # Workflow definitions
│   ├── order_workflow.go
│   └── payment_workflow.go
├── worker/             # Worker entry point
├── starter/            # CLI to start workflows
├── tests/              # Unit tests
└── docker-compose.yml  # Infrastructure
```

## Key Features

### 1. Signals & Queries
- **Cancel Signal**: Stop order processing
- **Expedite Signal**: Reduce processing time from 5s to 2s
- **Status Query**: Get real-time order status

### 2. Child Workflow
Payment processing runs as an independent child workflow with:
- Separate lifecycle and retry policies
- Independent monitoring in Temporal UI
- Dedicated workflow ID: `payment-{order-id}`

### 3. Workflow Versioning
Safe evolution from activity-based to child workflow payment:
```go
version := workflow.GetVersion(ctx, "payment-processing-change", workflow.DefaultVersion, 2)
if version == workflow.DefaultVersion {
    // Old: Activity-based payment
} else {
    // New: Child workflow payment
}
```

### 4. Encryption
AES-256-GCM encryption for workflow inputs/outputs:
- Transparent to workflow logic
- Development key stored in `.encryption.key`
- Production: Use KMS or Vault for key management

### 5. Health Checks
Production-ready health endpoints for Kubernetes:
- `/health` - Detailed component health
- `/health/live` - Liveness probe
- `/health/ready` - Readiness probe

## Testing

```bash
# Run all tests
go test ./tests/... -v

# With coverage
go test ./tests/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Monitoring

View workflows in Temporal UI at http://localhost:8080:
- Workflow execution history
- Activity retries and failures
- Input/output data (encrypted if enabled)
- Child workflow relationships
- Version markers

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `TEMPORAL_HOST` | `localhost:7233` | Temporal server address |
| `VALIDATION_URL` | `http://localhost:8081/validate` | Validation service URL |
| `ENCRYPTION_ENABLED` | `false` | Enable payload encryption |
| `HEALTH_PORT` | `8090` | Health check server port |

## Validation Rules (WireMock)

- Amount < $10,000: ✅ Valid
- Amount >= $10,000: ❌ Invalid

Customize in `wiremock/mappings/validate.json`

## Cleanup

```bash
# Stop all services
docker-compose down

# Remove volumes (reset database)
docker-compose down -v
```

## Troubleshooting

### Services not starting
```bash
docker-compose ps
docker-compose logs temporal
docker-compose restart
```

### Worker connection issues
```bash
# Verify Temporal is running
curl -I http://localhost:7233
```

### WireMock not responding
```bash
curl http://localhost:8081/__admin/
docker-compose logs wiremock
```

## What Makes This Production-Ready

- ✅ Proper retry policies with exponential backoff
- ✅ Activity timeouts and heartbeats
- ✅ Graceful shutdown with 30s timeout
- ✅ Health check endpoints for Kubernetes
- ✅ Comprehensive error handling
- ✅ Unit tests with mocked dependencies
- ✅ Encryption for sensitive data
- ✅ Structured logging
- ✅ Docker Compose for local development

## License

MIT License
