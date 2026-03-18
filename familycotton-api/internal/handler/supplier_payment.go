package handler

import (
	"net/http"

	"github.com/familycotton/api/internal/middleware"
	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type SupplierPaymentHandler struct {
	service *service.SupplierPaymentService
}

func NewSupplierPaymentHandler(service *service.SupplierPaymentService) *SupplierPaymentHandler {
	return &SupplierPaymentHandler{service: service}
}

func (h *SupplierPaymentHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req model.CreateSupplierPaymentRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	sp, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, sp)
}
