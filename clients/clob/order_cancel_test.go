package clob

import (
	"context"
	"testing"
)

func TestCancelOrdersMethods(t *testing.T) {
	// These tests verify the API call structure
	// Actual execution would require real orders

	t.Run("CancelOrders_Structure", func(t *testing.T) {
		// Verify the method signature exists
		client := &ClobClient{}
		orderIDs := []string{"0xtest1", "0xtest2"}

		_, err := client.CancelOrders(context.Background(), orderIDs)
		// Expected to fail with auth error since no creds
		if err == nil {
			t.Error("Expected auth error for unauthenticated call")
		}
		t.Log("✓ CancelOrders method structure verified")
	})

	t.Run("CancelAllOrders_Structure", func(t *testing.T) {
		// Verify the method signature exists
		client := &ClobClient{}

		err := client.CancelAllOrders(context.Background())
		// Expected to fail with auth error since no creds
		if err == nil {
			t.Error("Expected auth error for unauthenticated call")
		}
		t.Log("✓ CancelAllOrders method structure verified")
	})
}
