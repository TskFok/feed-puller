package rss

import (
	"os"
	"strings"
	"testing"
)

func TestMagnetFromTorrent(t *testing.T) {
	data, err := os.ReadFile("testdata/mikan_sample.torrent")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	magnet, err := MagnetFromTorrent(data)
	if err != nil {
		t.Fatalf("MagnetFromTorrent: %v", err)
	}
	if magnet == "" || magnet[:8] != "magnet:?" {
		t.Fatalf("magnet = %q", magnet)
	}
	if !strings.Contains(strings.ToLower(magnet), "btih") {
		t.Fatalf("magnet missing info hash: %q", magnet)
	}
}

func TestParseMikanTorrentTime(t *testing.T) {
	if parseMikanTorrentTime("2026-05-18T01:00:51.216428") == nil {
		t.Fatal("expected parsed time")
	}
}
