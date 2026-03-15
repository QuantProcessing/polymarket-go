package clob

import (
	"fmt"
)

// Common errors
var (
	ErrL1AuthUnavailable  = fmt.Errorf("L1 authentication not available")
	ErrL2AuthNotAvailable = fmt.Errorf("L2 authentication not available")
	ErrBuilderAuthFailed  = fmt.Errorf("builder authentication failed")
	ErrInvalidSignature   = fmt.Errorf("invalid signature")
	ErrInsufficientBalance = fmt.Errorf("insufficient balance/allowance")
	ErrOrderNotFound      = fmt.Errorf("order not found")
	ErrMarketNotFound     = fmt.Errorf("market not found")
	ErrNotConnected       = fmt.Errorf("WebSocket not connected")
)

// APIError represents a structured API error response
type APIError struct {
	StatusCode int
	Message    string
	Code       int
}

func (e *APIError) Error() string {
	if e.Code > 0 {
		return fmt.Sprintf("API error: status %d, code %d, msg: %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("API error: status %d, msg: %s", e.StatusCode, e.Message)
}

// NewAPIError creates a new API error
func NewAPIError(statusCode int, message string, code int) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Message:    message,
		Code:       code,
	}
}
