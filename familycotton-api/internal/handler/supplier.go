package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type SupplierHandler struct {
	service *service.SupplierService
}

func NewSupplierHandler(service *service.SupplierService) *SupplierHandler {
	return &SupplierHandler{service: service}
}

func (h *SupplierHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationParams(r)
	suppliers, total, err := h.service.List(r.Context(), page, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	respondList(w, suppliers, page, limit, total)
}

func (h *SupplierHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid supplier id"))
		return
	}
	supplier, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, supplier)
}

func (h *SupplierHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateSupplierRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	supplier, err := h.service.Create(r.Context(), &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, supplier)
}

func (h *SupplierHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid supplier id"))
		return
	}
	var req model.UpdateSupplierRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	supplier, err := h.service.Update(r.Context(), id, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, supplier)
}

func (h *SupplierHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid supplier id"))
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, map[string]string{"message": "supplier deleted"})
}
