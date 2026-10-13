package domain

// User mirrors an HN profile from /v0/user/<id>.json. SubmittedCount is
// derived; Submitted may be dropped from captures to keep archives small.
type User struct {
	ID             string `json:"id"`
	Created        int64  `json:"created"`
	Karma          int    `json:"karma"`
	About          string `json:"about,omitempty"`
	SubmittedCount int    `json:"submitted_count"`
	Submitted      []int  `json:"submitted,omitempty"`
}

// MissingUser records an author whose profile could not be captured.
type MissingUser struct {
	ID     string `json:"id"`
	Status Status `json:"status"`
	Reason string `json:"reason,omitempty"`
}
