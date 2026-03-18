package handler

import (
	"net/http"

	"github.com/familycotton/api/internal/service"
)

type SafeHandler struct {
	svc *service.SafeService
}

func NewSafeHandler(svc *service.SafeService) *SafeHandler {
	return &SafeHandler{svc: svc}
}

func (h *SafeHandler) Balance(w http.ResponseWriter, r *http.Request) {
	balance, err := h.svc.GetBalance(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, balance)
}

func (h *SafeHandler) Transactions(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationParams(r)
	txns, total, err := h.svc.ListTransactions(r.Context(), page, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	respondList(w, txns, page, limit, total)
}

func (h *SafeHandler) OwnerDebts(w http.ResponseWriter, r *http.Request) {
	debts, err := h.svc.ListOwnerDebts(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, debts)
}

func (h *SafeHandler) OwnerDeposit(w http.ResponseWriter, r *http.Request) {
	var req service.OwnerDepositRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	if err := h.svc.OwnerDeposit(r.Context(), &req); err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, map[string]string{"message": "deposit recorded"})
}
