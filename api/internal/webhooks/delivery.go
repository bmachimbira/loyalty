package webhooks

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// DeliveryService handles webhook delivery with worker pools and retry logic
type DeliveryService struct {
	queries     *db.Queries
	dbConn      *sql.DB
	client      *http.Client
	queue       chan *DeliveryJob
	workers     int
	stopChan    chan struct{}
	logger      *slog.Logger
}

// DeliveryJob represents a webhook delivery job
type DeliveryJob struct {
	WebhookID pgtype.UUID
	TenantID  uuid.UUID
	Event     string
	Payload   EventPayload
}

// NewDeliveryService creates a new webhook delivery service
func NewDeliveryService(queries *db.Queries, dbConn *sql.DB, logger *slog.Logger) *DeliveryService {
	return &DeliveryService{
		queries:  queries,
		dbConn:   dbConn,
		client:   &http.Client{Timeout: 30 * time.Second},
		queue:    make(chan *DeliveryJob, 100),
		workers:  5,
		stopChan: make(chan struct{}),
		logger:   logger,
	}
}

// Start starts the webhook delivery workers
func (s *DeliveryService) StartWorkers() {
	for i := 0; i < s.workers; i++ {
		go s.worker(i)
	}
	s.logger.Info("webhook delivery workers started", "count", s.workers)
}

// Stop gracefully stops all workers
func (s *DeliveryService) Stop() {
	close(s.stopChan)
	s.logger.Info("webhook delivery service stopping")
}

// SendWebhook queues a webhook for async delivery
func (s *DeliveryService) SendWebhook(ctx context.Context, tenantID uuid.UUID, eventType string, payload EventPayload) error {
	// Set tenant context for RLS
	_, err := s.dbConn.ExecContext(ctx, "SET LOCAL app.tenant_id = $1", tenantID)
	if err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	// Get all webhooks subscribed to this event
	webhooks, err := s.queries.GetWebhooksByEvent(ctx, eventType)
	if err != nil {
		return fmt.Errorf("failed to get webhooks: %w", err)
	}

	// Queue delivery for each webhook
	for _, webhook := range webhooks {
		job := &DeliveryJob{
			WebhookID: webhook.ID,
			TenantID:  tenantID,
			Event:     eventType,
			Payload:   payload,
		}

		select {
		case s.queue <- job:
			s.logger.Debug("webhook queued", "webhook_id", webhook.ID, "event", eventType)
		case <-time.After(5 * time.Second):
			s.logger.Error("webhook queue full, dropping job", "webhook_id", webhook.ID)
		}
	}

	return nil
}

// worker processes webhook delivery jobs
func (s *DeliveryService) worker(id int) {
	s.logger.Debug("webhook worker started", "worker_id", id)

	for {
		select {
		case <-s.stopChan:
			s.logger.Debug("webhook worker stopped", "worker_id", id)
			return
		case job := <-s.queue:
			s.deliverWebhook(context.Background(), job)
		}
	}
}

// deliverWebhook delivers a webhook with retry logic
func (s *DeliveryService) deliverWebhook(ctx context.Context, job *DeliveryJob) {
	// Set tenant context
	ctx = context.WithValue(ctx, "tenant_id", job.TenantID)
	_, err := s.dbConn.ExecContext(ctx, "SET LOCAL app.tenant_id = $1", job.TenantID)
	if err != nil {
		s.logger.Error("failed to set tenant context", "error", err)
		return
	}

	// Get webhook config
	webhook, err := s.queries.GetWebhookByID(ctx, job.WebhookID)
	if err != nil {
		s.logger.Error("failed to get webhook", "webhook_id", job.WebhookID, "error", err)
		return
	}

	if !webhook.Active {
		s.logger.Debug("webhook inactive, skipping", "webhook_id", webhook.ID)
		return
	}

	// Marshal payload
	body, err := json.Marshal(job.Payload)
	if err != nil {
		s.logger.Error("failed to marshal payload", "error", err)
		return
	}

	// Retry delivery up to 3 times with exponential backoff
	maxAttempts := 3
	delay := 1 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		status, statusCode, respBody, err := s.sendRequest(ctx, webhook.Url, webhook.Secret, job.Event, body)

		// Record delivery attempt
		_, dbErr := s.queries.InsertWebhookDelivery(ctx, db.InsertWebhookDeliveryParams{
			WebhookID:    webhook.ID,
			EventType:    job.Event,
			Attempt:      int32(attempt),
			Status:       status,
			ResponseCode: pgtype.Int4{Int32: int32(statusCode), Valid: statusCode > 0},
			ResponseBody: pgtype.Text{String: respBody, Valid: respBody != ""},
			ErrorMessage: pgtype.Text{String: getErrorMessage(err), Valid: err != nil},
		})

		if dbErr != nil {
			s.logger.Error("failed to record webhook delivery", "error", dbErr)
		}

		if err == nil && statusCode >= 200 && statusCode < 300 {
			s.logger.Info("webhook delivered successfully",
				"webhook_id", webhook.ID,
				"event", job.Event,
				"attempt", attempt,
				"status_code", statusCode)
			return
		}

		// Log failure
		s.logger.Warn("webhook delivery failed",
			"webhook_id", webhook.ID,
			"event", job.Event,
			"attempt", attempt,
			"status_code", statusCode,
			"error", err)

		// Don't retry on final attempt
		if attempt == maxAttempts {
			break
		}

		// Exponential backoff
		time.Sleep(delay)
		delay *= 2
	}

	s.logger.Error("webhook delivery failed after all attempts",
		"webhook_id", webhook.ID,
		"event", job.Event,
		"attempts", maxAttempts)
}

// sendRequest sends a single HTTP request to the webhook endpoint
func (s *DeliveryService) sendRequest(ctx context.Context, url, secret, event string, body []byte) (status string, statusCode int, respBody string, err error) {
	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "failed", 0, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Event", event)
	req.Header.Set("X-Signature", GenerateSignature(secret, body))
	req.Header.Set("User-Agent", "ZW-Loyalty-Platform/1.0")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return "failed", 0, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // Max 1MB
	if err != nil {
		return "failed", resp.StatusCode, "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return "success", resp.StatusCode, string(responseBody), nil
	}

	return "failed", resp.StatusCode, string(responseBody), fmt.Errorf("non-2xx status code: %d", resp.StatusCode)
}

// NotifyRewardIssued sends reward.issued webhook notifications
func (s *DeliveryService) NotifyRewardIssued(ctx context.Context, tenantID uuid.UUID, data RewardIssuedData) error {
	payload := NewRewardIssuedEvent(tenantID, data)
	return s.SendWebhook(ctx, tenantID, EventRewardIssued, payload)
}

// NotifyRewardRedeemed sends reward.redeemed webhook notifications
func (s *DeliveryService) NotifyRewardRedeemed(ctx context.Context, tenantID uuid.UUID, data RewardRedeemedData) error {
	payload := NewRewardRedeemedEvent(tenantID, data)
	return s.SendWebhook(ctx, tenantID, EventRewardRedeemed, payload)
}

// NotifyRewardExpired sends reward.expired webhook notifications
func (s *DeliveryService) NotifyRewardExpired(ctx context.Context, tenantID uuid.UUID, data RewardExpiredData) error {
	payload := NewRewardExpiredEvent(tenantID, data)
	return s.SendWebhook(ctx, tenantID, EventRewardExpired, payload)
}

// NotifyCustomerEnrolled sends customer.enrolled webhook notifications
func (s *DeliveryService) NotifyCustomerEnrolled(ctx context.Context, tenantID uuid.UUID, data CustomerEnrolledData) error {
	payload := NewCustomerEnrolledEvent(tenantID, data)
	return s.SendWebhook(ctx, tenantID, EventCustomerEnrolled, payload)
}

// NotifyBudgetThreshold sends budget.threshold webhook notifications
func (s *DeliveryService) NotifyBudgetThreshold(ctx context.Context, tenantID uuid.UUID, data BudgetThresholdData) error {
	payload := NewBudgetThresholdEvent(tenantID, data)
	return s.SendWebhook(ctx, tenantID, EventBudgetThreshold, payload)
}

func getErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
