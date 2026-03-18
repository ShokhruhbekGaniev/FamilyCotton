package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/familycotton/api/internal/middleware"
	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type InventoryCheckHandler struct {
	service *service.InventoryCheckService
}

func NewInventoryCheckHandler(service *service.InventoryCheckService) *InventoryCheckHandler {
	return &InventoryCheckHandler{service: service}
}

func (h *InventoryCheckHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req model.CreateInventoryCheckRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	ic, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, ic)
}

func (h *InventoryCheckHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid inventory check id"))
		return
	}
	var req model.UpdateInventoryCheckRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	ic, err := h.service.Update(r.Context(), id, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, ic)
}
