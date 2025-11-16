package handlers

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/budget"
	"github.com/bmachimbira/loyalty/api/internal/db"
	"github.com/bmachimbira/loyalty/api/internal/httputil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BudgetsHandler handles budget-related API endpoints
type BudgetsHandler struct {
	pool    *pgxpool.Pool
	service *budget.Service
	queries *db.Queries
}

// NewBudgetsHandler creates a new budgets handler
func NewBudgetsHandler(pool *pgxpool.Pool, logger *slog.Logger) *BudgetsHandler {
	queries := db.New(pool)
	return &BudgetsHandler{
		pool:    pool,
		service: budget.NewService(pool, queries, logger),
		queries: queries,
	}
}

// CreateBudgetRequest represents the request to create a budget
type CreateBudgetRequest struct {
	Name     string  `json:"name" binding:"required"`
	Currency string  `json:"currency" binding:"required"`
	SoftCap  float64 `json:"soft_cap"`
	HardCap  float64 `json:"hard_cap" binding:"required"`
	Period   string  `json:"period"`
}

// TopupBudgetRequest represents the request to topup a budget
type TopupBudgetRequest struct {
	Amount      float64 `json:"amount" binding:"required"`
	Description string  `json:"description"`
}

// Create handles POST /v1/tenants/:tid/budgets
func (h *BudgetsHandler) Create(c *gin.Context) {
	tenantID := c.Param("tid")
	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	var req CreateBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	// Validate currency
	if err := httputil.ValidateCurrency(req.Currency); err != nil {
		httputil.BadRequest(c, err.Error(), nil)
		return
	}

	// Validate period
	validPeriods := map[string]bool{
		"rolling": true,
		"monthly": true,
	}
	if req.Period != "" && !validPeriods[req.Period] {
		httputil.BadRequest(c, "Period must be 'rolling' or 'monthly'", nil)
		return
	}
	if req.Period == "" {
		req.Period = "rolling"
	}

	// Validate caps
	if req.HardCap <= 0 {
		httputil.BadRequest(c, "Hard cap must be greater than 0", nil)
		return
	}
	if req.SoftCap > req.HardCap {
		httputil.BadRequest(c, "Soft cap cannot exceed hard cap", nil)
		return
	}

	// Parse tenant UUID
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}

	// Prepare numeric values
	var softCap, hardCap, balance pgtype.Numeric
	if err := softCap.Scan(req.SoftCap); err != nil {
		httputil.BadRequest(c, "Invalid soft cap value", nil)
		return
	}
	if err := hardCap.Scan(req.HardCap); err != nil {
		httputil.BadRequest(c, "Invalid hard cap value", nil)
		return
	}
	if err := balance.Scan(0.0); err != nil {
		httputil.InternalError(c, "Failed to initialize balance")
		return
	}

	// Create budget using queries
	budget, err := h.queries.CreateBudget(c.Request.Context(), db.CreateBudgetParams{
		TenantID: tenantUUID,
		Name:     req.Name,
		Currency: req.Currency,
		SoftCap:  softCap,
		HardCap:  hardCap,
		Balance:  balance,
		Period:   req.Period,
	})
	if err != nil {
		httputil.InternalError(c, "Failed to create budget")
		return
	}

	c.JSON(201, gin.H{
		"id":         formatUUID(budget.ID),
		"tenant_id":  formatUUID(budget.TenantID),
		"name":       budget.Name,
		"currency":   budget.Currency,
		"soft_cap":   budget.SoftCap.Int.String(),
		"hard_cap":   budget.HardCap.Int.String(),
		"balance":    budget.Balance.Int.String(),
		"period":     budget.Period,
		"created_at": formatTimestamp(budget.CreatedAt),
	})
}

// List handles GET /v1/tenants/:tid/budgets
func (h *BudgetsHandler) List(c *gin.Context) {
	tenantID := c.Param("tid")
	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	// Parse tenant UUID
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}

	// List budgets using queries
	budgets, err := h.queries.ListBudgets(c.Request.Context(), tenantUUID)
	if err != nil {
		httputil.InternalError(c, "Failed to list budgets")
		return
	}

	// Format response
	budgetsList := make([]gin.H, len(budgets))
	for i, budget := range budgets {
		budgetsList[i] = gin.H{
			"id":         formatUUID(budget.ID),
			"tenant_id":  formatUUID(budget.TenantID),
			"name":       budget.Name,
			"currency":   budget.Currency,
			"soft_cap":   budget.SoftCap.Int.String(),
			"hard_cap":   budget.HardCap.Int.String(),
			"balance":    budget.Balance.Int.String(),
			"period":     budget.Period,
			"created_at": formatTimestamp(budget.CreatedAt),
		}
	}

	c.JSON(200, gin.H{
		"budgets": budgetsList,
		"total":   len(budgets),
	})
}

// Get handles GET /v1/tenants/:tid/budgets/:id
func (h *BudgetsHandler) Get(c *gin.Context) {
	tenantID := c.Param("tid")
	budgetID := c.Param("id")

	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	if err := httputil.ValidateUUID(budgetID); err != nil {
		httputil.BadRequest(c, "Invalid budget ID", nil)
		return
	}

	// Parse UUIDs
	var tenantUUID, budgetUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}
	if err := budgetUUID.Scan(budgetID); err != nil {
		httputil.BadRequest(c, "Invalid budget ID format", nil)
		return
	}

	// Get budget using queries
	budget, err := h.queries.GetBudgetByID(c.Request.Context(), db.GetBudgetByIDParams{
		ID:       budgetUUID,
		TenantID: tenantUUID,
	})
	if err != nil {
		httputil.NotFound(c, "Budget not found")
		return
	}

	c.JSON(200, gin.H{
		"id":         formatUUID(budget.ID),
		"tenant_id":  formatUUID(budget.TenantID),
		"name":       budget.Name,
		"currency":   budget.Currency,
		"soft_cap":   budget.SoftCap.Int.String(),
		"hard_cap":   budget.HardCap.Int.String(),
		"balance":    budget.Balance.Int.String(),
		"period":     budget.Period,
		"created_at": formatTimestamp(budget.CreatedAt),
	})
}

// Topup handles POST /v1/tenants/:tid/budgets/:id/topup
func (h *BudgetsHandler) Topup(c *gin.Context) {
	tenantID := c.Param("tid")
	budgetID := c.Param("id")

	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	if err := httputil.ValidateUUID(budgetID); err != nil {
		httputil.BadRequest(c, "Invalid budget ID", nil)
		return
	}

	var req TopupBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	if req.Amount <= 0 {
		httputil.BadRequest(c, "Amount must be greater than 0", nil)
		return
	}

	// Parse UUIDs
	var tenantUUID, budgetUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}
	if err := budgetUUID.Scan(budgetID); err != nil {
		httputil.BadRequest(c, "Invalid budget ID format", nil)
		return
	}

	// Get budget to verify currency
	bgt, err := h.queries.GetBudgetByID(c.Request.Context(), db.GetBudgetByIDParams{
		ID:       budgetUUID,
		TenantID: tenantUUID,
	})
	if err != nil {
		httputil.NotFound(c, "Budget not found")
		return
	}

	// Topup budget using service
	result, err := h.service.TopupBudget(c.Request.Context(), budget.TopupBudgetParams{
		TenantID: tenantUUID,
		BudgetID: budgetUUID,
		Amount:   strconv.FormatFloat(req.Amount, 'f', -1, 64),
		Currency: bgt.Currency,
	})
	if err != nil {
		httputil.InternalError(c, "Failed to topup budget")
		return
	}

	c.JSON(200, gin.H{
		"budget_id":   formatUUID(result.BudgetID),
		"amount":      result.Amount,
		"currency":    result.Currency,
		"new_balance": result.NewBalance,
	})
}

// ListLedger handles GET /v1/tenants/:tid/ledger
func (h *BudgetsHandler) ListLedger(c *gin.Context) {
	tenantID := c.Param("tid")
	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	// Get filter params
	budgetID := c.Query("budget_id")
	fromDate := c.Query("from")
	toDate := c.Query("to")
	limit := c.DefaultQuery("limit", "100")
	offset := c.DefaultQuery("offset", "0")

	// Budget ID is required
	if budgetID == "" {
		httputil.BadRequest(c, "budget_id query parameter is required", nil)
		return
	}

	if err := httputil.ValidateUUID(budgetID); err != nil {
		httputil.BadRequest(c, "Invalid budget ID", nil)
		return
	}

	// Parse UUIDs
	var tenantUUID, budgetUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID format", nil)
		return
	}
	if err := budgetUUID.Scan(budgetID); err != nil {
		httputil.BadRequest(c, "Invalid budget ID format", nil)
		return
	}

	// Parse limit and offset
	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt < 1 {
		limitInt = 100
	}
	if limitInt > 1000 {
		limitInt = 1000
	}

	offsetInt, err := strconv.Atoi(offset)
	if err != nil || offsetInt < 0 {
		offsetInt = 0
	}

	var entries []db.LedgerEntry

	// If date range is provided, use GetLedgerEntries
	if fromDate != "" && toDate != "" {
		fromTime, err := time.Parse(time.RFC3339, fromDate)
		if err != nil {
			httputil.BadRequest(c, "Invalid from date format", nil)
			return
		}
		toTime, err := time.Parse(time.RFC3339, toDate)
		if err != nil {
			httputil.BadRequest(c, "Invalid to date format", nil)
			return
		}

		entries, err = h.queries.GetLedgerEntries(c.Request.Context(), db.GetLedgerEntriesParams{
			TenantID:    tenantUUID,
			BudgetID:    budgetUUID,
			CreatedAt:   pgtype.Timestamptz{Time: fromTime, Valid: true},
			CreatedAt_2: pgtype.Timestamptz{Time: toTime, Valid: true},
			Limit:       int32(limitInt),
			Offset:      int32(offsetInt),
		})
	} else {
		// Otherwise use GetLedgerEntriesByDateRangeOnly (no date filter)
		entries, err = h.queries.GetLedgerEntriesByDateRangeOnly(c.Request.Context(), db.GetLedgerEntriesByDateRangeOnlyParams{
			TenantID: tenantUUID,
			BudgetID: budgetUUID,
			Limit:    int32(limitInt),
			Offset:   int32(offsetInt),
		})
	}

	if err != nil {
		httputil.InternalError(c, "Failed to list ledger entries")
		return
	}

	// Format response
	entriesList := make([]gin.H, len(entries))
	for i, entry := range entries {
		entriesList[i] = gin.H{
			"id":          entry.ID,
			"tenant_id":   formatUUID(entry.TenantID),
			"budget_id":   formatUUID(entry.BudgetID),
			"entry_type":  entry.EntryType,
			"currency":    entry.Currency,
			"amount":      entry.Amount.Int.String(),
			"ref_type":    entry.RefType.String,
			"ref_id":      formatUUID(entry.RefID),
			"created_at":  formatTimestamp(entry.CreatedAt),
		}
	}

	c.JSON(200, gin.H{
		"entries": entriesList,
		"total":   len(entries),
		"limit":   limit,
		"offset":  offset,
		"filters": gin.H{
			"budget_id": budgetID,
			"from":      fromDate,
			"to":        toDate,
		},
	})
}
