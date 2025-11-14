package http

import (
	"os"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/auth"
	"github.com/bmachimbira/loyalty/api/internal/channels/ussd"
	"github.com/bmachimbira/loyalty/api/internal/channels/whatsapp"
	"github.com/bmachimbira/loyalty/api/internal/http/handlers"
	"github.com/bmachimbira/loyalty/api/internal/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SetupRouter configures all routes and middleware
func SetupRouter(pool *pgxpool.Pool, jwtSecret string, hmacKeys auth.HMACKeys) *gin.Engine {
	// Set Gin mode based on environment
	// gin.SetMode(gin.ReleaseMode) // Uncomment for production

	r := gin.New()

	// Global middleware
	r.Use(gin.Recovery()) // Recover from panics
	r.Use(middleware.CORS())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())

	// Health check endpoint (no auth required)
	r.GET("/health", HealthCheck)
	r.GET("/ready", ReadyCheck(pool))

	// Initialize handlers
	customersHandler := handlers.NewCustomersHandler(pool)
	eventsHandler := handlers.NewEventsHandler(pool)
	rulesHandler := handlers.NewRulesHandler(pool)
	rewardsHandler := handlers.NewRewardsHandler(pool)
	issuancesHandler := handlers.NewIssuancesHandler(pool)
	budgetsHandler := handlers.NewBudgetsHandler(pool)
	campaignsHandler := handlers.NewCampaignsHandler(pool)

	// Initialize channel handlers
	waHandler := whatsapp.NewHandler(
		pool,
		os.Getenv("WHATSAPP_VERIFY_TOKEN"),
		os.Getenv("WHATSAPP_APP_SECRET"),
		os.Getenv("WHATSAPP_PHONE_NUMBER_ID"),
		os.Getenv("WHATSAPP_ACCESS_TOKEN"),
	)
	ussdHandler := ussd.NewHandler(pool)

	// Public routes (no authentication)
	public := r.Group("/public")
	{
		// WhatsApp webhook endpoints
		wa := public.Group("/wa")
		{
			wa.GET("/webhook", waHandler.Verify)
			wa.POST("/webhook", waHandler.Webhook)
		}

		// USSD callback endpoint
		public.POST("/ussd/callback", ussdHandler.HandleCallback)
	}

	// V1 API routes (authenticated)
	v1 := r.Group("/v1")
	v1.Use(middleware.RequireAuth(jwtSecret))
	v1.Use(middleware.TenantContext(pool))
	v1.Use(middleware.IdempotencyCheck())

	// Tenant-scoped routes
	tenants := v1.Group("/tenants/:tid")
	{
		// Customers API
		customers := tenants.Group("/customers")
		{
			customers.POST("", customersHandler.Create)
			customers.GET("", customersHandler.List)
			customers.GET("/:id", customersHandler.Get)
			customers.PATCH("/:id/status", customersHandler.UpdateStatus)
		}

		// Events API
		events := tenants.Group("/events")
		{
			events.POST("", eventsHandler.Create) // Requires Idempotency-Key
			events.GET("", eventsHandler.List)
			events.GET("/:id", eventsHandler.Get)
		}

		// Rules API
		rules := tenants.Group("/rules")
		{
			rules.POST("", middleware.RequireRole("owner", "admin"), rulesHandler.Create)
			rules.GET("", rulesHandler.List)
			rules.GET("/:id", rulesHandler.Get)
			rules.PATCH("/:id", middleware.RequireRole("owner", "admin"), rulesHandler.Update)
			rules.DELETE("/:id", middleware.RequireRole("owner", "admin"), rulesHandler.Delete)
		}

		// Rewards Catalog API
		rewards := tenants.Group("/reward-catalog")
		{
			rewards.POST("", middleware.RequireRole("owner", "admin"), rewardsHandler.Create)
			rewards.GET("", rewardsHandler.List)
			rewards.GET("/:id", rewardsHandler.Get)
			rewards.PATCH("/:id", middleware.RequireRole("owner", "admin"), rewardsHandler.Update)
			rewards.POST("/:id/upload-codes", middleware.RequireRole("owner", "admin"), rewardsHandler.UploadCodes)
		}

		// Issuances API
		issuances := tenants.Group("/issuances")
		{
			issuances.GET("", issuancesHandler.List)
			issuances.GET("/:id", issuancesHandler.Get)
			issuances.POST("/:id/redeem", issuancesHandler.Redeem)
			issuances.POST("/:id/cancel", middleware.RequireRole("owner", "admin", "staff"), issuancesHandler.Cancel)
		}

		// Budgets API
		budgets := tenants.Group("/budgets")
		{
			budgets.POST("", middleware.RequireRole("owner", "admin"), budgetsHandler.Create)
			budgets.GET("", budgetsHandler.List)
			budgets.GET("/:id", budgetsHandler.Get)
			budgets.POST("/:id/topup", middleware.RequireRole("owner", "admin"), budgetsHandler.Topup)
		}

		// Ledger API
		tenants.GET("/ledger", budgetsHandler.ListLedger)

		// Campaigns API
		campaigns := tenants.Group("/campaigns")
		{
			campaigns.POST("", middleware.RequireRole("owner", "admin"), campaignsHandler.Create)
			campaigns.GET("", campaignsHandler.List)
			campaigns.GET("/:id", campaignsHandler.Get)
			campaigns.PATCH("/:id", middleware.RequireRole("owner", "admin"), campaignsHandler.Update)
		}
	}

	return r
}

// HealthCheck returns the health status of the API
func HealthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// ReadyCheck verifies database connectivity
func ReadyCheck(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			c.JSON(503, gin.H{
				"status": "unavailable",
				"error":  "database connection failed",
			})
			return
		}

		c.JSON(200, gin.H{
			"status": "ready",
			"time":   time.Now().Format(time.RFC3339),
		})
	}
}
