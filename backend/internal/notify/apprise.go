package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"container-updater/backend/internal/db"
	"container-updater/backend/internal/logger"
)

type ApprisePayload struct {
	URLs   string `json:"urls"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	Format string `json:"format"` // "text", "markdown", "html"
}

func SendNotification(ctx context.Context, title, body string) error {
	// 1. Fetch enabled notification services from DB
	services, err := db.ListNotificationServices()
	if err != nil {
		return fmt.Errorf("failed to retrieve notification services: %w", err)
	}

	// Filter enabled ones
	var urls []string
	for _, s := range services {
		if s.IsEnabled && s.AppriseURL != "" {
			urls = append(urls, s.AppriseURL)
		}
	}

	if len(urls) == 0 {
		logger.Log.Info("No notification services enabled, skipping dispatch.")
		return nil
	}

	// 2. Prepare payload
	appriseAPIURL := os.Getenv("APPRISE_API_URL")
	if appriseAPIURL == "" {
		appriseAPIURL = "http://localhost:8000" // Default fallback
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for _, appriseURL := range urls {
		payload := ApprisePayload{
			URLs:   appriseURL,
			Title:  title,
			Body:   body,
			Format: "markdown",
		}

		jsonBytes, err := json.Marshal(payload)
		if err != nil {
			logger.Log.Error("failed to marshal apprise payload", "error", err)
			continue
		}

		reqURL := fmt.Sprintf("%s/notify", appriseAPIURL)
		req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewBuffer(jsonBytes))
		if err != nil {
			logger.Log.Error("failed to create http request for apprise", "endpoint", reqURL, "error", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			logger.Log.Error("failed to deliver notification via apprise", "apprise_url", appriseURL, "error", err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			logger.Log.Error("apprise API returned error status code", "status", resp.Status, "apprise_url", appriseURL)
			continue
		}

		logger.Log.Info("Notification successfully dispatched via Apprise", "title", title, "service_url", appriseURL)
	}

	return nil
}
