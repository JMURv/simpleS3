package utils

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type Response struct {
	URL any `json:"url"`
}

type PaginatedResponse struct {
	Data        any  `json:"data"`
	Count       int  `json:"count"`
	TotalPages  int  `json:"total_pages"`
	CurrentPage int  `json:"current_page"`
	HasNextPage bool `json:"has_next_page"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func SuccessPaginatedResponse(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func SuccessResponse(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(
		&Response{
			URL: data,
		},
	)
}

func ErrResponse(w http.ResponseWriter, statusCode int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(
		&ErrorResponse{
			Error: err.Error(),
		},
	)
}

func ParsePaginationParams(r *http.Request, page, size int) (int, int) {
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}

	if s, err := strconv.Atoi(r.URL.Query().Get("size")); err == nil && s > 0 {
		size = s
	}

	return page, size
}
