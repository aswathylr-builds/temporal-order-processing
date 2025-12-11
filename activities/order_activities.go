package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aswathylr-builds/temporal-order-processing/models"
	"go.temporal.io/sdk/activity"
)

// OrderActivities contains all order-related activities
type OrderActivities struct {
	HTTPClient      *http.Client
	ValidationURL   string
}

// NewOrderActivities creates a new instance of OrderActivities
func NewOrderActivities(validationURL string) *OrderActivities {
	return &OrderActivities{
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		ValidationURL: validationURL,
	}
}

// ValidateOrder validates an order by calling an external service
func (a *OrderActivities) ValidateOrder(ctx context.Context, order models.Order) (*models.ValidationResponse, error) {
	// Try to get activity logger, but don't panic if not in activity context
	if activity.IsActivity(ctx) {
		logger := activity.GetLogger(ctx)
		logger.Info("Validating order", "order_id", order.ID, "amount", order.Amount)
	}

	validationReq := models.ValidationRequest{
		OrderID: order.ID,
		Amount:  order.Amount,
		Items:   order.Items,
	}

	jsonData, err := json.Marshal(validationReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal validation request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.ValidationURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call validation service: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("validation service returned status %d: %s", resp.StatusCode, string(body))
	}

	var validationResp models.ValidationResponse
	if err := json.Unmarshal(body, &validationResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal validation response: %w", err)
	}

	if activity.IsActivity(ctx) {
		logger := activity.GetLogger(ctx)
		logger.Info("Order validation completed", "order_id", order.ID, "valid", validationResp.Valid)
	}
	return &validationResp, nil
}

// ProcessOrder processes the order (simulates business logic)
func (a *OrderActivities) ProcessOrder(ctx context.Context, order models.Order, isExpedited bool) error {
	isActivityCtx := activity.IsActivity(ctx)
	if isActivityCtx {
		logger := activity.GetLogger(ctx)
		logger.Info("Processing order", "order_id", order.ID, "expedited", isExpedited)
	}

	// Simulate processing time (reduced for demo)
	processingTime := 2 * time.Second
	if isExpedited {
		processingTime = 1 * time.Second
		if isActivityCtx {
			logger := activity.GetLogger(ctx)
			logger.Info("Expedited processing enabled", "order_id", order.ID)
		}
	}

	// Use activity heartbeat for long-running operations
	heartbeatInterval := 1 * time.Second
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	done := time.After(processingTime)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
			if isActivityCtx {
				logger := activity.GetLogger(ctx)
				logger.Info("Order processing completed", "order_id", order.ID)
			}
			return nil
		case <-ticker.C:
			if isActivityCtx {
				activity.RecordHeartbeat(ctx, "processing")
			}
		}
	}
}

// NotifyOrderComplete sends a notification that the order is complete
func (a *OrderActivities) NotifyOrderComplete(ctx context.Context, order models.Order) error {
	if activity.IsActivity(ctx) {
		logger := activity.GetLogger(ctx)
		logger.Info("Sending completion notification", "order_id", order.ID)
	}

	// Simulate notification logic (reduced for demo)
	time.Sleep(200 * time.Millisecond)

	if activity.IsActivity(ctx) {
		logger := activity.GetLogger(ctx)
		logger.Info("Notification sent successfully", "order_id", order.ID)
	}
	return nil
}

// ProcessPayment handles payment processing
func (a *OrderActivities) ProcessPayment(ctx context.Context, paymentReq models.PaymentRequest) (*models.PaymentResponse, error) {
	// Simulate payment processing (reduced for demo)
	time.Sleep(500 * time.Millisecond)

	// Generate a mock transaction ID
	transactionID := fmt.Sprintf("TXN-%s-%d", paymentReq.OrderID, time.Now().Unix())

	response := &models.PaymentResponse{
		Success:       true,
		TransactionID: transactionID,
		Message:       "Payment processed successfully",
	}

	return response, nil
}
