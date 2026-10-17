//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/catalog"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/customers"
)

type customerEnvelope struct {
	Data customers.Customer `json:"data"`
}

type customerListEnvelope struct {
	Data struct {
		Customers  []customers.Customer `json:"customers"`
		NextCursor *string              `json:"nextCursor"`
	} `json:"data"`
}

type productEnvelope struct {
	Data catalog.Product `json:"data"`
}

type productListEnvelope struct {
	Data struct {
		Products   []catalog.Product `json:"products"`
		NextCursor *string           `json:"nextCursor"`
	} `json:"data"`
}

type inventoryListEnvelope struct {
	Data struct {
		Inventory  []catalog.ProductInventoryItem `json:"inventory"`
		NextCursor *string                        `json:"nextCursor"`
	} `json:"data"`
}

type stockMovementEnvelope struct {
	Data catalog.StockMovement `json:"data"`
}

type stockMovementsListEnvelope struct {
	Data struct {
		Movements  []catalog.StockMovement `json:"movements"`
		NextCursor *string                 `json:"nextCursor"`
	} `json:"data"`
}

func TestCustomersAndCatalogLifecycle(t *testing.T) {
	handler, db, redisClient, _ := setupTestApp(t)
	defer db.Close()
	defer redisClient.Close()

	uniqueSuffix := fmt.Sprintf("%d", time.Now().UnixNano())
	aliceEmail := fmt.Sprintf("alice_cc_%s@example.com", uniqueSuffix)
	bobEmail := fmt.Sprintf("bob_cc_%s@example.com", uniqueSuffix)

	// Register Alice with Business A
	aliceReg := doRequest(t, handler, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"email":        aliceEmail,
		"password":     "PasswordAlice123!",
		"fullName":     "Alice Merchant",
		"businessName": "Alice Store",
	})
	if aliceReg.Code != http.StatusCreated {
		t.Fatalf("alice register status = %d; body = %s", aliceReg.Code, aliceReg.Body.String())
	}
	var aliceAuth authResponseEnvelope
	_ = json.Unmarshal(aliceReg.Body.Bytes(), &aliceAuth)
	aliceToken := aliceAuth.Data.AccessToken

	aliceMe := doRequest(t, handler, http.MethodGet, "/api/v1/me", aliceToken, nil)
	var aliceMeResp meResponseEnvelope
	_ = json.Unmarshal(aliceMe.Body.Bytes(), &aliceMeResp)
	aliceBizID := aliceMeResp.Data.Memberships[0].BusinessID

	// Register Bob with Business B
	bobReg := doRequest(t, handler, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"email":        bobEmail,
		"password":     "PasswordBob123!",
		"fullName":     "Bob Competitor",
		"businessName": "Bob Store",
	})
	if bobReg.Code != http.StatusCreated {
		t.Fatalf("bob register status = %d; body = %s", bobReg.Code, bobReg.Body.String())
	}
	var bobAuth authResponseEnvelope
	_ = json.Unmarshal(bobReg.Body.Bytes(), &bobAuth)
	bobToken := bobAuth.Data.AccessToken

	bobMe := doRequest(t, handler, http.MethodGet, "/api/v1/me", bobToken, nil)
	var bobMeResp meResponseEnvelope
	_ = json.Unmarshal(bobMe.Body.Bytes(), &bobMeResp)
	bobBizID := bobMeResp.Data.Memberships[0].BusinessID

	var customerID string

	// 1. Customer CRUD & Tenant Isolation
	t.Run("Create Customer in Alice Business", func(t *testing.T) {
		rec := doRequest(t, handler, http.MethodPost, "/api/v1/businesses/"+aliceBizID+"/customers", aliceToken, map[string]any{
			"name":    "Acme Corp",
			"phone":   "+919876543210",
			"email":   "contact@acme.com",
			"address": "123 Market Street, Mumbai",
			"notes":   "VIP wholesale client",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("create customer status = %d; body = %s", rec.Code, rec.Body.String())
		}
		var resp customerEnvelope
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal customer response: %v", err)
		}
		if resp.Data.ID == "" || resp.Data.Name != "Acme Corp" {
			t.Fatalf("unexpected customer response: %+v", resp.Data)
		}
		customerID = resp.Data.ID
	})

	t.Run("Cross-Tenant Customer Isolation", func(t *testing.T) {
		// Bob attempts to get Alice's customer via Alice's business endpoint (403 forbidden)
		rec1 := doRequest(t, handler, http.MethodGet, "/api/v1/businesses/"+aliceBizID+"/customers/"+customerID, bobToken, nil)
		if rec1.Code != http.StatusForbidden {
			t.Fatalf("bob accessing alice business customer status = %d, want 403", rec1.Code)
		}

		// Bob attempts to get Alice's customer via Bob's business endpoint (404 not found)
		rec2 := doRequest(t, handler, http.MethodGet, "/api/v1/businesses/"+bobBizID+"/customers/"+customerID, bobToken, nil)
		if rec2.Code != http.StatusNotFound {
			t.Fatalf("bob querying alice customer under bob biz status = %d, want 404", rec2.Code)
		}
	})

	t.Run("Update and Search Customer", func(t *testing.T) {
		rec := doRequest(t, handler, http.MethodPatch, "/api/v1/businesses/"+aliceBizID+"/customers/"+customerID, aliceToken, map[string]any{
			"name": "Acme Global Corp",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("update customer status = %d; body = %s", rec.Code, rec.Body.String())
		}

		// Search customers
		searchRec := doRequest(t, handler, http.MethodGet, "/api/v1/businesses/"+aliceBizID+"/customers?q=Acme", aliceToken, nil)
		if searchRec.Code != http.StatusOK {
			t.Fatalf("search customer status = %d; body = %s", searchRec.Code, searchRec.Body.String())
		}
		var listResp customerListEnvelope
		_ = json.Unmarshal(searchRec.Body.Bytes(), &listResp)
		if len(listResp.Data.Customers) != 1 || listResp.Data.Customers[0].Name != "Acme Global Corp" {
			t.Fatalf("search customer result unexpected: %+v", listResp.Data.Customers)
		}
	})

	// 2. Product Catalog, SKU Uniqueness & Stock Tracking
	var productID string
	testSKU := fmt.Sprintf("SKU-%s", uniqueSuffix)

	t.Run("Create Product with Initial Stock", func(t *testing.T) {
		rec := doRequest(t, handler, http.MethodPost, "/api/v1/businesses/"+aliceBizID+"/products", aliceToken, map[string]any{
			"name":               "Mechanical Keyboard",
			"sku":                testSKU,
			"description":        "RGB mechanical keyboard with blue switches",
			"unit":               "piece",
			"salePriceMinor":     450000, // 4500.00 INR
			"costPriceMinor":     280000, // 2800.00 INR
			"currency":           "INR",
			"taxRateBasisPoints": 1800,   // 18.00%
			"reorderLevel":       5,
			"initialStock":       20,
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("create product status = %d; body = %s", rec.Code, rec.Body.String())
		}
		var resp productEnvelope
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp.Data.ID == "" || resp.Data.SKU != testSKU || resp.Data.OnHand != 20 {
			t.Fatalf("unexpected product data: %+v", resp.Data)
		}
		productID = resp.Data.ID
	})

	t.Run("Prevent Duplicate SKU in Same Business", func(t *testing.T) {
		rec := doRequest(t, handler, http.MethodPost, "/api/v1/businesses/"+aliceBizID+"/products", aliceToken, map[string]any{
			"name":           "Another Keyboard",
			"sku":            testSKU,
			"salePriceMinor": 300000,
		})
		if rec.Code != http.StatusConflict {
			t.Fatalf("duplicate SKU in same business status = %d, want 409", rec.Code)
		}
	})

	t.Run("Allow Same SKU in Different Business", func(t *testing.T) {
		rec := doRequest(t, handler, http.MethodPost, "/api/v1/businesses/"+bobBizID+"/products", bobToken, map[string]any{
			"name":           "Bob Keyboard",
			"sku":            testSKU,
			"salePriceMinor": 500000,
			"initialStock":   10,
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("same SKU in different business status = %d, want 201; body = %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Record Stock Adjustment and Prevent Negative Stock", func(t *testing.T) {
		// Attempting to deduct more stock than available (current = 20, deduct 25)
		failRec := doRequest(t, handler, http.MethodPost, "/api/v1/businesses/"+aliceBizID+"/inventory/adjustments", aliceToken, map[string]any{
			"productId": productID,
			"quantity":  -25,
			"notes":     "Damaged batch",
		})
		if failRec.Code != http.StatusConflict {
			t.Fatalf("oversell deduction status = %d, want 409; body = %s", failRec.Code, failRec.Body.String())
		}

		// Valid deduction: deduct 12 units
		idempotencyKey := fmt.Sprintf("adj_%s", uniqueSuffix)
		rec := doRequest(t, handler, http.MethodPost, "/api/v1/businesses/"+aliceBizID+"/inventory/adjustments", aliceToken, map[string]any{
			"productId":      productID,
			"quantity":       -12,
			"movementType":   "sale",
			"notes":          "Direct counter sale",
			"idempotencyKey": idempotencyKey,
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("valid stock deduction status = %d; body = %s", rec.Code, rec.Body.String())
		}

		// Idempotency replay check: repeating the same request returns 200 without double-deducting
		replayRec := doRequest(t, handler, http.MethodPost, "/api/v1/businesses/"+aliceBizID+"/inventory/adjustments", aliceToken, map[string]any{
			"productId":      productID,
			"quantity":       -12,
			"movementType":   "sale",
			"notes":          "Direct counter sale",
			"idempotencyKey": idempotencyKey,
		})
		if replayRec.Code != http.StatusOK {
			t.Fatalf("idempotent replay status = %d, want 200", replayRec.Code)
		}

		// Check product balance: was 20, deducted 12 -> should be 8
		prodRec := doRequest(t, handler, http.MethodGet, "/api/v1/businesses/"+aliceBizID+"/products/"+productID, aliceToken, nil)
		var prodResp productEnvelope
		_ = json.Unmarshal(prodRec.Body.Bytes(), &prodResp)
		if prodResp.Data.OnHand != 8 {
			t.Fatalf("expected on_hand = 8, got %d", prodResp.Data.OnHand)
		}
	})

	t.Run("List Stock Movements for Product", func(t *testing.T) {
		rec := doRequest(t, handler, http.MethodGet, "/api/v1/businesses/"+aliceBizID+"/products/"+productID+"/movements", aliceToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("list movements status = %d; body = %s", rec.Code, rec.Body.String())
		}
		var moveResp stockMovementsListEnvelope
		_ = json.Unmarshal(rec.Body.Bytes(), &moveResp)
		// Expected movements: initial stock (+20) and adjustment (-12)
		if len(moveResp.Data.Movements) != 2 {
			t.Fatalf("expected 2 movements, got %d", len(moveResp.Data.Movements))
		}
	})

	t.Run("List Inventory and Low Stock Check", func(t *testing.T) {
		// Reorder level is 5, onHand is 8 -> not low stock yet
		rec := doRequest(t, handler, http.MethodGet, "/api/v1/businesses/"+aliceBizID+"/inventory?lowStock=true", aliceToken, nil)
		var invResp inventoryListEnvelope
		_ = json.Unmarshal(rec.Body.Bytes(), &invResp)
		if len(invResp.Data.Inventory) != 0 {
			t.Fatalf("expected 0 low stock items, got %d", len(invResp.Data.Inventory))
		}

		// Deduct 4 more units -> onHand becomes 4 (<= reorder level 5)
		doRequest(t, handler, http.MethodPost, "/api/v1/businesses/"+aliceBizID+"/inventory/adjustments", aliceToken, map[string]any{
			"productId": productID,
			"quantity":  -4,
		})

		// Query low stock again -> must show the product
		lowRec := doRequest(t, handler, http.MethodGet, "/api/v1/businesses/"+aliceBizID+"/inventory?lowStock=true", aliceToken, nil)
		var lowResp inventoryListEnvelope
		_ = json.Unmarshal(lowRec.Body.Bytes(), &lowResp)
		if len(lowResp.Data.Inventory) != 1 || lowResp.Data.Inventory[0].ProductID != productID {
			t.Fatalf("expected 1 low stock item, got %+v", lowResp.Data.Inventory)
		}
	})

	t.Run("Archive Customer and Product", func(t *testing.T) {
		delCust := doRequest(t, handler, http.MethodDelete, "/api/v1/businesses/"+aliceBizID+"/customers/"+customerID, aliceToken, nil)
		if delCust.Code != http.StatusOK {
			t.Fatalf("archive customer status = %d", delCust.Code)
		}

		// Subsequent get must return 404
		getCust := doRequest(t, handler, http.MethodGet, "/api/v1/businesses/"+aliceBizID+"/customers/"+customerID, aliceToken, nil)
		if getCust.Code != http.StatusNotFound {
			t.Fatalf("archived customer get status = %d, want 404", getCust.Code)
		}

		delProd := doRequest(t, handler, http.MethodDelete, "/api/v1/businesses/"+aliceBizID+"/products/"+productID, aliceToken, nil)
		if delProd.Code != http.StatusOK {
			t.Fatalf("archive product status = %d", delProd.Code)
		}

		// Subsequent get must return 404
		getProd := doRequest(t, handler, http.MethodGet, "/api/v1/businesses/"+aliceBizID+"/products/"+productID, aliceToken, nil)
		if getProd.Code != http.StatusNotFound {
			t.Fatalf("archived product get status = %d, want 404", getProd.Code)
		}
	})
}
