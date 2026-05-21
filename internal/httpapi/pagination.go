package httpapi

import (
	"net/http"
	"strconv"
)

const (
	defaultPageSize = 30
	maxPageSize     = 100
)

type pageParams struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type paginatedResponse[T any] struct {
	Items    []T `json:"items"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

func parsePageParams(r *http.Request) pageParams {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return pageParams{Page: page, PageSize: pageSize}
}

func writePaginatedJSON[T any](w http.ResponseWriter, status int, items []T, total, page, pageSize int) {
	if items == nil {
		items = []T{}
	}
	writeJSON(w, status, paginatedResponse[T]{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}
