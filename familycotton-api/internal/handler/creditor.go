package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type CreditorHandler struct {
	service *service.CreditorService
}

func NewCreditorHandler(service *service.CreditorService) *CreditorHandler {
	return &CreditorHandler{service: service}
}

func (h *CreditorHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationParams(r)
	creditors, total, err := h.service.List(r.Context(), page, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	respondList(w, creditors, page, limit, total)
}

func (h *CreditorHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "Некорректный ID кредитора"))
		return
	}
	creditor, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, creditor)
}

func (h *CreditorHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateCreditorRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	creditor, err := h.service.Create(r.Context(), &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, creditor)
}

func (h *CreditorHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "Некорректный ID кредитора"))
		return
	}
	var req model.UpdateCreditorRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	creditor, err := h.service.Update(r.Context(), id, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, creditor)
}

func (h *CreditorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "Некорректный ID кредитора"))
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, map[string]string{"message": "creditor deleted"})
}
