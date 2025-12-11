# Health Check System

This document describes the health check system implemented for production readiness.

## Overview

The worker exposes HTTP health check endpoints that monitor the application's health and readiness. These endpoints are designed for:
- Load balancer health probes
- Kubernetes liveness and readiness probes
- Monitoring and alerting systems
- Operational visibility

## Endpoints

### 1. `/health` - Detailed Health Status

**Purpose:** Comprehensive health check with component-level details

**Response:** HTTP 200 (healthy/degraded) or 503 (unhealthy)

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "timestamp": "2025-12-11T16:24:43.295709+11:00",
  "components": {
    "temporal": {
      "status": "healthy",
      "message": "Connected to Temporal server",
      "latency": "5.213071ms"
    },
    "wiremock": {
      "status": "healthy",
      "message": "HTTP 200",
      "latency": "15.018815ms"
    }
  }
}
```

**Status Values:**
- `healthy` - All components functioning normally
- `degraded` - Some components degraded but service operational
- `unhealthy` - Critical components failed, service unavailable

### 2. `/health/live` - Liveness Probe

**Purpose:** Kubernetes liveness probe - checks if the application is alive

**Response:** HTTP 200 (always, if server is running)

```json
{
  "status": "alive"
}
```

**Use Case:**
- Kubernetes liveness probe
- Detects if application has deadlocked or crashed
- Triggers container restart if failing

```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 8090
  initialDelaySeconds: 10
  periodSeconds: 10
```

### 3. `/health/ready` - Readiness Probe

**Purpose:** Kubernetes readiness probe - checks if the application can serve traffic

**Response:** HTTP 200 (ready) or 503 (not ready)

```json
{
  "status": "ready"
}
```

**Use Case:**
- Kubernetes readiness probe
- Load balancer backend health checks
- Traffic routing decisions

```yaml
readinessProbe:
  httpGet:
    path: /health/ready
    port: 8090
  initialDelaySeconds: 5
  periodSeconds: 5
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HEALTH_PORT` | `8090` | Port for health check HTTP server |
| `TEMPORAL_HOST` | `localhost:7233` | Temporal server address (checked) |
| `WIREMOCK_URL` | `http://localhost:8081` | WireMock server URL (checked) |

### Example

```bash
# Custom health check port
HEALTH_PORT=9090 go run worker/main.go

# Test health endpoint
curl http://localhost:9090/health
```

## Component Health Checks

### 1. Temporal Connectivity

**Checker:** `TemporalChecker`

**What it checks:**
- Connection to Temporal server
- Temporal server health
- Network latency

**Status Logic:**
- `healthy` - Successfully connected, latency < 5s
- `unhealthy` - Cannot connect or timeout

### 2. WireMock Service

**Checker:** `HTTPChecker`

**What it checks:**
- WireMock admin endpoint availability
- HTTP response time

**Status Logic:**
- `healthy` - HTTP 2xx response
- `degraded` - HTTP 3xx/4xx response
- `unhealthy` - Connection failed or timeout

## Adding Custom Health Checks

### Step 1: Implement the Checker Interface

```go
type MyServiceChecker struct {
    client *MyServiceClient
}

func (c *MyServiceChecker) Name() string {
    return "my-service"
}

func (c *MyServiceChecker) Check(ctx context.Context) health.ComponentHealth {
    start := time.Now()

    err := c.client.Ping(ctx)
    latency := time.Since(start)

    if err != nil {
        return health.ComponentHealth{
            Status:  health.StatusUnhealthy,
            Message: fmt.Sprintf("Ping failed: %v", err),
            Latency: latency.String(),
        }
    }

    return health.ComponentHealth{
        Status:  health.StatusHealthy,
        Message: "Service responsive",
        Latency: latency.String(),
    }
}
```

### Step 2: Register the Checker

```go
// In worker/main.go
healthServer.RegisterChecker(&MyServiceChecker{
    client: myServiceClient,
})
```

## Production Deployment

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: temporal-worker
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: worker
        image: temporal-worker:latest
        ports:
        - containerPort: 8090
          name: health
        env:
        - name: HEALTH_PORT
          value: "8090"
        - name: TEMPORAL_HOST
          value: "temporal.temporal-system:7233"
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8090
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8090
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 2
```

### Service Definition

```yaml
apiVersion: v1
kind: Service
metadata:
  name: temporal-worker-health
spec:
  type: ClusterIP
  ports:
  - port: 8090
    targetPort: 8090
    protocol: TCP
    name: health
  selector:
    app: temporal-worker
```

### Load Balancer Integration

For AWS ALB/ELB:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: temporal-worker
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-healthcheck-path: /health/ready
    service.beta.kubernetes.io/aws-load-balancer-healthcheck-port: "8090"
    service.beta.kubernetes.io/aws-load-balancer-healthcheck-interval: "10"
    service.beta.kubernetes.io/aws-load-balancer-healthcheck-timeout: "5"
    service.beta.kubernetes.io/aws-load-balancer-healthcheck-healthy-threshold: "2"
    service.beta.kubernetes.io/aws-load-balancer-healthcheck-unhealthy-threshold: "3"
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8090
```

## Graceful Shutdown

The health check system integrates with graceful shutdown:

1. **SIGTERM received** → Worker stops accepting new tasks
2. **In-flight tasks complete** → Up to 30 seconds
3. **Health server stops** → Endpoint becomes unavailable
4. **Worker exits** → Clean shutdown

### Shutdown Sequence

```
[SIGTERM] → Stop worker → Complete tasks → Stop health server → Exit
            (immediate)   (up to 30s)      (graceful)          (0s)
```

### Testing Graceful Shutdown

```bash
# Start worker
go run worker/main.go

# In another terminal, send SIGTERM
kill -TERM $(pgrep -f "worker/main.go")

# Watch logs
2025/12/11 16:25:00 Received shutdown signal, gracefully stopping...
2025/12/11 16:25:00 Stopping worker...
2025/12/11 16:25:00 Stopping health check server...
2025/12/11 16:25:00 Worker shutdown complete
```

## Monitoring and Alerting

### Prometheus Metrics (Future Enhancement)

```yaml
# Example metrics to add
temporal_worker_health_status{component="temporal"} 1  # 1=healthy, 0=unhealthy
temporal_worker_health_latency_ms{component="temporal"} 5.2
temporal_worker_up 1
```

### Alert Rules

```yaml
groups:
- name: temporal-worker
  rules:
  - alert: WorkerUnhealthy
    expr: up{job="temporal-worker"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Worker is down"

  - alert: TemporalConnectionDegraded
    expr: temporal_worker_health_latency_ms{component="temporal"} > 1000
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High latency to Temporal server"
```

## Testing

### Manual Testing

```bash
# Test all endpoints
curl http://localhost:8090/health
curl http://localhost:8090/health/live
curl http://localhost:8090/health/ready

# Test with formatting
curl -s http://localhost:8090/health | python3 -m json.tool

# Test readiness status code
curl -w "%{http_code}\n" -o /dev/null -s http://localhost:8090/health/ready
```

### Automated Testing

```bash
# Health check script for CI/CD
#!/bin/bash
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8090/health/ready)
if [ "$response" -eq 200 ]; then
  echo "Worker is ready"
  exit 0
else
  echo "Worker is not ready (HTTP $response)"
  exit 1
fi
```

## Troubleshooting

### Health Endpoint Not Responding

1. Check if worker is running:
   ```bash
   ps aux | grep worker
   ```

2. Check if port is open:
   ```bash
   lsof -i :8090
   netstat -an | grep 8090
   ```

3. Check worker logs:
   ```bash
   tail -f worker.log | grep "Health check server"
   ```

### Component Showing Unhealthy

1. **Temporal unhealthy:**
   - Check Temporal server is running
   - Verify network connectivity: `nc -zv localhost 7233`
   - Check Temporal UI: `http://localhost:8080`

2. **WireMock unhealthy:**
   - Check WireMock is running: `docker ps | grep wiremock`
   - Test directly: `curl http://localhost:8081/__admin/`
   - Check logs: `docker logs temporal-order-processing_wiremock_1`

## Security Considerations

1. **Network Access:**
   - Health endpoints should only be accessible from:
     - Load balancers
     - Kubernetes control plane
     - Monitoring systems
   - Use network policies to restrict access

2. **Information Disclosure:**
   - Health endpoints do not expose sensitive data
   - Latency information is operational metadata only
   - No authentication required (by design for k8s probes)

3. **Production Best Practices:**
   - Use separate network for health checks
   - Implement rate limiting if exposed publicly
   - Monitor access logs for anomalies

## Future Enhancements

- [ ] Prometheus metrics integration
- [ ] Custom health check timeout configuration
- [ ] Database connection pool health check
- [ ] Cache health check (Redis, etc.)
- [ ] Disk space health check
- [ ] Memory usage health check
- [ ] Goroutine leak detection

## References

- [Kubernetes Liveness and Readiness Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [AWS Load Balancer Health Checks](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/target-group-health-checks.html)
- [Temporal Health Check API](https://docs.temporal.io/references/sdk-go/client#checkhealthrequest)
