package businesses

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/authctx"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/httpserver"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	businessService *Service
}

func NewHandler(businessService *Service) *Handler {
	return &Handler{businessService: businessService}
}

func (h *Handler) Routes(jwtSecret []byte) chi.Router {
	r := chi.NewRouter()

	r.Use(httpserver.Authenticate(jwtSecret))
	r.Use(httpserver.RequireAuth)

	r.Get("/", h.ListBusinesses)
	r.Post("/", h.CreateBusiness)

	r.Route("/{businessId}", func(b chi.Router) {
		b.Use(h.ResolveTenant)

		b.Get("/", h.GetBusiness)
		b.With(httpserver.RequireTenantRole("owner")).Patch("/", h.UpdateBusiness)

		b.Get("/members", h.ListMembers)
		b.With(httpserver.RequireTenantRole("owner")).Post("/invitations", h.InviteMember)
		b.With(httpserver.RequireTenantRole("owner")).Patch("/members/{memberId}", h.UpdateMember)
		b.With(httpserver.RequireTenantRole("owner")).Delete("/members/{memberId}", h.RemoveMember)
	})

	return r
}

func (h *Handler) ResolveTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		businessID := chi.URLParam(r, "businessId")
		if businessID == "" {
			next.ServeHTTP(w, r)
			return
		}

		user, ok := authctx.UserFromContext(r.Context())
		if !ok {
			httpserver.RespondError(w, r, http.StatusUnauthorized, "unauthorized", "Authentication required", nil, nil)
			return
		}

		_, role, err := h.businessService.GetBusiness(r.Context(), businessID, user.ID)
		if err != nil {
			httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "You do not have access to this business", nil, nil)
			return
		}

		ctx := authctx.WithTenant(r.Context(), authctx.TenantContext{
			BusinessID: businessID,
			Role:       role,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) ListBusinesses(w http.ResponseWriter, r *http.Request) {
	user, ok := authctx.UserFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusUnauthorized, "unauthorized", "Authentication required", nil, nil)
		return
	}

	list, err := h.businessService.ListUserBusinesses(r.Context(), user.ID)
	if err != nil {
		httpserver.RespondError(w, r, http.StatusInternalServerError, "database_error", "Failed to retrieve businesses", nil, nil)
		return
	}

	if list == nil {
		list = []UserBusinessSummary{}
	}
	httpserver.RespondJSON(w, r, http.StatusOK, list)
}

type createBusinessPayload struct {
	Name          string  `json:"name"`
	LegalName     *string `json:"legalName"`
	Phone         *string `json:"phone"`
	Address       *string `json:"address"`
	Timezone      string  `json:"timezone"`
	Currency      string  `json:"currency"`
	InvoicePrefix string  `json:"invoicePrefix"`
}

func (h *Handler) CreateBusiness(w http.ResponseWriter, r *http.Request) {
	user, ok := authctx.UserFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusUnauthorized, "unauthorized", "Authentication required", nil, nil)
		return
	}

	var req createBusinessPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, r, http.StatusBadRequest, "invalid_json", "Malformed request body", nil, nil)
		return
	}

	if req.Name == "" {
		httpserver.RespondError(w, r, http.StatusUnprocessableEntity, "validation_failed", "Business name is required", nil, map[string]string{"name": "is required"})
		return
	}

	meta := RequestMeta{
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		RequestID: httpserver.RequestIDFromContext(r.Context()),
	}

	biz, err := h.businessService.CreateBusiness(r.Context(), user.ID, CreateBusinessInput(req), meta)
	if err != nil {
		httpserver.RespondError(w, r, http.StatusBadRequest, "creation_failed", err.Error(), nil, nil)
		return
	}

	httpserver.RespondJSON(w, r, http.StatusCreated, biz)
}

func (h *Handler) GetBusiness(w http.ResponseWriter, r *http.Request) {
	tenant, ok := authctx.TenantFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Tenant context missing", nil, nil)
		return
	}
	user, _ := authctx.UserFromContext(r.Context())

	biz, _, err := h.businessService.GetBusiness(r.Context(), tenant.BusinessID, user.ID)
	if err != nil {
		httpserver.RespondError(w, r, http.StatusNotFound, "business_not_found", "Business not found or inaccessible", nil, nil)
		return
	}

	httpserver.RespondJSON(w, r, http.StatusOK, biz)
}

type updateBusinessPayload struct {
	Name          *string `json:"name"`
	LegalName     *string `json:"legalName"`
	Phone         *string `json:"phone"`
	Address       *string `json:"address"`
	Timezone      *string `json:"timezone"`
	InvoicePrefix *string `json:"invoicePrefix"`
}

func (h *Handler) UpdateBusiness(w http.ResponseWriter, r *http.Request) {
	tenant, ok := authctx.TenantFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Tenant context missing", nil, nil)
		return
	}
	user, _ := authctx.UserFromContext(r.Context())

	var req updateBusinessPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, r, http.StatusBadRequest, "invalid_json", "Malformed request body", nil, nil)
		return
	}

	meta := RequestMeta{
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		RequestID: httpserver.RequestIDFromContext(r.Context()),
	}

	biz, err := h.businessService.UpdateBusiness(r.Context(), tenant.BusinessID, user.ID, UpdateBusinessInput(req), meta)
	if err != nil {
		if errors.Is(err, ErrUnauthorizedAction) {
			httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Only owners can modify business settings", nil, nil)
			return
		}
		httpserver.RespondError(w, r, http.StatusBadRequest, "update_failed", err.Error(), nil, nil)
		return
	}

	httpserver.RespondJSON(w, r, http.StatusOK, biz)
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	tenant, ok := authctx.TenantFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Tenant context missing", nil, nil)
		return
	}
	user, _ := authctx.UserFromContext(r.Context())

	members, err := h.businessService.ListMembers(r.Context(), tenant.BusinessID, user.ID)
	if err != nil {
		httpserver.RespondError(w, r, http.StatusBadRequest, "list_members_failed", err.Error(), nil, nil)
		return
	}

	if members == nil {
		members = []MemberDetail{}
	}
	httpserver.RespondJSON(w, r, http.StatusOK, members)
}

type inviteMemberPayload struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (h *Handler) InviteMember(w http.ResponseWriter, r *http.Request) {
	tenant, ok := authctx.TenantFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Tenant context missing", nil, nil)
		return
	}
	user, _ := authctx.UserFromContext(r.Context())

	var req inviteMemberPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, r, http.StatusBadRequest, "invalid_json", "Malformed request body", nil, nil)
		return
	}

	meta := RequestMeta{
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		RequestID: httpserver.RequestIDFromContext(r.Context()),
	}

	member, err := h.businessService.InviteMember(r.Context(), tenant.BusinessID, user.ID, req.Email, req.Role, meta)
	if err != nil {
		if errors.Is(err, ErrMemberAlreadyExists) {
			httpserver.RespondError(w, r, http.StatusConflict, "member_already_exists", "This user is already a member or invited", nil, nil)
			return
		}
		httpserver.RespondError(w, r, http.StatusBadRequest, "invitation_failed", err.Error(), nil, nil)
		return
	}

	httpserver.RespondJSON(w, r, http.StatusCreated, member)
}

type updateMemberPayload struct {
	Role   *string `json:"role"`
	Status *string `json:"status"`
}

func (h *Handler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	tenant, ok := authctx.TenantFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Tenant context missing", nil, nil)
		return
	}
	user, _ := authctx.UserFromContext(r.Context())
	memberID := chi.URLParam(r, "memberId")

	var req updateMemberPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, r, http.StatusBadRequest, "invalid_json", "Malformed request body", nil, nil)
		return
	}

	meta := RequestMeta{
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		RequestID: httpserver.RequestIDFromContext(r.Context()),
	}

	member, err := h.businessService.UpdateMember(r.Context(), tenant.BusinessID, user.ID, memberID, req.Role, req.Status, meta)
	if err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			httpserver.RespondError(w, r, http.StatusNotFound, "member_not_found", "Member not found in this business", nil, nil)
			return
		}
		httpserver.RespondError(w, r, http.StatusBadRequest, "update_member_failed", err.Error(), nil, nil)
		return
	}

	httpserver.RespondJSON(w, r, http.StatusOK, member)
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	tenant, ok := authctx.TenantFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusForbidden, "forbidden", "Tenant context missing", nil, nil)
		return
	}
	user, _ := authctx.UserFromContext(r.Context())
	memberID := chi.URLParam(r, "memberId")

	meta := RequestMeta{
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		RequestID: httpserver.RequestIDFromContext(r.Context()),
	}

	if err := h.businessService.RemoveMember(r.Context(), tenant.BusinessID, user.ID, memberID, meta); err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			httpserver.RespondError(w, r, http.StatusNotFound, "member_not_found", "Member not found in this business", nil, nil)
			return
		}
		if errors.Is(err, ErrCannotRemoveOwner) {
			httpserver.RespondError(w, r, http.StatusBadRequest, "cannot_remove_owner", "The business owner cannot be removed", nil, nil)
			return
		}
		httpserver.RespondError(w, r, http.StatusBadRequest, "remove_member_failed", err.Error(), nil, nil)
		return
	}

	httpserver.RespondJSON(w, r, http.StatusOK, map[string]string{"message": "Member removed successfully"})
}
