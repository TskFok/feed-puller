package store

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestListProwlarrSubmittedGuids_ReturnsInProgressAndCompleted(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	s := New(db)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COALESCE(guid, ''), dedupe_key
		FROM feed_items
		WHERE dedupe_key IN (?,?)
		  AND download_status IN (?, ?, ?)
	`)).WithArgs("prowlarr:g1", "prowlarr:g2", "submitting", "submitted", "completed").
		WillReturnRows(sqlmock.NewRows([]string{"guid", "dedupe_key"}).
			AddRow("g1", "prowlarr:g1"))

	got, err := s.ListProwlarrSubmittedGuids(context.Background(), []string{"g1", "g2", "g1"})
	if err != nil {
		t.Fatalf("ListProwlarrSubmittedGuids() error = %v", err)
	}
	if len(got) != 1 || got[0] != "g1" {
		t.Fatalf("got = %#v, want [g1]", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListProwlarrSubmittedGuids_EmptyInput(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	s := New(db)
	got, err := s.ListProwlarrSubmittedGuids(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListProwlarrSubmittedGuids() error = %v", err)
	}
	if got != nil {
		t.Fatalf("got = %#v, want nil", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
