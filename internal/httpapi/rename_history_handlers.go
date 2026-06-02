package httpapi

import (
	"net/http"

	"feed-puller/internal/store"
)

func (s *Server) handleRenameHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	params := parsePageParams(r)
	rows, total, err := s.store.ListRenameHistoryPage(r.Context(), params.Page, params.PageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rows == nil {
		rows = []store.RenameHistory{}
	}
	writePaginatedJSON(w, http.StatusOK, rows, total, params.Page, params.PageSize)
}
