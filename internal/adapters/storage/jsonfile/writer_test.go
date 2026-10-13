package jsonfile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"interview-process/internal/core/domain"
)

func TestWriteCreatesTimestampedFileThatRoundTrips(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "out")
	c := domain.Capture{
		CapturedAt: time.Date(2026, 10, 13, 9, 30, 15, 0, time.UTC),
		Stories: []domain.Thread{{Root: domain.Node{ID: 1, Status: domain.StatusOK,
			Item: &domain.Item{ID: 1, By: "alice", Title: "Hi"}}}},
		Users: map[string]domain.User{"alice": {ID: "alice", Karma: 5}},
	}

	path, err := Writer{}.Write(context.Background(), c, dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "capture-20261013T093015Z.json"); path != want {
		t.Fatalf("path = %s, want %s", path, want)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got domain.Capture
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Stories[0].Root.Item.By != "alice" || got.Users["alice"].Karma != 5 {
		t.Fatalf("round trip = %+v", got)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Fatalf("expected only the final file, got %d entries", len(entries))
	}
}
