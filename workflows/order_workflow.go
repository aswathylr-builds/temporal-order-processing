package workflows

import (
	"fmt"
	"time"

	"github.com/aswathylr-builds/temporal-order-processing/models"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// RetryPolicy configuration
type RetryPolicy = temporal.RetryPolicy

const (
	OrderWorkflowName = "OrderProcessingWorkflow"
	PaymentWorkflowName = "PaymentWorkflow"
)

// OrderWorkflow is the main workflow for processing orders
func OrderWorkflow(ctx workflow.Context, order models.Order) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Order workflow started", "order_id", order.ID)

	// Initialize workflow state
	state := &models.OrderStatus{
		OrderID:       order.ID,
		Status:        models.StatusPending,
		Stage:         models.StageValidation,
		IsExpedited:   false,
		PaymentStatus: "pending",
		LastUpdated:   workflow.Now(ctx),
	}

	// Set up signal and query handlers
	cancelRequested := false

	// Signal handler for cancellation
	cancelChannel := workflow.GetSignalChannel(ctx, models.SignalCancel)
	workflow.Go(ctx, func(ctx workflow.Context) {
		for {
			cancelChannel.Receive(ctx, nil)
			logger.Info("Cancel signal received", "order_id", order.ID)
			cancelRequested = true
		}
	})

	// Signal handler for expedited processing
	expediteChannel := workflow.GetSignalChannel(ctx, models.SignalExpedite)
	workflow.Go(ctx, func(ctx workflow.Context) {
		for {
			expediteChannel.Receive(ctx, nil)
			logger.Info("Expedite signal received", "order_id", order.ID)
			state.IsExpedited = true
			state.LastUpdated = workflow.Now(ctx)
		}
	})

	// Query handler for workflow status
	err := workflow.SetQueryHandler(ctx, "getStatus", func() (*models.OrderStatus, error) {
		return state, nil
	})
	if err != nil {
		logger.Error("Failed to register query handler", "error", err)
		return err
	}

	// Check for cancellation
	if cancelRequested {
		state.Status = models.StatusCancelled
		state.LastUpdated = workflow.Now(ctx)
		logger.Info("Order cancelled", "order_id", order.ID)
		return nil
	}

	// Configure activity options with retry policy (optimized for demo)
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

	// Step 1: Validate Order
	state.Status = models.StatusValidating
	state.Stage = models.StageValidation
	state.LastUpdated = workflow.Now(ctx)
	logger.Info("Starting order validation", "order_id", order.ID)

	var validationResp models.ValidationResponse
	err = workflow.ExecuteActivity(ctx, "ValidateOrder", order).Get(ctx, &validationResp)
	if err != nil {
		state.Status = models.StatusFailed
		state.LastUpdated = workflow.Now(ctx)
		logger.Error("Order validation failed", "order_id", order.ID, "error", err)
		return err
	}

	if !validationResp.Valid {
		state.Status = models.StatusFailed
		state.LastUpdated = workflow.Now(ctx)
		logger.Error("Order validation rejected", "order_id", order.ID, "reason", validationResp.Message)
		return fmt.Errorf("order validation failed: %s", validationResp.Message)
	}

	// Check for cancellation after validation
	if cancelRequested {
		state.Status = models.StatusCancelled
		state.LastUpdated = workflow.Now(ctx)
		logger.Info("Order cancelled after validation", "order_id", order.ID)
		return nil
	}

	// Step 2: Process payment with versioning for backward compatibility
	state.Stage = models.StagePayment
	state.LastUpdated = workflow.Now(ctx)

	// Workflow versioning: Allows safe evolution from activity to child workflow
	// Version 1 (DefaultVersion): Used activity directly (old behavior)
	// Version 2: Uses child workflow (new behavior)
	version := workflow.GetVersion(ctx, "payment-processing-change", workflow.DefaultVersion, 2)

	var paymentResp *models.PaymentResponse

	if version == workflow.DefaultVersion {
		// OLD VERSION: Process payment using activity directly
		// This path ensures running workflows continue to work when we deploy new code
		logger.Info("Processing payment via activity (legacy version)", "order_id", order.ID)

		paymentReq := models.PaymentRequest{
			OrderID: order.ID,
			Amount:  order.Amount,
		}

		var activityResp models.PaymentResponse
		err = workflow.ExecuteActivity(ctx, "ProcessPayment", paymentReq).Get(ctx, &activityResp)
		if err != nil {
			state.Status = models.StatusFailed
			state.PaymentStatus = "failed"
			state.LastUpdated = workflow.Now(ctx)
			logger.Error("Payment processing failed", "order_id", order.ID, "error", err)
			return err
		}
		paymentResp = &activityResp
		logger.Info("Payment completed via activity", "order_id", order.ID, "transaction_id", paymentResp.TransactionID)

	} else {
		// NEW VERSION: Process payment using child workflow
		// All new workflow executions will use this path
		logger.Info("Processing payment via child workflow (v2)", "order_id", order.ID)

		// Configure child workflow options
		childWorkflowOptions := workflow.ChildWorkflowOptions{
			WorkflowID:               fmt.Sprintf("payment-%s", order.ID),
			WorkflowExecutionTimeout: 2 * time.Minute,
			RetryPolicy: &RetryPolicy{
				InitialInterval:    time.Second,
				BackoffCoefficient: 2.0,
				MaximumInterval:    10 * time.Second,
				MaximumAttempts:    3,
			},
		}
		childCtx := workflow.WithChildOptions(ctx, childWorkflowOptions)

		// Execute payment as child workflow
		err = workflow.ExecuteChildWorkflow(childCtx, PaymentWorkflowName, order).Get(ctx, &paymentResp)
		if err != nil {
			state.Status = models.StatusFailed
			state.PaymentStatus = "failed"
			state.LastUpdated = workflow.Now(ctx)
			logger.Error("Payment child workflow failed", "order_id", order.ID, "error", err)
			return err
		}
		logger.Info("Payment completed via child workflow", "order_id", order.ID, "transaction_id", paymentResp.TransactionID)
	}

	state.PaymentStatus = "completed"

	// Check for cancellation after payment
	if cancelRequested {
		state.Status = models.StatusCancelled
		state.LastUpdated = workflow.Now(ctx)
		logger.Info("Order cancelled after payment", "order_id", order.ID)
		return nil
	}

	// Step 3: Process Order
	state.Status = models.StatusProcessing
	state.Stage = models.StageProcessing
	state.LastUpdated = workflow.Now(ctx)
	logger.Info("Starting order processing", "order_id", order.ID, "expedited", state.IsExpedited)

	err = workflow.ExecuteActivity(ctx, "ProcessOrder", order, state.IsExpedited).Get(ctx, nil)
	if err != nil {
		state.Status = models.StatusFailed
		state.LastUpdated = workflow.Now(ctx)
		logger.Error("Order processing failed", "order_id", order.ID, "error", err)
		return err
	}

	// Step 4: Notify completion
	err = workflow.ExecuteActivity(ctx, "NotifyOrderComplete", order).Get(ctx, nil)
	if err != nil {
		logger.Warn("Notification failed but order completed", "order_id", order.ID, "error", err)
		// Don't fail the workflow if notification fails
	}

	// Mark as completed
	state.Status = models.StatusCompleted
	state.Stage = models.StageCompleted
	state.LastUpdated = workflow.Now(ctx)
	logger.Info("Order workflow completed successfully", "order_id", order.ID)

	return nil
}
