package httputil

import (
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var (
	// E.164 phone number regex (with optional + prefix)
	e164Regex = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)

	// Valid currencies
	validCurrencies = map[string]bool{
		"ZWG": true,
		"USD": true,
	}

	// Valid event types
	validEventTypes = map[string]bool{
		"purchase":       true,
		"visit":          true,
		"referral":       true,
		"signup":         true,
		"review":         true,
		"share":          true,
		"app_open":       true,
		"custom":         true,
	}

	// Valid reward types
	validRewardTypes = map[string]bool{
		"discount":         true,
		"voucher_code":     true,
		"points_credit":    true,
		"external_voucher": true,
		"physical_item":    true,
		"webhook_custom":   true,
	}

	// Valid inventory types
	validInventoryTypes = map[string]bool{
		"none":         true,
		"pool":         true,
		"jit_external": true,
	}

	// Valid user roles
	validRoles = map[string]bool{
		"owner":  true,
		"admin":  true,
		"staff":  true,
		"viewer": true,
	}
)

// ValidateE164Phone validates phone number in E.164 format
func ValidateE164Phone(phone string) error {
	if phone == "" {
		return nil // Empty is allowed in some contexts
	}

	// Ensure it starts with +
	if !strings.HasPrefix(phone, "+") {
		phone = "+" + phone
	}

	if !e164Regex.MatchString(phone) {
		return errors.New("phone number must be in E.164 format (e.g., +263771234567)")
	}

	return nil
}

// ValidateUUID validates UUID format
func ValidateUUID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return errors.New("invalid UUID format")
	}
	return nil
}

// ValidateCurrency checks if currency is ZWG or USD
func ValidateCurrency(currency string) error {
	if !validCurrencies[currency] {
		return errors.New("currency must be ZWG or USD")
	}
	return nil
}

// ValidateEventType checks if event type is allowed
func ValidateEventType(eventType string) error {
	if !validEventTypes[eventType] {
		return errors.New("invalid event type")
	}
	return nil
}

// ValidateRewardType checks if reward type is allowed
func ValidateRewardType(rewardType string) error {
	if !validRewardTypes[rewardType] {
		return errors.New("invalid reward type")
	}
	return nil
}

// ValidateInventoryType checks if inventory type is allowed
func ValidateInventoryType(inventoryType string) error {
	if !validInventoryTypes[inventoryType] {
		return errors.New("invalid inventory type")
	}
	return nil
}

// ValidateRole checks if role is valid
func ValidateRole(role string) error {
	if !validRoles[role] {
		return errors.New("invalid role")
	}
	return nil
}

// NormalizeE164Phone ensures phone number has + prefix
func NormalizeE164Phone(phone string) string {
	if phone == "" {
		return ""
	}
	if !strings.HasPrefix(phone, "+") {
		return "+" + phone
	}
	return phone
}
