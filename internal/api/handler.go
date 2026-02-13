package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/tsv-processor/internal/db"
)

type Handler struct {
	db *db.MongoDB
}

func NewHandler(db *db.MongoDB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/devices/{unit_guid}", h.getDeviceDataByGUID).Methods("GET")
}

func (h *Handler) getDeviceDataByGUID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	unitGUID := vars["unit_guid"]

	page, limit := getPaginationParams(r)

	data, err := h.db.GetDeviceDataByUnitGUID(r.Context(), unitGUID, page, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(data)
}

func getPaginationParams(r *http.Request) (page, limit int64) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page = 1
	if pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 64); err == nil && p > 0 {
			page = p
		}
	}

	limit = 10
	if limitStr != "" {
		if l, err := strconv.ParseInt(limitStr, 10, 64); err == nil && l > 0 {
			limit = l
		}
	}

	if limit > 100 {
		limit = 100
	}

	return page, limit
}
