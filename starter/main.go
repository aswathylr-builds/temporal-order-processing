package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aswathylr-builds/temporal-order-processing/codec"
	"github.com/aswathylr-builds/temporal-order-processing/models"
	"github.com/aswathylr-builds/temporal-order-processing/workflows"
	"go.temporal.io/sdk/client"
)

const (
	taskQueue = "order-processing-queue"
)

func main() {
	// Command line flags
	orderID := flag.String("order-id", "", "Order ID (generated if not provided)")
	amount := flag.Float64("amount", 100.0, "Order amount")
	items := flag.String("items", "item1,item2", "Comma-separated list of items")
	action := flag.String("action", "start", "Action to perform: start, cancel, expedite, query")
	workflowID := flag.String("workflow-id", "", "Workflow ID for signal/query operations")
	flag.Parse()

	// Get configuration from environment variables
	temporalHost := getEnv("TEMPORAL_HOST", "localhost:7233")
	encryptionEnabled := getEnv("ENCRYPTION_ENABLED", "false") == "true"

	// Create Temporal client options
	clientOptions := client.Options{
		HostPort: temporalHost,
	}

	// Enable encryption if configured
	if encryptionEnabled {
		encryptionKey := loadEncryptionKey()
		dataConverter, err := codec.NewEncryptionDataConverter(encryptionKey)
		if err != nil {
			log.Fatalf("Failed to create encryption data converter: %v", err)
		}
		clientOptions.DataConverter = dataConverter
		log.Println("Encryption enabled for starter")
	}

	// Create the Temporal client
	c, err := client.Dial(clientOptions)
	if err != nil {
		log.Fatalf("Unable to create Temporal client: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	switch *action {
	case "start":
		startWorkflow(ctx, c, orderID, amount, items)
	case "cancel":
		sendSignal(ctx, c, *workflowID, models.SignalCancel)
	case "expedite":
		sendSignal(ctx, c, *workflowID, models.SignalExpedite)
	case "query":
		queryWorkflow(ctx, c, *workflowID)
	default:
		log.Fatalf("Unknown action: %s", *action)
	}
}

func startWorkflow(ctx context.Context, c client.Client, orderID *string, amount *float64, itemsStr *string) {
	// Generate order ID if not provided
	if *orderID == "" {
		*orderID = fmt.Sprintf("ORD-%d", time.Now().Unix())
	}

	// Parse items
	items := []string{}
	if *itemsStr != "" {
		json.Unmarshal([]byte(fmt.Sprintf("[\"%s\"]", *itemsStr)), &items)
	}

	// Create order
	order := models.Order{
		ID:        *orderID,
		Items:     items,
		Amount:    *amount,
		Status:    models.StatusPending,
		CreatedAt: time.Now(),
	}

	// Workflow options
	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("order-workflow-%s", order.ID),
		TaskQueue: taskQueue,
	}

	// Start workflow
	we, err := c.ExecuteWorkflow(ctx, workflowOptions, workflows.OrderWorkflow, order)
	if err != nil {
		log.Fatalf("Unable to execute workflow: %v", err)
	}

	log.Printf("Started workflow successfully")
	log.Printf("  Workflow ID: %s", we.GetID())
	log.Printf("  Run ID: %s", we.GetRunID())
	log.Printf("  Order ID: %s", order.ID)
	log.Printf("  Amount: $%.2f", order.Amount)
	log.Printf("  Items: %v", order.Items)
	log.Println()
	log.Println("To query the workflow status, run:")
	log.Printf("  go run starter/main.go -action=query -workflow-id=%s", we.GetID())
	log.Println()
	log.Println("To expedite the order, run:")
	log.Printf("  go run starter/main.go -action=expedite -workflow-id=%s", we.GetID())
	log.Println()
	log.Println("To cancel the order, run:")
	log.Printf("  go run starter/main.go -action=cancel -workflow-id=%s", we.GetID())
}

func sendSignal(ctx context.Context, c client.Client, workflowID, signalName string) {
	if workflowID == "" {
		log.Fatal("workflow-id is required for signal operations")
	}

	err := c.SignalWorkflow(ctx, workflowID, "", signalName, nil)
	if err != nil {
		log.Fatalf("Unable to signal workflow: %v", err)
	}

	log.Printf("Signal '%s' sent successfully to workflow: %s", signalName, workflowID)
}

func queryWorkflow(ctx context.Context, c client.Client, workflowID string) {
	if workflowID == "" {
		log.Fatal("workflow-id is required for query operations")
	}

	// Create a context with longer timeout for query
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := c.QueryWorkflow(queryCtx, workflowID, "", "getStatus")
	if err != nil {
		log.Fatalf("Unable to query workflow: %v", err)
	}

	var status models.OrderStatus
	if err := response.Get(&status); err != nil {
		log.Fatalf("Unable to decode query result: %v", err)
	}

	// Pretty print the status
	statusJSON, _ := json.MarshalIndent(status, "", "  ")
	log.Println("Workflow Status:")
	fmt.Println(string(statusJSON))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func loadEncryptionKey() []byte {
	keyFile := ".encryption.key"

	// Try to read existing key
	if key, err := os.ReadFile(keyFile); err == nil && len(key) == 32 {
		log.Println("Using existing encryption key")
		return key
	}

	// Generate new key if not found
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
