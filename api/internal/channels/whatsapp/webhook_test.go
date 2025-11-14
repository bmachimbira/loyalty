package whatsapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHandler_Verify(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		mode           string
		token          string
		challenge      string
		verifyToken    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "valid verification",
			mode:           "subscribe",
			token:          "test-token",
			challenge:      "challenge123",
			verifyToken:    "test-token",
			expectedStatus: http.StatusOK,
			expectedBody:   "challenge123",
		},
		{
			name:           "invalid token",
			mode:           "subscribe",
			token:          "wrong-token",
			challenge:      "challenge123",
			verifyToken:    "test-token",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "wrong mode",
			mode:           "unsubscribe",
			token:          "test-token",
			challenge:      "challenge123",
			verifyToken:    "test-token",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{
				verifyToken: tt.verifyToken,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req := httptest.NewRequest("GET", "/webhook?hub.mode="+tt.mode+"&hub.verify_token="+tt.token+"&hub.challenge="+tt.challenge, nil)
			c.Request = req

			handler.Verify(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestVerifySignature(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		secret    string
		signature string
		expected  bool
	}{
		{
			name:      "valid signature",
			body:      "test body",
			secret:    "test-secret",
			signature: computeSignature("test body", "test-secret"),
			expected:  true,
		},
		{
			name:      "invalid signature",
			body:      "test body",
			secret:    "test-secret",
			signature: "sha256=invalid",
			expected:  false,
		},
		{
			name:      "wrong secret",
			body:      "test body",
			secret:    "test-secret",
			signature: computeSignature("test body", "wrong-secret"),
			expected:  false,
		},
		{
			name:      "empty secret skips verification",
			body:      "test body",
			secret:    "",
			signature: "sha256=anything",
			expected:  true, // Should pass when secret is empty (dev mode)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{
				appSecret: tt.secret,
			}

			result := handler.verifySignature([]byte(tt.body), tt.signature)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWebhookPayloadParsing(t *testing.T) {
	payload := `{
		"object": "whatsapp_business_account",
		"entry": [{
			"id": "123",
			"changes": [{
				"field": "messages",
				"value": {
					"messaging_product": "whatsapp",
					"metadata": {
						"display_phone_number": "263777123456",
						"phone_number_id": "123456"
					},
					"messages": [{
						"from": "263777123456",
						"id": "msg123",
						"timestamp": "1234567890",
						"type": "text",
						"text": {
							"body": "Hello"
						}
					}],
					"contacts": [{
						"profile": {
							"name": "John Doe"
						},
						"wa_id": "263777123456"
					}]
				}
			}]
		}]
	}`

	var result WebhookPayload
	err := json.Unmarshal([]byte(payload), &result)

	assert.NoError(t, err)
	assert.Equal(t, "whatsapp_business_account", result.Object)
	assert.Len(t, result.Entry, 1)
	assert.Len(t, result.Entry[0].Changes, 1)
	assert.Equal(t, "messages", result.Entry[0].Changes[0].Field)

	messages := result.Entry[0].Changes[0].Value.Messages
	assert.Len(t, messages, 1)
	assert.Equal(t, "263777123456", messages[0].From)
	assert.Equal(t, "text", messages[0].Type)
	assert.NotNil(t, messages[0].Text)
	assert.Equal(t, "Hello", messages[0].Text.Body)
}

func TestSendMessageRequest(t *testing.T) {
	tests := []struct {
		name    string
		request SendMessageRequest
		valid   bool
	}{
		{
			name: "text message",
			request: SendMessageRequest{
				MessagingProduct: "whatsapp",
				To:               "263777123456",
				Type:             "text",
				Text: &TextPayload{
					Body: "Hello",
				},
			},
			valid: true,
		},
		{
			name: "template message",
			request: SendMessageRequest{
				MessagingProduct: "whatsapp",
				To:               "263777123456",
				Type:             "template",
				Template: &TemplatePayload{
					Name: "welcome",
					Language: LanguagePayload{
						Code: "en",
					},
				},
			},
			valid: true,
		},
		{
			name: "interactive button message",
			request: SendMessageRequest{
				MessagingProduct: "whatsapp",
				To:               "263777123456",
				Type:             "interactive",
				Interactive: &InteractivePayload{
					Type: "button",
					Body: BodyPayload{
						Text: "Choose an option",
					},
					Action: ActionPayload{
						Buttons: []ButtonPayload{
							{
								Type: "reply",
								Reply: ReplyPayload{
									ID:    "btn1",
									Title: "Option 1",
								},
							},
						},
					},
				},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.request)
			assert.NoError(t, err)
			assert.NotEmpty(t, data)

			// Test unmarshaling
			var result SendMessageRequest
			err = json.Unmarshal(data, &result)
			assert.NoError(t, err)
			assert.Equal(t, tt.request.Type, result.Type)
		})
	}
}

// Helper function to compute HMAC-SHA256 signature
func computeSignature(body, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestGetMessageText(t *testing.T) {
	processor := &MessageProcessor{}

	tests := []struct {
		name     string
		message  Message
		expected string
	}{
		{
			name: "text message",
			message: Message{
				Type: "text",
				Text: &TextMsg{Body: "Hello"},
			},
			expected: "Hello",
		},
		{
			name: "button message",
			message: Message{
				Type:   "button",
				Button: &ButtonMsg{Text: "Button Text"},
			},
			expected: "Button Text",
		},
		{
			name: "image message",
			message: Message{
				Type:  "image",
				Image: &MediaMsg{ID: "123"},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.getMessageText(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommandParsing(t *testing.T) {
	tests := []struct {
		input           string
		expectedCommand string
		expectedArgs    []string
	}{
		{
			input:           "/help",
			expectedCommand: "/help",
			expectedArgs:    []string{},
		},
		{
			input:           "/redeem ABC123",
			expectedCommand: "/redeem",
			expectedArgs:    []string{"ABC123"},
		},
		{
			input:           "/redeem ABC123 extra",
			expectedCommand: "/redeem",
			expectedArgs:    []string{"ABC123", "extra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			parts := strings.Fields(tt.input)
			command := parts[0]
			args := parts[1:]

			assert.Equal(t, tt.expectedCommand, command)
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}
