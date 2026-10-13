// Package domain holds the pure data types of the capture pipeline. It has no
// I/O; JSON tags define the archive schema.
package domain

// ItemType is one of the HN item kinds.
type ItemType string

const (
	TypeStory   ItemType = "story"
	TypeComment ItemType = "comment"
	TypeJob     ItemType = "job"
	TypePoll    ItemType = "poll"
	TypePollOpt ItemType = "pollopt"
)

// Item mirrors an HN item as returned by /v0/item/<id>.json. Only ID is
// guaranteed; every other field is optional.
type Item struct {
	ID          int      `json:"id"`
	Deleted     bool     `json:"deleted,omitempty"`
	Type        ItemType `json:"type,omitempty"`
	By          string   `json:"by,omitempty"`
	Time        int64    `json:"time,omitempty"`
	Text        string   `json:"text,omitempty"`
	Dead        bool     `json:"dead,omitempty"`
	Parent      int      `json:"parent,omitempty"`
	Poll        int      `json:"poll,omitempty"`
	Kids        []int    `json:"kids,omitempty"`
	URL         string   `json:"url,omitempty"`
	Score       int      `json:"score,omitempty"`
	Title       string   `json:"title,omitempty"`
	Parts       []int    `json:"parts,omitempty"`
	Descendants int      `json:"descendants,omitempty"`
}
