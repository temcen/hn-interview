package domain

import "time"

// Capture is the archived result of one pipeline run.
type Capture struct {
	CapturedAt   time.Time       `json:"captured_at"`
	Stories      []Thread        `json:"stories"`
	Users        map[string]User `json:"users"`
	MissingUsers []MissingUser   `json:"missing_users"`
}
