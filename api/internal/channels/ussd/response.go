package ussd

import (
	"fmt"
	"strings"
)

// ResponseBuilder helps build USSD responses
type ResponseBuilder struct {
	lines []string
}

// NewResponseBuilder creates a new response builder
func NewResponseBuilder() *ResponseBuilder {
	return &ResponseBuilder{
		lines: make([]string, 0),
	}
}

// AddLine adds a line to the response
func (rb *ResponseBuilder) AddLine(text string) *ResponseBuilder {
	rb.lines = append(rb.lines, text)
	return rb
}

// AddLines adds multiple lines to the response
func (rb *ResponseBuilder) AddLines(texts ...string) *ResponseBuilder {
	rb.lines = append(rb.lines, texts...)
	return rb
}

// AddBlankLine adds a blank line for spacing
func (rb *ResponseBuilder) AddBlankLine() *ResponseBuilder {
	rb.lines = append(rb.lines, "")
	return rb
}

// AddOption adds a numbered option
func (rb *ResponseBuilder) AddOption(number int, text string) *ResponseBuilder {
	rb.lines = append(rb.lines, fmt.Sprintf("%d. %s", number, text))
	return rb
}

// AddOptions adds multiple numbered options
func (rb *ResponseBuilder) AddOptions(options []MenuOption) *ResponseBuilder {
	for i, opt := range options {
		rb.AddOption(i+1, opt.Label)
	}
	return rb
}

// Build builds the response text
func (rb *ResponseBuilder) Build() string {
	return strings.Join(rb.lines, "\n")
}

// Continue builds a CON response (session continues)
func (rb *ResponseBuilder) Continue() USSDResponse {
	return USSDResponse{
		Type:    Continue,
		Message: rb.Build(),
	}
}

// End builds an END response (session ends)
func (rb *ResponseBuilder) End() USSDResponse {
	return USSDResponse{
		Type:    End,
		Message: rb.Build(),
	}
}

// FormatContinue creates a CON response
func FormatContinue(message string) USSDResponse {
	return USSDResponse{
		Type:    Continue,
		Message: message,
	}
}

// FormatEnd creates an END response
func FormatEnd(message string) USSDResponse {
	return USSDResponse{
		Type:    End,
		Message: message,
	}
}

// FormatMenu creates a menu response
func FormatMenu(title string, options []MenuOption) USSDResponse {
	rb := NewResponseBuilder()
	rb.AddLine(title)
	rb.AddBlankLine()

	for i, opt := range options {
		rb.AddOption(i+1, opt.Label)
	}

	return rb.Continue()
}

// FormatMenuWithBack creates a menu with a back option
func FormatMenuWithBack(title string, options []MenuOption) USSDResponse {
	rb := NewResponseBuilder()
	rb.AddLine(title)
	rb.AddBlankLine()

	for i, opt := range options {
		rb.AddOption(i+1, opt.Label)
	}
	rb.AddOption(0, "Back")

	return rb.Continue()
}

// FormatPaginatedList creates a paginated list response
func FormatPaginatedList(title string, items []MenuItem, page, totalPages int) USSDResponse {
	rb := NewResponseBuilder()
	rb.AddLine(title)
	rb.AddLine(fmt.Sprintf("(Page %d/%d)", page, totalPages))
	rb.AddBlankLine()

	for i, item := range items {
		rb.AddOption(i+1, item.Label)
		if item.Description != "" {
			rb.AddLine("   " + item.Description)
		}
	}

	rb.AddBlankLine()

	// Add navigation options
	navStart := len(items) + 1
	if page < totalPages {
		rb.AddOption(navStart, "Next")
		navStart++
	}
	if page > 1 {
		rb.AddOption(navStart, "Previous")
		navStart++
	}
	rb.AddOption(0, "Back")

	return rb.Continue()
}

// FormatError creates an error response
func FormatError(message string) USSDResponse {
	return FormatEnd("Error: " + message + "\n\nPlease try again later.")
}

// FormatSuccess creates a success response
func FormatSuccess(message string) USSDResponse {
	return FormatEnd(message)
}

// FormatConfirmation creates a confirmation prompt
func FormatConfirmation(message string) USSDResponse {
	rb := NewResponseBuilder()
	rb.AddLine(message)
	rb.AddBlankLine()
	rb.AddOption(1, "Confirm")
	rb.AddOption(2, "Cancel")

	return rb.Continue()
}

// ParseMenuChoice parses a menu choice input
func ParseMenuChoice(input string, maxOptions int) (int, bool) {
	var choice int
	_, err := fmt.Sscanf(input, "%d", &choice)
	if err != nil {
		return 0, false
	}

	if choice < 0 || choice > maxOptions {
		return 0, false
	}

	return choice, true
}

// TruncateText truncates text to fit USSD character limits
// USSD typically supports ~182 characters per message
func TruncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	// Truncate and add ellipsis
	return text[:maxLength-3] + "..."
}

// FormatCurrency formats a currency amount
func FormatCurrency(amount float64, currency string) string {
	if currency == "USD" {
		return fmt.Sprintf("$%.2f", amount)
	}
	if currency == "ZWG" {
		return fmt.Sprintf("ZWG %.2f", amount)
	}
	return fmt.Sprintf("%s %.2f", currency, amount)
}

// FormatDate formats a date for display
func FormatDate(date string) string {
	// Simple date formatting for USSD
	// In production, this should parse and format properly
	return date
}

// ChunkText splits text into chunks suitable for USSD
func ChunkText(text string, chunkSize int) []string {
	if len(text) <= chunkSize {
		return []string{text}
	}

	chunks := make([]string, 0)
	words := strings.Fields(text)
	currentChunk := ""

	for _, word := range words {
		if len(currentChunk)+len(word)+1 <= chunkSize {
			if currentChunk != "" {
				currentChunk += " "
			}
			currentChunk += word
		} else {
			if currentChunk != "" {
				chunks = append(chunks, currentChunk)
			}
			currentChunk = word
		}
	}

	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}
