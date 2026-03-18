package handler

import (
	"net/http"
	"time"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type DashboardHandler struct {
	svc *service.DashboardService
}

func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

func parseDateRange(r *http.Request) (time.Time, time.Time, error) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		return time.Time{}, time.Time{}, model.NewAppError(model.ErrValidation, "from and to query params are required")
	}
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return time.Time{}, time.Time{}, model.NewAppError(model.ErrValidation, "invalid from date format, use YYYY-MM-DD")
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return time.Time{}, time.Time{}, model.NewAppError(model.ErrValidation, "invalid to date format, use YYYY-MM-DD")
	}
	// Set to end of day.
	to = to.Add(24*time.Hour - time.Nanosecond)
	return from, to, nil
}

func (h *DashboardHandler) Revenue(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseDateRange(r)
	if err != nil {
		respondError(w, err)
		return
	}
	rpt, err := h.svc.Revenue(r.Context(), from, to)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, rpt)
}

func (h *DashboardHandler) Profit(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseDateRange(r)
	if err != nil {
		respondError(w, err)
		return
	}
	rpt, err := h.svc.Profit(r.Context(), from, to)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, rpt)
}

func (h *DashboardHandler) StockValue(w http.ResponseWriter, r *http.Request) {
	rpt, err := h.svc.StockValue(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, rpt)
}

func (h *DashboardHandler) SalesBySupplier(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseDateRange(r)
	if err != nil {
		respondError(w, err)
		return
	}
	rpt, err := h.svc.SalesBySupplier(r.Context(), from, to)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, rpt)
}

func (h *DashboardHandler) PaidVsDebt(w http.ResponseWriter, r *http.Request) {
	rpt, err := h.svc.PaidVsDebt(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, rpt)
}
