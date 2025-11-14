package budget

import (
	"errors"

	"github.com/bmachimbira/loyalty/api/internal/db"
)

// Supported currencies
const (
	CurrencyZWG = "ZWG"
	CurrencyUSD = "USD"
)

var (
	// ErrUnsupportedCurrency is returned when an unsupported currency is used
	ErrUnsupportedCurrency = errors.New("unsupported currency")
)

// ValidCurrencies is a map of all supported currencies
var ValidCurrencies = map[string]bool{
	CurrencyZWG: true,
	CurrencyUSD: true,
}

// IsValidCurrency checks if a currency code is supported
func IsValidCurrency(currency string) bool {
	return ValidCurrencies[currency]
}

// ValidateCurrency validates that a currency matches the budget's currency
func ValidateCurrency(budget *db.Budget, currency string) error {
	if !IsValidCurrency(currency) {
		return ErrUnsupportedCurrency
	}

	if budget.Currency != currency {
		return ErrCurrencyMismatch
	}

	return nil
}

// GetSupportedCurrencies returns a list of all supported currencies
func GetSupportedCurrencies() []string {
	return []string{CurrencyZWG, CurrencyUSD}
}
