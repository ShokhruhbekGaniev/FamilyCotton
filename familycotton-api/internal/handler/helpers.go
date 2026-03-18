package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/familycotton/api/internal/model"
)

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func respondSuccess(w http.ResponseWriter, status int, data any) {
	respondJSON(w, status, model.SuccessResponse{Data: data})
}

func respondList(w http.ResponseWriter, data any, page, limit, total int) {
	respondJSON(w, http.StatusOK, model.SuccessResponse{
		Data: data,
		Meta: &model.Meta{Page: page, Limit: limit, Total: total},
	})
}

func respondError(w http.ResponseWriter, err error) {
	var appErr *model.AppError
	if errors.As(err, &appErr) {
		status := mapErrToStatus(appErr.Unwrap())
		code := mapErrToCode(appErr.Unwrap())
		respondJSON(w, status, model.ErrorResponse{
			Error: model.ErrorBody{Code: code, Message: appErr.Message},
		})
		return
	}
	slog.Error("unhandled error", "error", err)
	respondJSON(w, http.StatusInternalServerError, model.ErrorResponse{
		Error: model.ErrorBody{Code: "INTERNAL_ERROR", Message: "internal server error"},
	})
}

func mapErrToStatus(err error) int {
	switch {
	case errors.Is(err, model.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, model.ErrValidation):
		return http.StatusBadRequest
	case errors.Is(err, model.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, model.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, model.ErrConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func mapErrToCode(err error) string {
	switch {
	case errors.Is(err, model.ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, model.ErrValidation):
		return "VALIDATION_ERROR"
	case errors.Is(err, model.ErrForbidden):
		return "FORBIDDEN"
	case errors.Is(err, model.ErrUnauthorized):
		return "UNAUTHORIZED"
	case errors.Is(err, model.ErrConflict):
		return "CONFLICT"
	default:
		return "INTERNAL_ERROR"
	}
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return model.NewAppError(model.ErrValidation, "invalid JSON body")
	}
	return nil
}

func paginationParams(r *http.Request) (page, limit int) {
	page = 1
	limit = 20
	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	return
}
