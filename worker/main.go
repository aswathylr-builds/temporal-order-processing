package main

import (
	"crypto/rand"
	"log"
	"os"

	"github.com/aswathylr-builds/temporal-order-processing/activities"
	"github.com/aswathylr-builds/temporal-order-processing/codec"
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

	// Start worker
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalf("Unable to start worker: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
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
