//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/audit"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/businesses"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/identity"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/httpserver"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/observability"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/postgres"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/ratelimit"
	redisstore "github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/redis"
)

func setupTestApp(t *testing.T) (http.Handler, *postgres.Client, *redisstore.Client, []byte) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbURL := environmentOrDefault("TEST_DATABASE_URL", "postgres://aibos:aibos_local@localhost:5433/aibos?sslmode=disable")
	database, err := postgres.Open(ctx, postgres.Config{
		URL:             dbURL,
		PoolMax:         5,
		PoolMin:         1,
		ConnectTimeout:  3 * time.Second,
		HealthTimeout:   3 * time.Second,
		MaxConnLifetime: 5 * time.Minute,
		MaxConnIdleTime: time.Minute,
	})
	if err != nil {
		t.Fatalf("open PostgreSQL pool: %v", err)
	}

	redisURL := environmentOrDefault("TEST_REDIS_URL", "redis://localhost:6380/0")
	cache, err := redisstore.Open(redisstore.Config{
		URL:           redisURL,
		DialTimeout:   3 * time.Second,
		ReadTimeout:   3 * time.Second,
		WriteTimeout:  3 * time.Second,
		HealthTimeout: 3 * time.Second,
	})
	if err != nil {
		t.Fatalf("open Redis client: %v", err)
	}

	jwtSecret := []byte("test-suite-jwt-secret-key-at-least-32-chars-long")
	auditRepo := audit.NewRepository(database.RawPool())
	rateLimiter := ratelimit.New(cache.RawClient())

	bizRepo := businesses.NewRepository(database.RawPool())
	bizService := businesses.NewService(bizRepo, auditRepo)
	bizHandler := businesses.NewHandler(bizService)

	userRepo := identity.NewRepository(database.RawPool())
	identityService := identity.NewService(
		userRepo,
		bizRepo,
		auditRepo,
		jwtSecret,
		15*time.Minute,
		7*24*time.Hour,
	)
	identityHandler := identity.NewHandler(identityService, bizService, rateLimiter, false)

	server := httpserver.New(httpserver.Config{
		Addr:              "127.0.0.1:0",
		FrontendOrigins:   []string{"http://localhost:3000"},
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       time.Second,
		WriteTimeout:      2 * time.Second,
		IdleTimeout:       time.Second,
		RequestTimeout:    10 * time.Second,
		ShutdownTimeout:   time.Second,
		MaxBodyBytes:      1024 * 1024,
		Production:        false,
		JWTSecret:         jwtSecret,
	}, slog.New(slog.NewJSONHandler(io.Discard, nil)), httpserver.Dependencies{
		Postgres:   database,
		Redis:      cache,
		Identity:   identityHandler.Routes(jwtSecret),
		Businesses: bizHandler.Routes(jwtSecret),
		Me:         http.HandlerFunc(identityHandler.GetMe),
	}, observability.NoopHooks())

	return server.Handler(), database, cache, jwtSecret
}

var testIPCounter uint64

func doRequest(t *testing.T, handler http.Handler, method, path string, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reqBody = bytes.NewReader(jsonBytes)
	}

	req := httptest.NewRequest(method, path, reqBody)
	testIPCounter++
	req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", testIPCounter%250+1)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

type authResponseEnvelope struct {
	Data identity.AuthTokens `json:"data"`
}

type businessResponseEnvelope struct {
	Data businesses.Business `json:"data"`
}

type meResponseEnvelope struct {
	Data struct {
		User        identity.User                    `json:"user"`
		Memberships []businesses.UserBusinessSummary `json:"memberships"`
	} `json:"data"`
}

func TestIdentityAndTenancyLifecycle(t *testing.T) {
	handler, db, redisClient, _ := setupTestApp(t)
	defer db.Close()
	defer redisClient.Close()

	uniqueSuffix := fmt.Sprintf("%d", time.Now().UnixNano())
	aliceEmail := fmt.Sprintf("alice_%s@example.com", uniqueSuffix)
	bobEmail := fmt.Sprintf("bob_%s@example.com", uniqueSuffix)
	carolEmail := fmt.Sprintf("carol_%s@example.com", uniqueSuffix)

	// 1. User A (Alice) Registration
	t.Run("Register Alice with new Business", func(t *testing.T) {
		rec := doRequest(t, handler, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
			"email":        aliceEmail,
			"password":     "SuperPassword123!",
			"name":         "Alice Baker",
			"businessName": "Alice's Artisan Bakery",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("register status = %d, want 201; body = %s", rec.Code, rec.Body.String())
		}

		var resp authResponseEnvelope
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal register response: %v", err)
		}
		if resp.Data.AccessToken == "" || resp.Data.RefreshToken == "" {
			t.Fatal("expected access and refresh tokens")
		}
		if resp.Data.User.Email != aliceEmail {
			t.Fatalf("user email = %q, want %q", resp.Data.User.Email, aliceEmail)
		}
	})

	// 2. Prevent Duplicate Registration
	t.Run("Reject Duplicate Email Registration", func(t *testing.T) {
		rec := doRequest(t, handler, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
			"email":    aliceEmail,
			"password": "AnotherPassword123!",
			"name":     "Alice Clone",
		})
		if rec.Code != http.StatusConflict {
			t.Fatalf("duplicate register status = %d, want 409", rec.Code)
		}
	})

	// 3. User Login
	var aliceToken, aliceRefreshToken string
	t.Run("Login Alice with valid credentials", func(t *testing.T) {
		// Wrong password fails
		failRec := doRequest(t, handler, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
			"email":    aliceEmail,
			"password": "WrongPassword!",
		})
		if failRec.Code != http.StatusUnauthorized {
			t.Fatalf("wrong password status = %d, want 401", failRec.Code)
		}

		// Correct password succeeds
		rec := doRequest(t, handler, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
			"email":    aliceEmail,
			"password": "SuperPassword123!",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("login status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}

		var resp authResponseEnvelope
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		aliceToken = resp.Data.AccessToken
		aliceRefreshToken = resp.Data.RefreshToken
	})

	// 4. Token Rotation / Refresh
	t.Run("Token Refresh and Rotation", func(t *testing.T) {
		rec := doRequest(t, handler, http.MethodPost, "/api/v1/auth/refresh", "", map[string]string{
			"refreshToken": aliceRefreshToken,
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("refresh status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}

		var resp authResponseEnvelope
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		newRefreshToken := resp.Data.RefreshToken
		aliceToken = resp.Data.AccessToken // update active token

		// Attempting to reuse old refresh token MUST fail
		reuseRec := doRequest(t, handler, http.MethodPost, "/api/v1/auth/refresh", "", map[string]string{
			"refreshToken": aliceRefreshToken,
		})
		if reuseRec.Code != http.StatusUnauthorized {
			t.Fatalf("reused token status = %d, want 401", reuseRec.Code)
		}
		aliceRefreshToken = newRefreshToken
	})

	// 5. Get Current User (/api/v1/me)
	var aliceBusinessID string
	t.Run("Get Alice Profile and Businesses", func(t *testing.T) {
		rec := doRequest(t, handler, http.MethodGet, "/api/v1/me", aliceToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("me status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}

		var resp meResponseEnvelope
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal me response: %v", err)
		}
		if len(resp.Data.Memberships) == 0 {
			t.Fatal("expected Alice to have at least 1 membership")
		}
		aliceBusinessID = resp.Data.Memberships[0].BusinessID
		if resp.Data.Memberships[0].Role != "owner" {
			t.Fatalf("expected role owner, got %s", resp.Data.Memberships[0].Role)
		}
	})

	// 6. Register User B (Bob) with separate business
	var bobToken, bobBusinessID string
	t.Run("Register Bob with separate Business", func(t *testing.T) {
		rec := doRequest(t, handler, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
			"email":        bobEmail,
			"password":     "BobsPassword123!",
			"name":         "Bob Builder",
			"businessName": "Bob's Auto Garage",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("bob register status = %d, want 201; body = %s", rec.Code, rec.Body.String())
		}
		var resp authResponseEnvelope
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		bobToken = resp.Data.AccessToken

		meRec := doRequest(t, handler, http.MethodGet, "/api/v1/me", bobToken, nil)
		var meResp meResponseEnvelope
		_ = json.Unmarshal(meRec.Body.Bytes(), &meResp)
		bobBusinessID = meResp.Data.Memberships[0].BusinessID
	})

	// 7. Multi-Tenant Isolation Verification
	t.Run("Cross-Tenant IDOR Protection", func(t *testing.T) {
		// Bob attempts to read Alice's business
		rec := doRequest(t, handler, http.MethodGet, "/api/v1/businesses/"+aliceBusinessID, bobToken, nil)
		if rec.Code != http.StatusForbidden && rec.Code != http.StatusNotFound {
			t.Fatalf("cross-tenant read status = %d, want 403 or 404; body = %s", rec.Code, rec.Body.String())
		}

		// Bob attempts to update Alice's business settings
		patchRec := doRequest(t, handler, http.MethodPatch, "/api/v1/businesses/"+aliceBusinessID, bobToken, map[string]string{
			"name": "Hacked Bakery",
		})
		if patchRec.Code != http.StatusForbidden && patchRec.Code != http.StatusNotFound {
			t.Fatalf("cross-tenant update status = %d, want 403 or 404", patchRec.Code)
		}

		// Bob attempts to list members of Alice's business
		membersRec := doRequest(t, handler, http.MethodGet, "/api/v1/businesses/"+aliceBusinessID+"/members", bobToken, nil)
		if membersRec.Code != http.StatusForbidden && membersRec.Code != http.StatusNotFound {
			t.Fatalf("cross-tenant list members status = %d, want 403 or 404", membersRec.Code)
		}

		// Alice successfully accesses her own business
		aliceRec := doRequest(t, handler, http.MethodGet, "/api/v1/businesses/"+aliceBusinessID, aliceToken, nil)
		if aliceRec.Code != http.StatusOK {
			t.Fatalf("alice read own business status = %d, want 200", aliceRec.Code)
		}
		var bizResp businessResponseEnvelope
		_ = json.Unmarshal(aliceRec.Body.Bytes(), &bizResp)
		if bizResp.Data.Name != "Alice's Artisan Bakery" {
			t.Fatalf("business name = %q, want Alice's Artisan Bakery", bizResp.Data.Name)
		}
	})

	// 8. Role-Based Access Control (Owner vs Staff)
	t.Run("Role Authorization in Tenant", func(t *testing.T) {
		// Register Carol first
		carolReg := doRequest(t, handler, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
			"email":    carolEmail,
			"password": "CarolsPassword123!",
			"name":     "Carol Cashier",
		})
		if carolReg.Code != http.StatusCreated {
			t.Fatalf("carol register status = %d", carolReg.Code)
		}
		var carolResp authResponseEnvelope
		_ = json.Unmarshal(carolReg.Body.Bytes(), &carolResp)
		carolToken := carolResp.Data.AccessToken

		// Alice (Owner) invites Carol to Alice's Bakery as 'staff'
		inviteRec := doRequest(t, handler, http.MethodPost, "/api/v1/businesses/"+aliceBusinessID+"/invitations", aliceToken, map[string]string{
			"email": carolEmail,
			"role":  "staff",
		})
		if inviteRec.Code != http.StatusCreated {
			t.Fatalf("invite member status = %d, want 201; body = %s", inviteRec.Code, inviteRec.Body.String())
		}

		// Carol (Staff) attempts to modify business settings (owner-only operation)
		// Note: first Carol's membership is invited, let's update Carol status to active directly or test role check
		// Update Carol's membership status to active
		_, err := db.RawPool().Exec(context.Background(),
			"UPDATE memberships SET status = 'active' WHERE business_id = $1 AND user_id = $2",
			aliceBusinessID, carolResp.Data.User.ID,
		)
		if err != nil {
			t.Fatalf("activate membership: %v", err)
		}

		// Carol attempts owner-only update
		staffPatch := doRequest(t, handler, http.MethodPatch, "/api/v1/businesses/"+aliceBusinessID, carolToken, map[string]string{
			"name": "Staff Changed Name",
		})
		if staffPatch.Code != http.StatusForbidden {
			t.Fatalf("staff update business status = %d, want 403; body = %s", staffPatch.Code, staffPatch.Body.String())
		}
	})

	// 9. Unauthenticated Access Rejection
	t.Run("Reject Unauthenticated Requests", func(t *testing.T) {
		rec := doRequest(t, handler, http.MethodGet, "/api/v1/me", "", nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated status = %d, want 401", rec.Code)
		}

		recBiz := doRequest(t, handler, http.MethodGet, "/api/v1/businesses", "invalid-token", nil)
		if recBiz.Code != http.StatusUnauthorized {
			t.Fatalf("invalid token status = %d, want 401", recBiz.Code)
		}
	})

	// 10. Rate Limiting Verification
	t.Run("Rate Limiting on Auth Endpoints", func(t *testing.T) {
		spamIP := fmt.Sprintf("192.0.2.%d:54321", time.Now().UnixNano()%200+1)
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(`{"email":"nonexistent@example.com","password":"bad"}`)))
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = spamIP
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}

		// 11th attempt from same IP must return 429 Too Many Requests
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(`{"email":"nonexistent@example.com","password":"bad"}`)))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = spamIP
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("expected 429 rate limit exceeded, got %d; body = %s", rec.Code, rec.Body.String())
		}
	})

	_ = bobBusinessID
}
