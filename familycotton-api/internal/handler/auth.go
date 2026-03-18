package handler

import (
	"net/http"

	"github.com/familycotton/api/internal/middleware"
	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
	userService *service.UserService
}

func NewAuthHandler(authService *service.AuthService, userService *service.UserService) *AuthHandler {
	return &AuthHandler{authService: authService, userService: userService}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		respondError(w, err)
		return
	}

	tokens, err := h.authService.Login(r.Context(), &req)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, tokens)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		respondError(w, err)
		return
	}

	tokens, err := h.authService.Refresh(r.Context(), &req)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, tokens)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	if err := h.authService.Logout(r.Context(), req.RefreshToken); err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	user, err := h.userService.GetByID(r.Context(), userID)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, user)
}
