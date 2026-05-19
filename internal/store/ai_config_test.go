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
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO ai_configs (name, base_url, model, api_key)`)).
		WithArgs("Demo", "https://api.example.com/v1", "gpt-4o-mini", "sk-test").
		WillReturnResult(sqlmock.NewResult(7, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, base_url, model, api_key, created_at, updated_at`)).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "base_url", "model", "api_key", "created_at", "updated_at"}).
			AddRow(7, "Demo", "https://api.example.com/v1", "gpt-4o-mini", "sk-test", now, now))

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
