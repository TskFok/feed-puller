package httpapi

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"feed-puller/internal/app"
	"feed-puller/internal/config"
	"feed-puller/internal/downloader"
	"feed-puller/internal/opensubtitles"
	"feed-puller/internal/store"
)

func newOpenSubtitlesServer(t *testing.T) (*Server, sqlmock.Sqlmock, func()) {
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

func expectEmptyOpenSubtitlesSettings(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT name, value FROM settings WHERE name IN (?, ?, ?, ?)`)).
		WithArgs("opensubtitles_username", "opensubtitles_password", "opensubtitles_api_key", "opensubtitles_download_dir").
		WillReturnRows(sqlmock.NewRows([]string{"name", "value"}))
}

func expectConfiguredOpenSubtitlesSettings(mock sqlmock.Sqlmock, downloadDir ...string) {
	dir := "/data/subtitles"
	if len(downloadDir) > 0 {
		dir = downloadDir[0]
	}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT name, value FROM settings WHERE name IN (?, ?, ?, ?)`)).
		WithArgs("opensubtitles_username", "opensubtitles_password", "opensubtitles_api_key", "opensubtitles_download_dir").
		WillReturnRows(sqlmock.NewRows([]string{"name", "value"}).
			AddRow("opensubtitles_username", "u").
			AddRow("opensubtitles_password", "p").
			AddRow("opensubtitles_api_key", "k").
			AddRow("opensubtitles_download_dir", dir))
}

func TestOpenSubtitlesSettingRequiresAuth(t *testing.T) {
	srv, _, cleanup := newOpenSubtitlesServer(t)
	defer cleanup()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/settings/opensubtitles", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestOpenSubtitlesSearch_NotConfigured(t *testing.T) {
	srv, mock, cleanup := newOpenSubtitlesServer(t)
	defer cleanup()
	expectEmptyOpenSubtitlesSettings(mock)
	req := authRequest(httptest.NewRequest(http.MethodGet, "/api/subtitles/search?query=Inception&languages=zh-CN", nil))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestOpenSubtitlesSearch_EmptyQuery(t *testing.T) {
	srv, mock, cleanup := newOpenSubtitlesServer(t)
	defer cleanup()
	expectConfiguredOpenSubtitlesSettings(mock)
	req := authRequest(httptest.NewRequest(http.MethodGet, "/api/subtitles/search?query=&languages=zh-CN", nil))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "query 不能为空") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestOpenSubtitlesSettingPut_Incomplete(t *testing.T) {
	srv, mock, cleanup := newOpenSubtitlesServer(t)
	defer cleanup()
	req := authRequest(httptest.NewRequest(http.MethodPut, "/api/settings/opensubtitles", strings.NewReader(`{"username":"u"}`)))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenSubtitlesSearch_FlattensItems(t *testing.T) {
	osAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subtitles" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"attributes":{"release":"Inception","language":"zh-CN","download_count":1,"ratings":9,"files":[{"file_id":7,"file_name":"a.srt"}]}}]}`))
	}))
	defer osAPI.Close()
	orig := opensubtitles.APIBaseURL
	opensubtitles.APIBaseURL = osAPI.URL
	t.Cleanup(func() { opensubtitles.APIBaseURL = orig })
	srv, mock, cleanup := newOpenSubtitlesServer(t)
	defer cleanup()
	expectConfiguredOpenSubtitlesSettings(mock)
	req := authRequest(httptest.NewRequest(http.MethodGet, "/api/subtitles/search?query=Inception&languages=zh-CN", nil))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Items []opensubtitles.SubtitleFile `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Items) != 1 || payload.Items[0].FileID != 7 {
		t.Fatalf("payload=%+v", payload)
	}
}

func TestOpenSubtitlesDownload_UsesFallbackBaseName(t *testing.T) {
	dir := t.TempDir()
	var osAPI *httptest.Server
	osAPI = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			_, _ = w.Write([]byte(`{"token":"tok"}`))
		case "/download":
			fmt.Fprintf(w, `{"link":"%s/payload.srt","file_name":""}`, osAPI.URL)
		case "/payload.srt":
			_, _ = w.Write([]byte("ok"))
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer osAPI.Close()
	orig := opensubtitles.APIBaseURL
	opensubtitles.APIBaseURL = osAPI.URL
	t.Cleanup(func() { opensubtitles.APIBaseURL = orig })
	srv, mock, cleanup := newOpenSubtitlesServer(t)
	defer cleanup()
	expectConfiguredOpenSubtitlesSettings(mock, dir)
	req := authRequest(httptest.NewRequest(http.MethodPost, "/api/subtitles/download", strings.NewReader(`{"file_id":9,"file_name":"../../etc/passwd"}`)))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	body, err := os.ReadFile(filepath.Join(dir, "passwd"))
	if err != nil || string(body) != "ok" {
		t.Fatalf("file err=%v body=%s", err, body)
	}
	var payload struct {
		Path     string `json:"path"`
		FileName string `json:"file_name"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.FileName != "passwd" || payload.Path != filepath.Join(dir, "passwd") {
		t.Fatalf("payload=%+v", payload)
	}
}

func TestOpenSubtitlesDownload_RejectsDotDotFileName(t *testing.T) {
	dir := t.TempDir()
	osAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			_, _ = w.Write([]byte(`{"token":"tok"}`))
			return
		}
		if r.URL.Path == "/download" {
			_, _ = w.Write([]byte(`{"link":"http://example/x","file_name":""}`))
			return
		}
		t.Fatalf("path %s", r.URL.Path)
	}))
	defer osAPI.Close()
	orig := opensubtitles.APIBaseURL
	opensubtitles.APIBaseURL = osAPI.URL
	t.Cleanup(func() { opensubtitles.APIBaseURL = orig })
	srv, mock, cleanup := newOpenSubtitlesServer(t)
	defer cleanup()
	expectConfiguredOpenSubtitlesSettings(mock, dir)
	req := authRequest(httptest.NewRequest(http.MethodPost, "/api/subtitles/download", strings.NewReader(`{"file_id":9,"file_name":".."}`)))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "文件名无效") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestOpenSubtitlesDownload_ZeroFileID(t *testing.T) {
	srv, _, cleanup := newOpenSubtitlesServer(t)
	defer cleanup()
	req := authRequest(httptest.NewRequest(http.MethodPost, "/api/subtitles/download", strings.NewReader(`{"file_id":0,"file_name":"a.srt"}`)))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestOpenSubtitlesDownload_LoginFailed(t *testing.T) {
	dir := t.TempDir()
	osAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"unauthorized"}`))
			return
		}
		t.Fatalf("path %s", r.URL.Path)
	}))
	defer osAPI.Close()
	orig := opensubtitles.APIBaseURL
	opensubtitles.APIBaseURL = osAPI.URL
	t.Cleanup(func() { opensubtitles.APIBaseURL = orig })
	srv, mock, cleanup := newOpenSubtitlesServer(t)
	defer cleanup()
	expectConfiguredOpenSubtitlesSettings(mock, dir)
	req := authRequest(httptest.NewRequest(http.MethodPost, "/api/subtitles/download", strings.NewReader(`{"file_id":9,"file_name":"a.srt"}`)))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "OpenSubtitles 登录失败") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestOpenSubtitlesSettingGet_ReturnsSnakeCaseAndConfigured(t *testing.T) {
	srv, mock, cleanup := newOpenSubtitlesServer(t)
	defer cleanup()
	expectConfiguredOpenSubtitlesSettings(mock)
	req := authRequest(httptest.NewRequest(http.MethodGet, "/api/settings/opensubtitles", nil))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["username"] != "u" || payload["password"] != "p" || payload["api_key"] != "k" || payload["download_dir"] != "/data/subtitles" {
		t.Fatalf("payload=%v", payload)
	}
	configured, ok := payload["configured"].(bool)
	if !ok || !configured {
		t.Fatalf("configured=%v", payload["configured"])
	}
	if _, hasCamel := payload["apiKey"]; hasCamel {
		t.Fatalf("expected snake_case keys, got %v", payload)
	}
}

func TestOpenSubtitlesSearch_UpstreamNon2xxIs502(t *testing.T) {
	osAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subtitles" {
			t.Fatalf("path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"文件名无效"}`))
	}))
	defer osAPI.Close()
	orig := opensubtitles.APIBaseURL
	opensubtitles.APIBaseURL = osAPI.URL
	t.Cleanup(func() { opensubtitles.APIBaseURL = orig })
	srv, mock, cleanup := newOpenSubtitlesServer(t)
	defer cleanup()
	expectConfiguredOpenSubtitlesSettings(mock)
	req := authRequest(httptest.NewRequest(http.MethodGet, "/api/subtitles/search?query=Inception&languages=zh-CN", nil))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestOpenSubtitlesDownload_FetchFailureDoesNotLeakLink(t *testing.T) {
	dir := t.TempDir()
	secretLink := "http://127.0.0.1:1/dl?token=one-time-secret"
	osAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			_, _ = w.Write([]byte(`{"token":"tok"}`))
		case "/download":
			fmt.Fprintf(w, `{"link":%q,"file_name":"a.srt"}`, secretLink)
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer osAPI.Close()
	orig := opensubtitles.APIBaseURL
	opensubtitles.APIBaseURL = osAPI.URL
	t.Cleanup(func() { opensubtitles.APIBaseURL = orig })
	srv, mock, cleanup := newOpenSubtitlesServer(t)
	defer cleanup()
	expectConfiguredOpenSubtitlesSettings(mock, dir)
	req := authRequest(httptest.NewRequest(http.MethodPost, "/api/subtitles/download", strings.NewReader(`{"file_id":9,"file_name":"a.srt"}`)))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "下载字幕文件失败") {
		t.Fatalf("body=%s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "one-time-secret") || strings.Contains(rec.Body.String(), secretLink) {
		t.Fatalf("leaked link: %s", rec.Body.String())
	}
}
