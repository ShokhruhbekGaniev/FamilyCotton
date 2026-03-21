package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type BrandHandler struct {
	service *service.BrandService
}

func NewBrandHandler(service *service.BrandService) *BrandHandler {
	return &BrandHandler{service: service}
}

func (h *BrandHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationParams(r)
	brands, total, err := h.service.List(r.Context(), page, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	respondList(w, brands, page, limit, total)
}

func (h *BrandHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "Некорректный ID бренда"))
		return
	}
	brand, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, brand)
}

func (h *BrandHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateBrandRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	brand, err := h.service.Create(r.Context(), &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, brand)
}

func (h *BrandHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "Некорректный ID бренда"))
		return
	}
	var req model.UpdateBrandRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	brand, err := h.service.Update(r.Context(), id, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, brand)
}

func (h *BrandHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "Некорректный ID бренда"))
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, map[string]string{"message": "brand deleted"})
}
