package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/familycotton/api/internal/middleware"
	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type PurchaseOrderHandler struct {
	service *service.PurchaseOrderService
}

func NewPurchaseOrderHandler(service *service.PurchaseOrderService) *PurchaseOrderHandler {
	return &PurchaseOrderHandler{service: service}
}

func (h *PurchaseOrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req model.CreatePurchaseOrderRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	po, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, po)
}

func (h *PurchaseOrderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid purchase order id"))
		return
	}
	po, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, po)
}

func (h *PurchaseOrderHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationParams(r)

	var supplierID *uuid.UUID
	if sid := r.URL.Query().Get("supplier_id"); sid != "" {
		if id, err := uuid.Parse(sid); err == nil {
			supplierID = &id
		}
	}
	status := r.URL.Query().Get("status")

	orders, total, err := h.service.List(r.Context(), supplierID, status, page, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	respondList(w, orders, page, limit, total)
}
