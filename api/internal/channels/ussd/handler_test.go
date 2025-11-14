package ussd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUSSDRequestParsing(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "empty text",
			text:     "",
			expected: []string{""},
		},
		{
			name:     "single input",
			text:     "1",
			expected: []string{"1"},
		},
		{
			name:     "multiple inputs",
			text:     "1*2*3",
			expected: []string{"1", "2", "3"},
		},
		{
			name:     "complex navigation",
			text:     "1*1*0*2",
			expected: []string{"1", "1", "0", "2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := ParseInputSequence(tt.text)
			assert.Equal(t, tt.expected, parts)
		})
	}
}

func TestResponseFormatting(t *testing.T) {
	tests := []struct {
		name     string
		response USSDResponse
		expected string
	}{
		{
			name: "continue response",
			response: USSDResponse{
				Type:    Continue,
				Message: "Welcome",
			},
			expected: "CON Welcome",
		},
		{
			name: "end response",
			response: USSDResponse{
				Type:    End,
				Message: "Thank you",
			},
			expected: "END Thank you",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.response.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMenuOptionParsing(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		maxOptions int
		expected   int
		valid      bool
	}{
		{
			name:       "valid choice 1",
			input:      "1",
			maxOptions: 4,
			expected:   1,
			valid:      true,
		},
		{
			name:       "valid choice 0 (back)",
			input:      "0",
			maxOptions: 4,
			expected:   0,
			valid:      true,
		},
		{
			name:       "invalid - too high",
			input:      "5",
			maxOptions: 4,
			expected:   0,
			valid:      false,
		},
		{
			name:       "invalid - not a number",
			input:      "abc",
			maxOptions: 4,
			expected:   0,
			valid:      false,
		},
		{
			name:       "invalid - negative",
			input:      "-1",
			maxOptions: 4,
			expected:   0,
			valid:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			choice, valid := ParseMenuChoice(tt.input, tt.maxOptions)
			assert.Equal(t, tt.valid, valid)
			if valid {
				assert.Equal(t, tt.expected, choice)
			}
		})
	}
}

func TestSessionDataManagement(t *testing.T) {
	t.Run("new session data", func(t *testing.T) {
		data := NewSessionData()
		assert.Equal(t, "main", data.CurrentMenu)
		assert.Empty(t, data.MenuStack)
		assert.NotNil(t, data.Data)
	})

	t.Run("push and pop menu", func(t *testing.T) {
		data := NewSessionData()
		assert.Equal(t, "main", data.CurrentMenu)

		// Push to rewards menu
		data.PushMenu("rewards")
		assert.Equal(t, "rewards", data.CurrentMenu)
		assert.Equal(t, []string{"main"}, data.MenuStack)

		// Push to details menu
		data.PushMenu("details")
		assert.Equal(t, "details", data.CurrentMenu)
		assert.Equal(t, []string{"main", "rewards"}, data.MenuStack)

		// Pop back to rewards
		previous := data.PopMenu()
		assert.Equal(t, "rewards", previous)
		assert.Equal(t, "rewards", data.CurrentMenu)
		assert.Equal(t, []string{"main"}, data.MenuStack)

		// Pop back to main
		previous = data.PopMenu()
		assert.Equal(t, "main", previous)
		assert.Equal(t, "main", data.CurrentMenu)
		assert.Empty(t, data.MenuStack)

		// Pop when empty returns main
		previous = data.PopMenu()
		assert.Equal(t, "main", previous)
	})

	t.Run("set and get data", func(t *testing.T) {
		data := NewSessionData()

		// Set string data
		data.SetData("key1", "value1")
		value, ok := data.GetData("key1")
		assert.True(t, ok)
		assert.Equal(t, "value1", value)

		// Get as string
		strValue, ok := data.GetDataString("key1")
		assert.True(t, ok)
		assert.Equal(t, "value1", strValue)

		// Get non-existent key
		_, ok = data.GetData("nonexistent")
		assert.False(t, ok)

		// Set other types
		data.SetData("number", 123)
		value, ok = data.GetData("number")
		assert.True(t, ok)
		assert.Equal(t, 123, value)
	})
}

func TestResponseBuilder(t *testing.T) {
	t.Run("build simple response", func(t *testing.T) {
		rb := NewResponseBuilder()
		rb.AddLine("Welcome")
		rb.AddLine("to Loyalty")

		result := rb.Build()
		assert.Equal(t, "Welcome\nto Loyalty", result)
	})

	t.Run("build with options", func(t *testing.T) {
		rb := NewResponseBuilder()
		rb.AddLine("Main Menu")
		rb.AddBlankLine()
		rb.AddOption(1, "Option 1")
		rb.AddOption(2, "Option 2")

		result := rb.Build()
		assert.Contains(t, result, "1. Option 1")
		assert.Contains(t, result, "2. Option 2")
	})

	t.Run("continue response", func(t *testing.T) {
		rb := NewResponseBuilder()
		rb.AddLine("Choose option")
		response := rb.Continue()

		assert.Equal(t, Continue, response.Type)
		assert.Equal(t, "Choose option", response.Message)
	})

	t.Run("end response", func(t *testing.T) {
		rb := NewResponseBuilder()
		rb.AddLine("Thank you")
		response := rb.End()

		assert.Equal(t, End, response.Type)
		assert.Equal(t, "Thank you", response.Message)
	})
}

func TestFormatMenu(t *testing.T) {
	options := []MenuOption{
		{Key: "1", Label: "Option 1"},
		{Key: "2", Label: "Option 2"},
	}

	response := FormatMenu("Test Menu", options)

	assert.Equal(t, Continue, response.Type)
	assert.Contains(t, response.Message, "Test Menu")
	assert.Contains(t, response.Message, "1. Option 1")
	assert.Contains(t, response.Message, "2. Option 2")
}

func TestPhoneNumberNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "local format with leading zero",
			input:    "0771234567",
			expected: "+263771234567",
		},
		{
			name:     "without country code",
			input:    "771234567",
			expected: "+263771234567",
		},
		{
			name:     "with country code no plus",
			input:    "263771234567",
			expected: "+263771234567",
		},
		{
			name:     "already E.164",
			input:    "+263771234567",
			expected: "+263771234567",
		},
		{
			name:     "with spaces",
			input:    "077 123 4567",
			expected: "+263771234567",
		},
		{
			name:     "with dashes",
			input:    "077-123-4567",
			expected: "+263771234567",
		},
	}

	handler := &Handler{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.normalizePhoneNumber(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxLength int
		expected  string
	}{
		{
			name:      "text within limit",
			text:      "Short text",
			maxLength: 20,
			expected:  "Short text",
		},
		{
			name:      "text exceeds limit",
			text:      "This is a very long text that exceeds the maximum length",
			maxLength: 20,
			expected:  "This is a very lo...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateText(tt.text, tt.maxLength)
			assert.Equal(t, tt.expected, result)
			assert.LessOrEqual(t, len(result), tt.maxLength)
		})
	}
}

func TestFormatCurrency(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		currency string
		expected string
	}{
		{
			name:     "USD",
			amount:   10.50,
			currency: "USD",
			expected: "$10.50",
		},
		{
			name:     "ZWG",
			amount:   100.00,
			currency: "ZWG",
			expected: "ZWG 100.00",
		},
		{
			name:     "other currency",
			amount:   50.25,
			currency: "EUR",
			expected: "EUR 50.25",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCurrency(tt.amount, tt.currency)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to parse input sequence
func ParseInputSequence(text string) []string {
	if text == "" {
		return []string{""}
	}
	return splitByAsterisk(text)
}

func splitByAsterisk(text string) []string {
	parts := make([]string, 0)
	current := ""

	for _, char := range text {
		if char == '*' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(char)
		}
	}

	if current != "" || len(parts) > 0 {
		parts = append(parts, current)
	}

	return parts
}
