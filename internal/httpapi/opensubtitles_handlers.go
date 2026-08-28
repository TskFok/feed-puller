package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"feed-puller/internal/app"
	"feed-puller/internal/opensubtitles"
	"feed-puller/internal/store"
)

func (s *Server) handleOpenSubtitlesSetting(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := s.service.GetOpenSubtitlesConfig(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, cfg)
	case http.MethodPut:
		var input store.OpenSubtitlesConfig
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "请求体无效")
			return
		}
		cfg, err := s.service.SaveOpenSubtitlesConfig(r.Context(), input)
		if err != nil {
			writeOpenSubtitlesError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, cfg)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleSubtitlesSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		writeError(w, http.StatusBadRequest, "query 不能为空")
		return
	}
	language := strings.TrimSpace(r.URL.Query().Get("languages"))
	if language == "" {
		writeError(w, http.StatusBadRequest, "languages 不能为空")
		return
	}
	result, err := s.service.SearchSubtitles(r.Context(), query, language, 1)
	if err != nil {
		writeOpenSubtitlesError(w, err)
		return
	}
	items := result.Items
	if items == nil {
		items = []opensubtitles.SubtitleFile{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleSubtitlesDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var input struct {
		FileID   int64  `json:"file_id"`
		FileName string `json:"file_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效")
		return
	}
	if input.FileID <= 0 {
		writeError(w, http.StatusBadRequest, "file_id 无效")
		return
	}
	path, fileName, err := s.service.DownloadSubtitle(r.Context(), input.FileID, input.FileName)
	if err != nil {
		writeOpenSubtitlesError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": path, "file_name": fileName})
}

func writeOpenSubtitlesError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, app.ErrOpenSubtitlesNotConfigured):
		writeError(w, http.StatusServiceUnavailable, err.Error())
	case errors.Is(err, store.ErrOpenSubtitlesConfigIncomplete):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, opensubtitles.ErrLoginFailed):
		writeError(w, http.StatusBadGateway, err.Error())
	case errors.Is(err, opensubtitles.ErrInvalidFileName),
		errors.Is(err, opensubtitles.ErrInvalidFileID),
		errors.Is(err, opensubtitles.ErrEmptyQuery):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, app.ErrSubtitleWriteFailed):
		writeError(w, http.StatusInternalServerError, err.Error())
	default:
		writeError(w, http.StatusBadGateway, err.Error())
	}
}
