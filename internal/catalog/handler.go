package catalog

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/authctx"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/httpserver"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ProductRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListProducts)
	r.Post("/", h.CreateProduct)
	r.Get("/{productId}", h.GetProduct)
	r.Patch("/{productId}", h.UpdateProduct)
	r.Delete("/{productId}", h.ArchiveProduct)
	r.Get("/{productId}/movements", h.ListMovements)

	return r
}

func (h *Handler) InventoryRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListInventory)
	r.Post("/adjustments", h.RecordStockAdjustment)

	return r
}

func extractMeta(r *http.Request) RequestMeta {
	return RequestMeta{
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		RequestID: httpserver.RequestIDFromContext(r.Context()),
	}
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	tenant, ok := authctx.TenantFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Tenant context missing", nil, nil)
		return
	}

	limit := 25
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	products, nextCursor, err := h.service.ListProducts(r.Context(), tenant.BusinessID, ListProductsQuery{
		Query:  r.URL.Query().Get("q"),
		Limit:  limit,
		Cursor: r.URL.Query().Get("cursor"),
	})
	if err != nil {
		httpserver.RespondError(w, r, http.StatusInternalServerError, "database_error", "Failed to list products", nil, nil)
		return
	}

	type listResponse struct {
		Products   []Product `json:"products"`
		NextCursor *string   `json:"nextCursor"`
	}

	var cursorPtr *string
	if nextCursor != "" {
		cursorPtr = &nextCursor
	}

	httpserver.RespondJSON(w, r, http.StatusOK, listResponse{
		Products:   products,
		NextCursor: cursorPtr,
	})
}

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	tenant, ok := authctx.TenantFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Tenant context missing", nil, nil)
		return
	}
	user, _ := authctx.UserFromContext(r.Context())

	var input CreateProductInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpserver.RespondError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON payload", nil, nil)
		return
	}

	product, err := h.service.CreateProduct(r.Context(), tenant.BusinessID, user.ID, input, extractMeta(r))
	if err != nil {
		if errors.Is(err, ErrInvalidProductData) {
			httpserver.RespondError(w, r, http.StatusUnprocessableEntity, "validation_failed", err.Error(), nil, map[string]string{
				"name": "name and SKU are required",
			})
			return
		}
		if errors.Is(err, ErrSKUAlreadyExists) {
			httpserver.RespondError(w, r, http.StatusConflict, "sku_exists", "A product with this SKU already exists in this business", nil, map[string]string{
				"sku": "SKU already in use",
			})
			return
		}
		httpserver.RespondError(w, r, http.StatusInternalServerError, "database_error", err.Error(), nil, nil)
		return
	}

	httpserver.RespondJSON(w, r, http.StatusCreated, product)
}

func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	tenant, ok := authctx.TenantFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Tenant context missing", nil, nil)
		return
	}
	productID := chi.URLParam(r, "productId")

	product, err := h.service.GetProduct(r.Context(), tenant.BusinessID, productID)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			httpserver.RespondError(w, r, http.StatusNotFound, "not_found", "Product not found", nil, nil)
			return
		}
		httpserver.RespondError(w, r, http.StatusInternalServerError, "database_error", "Failed to retrieve product", nil, nil)
		return
	}

	httpserver.RespondJSON(w, r, http.StatusOK, product)
}

func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	tenant, ok := authctx.TenantFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Tenant context missing", nil, nil)
		return
	}
	user, _ := authctx.UserFromContext(r.Context())
	productID := chi.URLParam(r, "productId")

	var input UpdateProductInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpserver.RespondError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON payload", nil, nil)
		return
	}

	product, err := h.service.UpdateProduct(r.Context(), tenant.BusinessID, productID, user.ID, input, extractMeta(r))
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			httpserver.RespondError(w, r, http.StatusNotFound, "not_found", "Product not found", nil, nil)
			return
		}
		if errors.Is(err, ErrInvalidProductData) {
			httpserver.RespondError(w, r, http.StatusUnprocessableEntity, "validation_failed", err.Error(), nil, nil)
			return
		}
		httpserver.RespondError(w, r, http.StatusInternalServerError, "database_error", err.Error(), nil, nil)
		return
	}

	httpserver.RespondJSON(w, r, http.StatusOK, product)
}

func (h *Handler) ArchiveProduct(w http.ResponseWriter, r *http.Request) {
	tenant, ok := authctx.TenantFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Tenant context missing", nil, nil)
		return
	}
	user, _ := authctx.UserFromContext(r.Context())
	productID := chi.URLParam(r, "productId")

	if err := h.service.ArchiveProduct(r.Context(), tenant.BusinessID, productID, user.ID, extractMeta(r)); err != nil {
		if errors.Is(err, ErrProductNotFound) {
			httpserver.RespondError(w, r, http.StatusNotFound, "not_found", "Product not found", nil, nil)
			return
		}
		httpserver.RespondError(w, r, http.StatusInternalServerError, "database_error", "Failed to archive product", nil, nil)
		return
	}

	httpserver.RespondJSON(w, r, http.StatusOK, map[string]string{
		"message": "Product archived successfully",
	})
}

func (h *Handler) ListInventory(w http.ResponseWriter, r *http.Request) {
	tenant, ok := authctx.TenantFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Tenant context missing", nil, nil)
		return
	}

	limit := 25
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	lowStock := false
	if lowStockStr := r.URL.Query().Get("lowStock"); lowStockStr == "true" || lowStockStr == "1" {
		lowStock = true
	}

	inventory, nextCursor, err := h.service.ListInventory(r.Context(), tenant.BusinessID, ListInventoryQuery{
		Query:        r.URL.Query().Get("q"),
		LowStockOnly: lowStock,
		Limit:        limit,
		Cursor:       r.URL.Query().Get("cursor"),
	})
	if err != nil {
		httpserver.RespondError(w, r, http.StatusInternalServerError, "database_error", "Failed to list inventory", nil, nil)
		return
	}

	type listResponse struct {
		Inventory  []ProductInventoryItem `json:"inventory"`
		NextCursor *string                `json:"nextCursor"`
	}

	var cursorPtr *string
	if nextCursor != "" {
		cursorPtr = &nextCursor
	}

	httpserver.RespondJSON(w, r, http.StatusOK, listResponse{
		Inventory:  inventory,
		NextCursor: cursorPtr,
	})
}

func (h *Handler) RecordStockAdjustment(w http.ResponseWriter, r *http.Request) {
	tenant, ok := authctx.TenantFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Tenant context missing", nil, nil)
		return
	}
	user, _ := authctx.UserFromContext(r.Context())

	var input StockAdjustmentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpserver.RespondError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON payload", nil, nil)
		return
	}

	// Support idempotency key from header if not in body
	idempotencyHeader := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyHeader != "" && (input.IdempotencyKey == nil || *input.IdempotencyKey == "") {
		input.IdempotencyKey = &idempotencyHeader
	}

	movement, err := h.service.RecordStockAdjustment(r.Context(), tenant.BusinessID, user.ID, input, extractMeta(r))
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			httpserver.RespondError(w, r, http.StatusNotFound, "not_found", "Product not found", nil, nil)
			return
		}
		if errors.Is(err, ErrNegativeStockNotAllowed) {
			httpserver.RespondError(w, r, http.StatusConflict, "negative_stock", "Stock deduction would result in negative inventory", nil, nil)
			return
		}
		if errors.Is(err, ErrInsufficientStock) {
			httpserver.RespondError(w, r, http.StatusConflict, "insufficient_stock", "Insufficient stock for requested operation", nil, nil)
			return
		}
		httpserver.RespondError(w, r, http.StatusInternalServerError, "database_error", err.Error(), nil, nil)
		return
	}

	httpserver.RespondJSON(w, r, http.StatusOK, movement)
}

func (h *Handler) ListMovements(w http.ResponseWriter, r *http.Request) {
	tenant, ok := authctx.TenantFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Tenant context missing", nil, nil)
		return
	}
	productID := chi.URLParam(r, "productId")

	limit := 25
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	movements, nextCursor, err := h.service.ListMovements(r.Context(), tenant.BusinessID, productID, ListMovementsQuery{
		Limit:  limit,
		Cursor: r.URL.Query().Get("cursor"),
	})
	if err != nil {
		httpserver.RespondError(w, r, http.StatusInternalServerError, "database_error", "Failed to list stock movements", nil, nil)
		return
	}

	type listResponse struct {
		Movements  []StockMovement `json:"movements"`
		NextCursor *string         `json:"nextCursor"`
	}

	var cursorPtr *string
	if nextCursor != "" {
		cursorPtr = &nextCursor
	}

	httpserver.RespondJSON(w, r, http.StatusOK, listResponse{
		Movements:  movements,
		NextCursor: cursorPtr,
	})
}
