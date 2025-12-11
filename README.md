# Temporal Order Processing System

A production-ready order processing system built with Temporal demonstrating advanced workflow patterns, signals, queries, encryption, and child workflows.

## Features

### Core Requirements ✅
- **Workflows & Activities**: Complete order processing workflow with validation and processing activities
- **Retry Policies**: Configurable retry policies with exponential backoff
- **Worker Setup**: Production-ready worker with proper registration
- **Mock Server**: WireMock integration for external service simulation
- **Signals & Queries**: Real-time order status updates and queries
- **Unit Tests**: Comprehensive test coverage with mocked dependencies

### Advanced Features ✅
- **Encryption/Decryption**: AES-256-GCM payload codec for secure data handling
- **Child Workflows**: Separate payment processing workflow
- **Versioning**: Workflow versioning using `GetVersion` for backward compatibility
- **Advanced Messaging**: Expedited processing and real-time status queries

## Architecture

```
┌─────────────────┐
│   Starter CLI   │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│         Temporal Server                 │
│  ┌───────────────────────────────────┐  │
│  │   Order Processing Workflow       │  │
│  │                                   │  │
│  │  1. Validate Order ───────────┐  │  │
│  │  2. Process Payment (Child)   │  │  │
│  │  3. Process Order             │  │  │
│  │  4. Send Notification         │  │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
         │                    │
         ▼                    ▼
    ┌─────────┐         ┌──────────┐
    │ Worker  │         │ WireMock │
    └─────────┘         └──────────┘
```

## Project Structure

```
temporal-order-processing/
├── activities/              # Activity implementations
│   └── order_activities.go
├── codec/                   # Encryption/Decryption
│   └── encryption_codec.go
├── models/                  # Data models
│   └── order.go
├── workflows/               # Workflow definitions
│   ├── order_workflow.go
│   └── payment_workflow.go
├── worker/                  # Worker implementation
│   └── main.go
├── starter/                 # Workflow starter CLI
│   └── main.go
├── tests/                   # Unit tests
│   └── activities_test.go
├── wiremock/                # WireMock configuration
│   └── mappings/
│       └── validate.json
├── docker-compose.yml       # Infrastructure setup
├── go.mod
└── README.md
```

## Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- Make (optional, for convenience)

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/aswathylr-builds/temporal-order-processing.git
cd temporal-order-processing
```

### 2. Start Infrastructure

```bash
# Start Temporal Server, PostgreSQL, UI, and WireMock
docker-compose up -d

# Wait for services to be ready (30-60 seconds)
docker-compose ps
```

**Services:**
- Temporal Server: `localhost:7233`
- Temporal UI: `http://localhost:8080`
- WireMock: `http://localhost:8081`

### 3. Install Dependencies

```bash
go mod download
```

### 4. Start the Worker

```bash
# Terminal 1 - Start the worker
go run worker/main.go
```

You should see:
```
Worker starting on task queue: order-processing-queue
Validation URL: http://localhost:8081/validate
Temporal Host: localhost:7233
```

### 5. Start a Workflow

```bash
# Terminal 2 - Start an order workflow
go run starter/main.go -order-id=ORDER-001 -amount=500.00 -items="laptop,mouse,keyboard"
```

Output:
```
Started workflow successfully
  Workflow ID: order-workflow-ORDER-001
  Run ID: <run-id>
  Order ID: ORDER-001
  Amount: $500.00
  Items: [laptop mouse keyboard]
```

## Demo Scenarios

### Scenario 1: Basic Order Processing

```bash
# Start a basic order
go run starter/main.go -order-id=ORDER-001 -amount=100.00 -items="item1,item2"

# Check status
go run starter/main.go -action=query -workflow-id=order-workflow-ORDER-001
```

### Scenario 2: Expedited Processing

```bash
# Start an order
go run starter/main.go -order-id=ORDER-002 -amount=250.00 -items="urgent-item"

# Send expedite signal (while workflow is running)
go run starter/main.go -action=expedite -workflow-id=order-workflow-ORDER-002

# Query status to see expedited flag
go run starter/main.go -action=query -workflow-id=order-workflow-ORDER-002
```

### Scenario 3: Order Cancellation

```bash
# Start an order
go run starter/main.go -order-id=ORDER-003 -amount=150.00 -items="item1"

# Cancel the order (quickly, while it's processing)
go run starter/main.go -action=cancel -workflow-id=order-workflow-ORDER-003

# Verify cancellation
go run starter/main.go -action=query -workflow-id=order-workflow-ORDER-003
```

### Scenario 4: Validation Failure

```bash
# Try to create an order with amount > $10,000 (will fail validation)
go run starter/main.go -order-id=ORDER-004 -amount=15000.00 -items="expensive-item"

# Check the workflow status (should show failed validation)
go run starter/main.go -action=query -workflow-id=order-workflow-ORDER-004
```

### Scenario 5: Encryption Enabled

```bash
# Start worker with encryption
ENCRYPTION_ENABLED=true go run worker/main.go

# In another terminal, start workflow with encryption
ENCRYPTION_ENABLED=true go run starter/main.go -order-id=ORDER-005 -amount=100.00

# Data will be encrypted in Temporal Server
```

## Monitoring with Temporal UI

1. Open your browser to `http://localhost:8080`
2. Navigate to "Workflows" to see all running workflows
3. Click on a workflow ID to see:
   - Execution history
   - Activity attempts and retries
   - Input/Output data
   - Pending activities
   - Workflow timeline

## Advanced Features Demo

### Workflow Versioning

The workflow includes version management for backward compatibility:

```go
version := workflow.GetVersion(ctx, "payment-processing-change", workflow.DefaultVersion, 1)

if version == workflow.DefaultVersion {
    // Old version: Skip payment processing
} else {
    // New version: Process payment using child workflow
}
```

This allows you to:
- Deploy new workflow versions without breaking existing executions
- Gradually roll out changes
- Support multiple versions simultaneously

### Child Workflow (Payment Processing)

Payment processing is handled by a separate child workflow:
- Independent lifecycle
- Separate retry policies
- Dedicated transaction handling
- Can be monitored independently in Temporal UI

### Encryption Codec

AES-256-GCM encryption for sensitive data:
- Encrypts workflow inputs/outputs
- Transparent to workflow logic
- Key management (development key stored in `.encryption.key`)
- **Production**: Use proper key management (AWS KMS, HashiCorp Vault, etc.)

## Testing

### Run Unit Tests

```bash
go test ./tests/... -v
```

### Test Coverage

```bash
go test ./tests/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Test Scenarios Covered

- ✅ Successful order validation
- ✅ Validation failure (high amount)
- ✅ HTTP error handling
- ✅ Order processing (normal and expedited)
- ✅ Payment processing
- ✅ Notification delivery
- ✅ Complete workflow execution

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `TEMPORAL_HOST` | `localhost:7233` | Temporal server address |
| `VALIDATION_URL` | `http://localhost:8081/validate` | Validation service URL |
| `ENCRYPTION_ENABLED` | `false` | Enable payload encryption |

### WireMock Configuration

WireMock validates orders based on amount:
- **Valid**: Amount < $10,000
- **Invalid**: Amount >= $10,000

You can customize the validation logic in `wiremock/mappings/validate.json`.

## Project Highlights for Interview

### 1. Core Temporal Concepts
- ✅ Workflows with multiple activities
- ✅ Proper retry policies and timeouts
- ✅ Activity heartbeats for long-running operations
- ✅ Worker registration and task queues

### 2. Advanced Patterns
- ✅ **Signals**: Cancel and expedite orders in real-time
- ✅ **Queries**: Get workflow status without affecting execution
- ✅ **Child Workflows**: Separate payment processing logic
- ✅ **Versioning**: Backward-compatible workflow changes

### 3. Security
- ✅ **Encryption**: AES-256-GCM payload codec
- ✅ Secure key generation and management
- ✅ Transparent encryption/decryption

### 4. Testing
- ✅ Comprehensive unit tests with mocking
- ✅ HTTP client mocking
- ✅ Temporal test suite integration
- ✅ Activity isolation testing

### 5. Production Readiness
- ✅ Docker Compose for easy setup
- ✅ Proper error handling
- ✅ Structured logging
- ✅ Configuration via environment variables
- ✅ Clean project structure

## Troubleshooting

### Services Not Starting

```bash
# Check service status
docker-compose ps

# View logs
docker-compose logs temporal
docker-compose logs wiremock

# Restart services
docker-compose restart
```

### Worker Connection Issues

```bash
# Verify Temporal is running
curl -I http://localhost:7233

# Check worker logs for connection errors
go run worker/main.go 2>&1 | tee worker.log
```

### WireMock Not Responding

```bash
# Test WireMock health
curl http://localhost:8081/__admin/

# View WireMock logs
docker-compose logs wiremock

# Restart WireMock
docker-compose restart wiremock
```

## Cleanup

```bash
# Stop all services
docker-compose down

# Remove volumes (resets database)
docker-compose down -v

# Clean up Go build artifacts
go clean -cache -testcache
```

## Development Notes

### Adding New Activities

1. Define activity in `activities/order_activities.go`
2. Register in `worker/main.go`
3. Call from workflow using `workflow.ExecuteActivity`
4. Add unit tests in `tests/`

### Adding New Signals

1. Define signal name in `models/order.go`
2. Add signal handler in workflow
3. Update starter to send signal
4. Test signal behavior

### Modifying Validation Logic

Edit `wiremock/mappings/validate.json` to change validation rules:
```json
{
  "request": {
    "method": "POST",
    "urlPath": "/validate"
  },
  "response": {
    "status": 200,
    "jsonBody": {
      "valid": true,
      "message": "Custom validation logic"
    }
  }
}
```

## License

MIT License - feel free to use this code for your projects.

## Interview Presentation Tips

1. **Start with Architecture**: Show the high-level diagram
2. **Live Demo**: Run through the 5 scenarios above
3. **Code Walkthrough**: Highlight key patterns (signals, child workflows, versioning)
4. **Temporal UI**: Show workflow execution and history
5. **Testing**: Demonstrate unit tests with mocking
6. **Encryption**: Explain the security considerations
7. **Production Considerations**: Discuss scaling, monitoring, and deployment

## Time Investment

- ⏱️ Core implementation: ~3 hours
- ⏱️ Advanced features: ~2 hours
- ⏱️ Testing & documentation: ~1 hour
- ⏱️ **Total**: ~6 hours

## Questions?

For questions or issues, please create an issue in the GitHub repository.

Happy Temporal coding! 🚀
