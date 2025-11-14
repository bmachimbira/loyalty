package whatsapp

// WebhookPayload represents the incoming WhatsApp webhook payload
type WebhookPayload struct {
	Object string  `json:"object"`
	Entry  []Entry `json:"entry"`
}

// Entry represents a single entry in the webhook payload
type Entry struct {
	ID      string   `json:"id"`
	Changes []Change `json:"changes"`
}

// Change represents a change notification
type Change struct {
	Field string       `json:"field"`
	Value MessageValue `json:"value"`
}

// MessageValue contains the actual message data
type MessageValue struct {
	MessagingProduct string    `json:"messaging_product"`
	Metadata         Metadata  `json:"metadata"`
	Messages         []Message `json:"messages,omitempty"`
	Contacts         []Contact `json:"contacts,omitempty"`
	Statuses         []Status  `json:"statuses,omitempty"`
}

// Metadata contains phone number information
type Metadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

// Message represents an incoming WhatsApp message
type Message struct {
	From      string      `json:"from"`
	ID        string      `json:"id"`
	Timestamp string      `json:"timestamp"`
	Type      string      `json:"type"`
	Text      *TextMsg    `json:"text,omitempty"`
	Button    *ButtonMsg  `json:"button,omitempty"`
	Image     *MediaMsg   `json:"image,omitempty"`
	Document  *MediaMsg   `json:"document,omitempty"`
}

// TextMsg represents a text message
type TextMsg struct {
	Body string `json:"body"`
}

// ButtonMsg represents a button reply message
type ButtonMsg struct {
	Payload string `json:"payload"`
	Text    string `json:"text"`
}

// MediaMsg represents a media message
type MediaMsg struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	SHA256   string `json:"sha256"`
	Caption  string `json:"caption,omitempty"`
}

// Contact represents sender contact information
type Contact struct {
	Profile Profile `json:"profile"`
	WaID    string  `json:"wa_id"`
}

// Profile contains contact profile information
type Profile struct {
	Name string `json:"name"`
}

// Status represents a message status update
type Status struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Timestamp    string `json:"timestamp"`
	RecipientID  string `json:"recipient_id"`
	ConversationID string `json:"conversation,omitempty"`
}

// SendMessageRequest represents a request to send a message
type SendMessageRequest struct {
	MessagingProduct string                 `json:"messaging_product"`
	RecipientType    string                 `json:"recipient_type,omitempty"`
	To               string                 `json:"to"`
	Type             string                 `json:"type"`
	Text             *TextPayload           `json:"text,omitempty"`
	Template         *TemplatePayload       `json:"template,omitempty"`
	Interactive      *InteractivePayload    `json:"interactive,omitempty"`
}

// TextPayload represents text message content
type TextPayload struct {
	PreviewURL bool   `json:"preview_url,omitempty"`
	Body       string `json:"body"`
}

// TemplatePayload represents a template message
type TemplatePayload struct {
	Name       string                   `json:"name"`
	Language   LanguagePayload          `json:"language"`
	Components []TemplateComponentPayload `json:"components,omitempty"`
}

// LanguagePayload represents the language code
type LanguagePayload struct {
	Code string `json:"code"`
}

// TemplateComponentPayload represents template parameters
type TemplateComponentPayload struct {
	Type       string                     `json:"type"`
	Parameters []TemplateParameterPayload `json:"parameters,omitempty"`
}

// TemplateParameterPayload represents a single template parameter
type TemplateParameterPayload struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// InteractivePayload represents an interactive message (buttons, lists)
type InteractivePayload struct {
	Type   string          `json:"type"`
	Header *HeaderPayload  `json:"header,omitempty"`
	Body   BodyPayload     `json:"body"`
	Footer *FooterPayload  `json:"footer,omitempty"`
	Action ActionPayload   `json:"action"`
}

// HeaderPayload represents the header of an interactive message
type HeaderPayload struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// BodyPayload represents the body of an interactive message
type BodyPayload struct {
	Text string `json:"text"`
}

// FooterPayload represents the footer of an interactive message
type FooterPayload struct {
	Text string `json:"text"`
}

// ActionPayload represents the action of an interactive message
type ActionPayload struct {
	Buttons  []ButtonPayload  `json:"buttons,omitempty"`
	Button   string           `json:"button,omitempty"`
	Sections []SectionPayload `json:"sections,omitempty"`
}

// ButtonPayload represents a button
type ButtonPayload struct {
	Type  string       `json:"type"`
	Reply ReplyPayload `json:"reply"`
}

// ReplyPayload represents a button reply
type ReplyPayload struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// SectionPayload represents a list section
type SectionPayload struct {
	Title string        `json:"title"`
	Rows  []RowPayload  `json:"rows"`
}

// RowPayload represents a list row
type RowPayload struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// SendMessageResponse represents the API response
type SendMessageResponse struct {
	MessagingProduct string          `json:"messaging_product"`
	Contacts         []ContactResult `json:"contacts"`
	Messages         []MessageResult `json:"messages"`
}

// ContactResult represents a contact in the response
type ContactResult struct {
	Input string `json:"input"`
	WaID  string `json:"wa_id"`
}

// MessageResult represents a sent message result
type MessageResult struct {
	ID string `json:"id"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details
type ErrorDetail struct {
	Message      string      `json:"message"`
	Type         string      `json:"type"`
	Code         int         `json:"code"`
	ErrorData    ErrorData   `json:"error_data"`
	FBTraceID    string      `json:"fbtrace_id"`
}

// ErrorData contains additional error information
type ErrorData struct {
	MessagingProduct string `json:"messaging_product"`
	Details          string `json:"details"`
}
