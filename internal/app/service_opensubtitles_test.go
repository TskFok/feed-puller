package app

import (
	"context"
	"errors"
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

	"feed-puller/internal/downloader"
	"feed-puller/internal/opensubtitles"
	"feed-puller/internal/store"
)

func expectOpenSubtitlesSettings(mock sqlmock.Sqlmock, values map[string]string) {
	rows := sqlmock.NewRows([]string{"name", "value"})
	keys := []string{"opensubtitles_username", "opensubtitles_password", "opensubtitles_api_key", "opensubtitles_download_dir"}
	for _, key := range keys {
		if value, ok := values[key]; ok {
			rows.AddRow(key, value)
		}
	}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT name, value FROM settings WHERE name IN (?, ?, ?, ?)`)).
		WithArgs("opensubtitles_username", "opensubtitles_password", "opensubtitles_api_key", "opensubtitles_download_dir").
		WillReturnRows(rows)
}

func TestSearchSubtitles_NotConfigured(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT name, value FROM settings WHERE name IN (?, ?, ?, ?)`)).
		WithArgs("opensubtitles_username", "opensubtitles_password", "opensubtitles_api_key", "opensubtitles_download_dir").
		WillReturnRows(sqlmock.NewRows([]string{"name", "value"}))
	svc := NewService(store.New(db), downloader.NewAria2Client("", ""), slog.Default())
	_, err = svc.SearchSubtitles(context.Background(), "Inception", "zh-CN")
	if !errors.Is(err, ErrOpenSubtitlesNotConfigured) {
		t.Fatalf("err=%v", err)
	}
}

func TestSearchSubtitles_ReturnsFiles(t *testing.T) {
	osAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subtitles" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "Inception" || r.URL.Query().Get("languages") != "zh-CN" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"data":[{"attributes":{"release":"Inception","language":"zh-CN","download_count":1,"ratings":9,"files":[{"file_id":7,"file_name":"inception.srt"}]}}]}`))
	}))
	defer osAPI.Close()
	orig := opensubtitles.APIBaseURL
	opensubtitles.APIBaseURL = osAPI.URL
	t.Cleanup(func() { opensubtitles.APIBaseURL = orig })

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	expectOpenSubtitlesSettings(mock, map[string]string{
		"opensubtitles_username":     "u",
		"opensubtitles_password":     "p",
		"opensubtitles_api_key":      "k",
		"opensubtitles_download_dir": t.TempDir(),
	})
	svc := NewService(store.New(db), downloader.NewAria2Client("", ""), slog.Default())
	items, err := svc.SearchSubtitles(context.Background(), "Inception", "zh-CN")
	if err != nil || len(items) != 1 || items[0].FileID != 7 || items[0].FileName != "inception.srt" {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadSubtitle_WritesSanitizedFile(t *testing.T) {
	dir := t.TempDir()
	var sawDownload bool
	var osAPI *httptest.Server
	osAPI = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			_, _ = w.Write([]byte(`{"token":"tok"}`))
		case "/download":
			sawDownload = true
			fmt.Fprintf(w, `{"link":"%s/payload.srt","file_name":"/etc/../shown.srt"}`, osAPI.URL)
		case "/payload.srt":
			_, _ = w.Write([]byte("subtitle-bytes"))
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer osAPI.Close()
	orig := opensubtitles.APIBaseURL
	opensubtitles.APIBaseURL = osAPI.URL
	t.Cleanup(func() { opensubtitles.APIBaseURL = orig })

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT name, value FROM settings WHERE name IN (?, ?, ?, ?)`)).
		WithArgs("opensubtitles_username", "opensubtitles_password", "opensubtitles_api_key", "opensubtitles_download_dir").
		WillReturnRows(sqlmock.NewRows([]string{"name", "value"}).
			AddRow("opensubtitles_username", "u").
			AddRow("opensubtitles_password", "p").
			AddRow("opensubtitles_api_key", "k").
			AddRow("opensubtitles_download_dir", dir))
	svc := NewService(store.New(db), downloader.NewAria2Client("", ""), slog.Default())
	path, name, err := svc.DownloadSubtitle(context.Background(), 9, "fallback.srt")
	if err != nil {
		t.Fatal(err)
	}
	if name != "shown.srt" || path != filepath.Join(dir, "shown.srt") || !sawDownload {
		t.Fatalf("path=%s name=%s", path, name)
	}
	body, _ := os.ReadFile(path)
	if string(body) != "subtitle-bytes" {
		t.Fatalf("body=%s", body)
	}
}

func TestDownloadSubtitle_RejectsEmptySanitizedName(t *testing.T) {
	dir := t.TempDir()
	var osAPI *httptest.Server
	osAPI = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			_, _ = w.Write([]byte(`{"token":"tok"}`))
		case "/download":
			fmt.Fprintf(w, `{"link":"%s/payload.srt","file_name":".."}`, osAPI.URL)
		case "/payload.srt":
			t.Fatal("should not fetch file when name is invalid")
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer osAPI.Close()
	orig := opensubtitles.APIBaseURL
	opensubtitles.APIBaseURL = osAPI.URL
	t.Cleanup(func() { opensubtitles.APIBaseURL = orig })

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	expectOpenSubtitlesSettings(mock, map[string]string{
		"opensubtitles_username":     "u",
		"opensubtitles_password":     "p",
		"opensubtitles_api_key":      "k",
		"opensubtitles_download_dir": dir,
	})
	svc := NewService(store.New(db), downloader.NewAria2Client("", ""), slog.Default())
	path, name, err := svc.DownloadSubtitle(context.Background(), 9, "..")
	if !errors.Is(err, opensubtitles.ErrInvalidFileName) {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(err.Error(), "文件名无效") {
		t.Fatalf("err=%v", err)
	}
	if path != "" || name != "" {
		t.Fatalf("path=%s name=%s", path, name)
	}
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("temp dir should be empty, got %v", entries)
	}
}

func TestDownloadSubtitle_ZeroFileIDFailsBeforeNetwork(t *testing.T) {
	dir := t.TempDir()
	var hitAPI bool
	osAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitAPI = true
		t.Fatalf("path %s", r.URL.Path)
	}))
	defer osAPI.Close()
	orig := opensubtitles.APIBaseURL
	opensubtitles.APIBaseURL = osAPI.URL
	t.Cleanup(func() { opensubtitles.APIBaseURL = orig })

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	expectOpenSubtitlesSettings(mock, map[string]string{
		"opensubtitles_username":     "u",
		"opensubtitles_password":     "p",
		"opensubtitles_api_key":      "k",
		"opensubtitles_download_dir": dir,
	})
	svc := NewService(store.New(db), downloader.NewAria2Client("", ""), slog.Default())
	_, _, err = svc.DownloadSubtitle(context.Background(), 0, "fallback.srt")
	if !errors.Is(err, opensubtitles.ErrInvalidFileID) {
		t.Fatalf("err=%v", err)
	}
	if hitAPI {
		t.Fatal("fileID==0 should fail before network")
	}
}

func TestDownloadSubtitle_WriteFailsWhenDirMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing-subdir")
	var osAPI *httptest.Server
	osAPI = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			_, _ = w.Write([]byte(`{"token":"tok"}`))
		case "/download":
			fmt.Fprintf(w, `{"link":"%s/payload.srt","file_name":"shown.srt"}`, osAPI.URL)
		case "/payload.srt":
			_, _ = w.Write([]byte("subtitle-bytes"))
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer osAPI.Close()
	orig := opensubtitles.APIBaseURL
	opensubtitles.APIBaseURL = osAPI.URL
	t.Cleanup(func() { opensubtitles.APIBaseURL = orig })

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	expectOpenSubtitlesSettings(mock, map[string]string{
		"opensubtitles_username":     "u",
		"opensubtitles_password":     "p",
		"opensubtitles_api_key":      "k",
		"opensubtitles_download_dir": dir,
	})
	svc := NewService(store.New(db), downloader.NewAria2Client("", ""), slog.Default())
	_, _, err = svc.DownloadSubtitle(context.Background(), 9, "fallback.srt")
	if err == nil {
		t.Fatal("expected write error")
	}
	if !errors.Is(err, ErrSubtitleWriteFailed) {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(err.Error(), "保存字幕失败") {
		t.Fatalf("err=%v", err)
	}
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("want path error unwrap, err=%v", err)
	}
}

func TestGetOpenSubtitlesConfig_DelegatesToStore(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	expectOpenSubtitlesSettings(mock, map[string]string{
		"opensubtitles_username":     "alice",
		"opensubtitles_password":     "secret",
		"opensubtitles_api_key":      "key-1",
		"opensubtitles_download_dir": "/data/subtitles",
	})
	svc := NewService(store.New(db), downloader.NewAria2Client("", ""), slog.Default())
	cfg, err := svc.GetOpenSubtitlesConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Configured || cfg.Username != "alice" || cfg.DownloadDir != "/data/subtitles" {
		t.Fatalf("got %+v", cfg)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSaveOpenSubtitlesConfig_DelegatesToStore(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO settings (name, value) VALUES (?, ?), (?, ?), (?, ?), (?, ?) ON DUPLICATE KEY UPDATE value = VALUES(value), updated_at = CURRENT_TIMESTAMP`)).
		WithArgs(
			"opensubtitles_username", "alice",
			"opensubtitles_password", "secret",
			"opensubtitles_api_key", "key-1",
			"opensubtitles_download_dir", "/data/subtitles",
		).
		WillReturnResult(sqlmock.NewResult(0, 4))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT name, value FROM settings WHERE name IN (?, ?, ?, ?)`)).
		WithArgs("opensubtitles_username", "opensubtitles_password", "opensubtitles_api_key", "opensubtitles_download_dir").
		WillReturnRows(sqlmock.NewRows([]string{"name", "value"}).
			AddRow("opensubtitles_username", "alice").
			AddRow("opensubtitles_password", "secret").
			AddRow("opensubtitles_api_key", "key-1").
			AddRow("opensubtitles_download_dir", "/data/subtitles"))
	svc := NewService(store.New(db), downloader.NewAria2Client("", ""), slog.Default())
	cfg, err := svc.SaveOpenSubtitlesConfig(context.Background(), store.OpenSubtitlesConfig{
		Username: " alice ", Password: " secret ", APIKey: " key-1 ", DownloadDir: " /data/subtitles ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Configured || cfg.Username != "alice" {
		t.Fatalf("got %+v", cfg)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
