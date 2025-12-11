package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aswathylr-builds/temporal-order-processing/activities"
	"github.com/aswathylr-builds/temporal-order-processing/models"
	"github.com/aswathylr-builds/temporal-order-processing/workflows"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestValidateOrder_Success(t *testing.T) {
	// Create mock HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/validate", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Decode request
		var req models.ValidationRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Validate request
		assert.NotEmpty(t, req.OrderID)
		assert.Greater(t, req.Amount, 0.0)

		// Send success response
		resp := models.ValidationResponse{
			Valid:   true,
			Message: "Order validated successfully",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	// Create activities with mock server URL
	orderActivities := activities.NewOrderActivities(mockServer.URL + "/validate")

	// Create test order
	order := models.Order{
		ID:        "TEST-001",
		Items:     []string{"item1", "item2"},
		Amount:    100.0,
		Status:    models.StatusPending,
		CreatedAt: time.Now(),
	}

	// Test the activity
	ctx := context.Background()
	resp, err := orderActivities.ValidateOrder(ctx, order)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Valid)
	assert.Equal(t, "Order validated successfully", resp.Message)
}

func TestValidateOrder_ValidationFailed(t *testing.T) {
	// Create mock HTTP server that returns validation failure
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := models.ValidationResponse{
			Valid:   false,
			Message: "Amount exceeds maximum allowed",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	// Create activities with mock server URL
	orderActivities := activities.NewOrderActivities(mockServer.URL + "/validate")

	// Create test order with high amount
	order := models.Order{
		ID:        "TEST-002",
		Items:     []string{"item1"},
		Amount:    15000.0,
		Status:    models.StatusPending,
		CreatedAt: time.Now(),
	}

	// Test the activity
	ctx := context.Background()
	resp, err := orderActivities.ValidateOrder(ctx, order)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.False(t, resp.Valid)
	assert.Contains(t, resp.Message, "exceeds")
}

func TestValidateOrder_HTTPError(t *testing.T) {
	// Create mock HTTP server that returns an error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer mockServer.Close()

	// Create activities with mock server URL
	orderActivities := activities.NewOrderActivities(mockServer.URL + "/validate")

	// Create test order
	order := models.Order{
		ID:        "TEST-003",
		Items:     []string{"item1"},
		Amount:    100.0,
		Status:    models.StatusPending,
		CreatedAt: time.Now(),
	}

	// Test the activity
	ctx := context.Background()
	resp, err := orderActivities.ValidateOrder(ctx, order)

	// Assertions
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "validation service returned status 500")
}

func TestProcessOrder(t *testing.T) {
	// Create activities
	orderActivities := activities.NewOrderActivities("http://mock-url")

	// Create test order
	order := models.Order{
		ID:        "TEST-004",
		Items:     []string{"item1", "item2"},
		Amount:    100.0,
		Status:    models.StatusPending,
		CreatedAt: time.Now(),
	}

	// Test without expedited processing
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	err := orderActivities.ProcessOrder(ctx, order, false)
	duration := time.Since(start)

	// Assertions
	require.NoError(t, err)
	assert.GreaterOrEqual(t, duration, 2*time.Second)
}

func TestProcessOrder_Expedited(t *testing.T) {
	// Create activities
	orderActivities := activities.NewOrderActivities("http://mock-url")

	// Create test order
	order := models.Order{
		ID:        "TEST-005",
		Items:     []string{"item1", "item2"},
		Amount:    100.0,
		Status:    models.StatusPending,
		CreatedAt: time.Now(),
	}

	// Test with expedited processing
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	err := orderActivities.ProcessOrder(ctx, order, true)
	duration := time.Since(start)

	// Assertions
	require.NoError(t, err)
	assert.GreaterOrEqual(t, duration, 1*time.Second)
	assert.Less(t, duration, 2*time.Second)
}

func TestProcessPayment(t *testing.T) {
	// Create activities
	orderActivities := activities.NewOrderActivities("http://mock-url")

	// Create payment request
	paymentReq := models.PaymentRequest{
		OrderID: "TEST-006",
		Amount:  250.50,
	}

	// Test payment processing
	ctx := context.Background()
	resp, err := orderActivities.ProcessPayment(ctx, paymentReq)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.TransactionID)
	assert.Contains(t, resp.TransactionID, "TXN-TEST-006")
	assert.Equal(t, "Payment processed successfully", resp.Message)
}

func TestNotifyOrderComplete(t *testing.T) {
	// Create activities
	orderActivities := activities.NewOrderActivities("http://mock-url")

	// Create test order
	order := models.Order{
		ID:        "TEST-007",
		Items:     []string{"item1"},
		Amount:    100.0,
		Status:    models.StatusCompleted,
		CreatedAt: time.Now(),
	}

	// Test notification
	ctx := context.Background()
	err := orderActivities.NotifyOrderComplete(ctx, order)

	// Assertions
	require.NoError(t, err)
}

// Test workflow using Temporal test suite
func TestOrderWorkflow(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register activities
	orderActivities := activities.NewOrderActivities("http://mock-url")
	env.RegisterActivity(orderActivities.ValidateOrder)
	env.RegisterActivity(orderActivities.ProcessPayment)
	env.RegisterActivity(orderActivities.ProcessOrder)
	env.RegisterActivity(orderActivities.NotifyOrderComplete)

	// Mock the ValidateOrder activity
	env.OnActivity(orderActivities.ValidateOrder, mock.Anything, mock.Anything).Return(&models.ValidationResponse{
		Valid:   true,
		Message: "Order validated successfully",
	}, nil)

	// Mock the ProcessPayment activity
	env.OnActivity(orderActivities.ProcessPayment, mock.Anything, mock.Anything).Return(&models.PaymentResponse{
		Success:       true,
		TransactionID: "TXN-TEST-123",
		Message:       "Payment processed successfully",
	}, nil)

	// Mock the ProcessOrder activity
	env.OnActivity(orderActivities.ProcessOrder, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Mock the NotifyOrderComplete activity
	env.OnActivity(orderActivities.NotifyOrderComplete, mock.Anything, mock.Anything).Return(nil)

	// Create test order
	order := models.Order{
		ID:        "TEST-WF-001",
		Items:     []string{"item1", "item2"},
		Amount:    100.0,
		Status:    models.StatusPending,
		CreatedAt: time.Now(),
	}

	// Register workflows
	env.RegisterWorkflow(workflows.OrderWorkflow)
	env.RegisterWorkflow(workflows.PaymentWorkflow)

	// Execute workflow
	env.ExecuteWorkflow(workflows.OrderWorkflow, order)

	// Verify workflow completed successfully
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}
