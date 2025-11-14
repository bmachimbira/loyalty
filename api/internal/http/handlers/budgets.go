package handlers

import (
	httputil "github.com/bmachimbira/loyalty/api/internal/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BudgetsHandler handles budget-related API endpoints
type BudgetsHandler struct {
	pool *pgxpool.Pool
}

// NewBudgetsHandler creates a new budgets handler
func NewBudgetsHandler(pool *pgxpool.Pool) *BudgetsHandler {
	return &BudgetsHandler{pool: pool}
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

	// TODO: Once sqlc is generated, use queries.CreateBudget()
	c.JSON(201, gin.H{
		"id":         uuid.New().String(),
		"tenant_id":  tenantID,
		"name":       req.Name,
		"currency":   req.Currency,
		"soft_cap":   req.SoftCap,
		"hard_cap":   req.HardCap,
		"balance":    0.0,
		"period":     req.Period,
		"created_at": "2025-11-14T00:00:00Z",
	})
}

// List handles GET /v1/tenants/:tid/budgets
func (h *BudgetsHandler) List(c *gin.Context) {
	tenantID := c.Param("tid")
	if err := httputil.ValidateUUID(tenantID); err != nil {
		httputil.BadRequest(c, "Invalid tenant ID", nil)
		return
	}

	// TODO: Once sqlc is generated, use queries.ListBudgets()
	c.JSON(200, gin.H{
		"budgets": []gin.H{},
		"total":   0,
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

	// TODO: Once sqlc is generated, use queries.GetBudget()
	c.JSON(200, gin.H{
		"id":         budgetID,
		"tenant_id":  tenantID,
		"name":       "Sample Budget",
		"currency":   "USD",
		"soft_cap":   1000.0,
		"hard_cap":   1500.0,
		"balance":    500.0,
		"period":     "rolling",
		"created_at": "2025-11-14T00:00:00Z",
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

	// TODO: Once sqlc is generated:
	// 1. Get budget
	// 2. Create ledger entry of type 'fund'
	// 3. Update budget balance
	// 4. Return updated budget

	c.JSON(200, gin.H{
		"budget_id":   budgetID,
		"amount":      req.Amount,
		"new_balance": 500.0 + req.Amount,
		"created_at":  "2025-11-14T00:00:00Z",
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

	if budgetID != "" {
		if err := httputil.ValidateUUID(budgetID); err != nil {
			httputil.BadRequest(c, "Invalid budget ID", nil)
			return
		}
	}

	// TODO: Once sqlc is generated, use queries.ListLedgerEntries()
	c.JSON(200, gin.H{
		"entries": []gin.H{},
		"total":   0,
		"limit":   limit,
		"offset":  offset,
		"filters": gin.H{
			"budget_id": budgetID,
			"from":      fromDate,
			"to":        toDate,
		},
	})
}
