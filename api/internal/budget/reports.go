package budget

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/bmachimbira/loyalty/api/internal/db"
)

// BudgetReport contains comprehensive budget statistics
type BudgetReport struct {
	BudgetID        pgtype.UUID            `json:"budget_id"`
	BudgetName      string                 `json:"budget_name"`
	Currency        string                 `json:"currency"`
	Period          string                 `json:"period"`
	CurrentBalance  float64                `json:"current_balance"`
	SoftCap         float64                `json:"soft_cap"`
	HardCap         float64                `json:"hard_cap"`
	Utilization     float64                `json:"utilization_percent"`
	TotalFunded     float64                `json:"total_funded"`
	TotalReserved   float64                `json:"total_reserved"`
	TotalCharged    float64                `json:"total_charged"`
	TotalReleased   float64                `json:"total_released"`
	NetCharged      float64                `json:"net_charged"`
	Available       float64                `json:"available"`
	EntryCount      map[string]int64       `json:"entry_count"`
	Summary         *LedgerSummary         `json:"ledger_summary"`
	DateRange       DateRange              `json:"date_range"`
	GeneratedAt     time.Time              `json:"generated_at"`
}

// LedgerSummary contains aggregated ledger statistics
type LedgerSummary struct {
	ByType     map[string]TypeSummary `json:"by_type"`
	TotalCount int64                  `json:"total_count"`
}

// TypeSummary contains statistics for a specific entry type
type TypeSummary struct {
	Count  int64   `json:"count"`
	Amount float64 `json:"amount"`
}

// DateRange represents a date range for reports
type DateRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// ReportFormat represents the output format for reports
type ReportFormat string

const (
	// FormatJSON outputs the report as JSON
	FormatJSON ReportFormat = "json"

	// FormatCSV outputs the report as CSV
	FormatCSV ReportFormat = "csv"
)

// GenerateBudgetReport generates a comprehensive budget report
func (s *Service) GenerateBudgetReport(ctx context.Context, tenantID, budgetID pgtype.UUID, dateRange DateRange) (*BudgetReport, error) {
	if !tenantID.Valid || !budgetID.Valid {
		return nil, fmt.Errorf("tenant_id and budget_id are required")
	}

	// Get budget
	budget, err := s.queries.GetBudgetByID(ctx, db.GetBudgetByIDParams{
		ID:       budgetID,
		TenantID: tenantID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrBudgetNotFound
		}
		return nil, fmt.Errorf("failed to get budget: %w", err)
	}

	// Convert budget numeric fields
	var balance, softCap, hardCap float64
	budget.Balance.Float(&balance)
	budget.SoftCap.Float(&softCap)
	budget.HardCap.Float(&hardCap)

	// Calculate utilization
	utilization := 0.0
	if hardCap > 0 {
		utilization = (balance / hardCap) * 100
	}

	// Get ledger summary
	fromTime := pgtype.Timestamptz{}
	toTime := pgtype.Timestamptz{}
	fromTime.Scan(dateRange.From)
	toTime.Scan(dateRange.To)

	summaryRows, err := s.queries.GetLedgerSummaryByType(ctx, db.GetLedgerSummaryByTypeParams{
		TenantID:    tenantID,
		BudgetID:    budgetID,
		CreatedAt:   fromTime,
		CreatedAt_2: toTime,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger summary: %w", err)
	}

	// Aggregate by type
	summary := &LedgerSummary{
		ByType:     make(map[string]TypeSummary),
		TotalCount: 0,
	}

	entryCount := make(map[string]int64)
	var totalFunded, totalReserved, totalCharged, totalReleased float64

	for _, row := range summaryRows {
		// Convert amount to float64
		var amount float64
		if numericAmount, ok := row.TotalAmount.(string); ok {
			fmt.Sscanf(numericAmount, "%f", &amount)
		}

		summary.ByType[row.EntryType] = TypeSummary{
			Count:  row.EntryCount,
			Amount: amount,
		}
		summary.TotalCount += row.EntryCount
		entryCount[row.EntryType] = row.EntryCount

		// Accumulate totals
		switch row.EntryType {
		case "fund":
			totalFunded += amount
		case "reserve":
			totalReserved += amount
		case "charge":
			totalCharged += amount
		case "release":
			totalReleased += -amount // Release amounts are negative
		}
	}

	// Net charged = total charged (actual redemptions)
	netCharged := totalCharged

	// Available = hard cap - current balance
	available := hardCap - balance

	report := &BudgetReport{
		BudgetID:       budgetID,
		BudgetName:     budget.Name,
		Currency:       budget.Currency,
		Period:         budget.Period,
		CurrentBalance: balance,
		SoftCap:        softCap,
		HardCap:        hardCap,
		Utilization:    utilization,
		TotalFunded:    totalFunded,
		TotalReserved:  totalReserved,
		TotalCharged:   totalCharged,
		TotalReleased:  totalReleased,
		NetCharged:     netCharged,
		Available:      available,
		EntryCount:     entryCount,
		Summary:        summary,
		DateRange:      dateRange,
		GeneratedAt:    time.Now(),
	}

	s.logger.Info("budget report generated",
		"budget_id", budgetID,
		"budget_name", budget.Name,
		"date_range", fmt.Sprintf("%s to %s", dateRange.From.Format("2006-01-02"), dateRange.To.Format("2006-01-02")),
		"total_entries", summary.TotalCount)

	return report, nil
}

// GenerateTenantReport generates a report for all budgets in a tenant
func (s *Service) GenerateTenantReport(ctx context.Context, tenantID pgtype.UUID, dateRange DateRange) ([]BudgetReport, error) {
	if !tenantID.Valid {
		return nil, fmt.Errorf("tenant_id is required")
	}

	// Get all budgets
	budgets, err := s.queries.ListBudgets(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list budgets: %w", err)
	}

	reports := make([]BudgetReport, 0, len(budgets))

	for _, budget := range budgets {
		report, err := s.GenerateBudgetReport(ctx, tenantID, budget.ID, dateRange)
		if err != nil {
			s.logger.Error("failed to generate budget report",
				"budget_id", budget.ID,
				"error", err)
			continue
		}
		reports = append(reports, *report)
	}

	return reports, nil
}

// ExportReportJSON exports a report as JSON
func (s *Service) ExportReportJSON(report *BudgetReport, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// ExportReportCSV exports a report as CSV
func (s *Service) ExportReportCSV(report *BudgetReport, w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write header
	header := []string{
		"Budget ID",
		"Budget Name",
		"Currency",
		"Period",
		"Current Balance",
		"Soft Cap",
		"Hard Cap",
		"Utilization %",
		"Total Funded",
		"Total Reserved",
		"Total Charged",
		"Total Released",
		"Net Charged",
		"Available",
		"Generated At",
	}
	if err := csvWriter.Write(header); err != nil {
		return err
	}

	// Write data
	row := []string{
		report.BudgetID.Bytes.String(),
		report.BudgetName,
		report.Currency,
		report.Period,
		fmt.Sprintf("%.2f", report.CurrentBalance),
		fmt.Sprintf("%.2f", report.SoftCap),
		fmt.Sprintf("%.2f", report.HardCap),
		fmt.Sprintf("%.2f", report.Utilization),
		fmt.Sprintf("%.2f", report.TotalFunded),
		fmt.Sprintf("%.2f", report.TotalReserved),
		fmt.Sprintf("%.2f", report.TotalCharged),
		fmt.Sprintf("%.2f", report.TotalReleased),
		fmt.Sprintf("%.2f", report.NetCharged),
		fmt.Sprintf("%.2f", report.Available),
		report.GeneratedAt.Format(time.RFC3339),
	}
	return csvWriter.Write(row)
}

// ExportTenantReportCSV exports multiple budget reports as CSV
func (s *Service) ExportTenantReportCSV(reports []BudgetReport, w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write header
	header := []string{
		"Budget ID",
		"Budget Name",
		"Currency",
		"Period",
		"Current Balance",
		"Soft Cap",
		"Hard Cap",
		"Utilization %",
		"Total Funded",
		"Total Reserved",
		"Total Charged",
		"Total Released",
		"Net Charged",
		"Available",
	}
	if err := csvWriter.Write(header); err != nil {
		return err
	}

	// Write data for each budget
	for _, report := range reports {
		row := []string{
			report.BudgetID.Bytes.String(),
			report.BudgetName,
			report.Currency,
			report.Period,
			fmt.Sprintf("%.2f", report.CurrentBalance),
			fmt.Sprintf("%.2f", report.SoftCap),
			fmt.Sprintf("%.2f", report.HardCap),
			fmt.Sprintf("%.2f", report.Utilization),
			fmt.Sprintf("%.2f", report.TotalFunded),
			fmt.Sprintf("%.2f", report.TotalReserved),
			fmt.Sprintf("%.2f", report.TotalCharged),
			fmt.Sprintf("%.2f", report.TotalReleased),
			fmt.Sprintf("%.2f", report.NetCharged),
			fmt.Sprintf("%.2f", report.Available),
		}
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// GetLedgerEntries retrieves ledger entries for a budget with pagination
func (s *Service) GetLedgerEntries(ctx context.Context, params GetLedgerEntriesParams) ([]db.LedgerEntry, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	fromTime := pgtype.Timestamptz{}
	toTime := pgtype.Timestamptz{}
	fromTime.Scan(params.From)
	toTime.Scan(params.To)

	entries, err := s.queries.GetLedgerEntriesByDateRange(ctx, db.GetLedgerEntriesByDateRangeParams{
		TenantID:    params.TenantID,
		BudgetID:    params.BudgetID,
		CreatedAt:   fromTime,
		CreatedAt_2: toTime,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger entries: %w", err)
	}

	return entries, nil
}

// GetLedgerEntriesParams contains parameters for retrieving ledger entries
type GetLedgerEntriesParams struct {
	TenantID pgtype.UUID
	BudgetID pgtype.UUID
	From     time.Time
	To       time.Time
	Limit    int32
	Offset   int32
}

// Validate validates the parameters
func (p GetLedgerEntriesParams) Validate() error {
	if !p.TenantID.Valid {
		return fmt.Errorf("tenant_id is required")
	}
	if !p.BudgetID.Valid {
		return fmt.Errorf("budget_id is required")
	}
	if p.From.IsZero() || p.To.IsZero() {
		return fmt.Errorf("from and to dates are required")
	}
	if p.From.After(p.To) {
		return fmt.Errorf("from date must be before to date")
	}
	return nil
}

// NewDateRange creates a date range for common periods
func NewDateRange(period string) DateRange {
	now := time.Now()
	var from, to time.Time

	switch period {
	case "today":
		from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		to = now

	case "week":
		from = now.AddDate(0, 0, -7)
		to = now

	case "month":
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		to = now

	case "quarter":
		from = now.AddDate(0, -3, 0)
		to = now

	case "year":
		from = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		to = now

	default:
		// Default to current month
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		to = now
	}

	return DateRange{
		From: from,
		To:   to,
	}
}
