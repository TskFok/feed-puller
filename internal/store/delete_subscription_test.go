package store

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDeleteSubscription_RemovesDependentRows(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	const subID int64 = 42
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM download_tasks WHERE subscription_id = ?`)).
		WithArgs(subID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM feed_items WHERE subscription_id = ?`)).
		WithArgs(subID).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM subscriptions WHERE id = ?`)).
		WithArgs(subID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	s := New(db)
	if err := s.DeleteSubscription(context.Background(), subID); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
