package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.List(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, users)
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	user, err := h.service.Create(r.Context(), &req)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusCreated, user)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid user id"))
		return
	}

	var req model.UpdateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	user, err := h.service.Update(r.Context(), id, &req)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, user)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid user id"))
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, map[string]string{"message": "user deleted"})
}
