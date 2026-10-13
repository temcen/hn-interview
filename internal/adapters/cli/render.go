package cli

import (
	"fmt"
	"html"
	"io"
	"regexp"
	"strings"
	"time"

	"interview-process/internal/core/domain"
)

const snippetLen = 120

// renderCapture prints each story as an indented tree. Every line carries the
// depth, item id and author; missing items are marked inline and listed
// again under the story.
func renderCapture(w io.Writer, c domain.Capture) {
	for i, th := range c.Stories {
		fmt.Fprintf(w, "━━ Story %d/%d ━━ %s\n", i+1, len(c.Stories), storyHeader(th.Root))
		if it := th.Root.Item; it != nil && it.URL != "" {
			fmt.Fprintf(w, "   %s\n", it.URL)
		}
		s := th.Stats
		fmt.Fprintf(w, "   comments: %d captured / %d reported by HN · missing: %d · max depth: %d · authors: %d\n",
			s.CapturedComments, s.ExpectedDescendants, s.MissingCount, s.MaxDepth, s.Authors)
		for j, child := range th.Root.Children {
			renderNode(w, child, "", j == len(th.Root.Children)-1)
		}
		if len(th.Missing) > 0 {
			fmt.Fprintln(w, "   missing items:")
			for _, m := range th.Missing {
				line := fmt.Sprintf("     #%d (parent #%d, depth %d): %s", m.ID, m.ParentID, m.Depth, m.Status)
				if m.ParentID == 0 {
					line = fmt.Sprintf("     #%d (story, depth 0): %s", m.ID, m.Status)
				}
				if m.Reason != "" {
					line += " — " + m.Reason
				}
				fmt.Fprintln(w, line)
			}
		}
		fmt.Fprintln(w)
	}
	if len(c.MissingUsers) > 0 {
		fmt.Fprintln(w, "missing author profiles:")
		for _, u := range c.MissingUsers {
			line := fmt.Sprintf("  %s: %s", u.ID, u.Status)
			if u.Reason != "" {
				line += " — " + u.Reason
			}
			fmt.Fprintln(w, line)
		}
	}
}

func storyHeader(n domain.Node) string {
	it := n.Item
	if it == nil {
		return fmt.Sprintf("#%d %s", n.ID, statusTag(n))
	}
	h := fmt.Sprintf("#%d [%s] %q by %s · %s · %d points", it.ID, it.Type, html.UnescapeString(it.Title), author(it), ts(it.Time), it.Score)
	if n.Status != domain.StatusOK {
		h += " " + statusTag(n)
	}
	return h
}

func renderNode(w io.Writer, n domain.Node, prefix string, last bool) {
	branch, next := "├─ ", "│  "
	if last {
		branch, next = "└─ ", "   "
	}
	fmt.Fprintf(w, "%s%s[d%d] #%d %s\n", prefix, branch, n.Depth, n.ID, nodeBody(n))
	for i, c := range n.Children {
		renderNode(w, c, prefix+next, i == len(n.Children)-1)
	}
}

func nodeBody(n domain.Node) string {
	if n.Item == nil || n.Status == domain.StatusDeleted {
		return statusTag(n)
	}
	body := fmt.Sprintf("by %s · %s: %s", author(n.Item), ts(n.Item.Time), cleanText(n.Item.Text, snippetLen))
	if n.Status == domain.StatusDead {
		body = statusTag(n) + " " + body
	}
	return body
}

func statusTag(n domain.Node) string {
	switch n.Status {
	case domain.StatusError:
		return "[ERROR: " + n.Reason + "]"
	case domain.StatusNotFound:
		return "[NOT FOUND]"
	default:
		return "[" + strings.ToUpper(string(n.Status)) + "]"
	}
}

func author(it *domain.Item) string {
	if it.By == "" {
		return "(unknown)"
	}
	return it.By
}

func ts(unix int64) string {
	if unix == 0 {
		return "(no time)"
	}
	return time.Unix(unix, 0).UTC().Format("2006-01-02 15:04 UTC")
}

var (
	tagRe   = regexp.MustCompile(`<[^>]*>`)
	spaceRe = regexp.MustCompile(`\s+`)
)

// cleanText turns HN's HTML into a single-line snippet of at most max runes.
func cleanText(s string, max int) string {
	s = strings.ReplaceAll(s, "<p>", " ")
	s = html.UnescapeString(tagRe.ReplaceAllString(s, ""))
	s = strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))
	if r := []rune(s); len(r) > max {
		return string(r[:max]) + "…"
	}
	return s
}

func renderSummary(w io.Writer, c domain.Capture, elapsed time.Duration) {
	var comments, missing int
	for _, th := range c.Stories {
		comments += th.Stats.CapturedComments
		missing += th.Stats.MissingCount
	}
	fmt.Fprintf(w, "captured %d stories, %d comments, %d author profiles (%d missing), %d missing/hidden items in %s\n",
		len(c.Stories), comments, len(c.Users), len(c.MissingUsers), missing, elapsed.Round(time.Millisecond))
}

func renderUser(w io.Writer, u domain.User) {
	fmt.Fprintf(w, "%s\n  karma: %d\n  created: %s\n  submissions: %d\n", u.ID, u.Karma, ts(u.Created), len(u.Submitted))
	if u.About != "" {
		fmt.Fprintf(w, "  about: %s\n", cleanText(u.About, 300))
	}
}
