package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"interview-process/internal/core/domain"
)

// threadFixture: story 1 with kids [2, 3, 4, 5];
// 2 -> [6 -> [7]], 3 deleted with kid 8, 4 is null, 5 errors, 8 is dead.
func threadFixture() *fakeClient {
	return &fakeClient{
		items: map[int]*domain.Item{
			1: {ID: 1, Type: domain.TypeStory, By: "alice", Title: "Story", Kids: []int{2, 3, 4, 5}, Descendants: 6},
			2: {ID: 2, Type: domain.TypeComment, By: "bob", Parent: 1, Kids: []int{6}},
			6: {ID: 6, Type: domain.TypeComment, By: "alice", Parent: 2, Kids: []int{7}},
			7: {ID: 7, Type: domain.TypeComment, By: "carol", Parent: 6},
			3: {ID: 3, Type: domain.TypeComment, Deleted: true, Parent: 1, Kids: []int{8}},
			8: {ID: 8, Type: domain.TypeComment, By: "dave", Dead: true, Parent: 3},
		},
		itemErrs: map[int]error{5: errors.New("boom")},
	}
}

func TestBuildThreadNestsKidsInOrderWithDepth(t *testing.T) {
	r := newRun(threadFixture(), 4)
	th := r.buildThread(context.Background(), 1)

	root := th.Root
	if root.Status != domain.StatusOK || root.Depth != 0 || root.Item.Title != "Story" {
		t.Fatalf("root = %+v", root)
	}
	var ids []int
	for _, c := range root.Children {
		ids = append(ids, c.ID)
		if c.Depth != 1 {
			t.Errorf("child %d depth = %d, want 1", c.ID, c.Depth)
		}
	}
	if !reflect.DeepEqual(ids, []int{2, 3, 4, 5}) {
		t.Fatalf("child order = %v", ids)
	}
	deep := root.Children[0].Children[0].Children[0]
	if deep.ID != 7 || deep.Depth != 3 || deep.Item.By != "carol" {
		t.Fatalf("deepest node = %+v", deep)
	}
}

func TestBuildThreadMarksMissingNodes(t *testing.T) {
	th := newRun(threadFixture(), 4).buildThread(context.Background(), 1)
	kids := th.Root.Children

	cases := []struct {
		node domain.Node
		want domain.Status
	}{
		{kids[1], domain.StatusDeleted},
		{kids[2], domain.StatusNotFound},
		{kids[3], domain.StatusError},
		{kids[1].Children[0], domain.StatusDead},
	}
	for _, c := range cases {
		if c.node.Status != c.want {
			t.Errorf("node %d status = %s, want %s", c.node.ID, c.node.Status, c.want)
		}
	}
	if kids[2].Item != nil || kids[3].Item != nil {
		t.Errorf("unfetched nodes should have no item")
	}
	if kids[3].Reason != "boom" {
		t.Errorf("error reason = %q", kids[3].Reason)
	}
	if len(kids[1].Children) != 1 {
		t.Errorf("deleted node should still recurse into kids")
	}
}

func TestBuildThreadSummarizesMissingAndStats(t *testing.T) {
	th := newRun(threadFixture(), 4).buildThread(context.Background(), 1)

	want := []domain.MissingItem{
		{ID: 3, ParentID: 1, Depth: 1, Status: domain.StatusDeleted},
		{ID: 8, ParentID: 3, Depth: 2, Status: domain.StatusDead},
		{ID: 4, ParentID: 1, Depth: 1, Status: domain.StatusNotFound},
		{ID: 5, ParentID: 1, Depth: 1, Status: domain.StatusError, Reason: "boom"},
	}
	if !reflect.DeepEqual(th.Missing, want) {
		t.Fatalf("missing =\n %+v\nwant\n %+v", th.Missing, want)
	}
	wantStats := domain.ThreadStats{
		ExpectedDescendants: 6,
		CapturedComments:    3, // 2, 6, 7
		MissingCount:        4,
		MaxDepth:            3,
		Authors:             4, // alice, bob, carol, dave
	}
	if th.Stats != wantStats {
		t.Fatalf("stats = %+v, want %+v", th.Stats, wantStats)
	}
}

func TestBuildThreadRootNotFound(t *testing.T) {
	th := newRun(&fakeClient{}, 2).buildThread(context.Background(), 99)
	if th.Root.Status != domain.StatusNotFound || len(th.Missing) != 1 {
		t.Fatalf("thread = %+v", th)
	}
}
