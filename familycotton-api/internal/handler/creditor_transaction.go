package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/familycotton/api/internal/middleware"
	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type CreditorTransactionHandler struct {
	service *service.CreditorTransactionService
}

func NewCreditorTransactionHandler(service *service.CreditorTransactionService) *CreditorTransactionHandler {
	return &CreditorTransactionHandler{service: service}
}

func (h *CreditorTransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req model.CreateCreditorTransactionRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	ct, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, ct)
}

func (h *CreditorTransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	creditorID, err := uuid.Parse(chi.URLParam(r, "creditorId"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid creditor id"))
		return
	}
	page, limit := paginationParams(r)
	txns, total, err := h.service.ListByCreditor(r.Context(), creditorID, page, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	respondList(w, txns, page, limit, total)
}
