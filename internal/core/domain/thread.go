package domain

// Node is one item in a thread tree. Depth is 0 for the story and grows by
// one per reply level. Item is nil when the item could not be fetched.
type Node struct {
	ID       int    `json:"id"`
	Depth    int    `json:"depth"`
	Status   Status `json:"status"`
	Reason   string `json:"reason,omitempty"`
	Item     *Item  `json:"item,omitempty"`
	Children []Node `json:"children,omitempty"`
}

// MissingItem records a node whose content is absent or hidden.
type MissingItem struct {
	ID       int    `json:"id"`
	ParentID int    `json:"parent_id,omitempty"`
	Depth    int    `json:"depth"`
	Status   Status `json:"status"`
	Reason   string `json:"reason,omitempty"`
}

// ThreadStats lets a reader check completeness at a glance.
// ExpectedDescendants is HN's own comment count for the story; HN does not
// count deleted/dead comments there, so it is a reference, not an exact target.
type ThreadStats struct {
	ExpectedDescendants int `json:"expected_descendants"`
	CapturedComments    int `json:"captured_comments"`
	MissingCount        int `json:"missing_count"`
	MaxDepth            int `json:"max_depth"`
	Authors             int `json:"authors"`
}

// Thread is a story with its full comment tree.
type Thread struct {
	Root    Node          `json:"root"`
	Missing []MissingItem `json:"missing"`
	Stats   ThreadStats   `json:"stats"`
}

// Walk visits n and all its descendants depth-first, passing each node's
// parent id (0 for the root).
func (n Node) Walk(fn func(node Node, parentID int)) {
	n.walk(0, fn)
}

func (n Node) walk(parentID int, fn func(Node, int)) {
	fn(n, parentID)
	for _, c := range n.Children {
		c.walk(n.ID, fn)
	}
}
