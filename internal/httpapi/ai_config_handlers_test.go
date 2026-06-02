package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"feed-puller/internal/store"
)

func TestHandleAIConfigModels_Success(t *testing.T) {
	t.Parallel()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4o-mini"}]}`))
	}))
	t.Cleanup(upstream.Close)

	server := &Server{}
	body, _ := json.Marshal(map[string]string{
		"url":     upstream.URL + "/v1",
		"api_key": "sk-test",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/ai-configs/models", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.handleAIConfigModels(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Models []string `json:"models"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Models) != 1 || payload.Models[0] != "gpt-4o-mini" {
		t.Fatalf("unexpected models: %#v", payload.Models)
	}
}

func TestHandleAIConfigModels_ValidatesURL(t *testing.T) {
	t.Parallel()
	server := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/ai-configs/models", bytes.NewReader([]byte(`{"api_key":"sk-test"}`)))
	rec := httptest.NewRecorder()
	server.handleAIConfigModels(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleAIConfigByID_Models(t *testing.T) {
	t.Parallel()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"id":"deepseek-chat"}]}`))
	}))
	t.Cleanup(upstream.Close)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Now().UTC().Truncate(time.Second)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, base_url, model, api_key, created_at, updated_at`)).
		WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "base_url", "model", "api_key", "created_at", "updated_at"}).
			AddRow(3, "Demo", upstream.URL+"/v1", "deepseek-chat", "sk-test", now, now))

	server := &Server{store: store.New(db)}
	req := httptest.NewRequest(http.MethodPost, "/api/ai-configs/3/models", nil)
	rec := httptest.NewRecorder()
	server.handleAIConfigByID(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Models []string `json:"models"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Models) != 1 || payload.Models[0] != "deepseek-chat" {
		t.Fatalf("unexpected models: %#v", payload.Models)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
