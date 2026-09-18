package identity

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/businesses"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/authctx"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/httpserver"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/ratelimit"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	identityService *Service
	businessService *businesses.Service
	rateLimiter     *ratelimit.Limiter
	isProduction    bool
}

func NewHandler(
	identityService *Service,
	businessService *businesses.Service,
	rateLimiter *ratelimit.Limiter,
	isProduction bool,
) *Handler {
	return &Handler{
		identityService: identityService,
		businessService: businessService,
		rateLimiter:     rateLimiter,
		isProduction:    isProduction,
	}
}

func (h *Handler) Routes(jwtSecret []byte) chi.Router {
	r := chi.NewRouter()

	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.Refresh)
	r.Post("/verify-email", h.VerifyEmail)
	r.Post("/forgot-password", h.ForgotPassword)
	r.Post("/reset-password", h.ResetPassword)

	r.Group(func(protected chi.Router) {
		protected.Use(httpserver.Authenticate(jwtSecret))
		protected.Use(httpserver.RequireAuth)
		protected.Post("/logout", h.Logout)
	})

	return r
}

type registerPayload struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	FullName     string `json:"fullName"`
	Mobile       string `json:"mobile"`
	BusinessName string `json:"businessName"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	// Rate limiting: 10 attempts per minute per IP
	if h.rateLimiter != nil {
		ip := r.RemoteAddr
		allowed, _ := h.rateLimiter.Allow(r.Context(), "register:"+ip, 10, time.Minute)
		if !allowed {
			httpserver.RespondError(w, r, http.StatusTooManyRequests, "rate_limited", "Too many registration attempts. Please try again later.", nil, nil)
			return
		}
	}

	var req registerPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, r, http.StatusBadRequest, "invalid_json", "Malformed request body", nil, nil)
		return
	}

	meta := RequestMeta{
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		RequestID: httpserver.RequestIDFromContext(r.Context()),
	}

	tokens, err := h.identityService.Register(r.Context(), RegisterInput(req), meta)
	if err != nil {
		if errors.Is(err, ErrUserAlreadyExists) {
			httpserver.RespondError(w, r, http.StatusConflict, "user_already_exists", "An account with this email already exists", nil, nil)
			return
		}
		if errors.Is(err, ErrPasswordTooShort) {
			httpserver.RespondError(w, r, http.StatusUnprocessableEntity, "password_too_short", "Password must be at least 8 characters long", nil, map[string]string{"password": "minimum 8 characters"})
			return
		}
		httpserver.RespondError(w, r, http.StatusBadRequest, "registration_failed", err.Error(), nil, nil)
		return
	}

	h.setSessionCookies(w, tokens)
	httpserver.RespondJSON(w, r, http.StatusCreated, tokens)
}

type loginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	// Rate limiting: 10 login attempts per minute per IP
	if h.rateLimiter != nil {
		ip := r.RemoteAddr
		allowed, _ := h.rateLimiter.Allow(r.Context(), "login:"+ip, 10, time.Minute)
		if !allowed {
			httpserver.RespondError(w, r, http.StatusTooManyRequests, "rate_limited", "Too many login attempts. Please try again later.", nil, nil)
			return
		}
	}

	var req loginPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, r, http.StatusBadRequest, "invalid_json", "Malformed request body", nil, nil)
		return
	}

	meta := RequestMeta{
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		RequestID: httpserver.RequestIDFromContext(r.Context()),
	}

	tokens, err := h.identityService.Login(r.Context(), LoginInput(req), meta)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpserver.RespondError(w, r, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password", nil, nil)
			return
		}
		if errors.Is(err, ErrAccountSuspended) {
			httpserver.RespondError(w, r, http.StatusForbidden, "account_suspended", "Account has been suspended", nil, nil)
			return
		}
		httpserver.RespondError(w, r, http.StatusBadRequest, "login_failed", err.Error(), nil, nil)
		return
	}

	h.setSessionCookies(w, tokens)
	httpserver.RespondJSON(w, r, http.StatusOK, tokens)
}

type refreshPayload struct {
	RefreshToken string `json:"refreshToken"`
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken := ""
	var req refreshPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.RefreshToken != "" {
		refreshToken = req.RefreshToken
	} else if cookie, err := r.Cookie("aibos_refresh_token"); err == nil {
		refreshToken = cookie.Value
	}

	if refreshToken == "" {
		httpserver.RespondError(w, r, http.StatusUnauthorized, "missing_refresh_token", "Refresh token is missing", nil, nil)
		return
	}

	meta := RequestMeta{
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		RequestID: httpserver.RequestIDFromContext(r.Context()),
	}

	tokens, err := h.identityService.Refresh(r.Context(), refreshToken, meta)
	if err != nil {
		httpserver.RespondError(w, r, http.StatusUnauthorized, "invalid_refresh_token", "Refresh session is invalid or expired", nil, nil)
		return
	}

	h.setSessionCookies(w, tokens)
	httpserver.RespondJSON(w, r, http.StatusOK, tokens)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	refreshToken := ""
	if cookie, err := r.Cookie("aibos_refresh_token"); err == nil {
		refreshToken = cookie.Value
	}

	user, _ := authctx.UserFromContext(r.Context())
	meta := RequestMeta{
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		RequestID: httpserver.RequestIDFromContext(r.Context()),
	}

	if refreshToken != "" {
		_ = h.identityService.Logout(r.Context(), refreshToken, user.ID, meta)
	}

	h.clearSessionCookies(w)
	httpserver.RespondJSON(w, r, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

type verifyEmailPayload struct {
	Token string `json:"token"`
}

func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req verifyEmailPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		httpserver.RespondError(w, r, http.StatusBadRequest, "invalid_input", "Verification token is required", nil, nil)
		return
	}

	meta := RequestMeta{
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		RequestID: httpserver.RequestIDFromContext(r.Context()),
	}

	if err := h.identityService.VerifyEmail(r.Context(), req.Token, meta); err != nil {
		httpserver.RespondError(w, r, http.StatusBadRequest, "invalid_token", "Verification token is invalid or expired", nil, nil)
		return
	}

	httpserver.RespondJSON(w, r, http.StatusOK, map[string]string{"message": "Email verified successfully"})
}

type forgotPasswordPayload struct {
	Email string `json:"email"`
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		httpserver.RespondError(w, r, http.StatusBadRequest, "invalid_input", "Email is required", nil, nil)
		return
	}

	meta := RequestMeta{
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		RequestID: httpserver.RequestIDFromContext(r.Context()),
	}

	_, _ = h.identityService.ForgotPassword(r.Context(), req.Email, meta)
	// Always return success to prevent email enumeration
	httpserver.RespondJSON(w, r, http.StatusOK, map[string]string{
		"message": "If the email is registered, a password reset link has been sent",
	})
}

type resetPasswordPayload struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" || req.NewPassword == "" {
		httpserver.RespondError(w, r, http.StatusBadRequest, "invalid_input", "Token and newPassword are required", nil, nil)
		return
	}

	meta := RequestMeta{
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		RequestID: httpserver.RequestIDFromContext(r.Context()),
	}

	if err := h.identityService.ResetPassword(r.Context(), req.Token, req.NewPassword, meta); err != nil {
		if errors.Is(err, ErrPasswordTooShort) {
			httpserver.RespondError(w, r, http.StatusUnprocessableEntity, "password_too_short", "Password must be at least 8 characters", nil, nil)
			return
		}
		httpserver.RespondError(w, r, http.StatusBadRequest, "invalid_token", "Reset token is invalid or expired", nil, nil)
		return
	}

	httpserver.RespondJSON(w, r, http.StatusOK, map[string]string{"message": "Password has been reset successfully"})
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	user, ok := authctx.UserFromContext(r.Context())
	if !ok {
		httpserver.RespondError(w, r, http.StatusUnauthorized, "unauthorized", "Authentication required", nil, nil)
		return
	}

	fullUser, err := h.identityService.GetCurrentUser(r.Context(), user.ID)
	if err != nil {
		httpserver.RespondError(w, r, http.StatusNotFound, "user_not_found", "User record not found", nil, nil)
		return
	}

	businessesList, err := h.businessService.ListUserBusinesses(r.Context(), user.ID)
	if err != nil {
		businessesList = []businesses.UserBusinessSummary{}
	}

	httpserver.RespondJSON(w, r, http.StatusOK, map[string]any{
		"user":        fullUser,
		"memberships": businessesList,
	})
}

func (h *Handler) setSessionCookies(w http.ResponseWriter, tokens *AuthTokens) {
	http.SetCookie(w, &http.Cookie{
		Name:     "aibos_access_token",
		Value:    tokens.AccessToken,
		Path:     "/",
		Expires:  time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second),
		HttpOnly: true,
		Secure:   h.isProduction,
		SameSite: http.SameSiteLaxMode,
	})

	if tokens.RefreshToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "aibos_refresh_token",
			Value:    tokens.RefreshToken,
			Path:     "/api/v1/auth",
			Expires:  time.Now().Add(7 * 24 * time.Hour),
			HttpOnly: true,
			Secure:   h.isProduction,
			SameSite: http.SameSiteStrictMode,
		})
	}
}

func (h *Handler) clearSessionCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "aibos_access_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   h.isProduction,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "aibos_refresh_token",
		Value:    "",
		Path:     "/api/v1/auth",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   h.isProduction,
		SameSite: http.SameSiteStrictMode,
	})
}
