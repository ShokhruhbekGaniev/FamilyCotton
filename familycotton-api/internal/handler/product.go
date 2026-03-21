package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type ProductHandler struct {
	service *service.ProductService
}

func NewProductHandler(service *service.ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationParams(r)
	filter := model.ProductFilter{
		Search: r.URL.Query().Get("search"),
		Brand:  r.URL.Query().Get("brand"),
	}
	if sid := r.URL.Query().Get("supplier_id"); sid != "" {
		if id, err := uuid.Parse(sid); err == nil {
			filter.SupplierID = &id
		}
	}
	products, total, err := h.service.List(r.Context(), filter, page, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	respondList(w, products, page, limit, total)
}

func (h *ProductHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "Некорректный ID товара"))
		return
	}
	product, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, product)
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateProductRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	product, err := h.service.Create(r.Context(), &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, product)
}

func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "Некорректный ID товара"))
		return
	}
	var req model.UpdateProductRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	product, err := h.service.Update(r.Context(), id, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, product)
}

func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "Некорректный ID товара"))
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, map[string]string{"message": "product deleted"})
}
