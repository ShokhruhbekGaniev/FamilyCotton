package handler

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/familycotton/api/internal/middleware"
	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type SaleReturnHandler struct {
	service *service.SaleReturnService
}

func NewSaleReturnHandler(service *service.SaleReturnService) *SaleReturnHandler {
	return &SaleReturnHandler{service: service}
}

func (h *SaleReturnHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req model.CreateSaleReturnRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	ret, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, ret)
}

func (h *SaleReturnHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationParams(r)
	var saleID *uuid.UUID
	if sid := r.URL.Query().Get("sale_id"); sid != "" {
		if id, err := uuid.Parse(sid); err == nil {
			saleID = &id
		}
	}
	returns, total, err := h.service.List(r.Context(), saleID, page, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	respondList(w, returns, page, limit, total)
}
