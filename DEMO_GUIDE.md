# Demo Guide for Interview Presentation

This guide will help you present the Temporal Order Processing System during your interview.

## Pre-Demo Setup (5 minutes before interview)

```bash
# 1. Start all services
docker-compose up -d

# 2. Wait for services to initialize
sleep 30

# 3. Verify services are running
docker-compose ps

# 4. Open Temporal UI in browser
open http://localhost:8080
```

## Demo Flow (20-30 minutes)

### 1. Introduction (2 minutes)

**Key Points:**
- Built a production-ready order processing system using Temporal
- Covers all basic requirements + advanced features
- Demonstrates real-world patterns and best practices

**Architecture Overview:**
```
User → Starter CLI → Temporal Server → Worker → Activities
                         ↓
                    WireMock (Mock External Service)
```

### 2. Project Structure Walkthrough (3 minutes)

```bash
# Show the clean, organized structure
tree -L 2

# Key components:
# - workflows/: Business logic orchestration
# - activities/: Individual task implementations
# - models/: Data structures
# - codec/: Encryption/decryption
# - tests/: Comprehensive unit tests
```

**Highlight:**
- Separation of concerns
- Testable architecture
- Production-ready structure

### 3. Live Demo - Basic Order Processing (5 minutes)

```bash
# Terminal 1 - Start Worker
go run worker/main.go

# Terminal 2 - Start an order
go run starter/main.go -order-id=DEMO-001 -amount=150.00 -items="laptop,mouse"

# Show in Temporal UI:
# - Workflow execution
# - Activity calls
# - Input/Output
# - Timeline
```

**Talking Points:**
- Worker polls task queue
- Workflow orchestrates multiple activities
- Each activity has retry policies
- Complete execution history visible in UI

### 4. Signals Demo - Expedited Processing (3 minutes)

```bash
# Start an order
go run starter/main.go -order-id=DEMO-002 -amount=200.00 -items="urgent-item"

# While it's running, send expedite signal
go run starter/main.go -action=expedite -workflow-id=order-workflow-DEMO-002

# Query the status
go run starter/main.go -action=query -workflow-id=order-workflow-DEMO-002
```

**Talking Points:**
- Signals allow external interaction with running workflows
- No workflow restarts needed
- Real-time updates to workflow state
- Processing time reduces from 5s to 2s when expedited

### 5. Queries Demo - Real-time Status (2 minutes)

```bash
# Query any running workflow
go run starter/main.go -action=query -workflow-id=order-workflow-DEMO-001

# Show the JSON output with:
# - Current status
# - Current stage
# - Is expedited?
# - Payment status
# - Last updated timestamp
```

**Talking Points:**
- Queries don't affect workflow execution
- Read-only operations
- Useful for monitoring and dashboards
- No workflow replay needed

### 6. Validation Failure Demo (2 minutes)

```bash
# Try an order with amount > $10,000
go run starter/main.go -order-id=FAIL-001 -amount=15000.00 -items="expensive"

# Show in UI:
# - Workflow failed
# - Validation activity returned false
# - Error message visible
```

**Talking Points:**
- WireMock simulates external validation service
- Business rules enforced
- Proper error handling and reporting

### 7. Code Walkthrough (8 minutes)

#### a) Main Workflow (`workflows/order_workflow.go`)

**Show:**
```go
// Workflow versioning - backward compatibility
version := workflow.GetVersion(ctx, "payment-processing-change", workflow.DefaultVersion, 1)

// Signal handlers
cancelChannel := workflow.GetSignalChannel(ctx, models.SignalCancel)
expediteChannel := workflow.GetSignalChannel(ctx, models.SignalExpedite)

// Query handler
workflow.SetQueryHandler(ctx, "getStatus", func() (*models.OrderStatus, error) {
    return state, nil
})

// Child workflow execution
workflow.ExecuteChildWorkflow(childCtx, PaymentWorkflowName, order)
```

**Explain:**
- Workflow.GetVersion allows safe updates to running workflows
- Signals for cancel/expedite
- Queries for status
- Child workflows for modular design

#### b) Activities (`activities/order_activities.go`)

**Show:**
```go
// Activity with proper context handling
func (a *OrderActivities) ValidateOrder(ctx context.Context, order models.Order) (*models.ValidationResponse, error) {
    logger := activity.GetLogger(ctx)
    // HTTP call to external service
    // Proper error handling
}

// Activity with heartbeats
func (a *OrderActivities) ProcessOrder(ctx context.Context, order models.Order, isExpedited bool) error {
    // Long-running operation with heartbeats
    activity.RecordHeartbeat(ctx, "processing")
}
```

**Explain:**
- Activities are idempotent
- Proper logging with activity logger
- Heartbeats for long-running operations
- Timeout and retry configurations

#### c) Encryption Codec (`codec/encryption_codec.go`)

**Show:**
```go
// AES-256-GCM encryption
func (e *EncryptionCodec) Encode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
    // Encrypts data before storing in Temporal
}

func (e *EncryptionCodec) Decode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
    // Decrypts data when reading from Temporal
}
```

**Explain:**
- Transparent encryption/decryption
- Sensitive data protected at rest
- No changes needed to workflow code
- Production-ready security pattern

#### d) Unit Tests (`tests/activities_test.go`)

**Show:**
```go
// Mock HTTP server for testing
mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Simulate external service
}))

// Test with mock
orderActivities := activities.NewOrderActivities(mockServer.URL + "/validate")
resp, err := orderActivities.ValidateOrder(ctx, order)
```

**Explain:**
- Activities tested in isolation
- HTTP client mocking
- No external dependencies needed
- Fast, reliable tests

### 8. Advanced Features Highlight (3 minutes)

#### Child Workflow
```bash
# Show payment_workflow.go
# Explain: Separate lifecycle, independent retry policies
```

#### Workflow Versioning
```bash
# Explain the GetVersion pattern
# How it allows deploying new code without breaking running workflows
```

#### Encryption
```bash
# Demo with encryption enabled
ENCRYPTION_ENABLED=true go run worker/main.go

# In another terminal
ENCRYPTION_ENABLED=true go run starter/main.go -order-id=SECURE-001 -amount=100.00

# Show: Data encrypted in Temporal UI
```

### 9. Testing Demo (2 minutes)

```bash
# Run all tests
go test ./tests/... -v

# Show coverage
go test ./tests/... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Highlight:
# - 7+ test cases
# - Mock HTTP servers
# - Activity isolation
# - Workflow testing with test suite
```

### 10. Production Considerations (2 minutes)

**Discuss:**

1. **Scalability**
   - Horizontal scaling of workers
   - Task queue partitioning
   - Activity pooling

2. **Monitoring**
   - Temporal UI for workflow visibility
   - Metrics export (Prometheus)
   - Custom logging

3. **Security**
   - Encryption at rest (demonstrated)
   - mTLS for communication
   - Key management (KMS integration)

4. **Deployment**
   - Docker Compose for local dev
   - Kubernetes for production
   - CI/CD integration

5. **Error Handling**
   - Retry policies with exponential backoff
   - Dead letter queues
   - Alerting on failures

## Q&A Preparation

### Expected Questions

**Q: Why did you choose Temporal over other orchestration tools?**
A: Temporal provides durable execution, built-in retry mechanisms, and excellent visibility. It's particularly good for long-running workflows and complex state management.

**Q: How would you handle workflow upgrades in production?**
A: Using `workflow.GetVersion` allows backward-compatible changes. New workers can handle both old and new workflow versions simultaneously.

**Q: What about performance at scale?**
A: Temporal scales horizontally. Add more workers to increase throughput. Activities can be optimized independently. Task queues can be partitioned by priority or type.

**Q: How do you ensure data consistency?**
A: Temporal's event sourcing ensures consistency. Activities are designed to be idempotent. The workflow state is fully reconstructed from event history.

**Q: What testing strategies did you use?**
A: Unit tests with mocked dependencies, integration tests with Temporal test suite, and end-to-end tests with Docker Compose.

**Q: How would you monitor this in production?**
A: Temporal UI for workflow visibility, export metrics to Prometheus/Grafana, structured logging with ELK stack, and custom alerting for business-critical workflows.

**Q: What about the encryption overhead?**
A: AES-GCM is efficient. The overhead is minimal compared to network/IO. For sensitive data, the security benefit outweighs the small performance cost.

**Q: How do you handle activity failures?**
A: Retry policies with exponential backoff, configurable maximum attempts, and proper error handling. Failed workflows can be manually retried or investigated in the UI.

## Time Breakdown Summary

- Introduction: 2 min
- Structure: 3 min
- Live Demo: 12 min (basic + signals + queries + failure)
- Code Walkthrough: 8 min
- Advanced Features: 3 min
- Testing: 2 min
- Production: 2 min
- **Total: ~30 minutes**

## Tips for Success

1. **Practice the demo flow** - Know exactly what you'll type
2. **Have terminals ready** - Pre-positioned with commands
3. **Keep Temporal UI visible** - Shows real-time execution
4. **Emphasize patterns** - Not just "what" but "why"
5. **Show enthusiasm** - You built something cool!
6. **Be ready to go deeper** - On any topic they're interested in
7. **Relate to production** - Always tie back to real-world usage

## Post-Demo

If they want to see more:
- Run tests live
- Show encryption in action
- Demonstrate workflow history replay
- Discuss specific production scenarios

## Emergency Backup

If live demo fails:
- Show pre-recorded video
- Walk through code in detail
- Use Temporal UI with existing workflows
- Focus on architecture and design decisions

---

**Good luck with your interview! 🚀**
