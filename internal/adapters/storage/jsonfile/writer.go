// Package jsonfile is an ports.ArchiveWriter that stores each capture as an
// indented JSON file in a directory.
package jsonfile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"interview-process/internal/core/domain"
	"interview-process/internal/core/ports"
)

type Writer struct{}

var _ ports.ArchiveWriter = Writer{}

// Write stores c under dir as capture-<UTC timestamp>.json. It writes to a
// temp file and renames, so a crash never leaves a half-written archive.
func (Writer) Write(_ context.Context, c domain.Capture, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode capture: %w", err)
	}
	path := filepath.Join(dir, "capture-"+c.CapturedAt.UTC().Format("20060102T150405Z")+".json")

	tmp, err := os.CreateTemp(dir, ".capture-*.tmp")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	// CreateTemp uses 0600; archives are meant to be shared.
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return "", err
	}
	return path, nil
}
