package models

import "time"

// Order represents an order in the system
type Order struct {
	ID        string    `json:"id"`
	Items     []string  `json:"items"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// OrderStatus represents the current state of an order
type OrderStatus struct {
	OrderID       string    `json:"order_id"`
	Status        string    `json:"status"`
	Stage         string    `json:"stage"`
	IsExpedited   bool      `json:"is_expedited"`
	PaymentStatus string    `json:"payment_status"`
	LastUpdated   time.Time `json:"last_updated"`
}

// ValidationRequest represents a request to validate an order
type ValidationRequest struct {
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
	Items   []string `json:"items"`
}

// ValidationResponse represents the response from validation service
type ValidationResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}

// PaymentRequest represents a payment processing request
type PaymentRequest struct {
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
}

// PaymentResponse represents a payment processing response
type PaymentResponse struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"transaction_id"`
	Message       string `json:"message"`
}

// Signal types
const (
	SignalCancel   = "cancel"
	SignalExpedite = "expedite"
)

// Order statuses
const (
	StatusPending    = "pending"
	StatusValidating = "validating"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusCancelled  = "cancelled"
	StatusFailed     = "failed"
)

// Stages
const (
	StageValidation = "validation"
	StagePayment    = "payment"
	StageProcessing = "processing"
	StageCompleted  = "completed"
)
