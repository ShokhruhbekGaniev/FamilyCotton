package handler

import (
	"net/http"

	"github.com/familycotton/api/internal/middleware"
	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type ClientPaymentHandler struct {
	service *service.ClientPaymentService
}

func NewClientPaymentHandler(service *service.ClientPaymentService) *ClientPaymentHandler {
	return &ClientPaymentHandler{service: service}
}

func (h *ClientPaymentHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req model.CreateClientPaymentRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	payment, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, payment)
}
