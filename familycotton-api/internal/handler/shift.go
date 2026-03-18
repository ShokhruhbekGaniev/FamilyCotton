package handler

import (
	"net/http"

	"github.com/familycotton/api/internal/middleware"
	"github.com/familycotton/api/internal/service"
)

type ShiftHandler struct {
	service *service.ShiftService
}

func NewShiftHandler(service *service.ShiftService) *ShiftHandler {
	return &ShiftHandler{service: service}
}

func (h *ShiftHandler) Open(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	shift, err := h.service.Open(r.Context(), userID)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, shift)
}

func (h *ShiftHandler) Close(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	shift, err := h.service.Close(r.Context(), userID)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, shift)
}

func (h *ShiftHandler) Current(w http.ResponseWriter, r *http.Request) {
	shift, err := h.service.GetCurrent(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, shift)
}

func (h *ShiftHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationParams(r)
	shifts, total, err := h.service.List(r.Context(), page, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	respondList(w, shifts, page, limit, total)
}
