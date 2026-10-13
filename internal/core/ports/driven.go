// Package ports declares the boundaries of the core: driven ports the core
// calls out through, and the driving Facade that adapters call in through.
package ports

import (
	"context"

	"interview-process/internal/core/domain"
)

// HNClient reads Hacker News data. Item and User return (nil, nil) when the
// API has no such entity.
type HNClient interface {
	TopStoryIDs(ctx context.Context) ([]int, error)
	Item(ctx context.Context, id int) (*domain.Item, error)
	User(ctx context.Context, id string) (*domain.User, error)
}

// ArchiveWriter persists a capture. dest is an adapter-specific location
// (e.g. a directory); the returned string says where it was written.
type ArchiveWriter interface {
	Write(ctx context.Context, c domain.Capture, dest string) (string, error)
}
