package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	whatsappAPIBaseURL = "https://graph.facebook.com/v18.0"
	maxRetries         = 3
	retryDelay         = time.Second * 2
)

// MessageSender handles sending messages via WhatsApp Business API
type MessageSender struct {
	client      *http.Client
	phoneID     string
	accessToken string
}

// NewMessageSender creates a new message sender
func NewMessageSender(phoneID, accessToken string) *MessageSender {
	return &MessageSender{
		client: &http.Client{
			Timeout: time.Second * 30,
		},
		phoneID:     phoneID,
		accessToken: accessToken,
	}
}

// SendText sends a text message
func (s *MessageSender) SendText(ctx context.Context, to, text string) error {
	req := SendMessageRequest{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               to,
		Type:             "text",
		Text: &TextPayload{
			PreviewURL: false,
			Body:       text,
		},
	}

	return s.send(ctx, req)
}

// SendTemplate sends a template message
func (s *MessageSender) SendTemplate(ctx context.Context, to, templateName string, params map[string]string) error {
	// Build template parameters
	var components []TemplateComponentPayload
	if len(params) > 0 {
		parameters := make([]TemplateParameterPayload, 0, len(params))
		for _, value := range params {
			parameters = append(parameters, TemplateParameterPayload{
				Type: "text",
				Text: value,
			})
		}

		components = append(components, TemplateComponentPayload{
			Type:       "body",
			Parameters: parameters,
		})
	}

	req := SendMessageRequest{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               to,
		Type:             "template",
		Template: &TemplatePayload{
			Name: templateName,
			Language: LanguagePayload{
				Code: "en",
			},
			Components: components,
		},
	}

	return s.send(ctx, req)
}

// SendInteractive sends an interactive message with buttons
func (s *MessageSender) SendInteractive(ctx context.Context, to, bodyText string, buttons []ButtonPayload) error {
	req := SendMessageRequest{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               to,
		Type:             "interactive",
		Interactive: &InteractivePayload{
			Type: "button",
			Body: BodyPayload{
				Text: bodyText,
			},
			Action: ActionPayload{
				Buttons: buttons,
			},
		},
	}

	return s.send(ctx, req)
}

// SendList sends an interactive list message
func (s *MessageSender) SendList(ctx context.Context, to, bodyText, buttonText string, sections []SectionPayload) error {
	req := SendMessageRequest{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               to,
		Type:             "interactive",
		Interactive: &InteractivePayload{
			Type: "list",
			Body: BodyPayload{
				Text: bodyText,
			},
			Action: ActionPayload{
				Button:   buttonText,
				Sections: sections,
			},
		},
	}

	return s.send(ctx, req)
}

// send sends a message request to WhatsApp API with retry logic
func (s *MessageSender) send(ctx context.Context, payload SendMessageRequest) error {
	url := fmt.Sprintf("%s/%s/messages", whatsappAPIBaseURL, s.phoneID)

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay * time.Duration(attempt-1)):
			}
		}

		// Marshal payload
		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal message payload: %w", err)
		}

		// Create request
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.accessToken)

		// Send request
		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to send request: %w", err)
			slog.Warn("WhatsApp API request failed, retrying",
				"attempt", attempt,
				"error", err,
			)
			continue
		}

		// Read response
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			slog.Warn("Failed to read WhatsApp API response",
				"attempt", attempt,
				"error", err,
			)
			continue
		}

		// Check status code
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// Success
			var result SendMessageResponse
			if err := json.Unmarshal(respBody, &result); err != nil {
				slog.Warn("Failed to parse successful response", "error", err)
			} else {
				slog.Info("WhatsApp message sent successfully",
					"to", payload.To,
					"type", payload.Type,
					"msg_id", result.Messages[0].ID,
				)
			}
			return nil
		}

		// Handle error response
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			lastErr = fmt.Errorf("WhatsApp API error (status %d): %s", resp.StatusCode, string(respBody))
		} else {
			lastErr = fmt.Errorf("WhatsApp API error (code %d): %s", errResp.Error.Code, errResp.Error.Message)
		}

		// Check if error is retryable
		if resp.StatusCode >= 500 || resp.StatusCode == 429 {
			slog.Warn("WhatsApp API returned retryable error",
				"attempt", attempt,
				"status", resp.StatusCode,
				"error", lastErr,
			)
			continue
		}

		// Non-retryable error
		break
	}

	slog.Error("WhatsApp message send failed after all retries",
		"attempts", maxRetries,
		"error", lastErr,
	)
	return lastErr
}

// MarkAsRead marks a message as read
func (s *MessageSender) MarkAsRead(ctx context.Context, messageID string) error {
	url := fmt.Sprintf("%s/%s/messages", whatsappAPIBaseURL, s.phoneID)

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"status":            "read",
		"message_id":        messageID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.accessToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mark as read failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}
