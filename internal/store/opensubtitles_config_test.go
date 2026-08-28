package store

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetOpenSubtitlesConfig_Empty(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT name, value FROM settings WHERE name IN (?, ?, ?, ?)`)).
		WithArgs("opensubtitles_username", "opensubtitles_password", "opensubtitles_api_key", "opensubtitles_download_dir").
		WillReturnRows(sqlmock.NewRows([]string{"name", "value"}))
	cfg, err := New(db).GetOpenSubtitlesConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Configured || cfg.Username != "" || cfg.APIKey != "" {
		t.Fatalf("got %+v", cfg)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSaveOpenSubtitlesConfig_RejectsIncompleteWithoutWrite(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = New(db).SaveOpenSubtitlesConfig(context.Background(), OpenSubtitlesConfig{
		Username: "u", Password: "p", APIKey: "", DownloadDir: "/data/subtitles",
	})
	if !errors.Is(err, ErrOpenSubtitlesConfigIncomplete) {
		t.Fatalf("err = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSaveOpenSubtitlesConfig_WritesAllKeysThenReadsBack(t *testing.T) {
	t.Parallel()
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
	cfg, err := New(db).SaveOpenSubtitlesConfig(context.Background(), OpenSubtitlesConfig{
		Username: " alice ", Password: " secret ", APIKey: " key-1 ", DownloadDir: " /data/subtitles ",
	})
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
