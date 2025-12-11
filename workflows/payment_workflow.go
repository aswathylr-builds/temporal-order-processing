package workflows

import (
	"time"

	"github.com/aswathylr-builds/temporal-order-processing/models"
	"go.temporal.io/sdk/workflow"
)

// PaymentWorkflow is a child workflow that handles payment processing
func PaymentWorkflow(ctx workflow.Context, order models.Order) (*models.PaymentResponse, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Payment workflow started", "order_id", order.ID)

	// Configure activity options (optimized for demo)
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout:    10 * time.Second,
		ScheduleToStartTimeout: 5 * time.Second,
		RetryPolicy: &RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    10 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Process payment
	paymentReq := models.PaymentRequest{
		OrderID: order.ID,
		Amount:  order.Amount,
	}

	var paymentResp models.PaymentResponse
	err := workflow.ExecuteActivity(ctx, "ProcessPayment", paymentReq).Get(ctx, &paymentResp)
	if err != nil {
		logger.Error("Payment processing failed", "order_id", order.ID, "error", err)
		return nil, err
	}

	logger.Info("Payment workflow completed", "order_id", order.ID, "transaction_id", paymentResp.TransactionID)
	return &paymentResp, nil
}
