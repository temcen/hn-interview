package cli

import (
	"bytes"
	"strings"
	"testing"

	"interview-process/internal/core/domain"
)

func TestRenderCaptureShowsTreeDepthAuthorsAndMissing(t *testing.T) {
	c := domain.Capture{
		Stories: []domain.Thread{{
			Root: domain.Node{ID: 1, Status: domain.StatusOK,
				Item: &domain.Item{ID: 1, Type: domain.TypeStory, By: "alice", Title: "Hello &amp; welcome", Score: 42, Time: 1760347800, URL: "https://x.test"},
				Children: []domain.Node{
					{ID: 2, Depth: 1, Status: domain.StatusOK, Item: &domain.Item{ID: 2, By: "bob", Time: 1760347860, Text: "First<p>para with <i>html</i>"},
						Children: []domain.Node{{ID: 3, Depth: 2, Status: domain.StatusDeleted, Item: &domain.Item{ID: 3, Deleted: true}}}},
					{ID: 4, Depth: 1, Status: domain.StatusError, Reason: "timeout"},
				}},
			Missing: []domain.MissingItem{
				{ID: 3, ParentID: 2, Depth: 2, Status: domain.StatusDeleted},
				{ID: 4, ParentID: 1, Depth: 1, Status: domain.StatusError, Reason: "timeout"},
			},
			Stats: domain.ThreadStats{ExpectedDescendants: 3, CapturedComments: 1, MissingCount: 2, MaxDepth: 2, Authors: 2},
		}},
		MissingUsers: []domain.MissingUser{{ID: "ghost", Status: domain.StatusNotFound}},
	}

	var buf bytes.Buffer
	renderCapture(&buf, c)
	out := buf.String()

	for _, want := range []string{
		`#1 [story] "Hello & welcome" by alice`,
		"42 points",
		"https://x.test",
		"1 captured / 3 reported by HN",
		"├─ [d1] #2 by bob · 2025-10-13 09:31 UTC: First para with html",
		"│  └─ [d2] #3 [DELETED]",
		"└─ [d1] #4 [ERROR: timeout]",
		"#3 (parent #2, depth 2): deleted",
		"#4 (parent #1, depth 1): error — timeout",
		"ghost: not_found",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestCleanTextTruncates(t *testing.T) {
	got := cleanText(strings.Repeat("a", 300), 10)
	if got != "aaaaaaaaaa…" {
		t.Fatalf("got %q", got)
	}
}
