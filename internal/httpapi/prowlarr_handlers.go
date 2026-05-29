package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"feed-puller/internal/app"
	"feed-puller/internal/prowlarr"
	"feed-puller/internal/store"
)

func (s *Server) handleProwlarrSetting(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := s.service.GetProwlarrConfig(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, cfg)
	case http.MethodPut:
		var input struct {
			URL                string  `json:"url"`
			APIKey             string  `json:"api_key"`
			DownloadDir        string  `json:"download_dir"`
			TVDownloadDir      string  `json:"tv_download_dir"`
			MovieRenameEnabled bool    `json:"movie_rename_enabled"`
			TMDBAPIKey         string  `json:"tmdb_api_key"`
			IndexerIDs         []int64 `json:"indexer_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "请求体无效")
			return
		}
		cfg, err := s.service.SaveProwlarrConfig(r.Context(), store.ProwlarrConfig{
			URL:                strings.TrimSpace(input.URL),
			APIKey:             strings.TrimSpace(input.APIKey),
			DownloadDir:        strings.TrimSpace(input.DownloadDir),
			TVDownloadDir:      strings.TrimSpace(input.TVDownloadDir),
			MovieRenameEnabled: input.MovieRenameEnabled,
			TMDBAPIKey:         strings.TrimSpace(input.TMDBAPIKey),
			IndexerIDs:         input.IndexerIDs,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, cfg)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleProwlarrSettingTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var input struct {
		URL    *string `json:"url"`
		APIKey *string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "请求体无效")
		return
	}
	cfg := store.ProwlarrConfig{}
	if input.URL == nil && input.APIKey == nil {
		current, err := s.service.GetProwlarrConfig(r.Context())
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		cfg.URL = current.URL
		cfg.APIKey = current.APIKey
	} else {
		if input.URL != nil {
			cfg.URL = strings.TrimSpace(*input.URL)
		}
		if input.APIKey != nil {
			cfg.APIKey = strings.TrimSpace(*input.APIKey)
		}
	}
	if err := s.service.TestProwlarrConnection(r.Context(), cfg); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "Prowlarr 连通正常"})
}

func (s *Server) handleProwlarrIndexers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	indexers, err := s.service.ListProwlarrIndexers(r.Context())
	if err != nil {
		if errors.Is(err, app.ErrProwlarrNotConfigured) {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if indexers == nil {
		indexers = []prowlarr.Indexer{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": indexers})
}

func (s *Server) handleProwlarrSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		writeError(w, http.StatusBadRequest, "query 不能为空")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	queryValues := r.URL.Query()
	indexerValues, indexerIDsSpecified := queryValues["indexer_ids"]
	indexerIDs := parseIndexerIDs(indexerValues)

	releases, err := s.service.SearchProwlarr(r.Context(), app.ProwlarrSearchRequest{
		Query:               query,
		Type:                prowlarr.NormalizeSearchType(queryValues.Get("type")),
		Sort:                prowlarr.NormalizeSortBy(queryValues.Get("sort")),
		IndexerIDs:          indexerIDs,
		IndexerIDsSpecified: indexerIDsSpecified,
		Limit:               limit,
		Offset:              offset,
	})
	if err != nil {
		if errors.Is(err, app.ErrProwlarrNotConfigured) {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if releases == nil {
		releases = []prowlarr.Release{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": releases})
}

func (s *Server) handleProwlarrSearchHistory(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		items, err := s.service.ListProwlarrSearchHistory(r.Context(), limit)
		if err != nil {
			if errors.Is(err, app.ErrProwlarrNotConfigured) {
				writeError(w, http.StatusServiceUnavailable, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if items == nil {
			items = []store.ProwlarrSearchHistory{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodDelete:
		if err := s.service.ClearProwlarrSearchHistory(r.Context()); err != nil {
			if errors.Is(err, app.ErrProwlarrNotConfigured) {
				writeError(w, http.StatusServiceUnavailable, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleProwlarrSearchHistoryByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}
	id, tail, ok := parseIDTail(r.URL.Path, "/api/prowlarr/search-history/")
	if !ok || tail != "" || id <= 0 {
		writeError(w, http.StatusNotFound, "历史记录不存在")
		return
	}
	if err := s.service.DeleteProwlarrSearchHistory(r.Context(), id); err != nil {
		if errors.Is(err, app.ErrProwlarrNotConfigured) {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		if store.IsProwlarrSearchHistoryNotFound(err) {
			writeError(w, http.StatusNotFound, "历史记录不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleProwlarrSubmittedGuids(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var input struct {
		GUIDs []string `json:"guids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效")
		return
	}
	guids, err := s.service.ListProwlarrSubmittedGuids(r.Context(), input.GUIDs)
	if err != nil {
		if errors.Is(err, app.ErrProwlarrNotConfigured) {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if guids == nil {
		guids = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"guids": guids})
}

func (s *Server) handleProwlarrDownloadBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var input struct {
		Releases []app.ProwlarrReleaseInput `json:"releases"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效")
		return
	}
	if len(input.Releases) == 0 {
		writeError(w, http.StatusBadRequest, "请至少选择一条资源")
		return
	}
	items, failures := s.service.SubmitProwlarrReleases(r.Context(), input.Releases)
	payload := map[string]any{"items": items}
	if len(failures) > 0 {
		out := make([]map[string]any, len(failures))
		for i, f := range failures {
			out[i] = map[string]any{"guid": f.GUID, "error": f.Error}
		}
		payload["failures"] = out
	}
	writeJSON(w, http.StatusOK, payload)
}

func parseIndexerIDs(values []string) []int64 {
	if len(values) == 0 {
		return nil
	}
	ids := make([]int64, 0)
	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.ParseInt(part, 10, 64)
			if err == nil && id > 0 {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

func (s *Server) handleProwlarrDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var input app.ProwlarrReleaseInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效")
		return
	}
	item, err := s.service.SubmitProwlarrRelease(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrProwlarrNotConfigured):
			writeError(w, http.StatusServiceUnavailable, err.Error())
		case errors.Is(err, app.ErrProwlarrReleaseInProgress):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, app.ErrProwlarrReleaseCompleted):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, item)
}
