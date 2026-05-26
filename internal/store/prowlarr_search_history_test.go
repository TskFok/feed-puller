package store

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRecordProwlarrSearchHistory_Upsert(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO prowlarr_search_history`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM prowlarr_search_history`)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = s.RecordProwlarrSearchHistory(context.Background(), ProwlarrSearchHistory{
		DisplayQuery: "Inception",
		Query:        "inception",
		MediaType:    ProwlarrMediaMovie,
		SortBy:       "seeders",
		IndexerIDs:   []int64{1, 2},
		ResultCount:  3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListProwlarrSearchHistory(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`FROM prowlarr_search_history`)).
		WithArgs(20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "display_query", "query", "media_type", "sort_by", "indexer_ids", "result_count", "updated_at",
		}).AddRow(int64(1), "Inception", "inception", "movie", "seeders", "[]", 5, now))

	items, err := s.ListProwlarrSearchHistory(context.Background(), 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].DisplayQuery != "Inception" {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestDeleteProwlarrSearchHistory_NotFound(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM prowlarr_search_history WHERE id = ?`)).
		WithArgs(int64(99)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = s.DeleteProwlarrSearchHistory(context.Background(), 99)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}
