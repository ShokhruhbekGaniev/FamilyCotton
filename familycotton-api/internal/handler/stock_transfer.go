package handler

import (
	"net/http"

	"github.com/familycotton/api/internal/middleware"
	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type StockTransferHandler struct {
	service *service.StockTransferService
}

func NewStockTransferHandler(service *service.StockTransferService) *StockTransferHandler {
	return &StockTransferHandler{service: service}
}

func (h *StockTransferHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req model.CreateStockTransferRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	st, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, st)
}
