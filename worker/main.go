package main

import (
	"context"
	"crypto/rand"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/aswathylr-builds/temporal-order-processing/activities"
	"github.com/aswathylr-builds/temporal-order-processing/codec"
	"github.com/aswathylr-builds/temporal-order-processing/health"
	"github.com/aswathylr-builds/temporal-order-processing/workflows"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

const (
	taskQueue = "order-processing-queue"
)

func main() {
	// Get configuration from environment variables
	temporalHost := getEnv("TEMPORAL_HOST", "localhost:7233")
	validationURL := getEnv("VALIDATION_URL", "http://localhost:8081/validate")
	encryptionEnabled := getEnv("ENCRYPTION_ENABLED", "false") == "true"
	healthPort := getEnvAsInt("HEALTH_PORT", 8090)

	// Create Temporal client options
	clientOptions := client.Options{
		HostPort: temporalHost,
	}

	// Enable encryption if configured
	if encryptionEnabled {
		encryptionKey := generateOrGetEncryptionKey()
		dataConverter, err := codec.NewEncryptionDataConverter(encryptionKey)
		if err != nil {
			log.Fatalf("Failed to create encryption data converter: %v", err)
		}
		clientOptions.DataConverter = dataConverter
		log.Println("Encryption enabled for worker")
	}

	// Create the Temporal client
	c, err := client.Dial(clientOptions)
	if err != nil {
		log.Fatalf("Unable to create Temporal client: %v", err)
	}
	defer c.Close()

	// Create worker
	w := worker.New(c, taskQueue, worker.Options{})

	// Register workflows
	w.RegisterWorkflow(workflows.OrderWorkflow)
	w.RegisterWorkflow(workflows.PaymentWorkflow)

	// Register activities
	orderActivities := activities.NewOrderActivities(validationURL)
	w.RegisterActivity(orderActivities.ValidateOrder)
	w.RegisterActivity(orderActivities.ProcessOrder)
	w.RegisterActivity(orderActivities.NotifyOrderComplete)
	w.RegisterActivity(orderActivities.ProcessPayment)

	log.Printf("Worker starting on task queue: %s", taskQueue)
	log.Printf("Validation URL: %s", validationURL)
	log.Printf("Temporal Host: %s", temporalHost)

	// Create and configure health check server
	healthServer := health.NewServer(healthPort)

	// Register Temporal health check
	healthServer.RegisterChecker(health.NewTemporalChecker(c))

	// Register WireMock health check
	wiremockHealthURL := getEnv("WIREMOCK_URL", "http://localhost:8081") + "/__admin/"
	healthServer.RegisterChecker(health.NewHTTPChecker("wiremock", wiremockHealthURL))

	// Start health check server
	if err := healthServer.Start(); err != nil {
		log.Fatalf("Failed to start health check server: %v", err)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Start worker in goroutine
	errCh := make(chan error, 1)
	go func() {
		log.Println("Worker started successfully")
		if err := w.Run(worker.InterruptCh()); err != nil {
			errCh <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigCh:
		log.Println("Received shutdown signal, gracefully stopping...")
	case err := <-errCh:
		log.Printf("Worker error: %v", err)
	}

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
	defer shutdownCancel()

	log.Println("Stopping worker...")
	w.Stop()

	log.Println("Stopping health check server...")
	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Health server shutdown error: %v", err)
	}

	log.Println("Worker shutdown complete")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func generateOrGetEncryptionKey() []byte {
	// In production, load this from a secure key management system
	keyFile := ".encryption.key"

	// Try to read existing key
	if key, err := os.ReadFile(keyFile); err == nil && len(key) == 32 {
		log.Println("Using existing encryption key")
		return key
	}

	// Generate new key
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("Failed to generate encryption key: %v", err)
	}

	// Save key for future use (development only!)
	if err := os.WriteFile(keyFile, key, 0600); err != nil {
		log.Printf("Warning: Failed to save encryption key: %v", err)
	}

	log.Println("Generated new encryption key")
	return key
}
