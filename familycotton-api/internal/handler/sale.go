package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/familycotton/api/internal/middleware"
	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type SaleHandler struct {
	service *service.SaleService
}

func NewSaleHandler(service *service.SaleService) *SaleHandler {
	return &SaleHandler{service: service}
}

func (h *SaleHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req model.CreateSaleRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	sale, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, sale)
}

func (h *SaleHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "Некорректный ID продажи"))
		return
	}
	sale, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, sale)
}

func (h *SaleHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationParams(r)
	var shiftID, clientID *uuid.UUID
	if sid := r.URL.Query().Get("shift_id"); sid != "" {
		if id, err := uuid.Parse(sid); err == nil {
			shiftID = &id
		}
	}
	if cid := r.URL.Query().Get("client_id"); cid != "" {
		if id, err := uuid.Parse(cid); err == nil {
			clientID = &id
		}
	}
	sales, total, err := h.service.List(r.Context(), shiftID, clientID, page, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	respondList(w, sales, page, limit, total)
}

func (h *SaleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "Некорректный ID продажи"))
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
