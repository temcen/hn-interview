package domain

import "errors"

// Status describes whether a node's content was captured.
type Status string

const (
	// StatusOK means the item was fetched and is live.
	StatusOK Status = "ok"
	// StatusDead means the item was fetched but HN has flagged it dead
	// (hidden by default on the site). Content is kept.
	StatusDead Status = "dead"
	// StatusDeleted means HN reports the item as deleted; no author or text.
	StatusDeleted Status = "deleted"
	// StatusNotFound means the API returned null for the id.
	StatusNotFound Status = "not_found"
	// StatusError means the fetch failed (after retries); see Reason.
	StatusError Status = "error"
)

// ErrNotFound is returned when a requested entity does not exist upstream.
var ErrNotFound = errors.New("not found")
