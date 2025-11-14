# Channels Agent

## Mission
Implement communication channels (WhatsApp and USSD) for customer engagement.

## Prerequisites
- Go 1.21+
- WhatsApp Business API access
- Understanding of webhook patterns

## Tasks

### 1. WhatsApp Integration

#### Webhook Handler
**File**: `api/internal/channels/whatsapp/webhook.go`

```go
package whatsapp

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/json"
    "io"
    "net/http"
)

type Handler struct {
    queries      *db.Queries
    verifyToken  string
    appSecret    string
    processor    *MessageProcessor
}

// Verify handles GET request for webhook verification
func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
    mode := r.URL.Query().Get("hub.mode")
    token := r.URL.Query().Get("hub.verify_token")
    challenge := r.URL.Query().Get("hub.challenge")

    if mode == "subscribe" && token == h.verifyToken {
        w.WriteHeader(200)
        w.Write([]byte(challenge))
        return
    }

    w.WriteHeader(403)
}

// Webhook handles POST request with incoming messages
func (h *Handler) Webhook(w http.ResponseWriter, r *http.Request) {
    // Verify signature
    signature := r.Header.Get("X-Hub-Signature-256")
    body, _ := io.ReadAll(r.Body)

    if !h.verifySignature(body, signature) {
        w.WriteHeader(403)
        return
    }

    // Parse webhook payload
    var payload WebhookPayload
    json.Unmarshal(body, &payload)

    // Process messages
    for _, entry := range payload.Entry {
        for _, change := range entry.Changes {
            if change.Field == "messages" {
                for _, msg := range change.Value.Messages {
                    h.processor.ProcessMessage(r.Context(), msg)
                }
            }
        }
    }

    w.WriteHeader(200)
}

func (h *Handler) verifySignature(body []byte, signature string) bool {
    mac := hmac.New(sha256.New, []byte(h.appSecret))
    mac.Write(body)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}
```

#### Message Types
**File**: `api/internal/channels/whatsapp/types.go`

```go
package whatsapp

type WebhookPayload struct {
    Object string  `json:"object"`
    Entry  []Entry `json:"entry"`
}

type Entry struct {
    ID      string   `json:"id"`
    Changes []Change `json:"changes"`
}

type Change struct {
    Field string       `json:"field"`
    Value MessageValue `json:"value"`
}

type MessageValue struct {
    MessagingProduct string    `json:"messaging_product"`
    Messages         []Message `json:"messages"`
    Contacts         []Contact `json:"contacts"`
}

type Message struct {
    From      string      `json:"from"`
    ID        string      `json:"id"`
    Timestamp string      `json:"timestamp"`
    Type      string      `json:"type"`
    Text      *TextMsg    `json:"text,omitempty"`
    Button    *ButtonMsg  `json:"button,omitempty"`
}

type TextMsg struct {
    Body string `json:"body"`
}
```

#### Message Processor
**File**: `api/internal/channels/whatsapp/processor.go`

```go
package whatsapp

type MessageProcessor struct {
    queries *db.Queries
    sender  *MessageSender
}

func (p *MessageProcessor) ProcessMessage(ctx context.Context, msg Message) error {
    // Get or create session
    session, err := p.queries.GetWASessionByWAID(ctx, msg.From)
    if err != nil {
        // Create new session
        session = p.createSession(ctx, msg.From)
    }

    // Parse command
    command := p.parseCommand(msg)

    // Route to handler
    switch command {
    case "enroll":
        return p.handleEnroll(ctx, session, msg)
    case "rewards":
        return p.handleRewards(ctx, session, msg)
    case "balance":
        return p.handleBalance(ctx, session, msg)
    case "referral":
        return p.handleReferral(ctx, session, msg)
    default:
        return p.handleHelp(ctx, session)
    }
}

func (p *MessageProcessor) handleEnroll(ctx context.Context, session *db.WASession, msg Message) error {
    // Check if customer exists
    customer, err := p.queries.GetCustomerByPhone(ctx, db.GetCustomerByPhoneParams{
        TenantID:  session.TenantID,
        PhoneE164: session.PhoneE164,
    })

    if err != nil {
        // Create customer
        customer, err = p.queries.CreateCustomer(ctx, db.CreateCustomerParams{
            TenantID:  session.TenantID,
            PhoneE164: session.PhoneE164,
        })
    }

    // Record consent
    p.queries.RecordConsent(ctx, db.RecordConsentParams{
        TenantID:   session.TenantID,
        CustomerID: customer.ID,
        Channel:    "whatsapp",
        Purpose:    "loyalty",
        Granted:    true,
    })

    // Send welcome message
    return p.sender.SendTemplate(ctx, session.PhoneE164, "LOYALTY_WELCOME", map[string]string{
        "name": customer.PhoneE164,
    })
}

func (p *MessageProcessor) handleRewards(ctx context.Context, session *db.WASession, msg Message) error {
    // Get customer
    customer, _ := p.queries.GetCustomerByPhone(ctx, ...)

    // List active issuances
    issuances, err := p.queries.ListActiveIssuances(ctx, db.ListActiveIssuancesParams{
        TenantID:   session.TenantID,
        CustomerID: customer.ID,
    })

    if len(issuances) == 0 {
        return p.sender.SendText(ctx, session.PhoneE164, "You have no rewards yet. Keep shopping!")
    }

    // Format rewards list
    text := "Your Rewards:\n\n"
    for i, iss := range issuances {
        text += fmt.Sprintf("%d. %s - Code: %s\n", i+1, iss.RewardName, iss.Code)
    }

    return p.sender.SendText(ctx, session.PhoneE164, text)
}
```

#### Message Sender
**File**: `api/internal/channels/whatsapp/sender.go`

```go
package whatsapp

type MessageSender struct {
    client      *http.Client
    phoneID     string
    accessToken string
}

func (s *MessageSender) SendText(ctx context.Context, to, text string) error {
    payload := map[string]interface{}{
        "messaging_product": "whatsapp",
        "to":                to,
        "type":              "text",
        "text": map[string]string{
            "body": text,
        },
    }

    return s.send(ctx, payload)
}

func (s *MessageSender) SendTemplate(ctx context.Context, to, templateName string, params map[string]string) error {
    components := []map[string]interface{}{
        {
            "type": "body",
            "parameters": buildTemplateParams(params),
        },
    }

    payload := map[string]interface{}{
        "messaging_product": "whatsapp",
        "to":                to,
        "type":              "template",
        "template": map[string]interface{}{
            "name":       templateName,
            "language":   map[string]string{"code": "en"},
            "components": components,
        },
    }

    return s.send(ctx, payload)
}

func (s *MessageSender) send(ctx context.Context, payload map[string]interface{}) error {
    url := fmt.Sprintf("https://graph.facebook.com/v18.0/%s/messages", s.phoneID)

    body, _ := json.Marshal(payload)
    req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+s.accessToken)

    resp, err := s.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return errors.New("failed to send message")
    }

    return nil
}
```

#### Templates
**File**: `api/internal/channels/whatsapp/templates.go`

Define message templates:
- `LOYALTY_WELCOME`: Welcome message on enrollment
- `REWARD_ISSUED`: Notification when reward is issued
- `REWARD_REMINDER`: Reminder before reward expires
- `REWARD_REDEEMED`: Confirmation of redemption

### 2. USSD Integration

#### USSD Handler
**File**: `api/internal/channels/ussd/handler.go`

```go
package ussd

type Handler struct {
    queries *db.Queries
    menus   *MenuSystem
}

type USSDRequest struct {
    SessionID   string `json:"sessionId"`
    PhoneNumber string `json:"phoneNumber"`
    Text        string `json:"text"`
}

type USSDResponse struct {
    Type    string `json:"type"` // "CON" (continue) or "END" (end)
    Message string `json:"message"`
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
    var req USSDRequest
    json.NewDecoder(r.Body).Decode(&req)

    // Get or create session
    session, err := h.queries.GetUSSDSession(ctx, req.SessionID)
    if err != nil {
        session = h.createSession(ctx, req)
    }

    // Parse input
    input := req.Text

    // Get current menu
    menu := h.menus.GetMenu(session.State["current_menu"])

    // Process input
    nextMenu, response := menu.Process(input)

    // Update session state
    session.State["current_menu"] = nextMenu
    h.queries.UpdateUSSDSessionState(ctx, session)

    // Send response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

#### Menu System
**File**: `api/internal/channels/ussd/menus.go`

```go
package ussd

type Menu interface {
    Process(input string) (nextMenu string, response USSDResponse)
}

type MainMenu struct{}

func (m *MainMenu) Process(input string) (string, USSDResponse) {
    if input == "" {
        return "main", USSDResponse{
            Type: "CON",
            Message: "Welcome to Loyalty\n" +
                "1. My Rewards\n" +
                "2. My Balance\n" +
                "3. Redeem\n" +
                "4. Help",
        }
    }

    switch input {
    case "1":
        return "rewards", USSDResponse{Type: "CON", Message: "Loading rewards..."}
    case "2":
        return "balance", USSDResponse{Type: "CON", Message: "Loading balance..."}
    case "3":
        return "redeem", USSDResponse{Type: "CON", Message: "Enter reward code:"}
    default:
        return "main", USSDResponse{Type: "END", Message: "Invalid option"}
    }
}
```

### 3. Testing

**File**: `api/internal/channels/whatsapp/webhook_test.go`

Test cases:
- [ ] Webhook verification
- [ ] Signature verification
- [ ] Message parsing
- [ ] Enrollment flow
- [ ] Rewards listing
- [ ] Template sending

**File**: `api/internal/channels/ussd/handler_test.go`

Test cases:
- [ ] Menu navigation
- [ ] Session management
- [ ] Input validation

## Completion Criteria

- [ ] WhatsApp webhook verified and working
- [ ] Message processing implemented
- [ ] Enrollment flow complete
- [ ] Rewards listing working
- [ ] Templates configured
- [ ] USSD menus implemented
- [ ] Session management working
- [ ] Tests passing
