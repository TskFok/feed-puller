package app

import (
	"log/slog"
	"testing"

	"feed-puller/internal/downloader"
	"feed-puller/internal/store"
)

func TestServiceSetAria2Client_UsesReplacementForSubsequentCalls(t *testing.T) {
	t.Parallel()
	oldClient := downloader.NewAria2Client("https://old.test/jsonrpc", "old-secret")
	newClient := downloader.NewAria2Client("https://new.test/jsonrpc", "new-secret")
	service := NewService(store.New(nil), oldClient, slog.Default())

	service.SetAria2Client(newClient)

	if got := service.Aria2Client(); got != newClient {
		t.Fatalf("Aria2Client() = %p, want replacement %p", got, newClient)
	}
}
