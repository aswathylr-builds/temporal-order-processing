.PHONY: help deps up down worker start test clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Install Go dependencies
	go mod download
	go mod tidy

up: ## Start all services (Temporal, WireMock, etc.)
	docker-compose up -d
	@echo "Waiting for services to start..."
	@sleep 10
	@echo "Services started!"
	@echo "  - Temporal UI: http://localhost:8080"
	@echo "  - Temporal Server: localhost:7233"
	@echo "  - WireMock: http://localhost:8081"

down: ## Stop all services
	docker-compose down

down-v: ## Stop all services and remove volumes
	docker-compose down -v

logs: ## Show logs from all services
	docker-compose logs -f

worker: ## Start the Temporal worker
	go run worker/main.go

worker-encrypted: ## Start the worker with encryption enabled
	ENCRYPTION_ENABLED=true go run worker/main.go

start: ## Start a sample workflow
	go run starter/main.go -order-id=DEMO-001 -amount=150.00 -items="item1,item2,item3"

start-encrypted: ## Start a workflow with encryption
	ENCRYPTION_ENABLED=true go run starter/main.go -order-id=DEMO-002 -amount=200.00 -items="secure-item"

query: ## Query workflow status (requires WORKFLOW_ID env var)
	go run starter/main.go -action=query -workflow-id=$(WORKFLOW_ID)

expedite: ## Send expedite signal (requires WORKFLOW_ID env var)
	go run starter/main.go -action=expedite -workflow-id=$(WORKFLOW_ID)

cancel: ## Send cancel signal (requires WORKFLOW_ID env var)
	go run starter/main.go -action=cancel -workflow-id=$(WORKFLOW_ID)

test: ## Run all tests
	go test ./tests/... -v

test-coverage: ## Run tests with coverage
	go test ./tests/... -coverprofile=coverage.out
	go tool cover -html=coverage.out

build: ## Build all binaries
	go build -o bin/worker worker/main.go
	go build -o bin/starter starter/main.go

clean: ## Clean up build artifacts and temporary files
	go clean -cache -testcache
	rm -rf bin/
	rm -f coverage.out
	rm -f .encryption.key

demo: up ## Run full demo (start services, worker, and workflow)
	@echo "Starting demo..."
	@sleep 10
	@echo "Starting worker in background..."
	@go run worker/main.go &
	@sleep 3
	@echo "Starting workflow..."
	@go run starter/main.go -order-id=DEMO-001 -amount=150.00 -items="laptop,mouse"
	@echo ""
	@echo "Demo started! Check Temporal UI at http://localhost:8080"

format: ## Format code
	go fmt ./...

lint: ## Run linter
	golangci-lint run ./...

install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

status: ## Show service status
	docker-compose ps

restart: ## Restart all services
	docker-compose restart

example-high-amount: ## Test with high amount (should fail validation)
	go run starter/main.go -order-id=FAIL-001 -amount=15000.00 -items="expensive-item"

example-expedited: ## Example of expedited order
	@echo "Starting order..."
	@go run starter/main.go -order-id=EXP-001 -amount=100.00 -items="urgent" &
	@sleep 2
	@echo "Sending expedite signal..."
	@go run starter/main.go -action=expedite -workflow-id=order-workflow-EXP-001
