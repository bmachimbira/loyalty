package api

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bmachimbira/loyalty/api/internal/db"
	httphandlers "github.com/bmachimbira/loyalty/api/internal/http/handlers"
	"github.com/bmachimbira/loyalty/api/internal/testutil"
)

func setupCustomersAPI(t *testing.T) (*gin.Engine, *db.Queries, db.Tenant) {
	pool, queries := testutil.SetupTestDB(t)

	router := testutil.NewTestRouter(t)

	// Setup handlers
	customersHandler := httphandlers.NewCustomersHandler(queries)

	// Register routes
	v1 := router.Group("/v1/tenants/:tid")
	{
		v1.POST("/customers", customersHandler.Create)
		v1.GET("/customers", customersHandler.List)
		v1.GET("/customers/:id", customersHandler.Get)
		v1.PATCH("/customers/:id/status", customersHandler.UpdateStatus)
	}

	// Create test tenant
	tenant := testutil.CreateTestTenant(t, queries)

	return router, queries, tenant
}

func TestCustomersAPI_Create(t *testing.T) {
	router, _, tenant := setupCustomersAPI(t)

	reqBody := map[string]interface{}{
		"phone_e164":   "+263771234567",
		"external_ref": "CUST001",
		"status":       "active",
	}

	url := fmt.Sprintf("/v1/tenants/%s/customers", tenant.ID.Bytes[:])
	w := testutil.MakeGinRequest(t, router, "POST", url, reqBody)

	assert.Equal(t, http.StatusCreated, w.Code, "Should return 201 Created")

	var response map[string]interface{}
	testutil.ParseGinResponse(t, w, &response)

	assert.NotEmpty(t, response["id"], "Response should include customer ID")
	assert.Equal(t, "+263771234567", response["phone_e164"], "Phone should match")
	assert.Equal(t, "CUST001", response["external_ref"], "External ref should match")
}

func TestCustomersAPI_Create_InvalidPhone(t *testing.T) {
	router, _, tenant := setupCustomersAPI(t)

	reqBody := map[string]interface{}{
		"phone_e164": "invalid-phone",
		"status":     "active",
	}

	url := fmt.Sprintf("/v1/tenants/%s/customers", tenant.ID.Bytes[:])
	w := testutil.MakeGinRequest(t, router, "POST", url, reqBody)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request for invalid phone")

	var errorResp map[string]interface{}
	testutil.ParseGinResponse(t, w, &errorResp)

	assert.Contains(t, errorResp["error"], "phone", "Error should mention phone validation")
}

func TestCustomersAPI_Create_MissingFields(t *testing.T) {
	router, _, tenant := setupCustomersAPI(t)

	reqBody := map[string]interface{}{}

	url := fmt.Sprintf("/v1/tenants/%s/customers", tenant.ID.Bytes[:])
	w := testutil.MakeGinRequest(t, router, "POST", url, reqBody)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request for missing fields")
}

func TestCustomersAPI_Get(t *testing.T) {
	router, queries, tenant := setupCustomersAPI(t)

	// Create a customer
	customer := testutil.CreateTestCustomer(t, queries, tenant.ID,
		testutil.WithPhone("+263771234567"),
		testutil.WithExternalRef("CUST001"),
	)

	url := fmt.Sprintf("/v1/tenants/%s/customers/%s", tenant.ID.Bytes[:], customer.ID.Bytes[:])
	w := testutil.MakeGinRequest(t, router, "GET", url, nil)

	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK")

	var response map[string]interface{}
	testutil.ParseGinResponse(t, w, &response)

	assert.Equal(t, customer.ID.Bytes[:], response["id"], "Customer ID should match")
	assert.Equal(t, "+263771234567", response["phone_e164"], "Phone should match")
}

func TestCustomersAPI_Get_NotFound(t *testing.T) {
	router, _, tenant := setupCustomersAPI(t)

	// Try to get non-existent customer
	fakeID := testutil.NewUUID(t)
	url := fmt.Sprintf("/v1/tenants/%s/customers/%s", tenant.ID.Bytes[:], fakeID.Bytes[:])
	w := testutil.MakeGinRequest(t, router, "GET", url, nil)

	assert.Equal(t, http.StatusNotFound, w.Code, "Should return 404 Not Found")
}

func TestCustomersAPI_List(t *testing.T) {
	router, queries, tenant := setupCustomersAPI(t)

	// Create multiple customers
	testutil.CreateTestCustomer(t, queries, tenant.ID,
		testutil.WithPhone("+263771111111"),
	)
	testutil.CreateTestCustomer(t, queries, tenant.ID,
		testutil.WithPhone("+263772222222"),
	)
	testutil.CreateTestCustomer(t, queries, tenant.ID,
		testutil.WithPhone("+263773333333"),
	)

	url := fmt.Sprintf("/v1/tenants/%s/customers?limit=10&offset=0", tenant.ID.Bytes[:])
	w := testutil.MakeGinRequest(t, router, "GET", url, nil)

	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK")

	var response map[string]interface{}
	testutil.ParseGinResponse(t, w, &response)

	customers, ok := response["customers"].([]interface{})
	require.True(t, ok, "Response should include customers array")
	assert.GreaterOrEqual(t, len(customers), 3, "Should have at least 3 customers")
}

func TestCustomersAPI_List_Pagination(t *testing.T) {
	router, queries, tenant := setupCustomersAPI(t)

	// Create 5 customers
	for i := 0; i < 5; i++ {
		testutil.CreateTestCustomer(t, queries, tenant.ID,
			testutil.WithPhone(fmt.Sprintf("+26377%07d", i)),
		)
	}

	// Get first page (limit 2)
	url := fmt.Sprintf("/v1/tenants/%s/customers?limit=2&offset=0", tenant.ID.Bytes[:])
	w := testutil.MakeGinRequest(t, router, "GET", url, nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	testutil.ParseGinResponse(t, w, &response)

	customers, ok := response["customers"].([]interface{})
	require.True(t, ok)
	assert.LessOrEqual(t, len(customers), 2, "Should have at most 2 customers")

	// Get second page (offset 2)
	url = fmt.Sprintf("/v1/tenants/%s/customers?limit=2&offset=2", tenant.ID.Bytes[:])
	w = testutil.MakeGinRequest(t, router, "GET", url, nil)

	assert.Equal(t, http.StatusOK, w.Code)

	testutil.ParseGinResponse(t, w, &response)
	customers2, ok := response["customers"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(customers2), 1, "Should have customers on second page")
}

func TestCustomersAPI_UpdateStatus(t *testing.T) {
	router, queries, tenant := setupCustomersAPI(t)

	// Create a customer
	customer := testutil.CreateTestCustomer(t, queries, tenant.ID,
		testutil.WithCustomerStatus("active"),
	)

	reqBody := map[string]interface{}{
		"status": "inactive",
	}

	url := fmt.Sprintf("/v1/tenants/%s/customers/%s/status", tenant.ID.Bytes[:], customer.ID.Bytes[:])
	w := testutil.MakeGinRequest(t, router, "PATCH", url, reqBody)

	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK")

	var response map[string]interface{}
	testutil.ParseGinResponse(t, w, &response)

	assert.Equal(t, "inactive", response["status"], "Status should be updated to inactive")
}

func TestCustomersAPI_UpdateStatus_InvalidStatus(t *testing.T) {
	router, queries, tenant := setupCustomersAPI(t)

	customer := testutil.CreateTestCustomer(t, queries, tenant.ID)

	reqBody := map[string]interface{}{
		"status": "invalid-status",
	}

	url := fmt.Sprintf("/v1/tenants/%s/customers/%s/status", tenant.ID.Bytes[:], customer.ID.Bytes[:])
	w := testutil.MakeGinRequest(t, router, "PATCH", url, reqBody)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request for invalid status")
}

func TestCustomersAPI_CrossTenantAccess(t *testing.T) {
	router, queries, tenant1 := setupCustomersAPI(t)

	// Create second tenant and customer
	tenant2 := testutil.CreateTestTenant(t, queries, testutil.WithTenantName("Tenant 2"))
	customer2 := testutil.CreateTestCustomer(t, queries, tenant2.ID)

	// Try to access tenant2's customer from tenant1's context
	url := fmt.Sprintf("/v1/tenants/%s/customers/%s", tenant1.ID.Bytes[:], customer2.ID.Bytes[:])
	w := testutil.MakeGinRequest(t, router, "GET", url, nil)

	// Should return 404 or 403 due to RLS
	assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusForbidden,
		"Should return 404 or 403 for cross-tenant access")
}
