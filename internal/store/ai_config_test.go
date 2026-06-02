package store

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateAIConfig_ValidatesRequiredFields(t *testing.T) {
	t.Parallel()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	s := New(db)
	_, err = s.CreateAIConfig(context.Background(), AIConfig{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCreateAIConfig_InsertsRow(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().UTC().Truncate(time.Second)
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO ai_configs (name, base_url, model, api_key, request_options)`)).
		WithArgs("Demo", "https://api.example.com/v1", "gpt-4o-mini", "sk-test", "").
		WillReturnResult(sqlmock.NewResult(7, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, base_url, model, api_key, request_options, created_at, updated_at`)).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "base_url", "model", "api_key", "request_options", "created_at", "updated_at"}).
			AddRow(7, "Demo", "https://api.example.com/v1", "gpt-4o-mini", "sk-test", "", now, now))

	s := New(db)
	cfg, err := s.CreateAIConfig(context.Background(), AIConfig{
		Name:    "Demo",
		BaseURL: "https://api.example.com/v1/",
		Model:   "gpt-4o-mini",
		APIKey:  "sk-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ID != 7 || cfg.Name != "Demo" || cfg.BaseURL != "https://api.example.com/v1" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateAIConfig_PersistsRequestOptions(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().UTC().Truncate(time.Second)
	options := `{"thinking":{"type":"disabled"}}`
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO ai_configs (name, base_url, model, api_key, request_options)`)).
		WithArgs("Demo", "https://api.example.com/v1", "kimi-k2.6", "sk-test", options).
		WillReturnResult(sqlmock.NewResult(7, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, base_url, model, api_key, request_options, created_at, updated_at`)).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "base_url", "model", "api_key", "request_options", "created_at", "updated_at"}).
			AddRow(7, "Demo", "https://api.example.com/v1", "kimi-k2.6", "sk-test", options, now, now))

	s := New(db)
	cfg, err := s.CreateAIConfig(context.Background(), AIConfig{
		Name:           "Demo",
		BaseURL:        "https://api.example.com/v1/",
		Model:          "kimi-k2.6",
		APIKey:         "sk-test",
		RequestOptions: options,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RequestOptions != options {
		t.Fatalf("request_options = %q, want %q", cfg.RequestOptions, options)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateAIConfig_ValidatesRequestOptionsJSON(t *testing.T) {
	t.Parallel()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	s := New(db)
	_, err = s.CreateAIConfig(context.Background(), AIConfig{
		Name:           "Demo",
		BaseURL:        "https://api.example.com/v1",
		Model:          "gpt-4o-mini",
		RequestOptions: `{"thinking":`,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCreateAIConfig_RejectsRequestOptionsArray(t *testing.T) {
	t.Parallel()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	s := New(db)
	_, err = s.CreateAIConfig(context.Background(), AIConfig{
		Name:           "Demo",
		BaseURL:        "https://api.example.com/v1",
		Model:          "gpt-4o-mini",
		RequestOptions: `[{"temperature":0.6}]`,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCreateAIConfig_AllowsEmptyAPIKey(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().UTC().Truncate(time.Second)
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO ai_configs (name, base_url, model, api_key, request_options)`)).
		WithArgs("Ollama", "http://localhost:11434/v1", "llama3.2", "", "").
		WillReturnResult(sqlmock.NewResult(8, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, base_url, model, api_key, request_options, created_at, updated_at`)).
		WithArgs(int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "base_url", "model", "api_key", "request_options", "created_at", "updated_at"}).
			AddRow(8, "Ollama", "http://localhost:11434/v1", "llama3.2", "", "", now, now))

	s := New(db)
	cfg, err := s.CreateAIConfig(context.Background(), AIConfig{
		Name:    "Ollama",
		BaseURL: "http://localhost:11434/v1",
		Model:   "llama3.2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "" {
		t.Fatalf("expected empty api key, got %q", cfg.APIKey)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteAIConfig_NotFound(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM ai_configs WHERE id = ?`)).
		WithArgs(int64(99)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	s := New(db)
	if err := s.DeleteAIConfig(context.Background(), 99); err == nil {
		t.Fatal("expected not found error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
