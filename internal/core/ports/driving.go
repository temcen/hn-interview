package ports

import (
	"context"

	"interview-process/internal/core/domain"
)

// CaptureOptions tunes a single capture run.
type CaptureOptions struct {
	// Concurrency caps in-flight upstream requests. Values < 1 mean 1.
	Concurrency int
	// FullProfiles keeps each author's full submission id list. Otherwise
	// only SubmittedCount is kept (the lists dominate archive size).
	FullProfiles bool
}

// Facade is the only entry point driving adapters (the CLI) use.
type Facade interface {
	// CaptureTop captures the first n top stories with full comment trees
	// and author profiles.
	CaptureTop(ctx context.Context, n int, opts CaptureOptions) (domain.Capture, error)
	// CaptureStory captures a single item's thread and author profiles.
	CaptureStory(ctx context.Context, id int, opts CaptureOptions) (domain.Capture, error)
	// GetUser fetches one profile; returns domain.ErrNotFound if absent.
	GetUser(ctx context.Context, id string) (domain.User, error)
	// Save archives a capture at dest and returns its location.
	Save(ctx context.Context, c domain.Capture, dest string) (string, error)
}
