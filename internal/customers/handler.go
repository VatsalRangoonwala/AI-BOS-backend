package customers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

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

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListCustomers)
	r.Post("/", h.CreateCustomer)
	r.Get("/{customerId}", h.GetCustomer)
	r.Patch("/{customerId}", h.UpdateCustomer)
	r.Delete("/{customerId}", h.ArchiveCustomer)
	r.Get("/{customerId}/ledger", h.GetCustomerLedger)

	return r
}

func extractMeta(r *http.Request) RequestMeta {
	return RequestMeta{
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		RequestID: httpserver.RequestIDFromContext(r.Context()),
	}
}

func (h *Handler) ListCustomers(w http.ResponseWriter, r *http.Request) {
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

	customers, nextCursor, err := h.service.ListCustomers(r.Context(), tenant.BusinessID, ListCustomersQuery{
		Query:  r.URL.Query().Get("q"),
		Limit:  limit,
		Cursor: r.URL.Query().Get("cursor"),
	})
	if err != nil {
		httpserver.RespondError(w, r, http.StatusInternalServerError, "database_error", "Failed to list customers", nil, nil)
		return
	}

	type listResponse struct {
		Customers  []Customer `json:"customers"`
		NextCursor *string    `json:"nextCursor"`
	}

	var cursorPtr *string
	if nextCursor != "" {
		cursorPtr = &nextCursor
	}

	httpserver.RespondJSON(w, r, http.StatusOK, listResponse{
		Customers:  customers,
		NextCursor: cursorPtr,
	})
}

func (h *Handler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	tenant, ok := authctx.TenantFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Tenant context missing", nil, nil)
		return
	}
	user, _ := authctx.UserFromContext(r.Context())

	var input CreateCustomerInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpserver.RespondError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON payload", nil, nil)
		return
	}

	customer, err := h.service.CreateCustomer(r.Context(), tenant.BusinessID, user.ID, input, extractMeta(r))
	if err != nil {
		if errors.Is(err, ErrInvalidCustomerName) {
			httpserver.RespondError(w, r, http.StatusUnprocessableEntity, "validation_failed", err.Error(), nil, map[string]string{
				"name": "name cannot be empty",
			})
			return
		}
		httpserver.RespondError(w, r, http.StatusInternalServerError, "database_error", "Failed to create customer", nil, nil)
		return
	}

	httpserver.RespondJSON(w, r, http.StatusCreated, customer)
}

func (h *Handler) GetCustomer(w http.ResponseWriter, r *http.Request) {
	tenant, ok := authctx.TenantFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Tenant context missing", nil, nil)
		return
	}
	customerID := chi.URLParam(r, "customerId")

	customer, err := h.service.GetCustomer(r.Context(), tenant.BusinessID, customerID)
	if err != nil {
		if errors.Is(err, ErrCustomerNotFound) {
			httpserver.RespondError(w, r, http.StatusNotFound, "not_found", "Customer not found", nil, nil)
			return
		}
		httpserver.RespondError(w, r, http.StatusInternalServerError, "database_error", "Failed to retrieve customer", nil, nil)
		return
	}

	httpserver.RespondJSON(w, r, http.StatusOK, customer)
}

func (h *Handler) UpdateCustomer(w http.ResponseWriter, r *http.Request) {
	tenant, ok := authctx.TenantFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Tenant context missing", nil, nil)
		return
	}
	user, _ := authctx.UserFromContext(r.Context())
	customerID := chi.URLParam(r, "customerId")

	var input UpdateCustomerInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpserver.RespondError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON payload", nil, nil)
		return
	}

	customer, err := h.service.UpdateCustomer(r.Context(), tenant.BusinessID, customerID, user.ID, input, extractMeta(r))
	if err != nil {
		if errors.Is(err, ErrCustomerNotFound) {
			httpserver.RespondError(w, r, http.StatusNotFound, "not_found", "Customer not found", nil, nil)
			return
		}
		if errors.Is(err, ErrInvalidCustomerName) {
			httpserver.RespondError(w, r, http.StatusUnprocessableEntity, "validation_failed", err.Error(), nil, map[string]string{
				"name": "name cannot be empty",
			})
			return
		}
		httpserver.RespondError(w, r, http.StatusInternalServerError, "database_error", "Failed to update customer", nil, nil)
		return
	}

	httpserver.RespondJSON(w, r, http.StatusOK, customer)
}

func (h *Handler) ArchiveCustomer(w http.ResponseWriter, r *http.Request) {
	tenant, ok := authctx.TenantFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Tenant context missing", nil, nil)
		return
	}
	user, _ := authctx.UserFromContext(r.Context())
	customerID := chi.URLParam(r, "customerId")

	if err := h.service.ArchiveCustomer(r.Context(), tenant.BusinessID, customerID, user.ID, extractMeta(r)); err != nil {
		if errors.Is(err, ErrCustomerNotFound) {
			httpserver.RespondError(w, r, http.StatusNotFound, "not_found", "Customer not found", nil, nil)
			return
		}
		httpserver.RespondError(w, r, http.StatusInternalServerError, "database_error", "Failed to archive customer", nil, nil)
		return
	}

	httpserver.RespondJSON(w, r, http.StatusOK, map[string]string{
		"message": "Customer archived successfully",
	})
}

func (h *Handler) GetCustomerLedger(w http.ResponseWriter, r *http.Request) {
	tenant, ok := authctx.TenantFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Tenant context missing", nil, nil)
		return
	}
	customerID := chi.URLParam(r, "customerId")

	ledger, err := h.service.GetCustomerLedger(r.Context(), tenant.BusinessID, customerID)
	if err != nil {
		if errors.Is(err, ErrCustomerNotFound) {
			httpserver.RespondError(w, r, http.StatusNotFound, "not_found", "Customer not found", nil, nil)
			return
		}
		httpserver.RespondError(w, r, http.StatusInternalServerError, "database_error", "Failed to retrieve customer ledger", nil, nil)
		return
	}

	httpserver.RespondJSON(w, r, http.StatusOK, ledger)
}
