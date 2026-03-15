package clob

import (
	"context"
	"fmt"
)

// Notification endpoints
const (
	EndpointNotifications     = "/notifications"
	EndpointDropNotifications = "/notifications"
)

// GetNotifications fetches user notifications
func (c *ClobClient) GetNotifications(ctx context.Context) ([]*Notification, error) {
	if err := c.EnsureAuth(ctx); err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointNotifications)
	
	var notifications []*Notification
	if err := c.get(ctx, url, nil, &notifications, true); err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}

	return notifications, nil
}

// DropNotifications marks notifications as read/dropped
func (c *ClobClient) DropNotifications(ctx context.Context, params DropNotificationParams) error {
	if err := c.EnsureAuth(ctx); err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, EndpointDropNotifications)
	
	// Convert params to query string
	queryParams := make(map[string]string)
	if len(params.IDs) > 0 {
		// Join IDs with commas
		ids := ""
		for i, id := range params.IDs {
			if i > 0 {
				ids += ","
			}
			ids += id
		}
		queryParams["ids"] = ids
	}
	
	if err := c.delete(ctx, url, params, nil, true); err != nil {
		return fmt.Errorf("failed to drop notifications: %w", err)
	}

	return nil
}
