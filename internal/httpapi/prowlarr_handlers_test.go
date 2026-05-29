package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"feed-puller/internal/app"
	"feed-puller/internal/config"
	"feed-puller/internal/downloader"
	"feed-puller/internal/store"
)

var prowlarrSettingKeys = []string{
	"prowlarr_url", "prowlarr_api_key", "prowlarr_download_dir", "prowlarr_tv_download_dir",
	"prowlarr_movie_rename_enabled", "prowlarr_tmdb_api_key", "prowlarr_indexer_ids",
	"prowlarr_subscription_id", "prowlarr_tv_subscription_id",
}

func newProwlarrServer(t *testing.T) (*Server, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	repo := store.New(db)
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	svc := app.NewService(repo, downloader.NewAria2Client("", ""), log)
	srv := New(config.Config{}, repo, svc, log)
	return srv, mock, func() { _ = db.Close() }
}

func authRequest(r *http.Request) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, store.User{ID: 1, Email: "admin@test.dev"})
	return r.WithContext(ctx)
}

func expectEmptyProwlarrSettings(mock sqlmock.Sqlmock) {
	for _, key := range prowlarrSettingKeys {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT value FROM settings WHERE name = ?`)).
			WithArgs(key).WillReturnRows(sqlmock.NewRows([]string{"value"}))
	}
}

func expectProwlarrSettings(mock sqlmock.Sqlmock, values map[string]string) {
	for _, key := range prowlarrSettingKeys {
		value := values[key]
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT value FROM settings WHERE name = ?`)).
			WithArgs(key).WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(value))
	}
}

func TestProwlarrSearch_RequiresAuth(t *testing.T) {
	srv, _, cleanup := newProwlarrServer(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodGet, "/api/prowlarr/search?query=test", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", rec.Code)
	}
}

func TestProwlarrSearch_NotConfigured(t *testing.T) {
	srv, mock, cleanup := newProwlarrServer(t)
	defer cleanup()
	expectEmptyProwlarrSettings(mock)
	req := authRequest(httptest.NewRequest(http.MethodGet, "/api/prowlarr/search?query=inception", nil))
	rec := httptest.NewRecorder()
	srv.handleProwlarrSearch(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("code = %d, want 503", rec.Code)
	}
}

func TestProwlarrSearch_EmptyIndexerParamSearchesAllTorrentIndexers(t *testing.T) {
	prowlarrSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query()["indexerIds"]; len(got) != 1 || got[0] != "-2" {
			t.Fatalf("indexerIds = %+v, want [-2]", got)
		}
		_, _ = w.Write([]byte(`[{"guid":"g1","title":"Inception","protocol":"torrent","infoHash":"abc"}]`))
	}))
	defer prowlarrSrv.Close()

	srv, mock, cleanup := newProwlarrServer(t)
	defer cleanup()
	expectProwlarrSettings(mock, map[string]string{
		"prowlarr_url":                prowlarrSrv.URL,
		"prowlarr_api_key":            "secret",
		"prowlarr_download_dir":       "/movies",
		"prowlarr_indexer_ids":        "[7,9]",
		"prowlarr_subscription_id":    "9",
		"prowlarr_tv_subscription_id": "10",
	})
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO prowlarr_search_history`)).
		WithArgs("inception", "inception", "movie", "seeders", "[]", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM prowlarr_search_history`)).
		WithArgs(50).
		WillReturnResult(sqlmock.NewResult(0, 0))

	req := authRequest(httptest.NewRequest(http.MethodGet, "/api/prowlarr/search?query=inception&indexer_ids=", nil))
	rec := httptest.NewRecorder()
	srv.handleProwlarrSearch(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProwlarrDownload_InProgress(t *testing.T) {
	srv, mock, cleanup := newProwlarrServer(t)
	defer cleanup()
	now := time.Now().UTC()
	for _, key := range prowlarrSettingKeys {
		value := ""
		switch key {
		case "prowlarr_url":
			value = "http://127.0.0.1:9696"
		case "prowlarr_api_key":
			value = "secret"
		case "prowlarr_download_dir":
			value = "/movies"
		case "prowlarr_indexer_ids":
			value = "[]"
		case "prowlarr_subscription_id":
			value = "9"
		case "prowlarr_tv_subscription_id":
			value = "10"
		}
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT value FROM settings WHERE name = ?`)).
			WithArgs(key).WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(value))
	}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM feed_items WHERE subscription_id = ? AND dedupe_key = ?`)).
		WithArgs(int64(9), "prowlarr:g1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE feed_items`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM feed_items WHERE id = ?`)).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "subscription_id", "guid", "title", "link", "download_url", "dedupe_key", "published_at", "download_status", "created_at", "updated_at",
		}).AddRow(int64(42), int64(9), "g1", "Movie", `{}`, "magnet:?xt=urn:btih:abc", "prowlarr:g1", nil, "submitted", now, now))

	req := authRequest(httptest.NewRequest(http.MethodPost, "/api/prowlarr/download", strings.NewReader(`{"guid":"g1","title":"Movie","info_hash":"abc"}`)))
	rec := httptest.NewRecorder()
	srv.handleProwlarrDownload(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("code = %d, want 409, body=%s", rec.Code, rec.Body.String())
	}
}

func TestProwlarrSettingTest_OK(t *testing.T) {
	prowlarrSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"version":"1.0.0"}`))
	}))
	defer prowlarrSrv.Close()

	srv, _, cleanup := newProwlarrServer(t)
	defer cleanup()
	body := strings.NewReader(`{"url":"` + prowlarrSrv.URL + `","api_key":"secret"}`)
	req := authRequest(httptest.NewRequest(http.MethodPost, "/api/settings/prowlarr/test", body))
	rec := httptest.NewRecorder()
	srv.handleProwlarrSettingTest(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["ok"] != true {
		t.Fatalf("expected ok=true, got %+v", payload)
	}
}

func TestProwlarrSettingTest_ExplicitEmptyAPIKeyDoesNotUseSavedKey(t *testing.T) {
	srv, mock, cleanup := newProwlarrServer(t)
	defer cleanup()
	body := strings.NewReader(`{"url":"http://127.0.0.1:9696","api_key":""}`)
	req := authRequest(httptest.NewRequest(http.MethodPost, "/api/settings/prowlarr/test", body))
	rec := httptest.NewRecorder()
	srv.handleProwlarrSettingTest(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["ok"] != false {
		t.Fatalf("expected ok=false, got %+v", payload)
	}
	if errorMessage, _ := payload["error"].(string); !strings.Contains(errorMessage, "Prowlarr API Key 不能为空") {
		t.Fatalf("expected API Key error, got %+v", payload)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestParseIndexerIDs(t *testing.T) {
	t.Parallel()
	ids := parseIndexerIDs([]string{"1,2", "5"})
	if len(ids) != 3 || ids[0] != 1 || ids[2] != 5 {
		t.Fatalf("unexpected ids: %+v", ids)
	}
}

func TestProwlarrSubmittedGuids_ReturnsMatches(t *testing.T) {
	srv, mock, cleanup := newProwlarrServer(t)
	defer cleanup()
	expectProwlarrSettings(mock, map[string]string{
		"prowlarr_url":      "http://127.0.0.1:9696",
		"prowlarr_api_key":  "secret",
		"prowlarr_download_dir": "/movies",
	})
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COALESCE(guid, ''), dedupe_key
		FROM feed_items
		WHERE dedupe_key IN (?)
		  AND download_status IN (?, ?, ?)
	`)).WithArgs("prowlarr:g1", "submitting", "submitted", "completed").
		WillReturnRows(sqlmock.NewRows([]string{"guid", "dedupe_key"}).AddRow("g1", "prowlarr:g1"))

	req := authRequest(httptest.NewRequest(http.MethodPost, "/api/prowlarr/submitted-guids", strings.NewReader(`{"guids":["g1"]}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleProwlarrSubmittedGuids(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		GUIDs []string `json:"guids"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.GUIDs) != 1 || payload.GUIDs[0] != "g1" {
		t.Fatalf("guids = %#v", payload.GUIDs)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
