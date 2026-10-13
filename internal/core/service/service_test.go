package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"interview-process/internal/core/domain"
	"interview-process/internal/core/ports"
)

type fakeArchive struct {
	got  domain.Capture
	dest string
}

func (a *fakeArchive) Write(_ context.Context, c domain.Capture, dest string) (string, error) {
	a.got, a.dest = c, dest
	return dest + "/file.json", nil
}

func twoStoryClient() *fakeClient {
	return &fakeClient{
		top: []int{10, 20, 30},
		items: map[int]*domain.Item{
			10: {ID: 10, Type: domain.TypeStory, By: "alice", Kids: []int{11, 12}},
			11: {ID: 11, Type: domain.TypeComment, By: "bob"},
			12: {ID: 12, Type: domain.TypeComment, By: "ghost"},
			20: {ID: 20, Type: domain.TypeJob, By: "bob"},
			30: {ID: 30, Type: domain.TypeStory, By: "zed"},
		},
		users: map[string]*domain.User{
			"alice": {ID: "alice", Karma: 10},
			"bob":   {ID: "bob", Karma: 20},
		},
		userErrs: map[string]error{},
	}
}

func TestCaptureTopTakesFirstNInOrderAndDedupesUsers(t *testing.T) {
	fc := twoStoryClient()
	svc := New(fc, &fakeArchive{})
	svc.now = func() time.Time { return time.Unix(100, 0).UTC() }

	c, err := svc.CaptureTop(context.Background(), 2, ports.CaptureOptions{Concurrency: 4})
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Stories) != 2 || c.Stories[0].Root.ID != 10 || c.Stories[1].Root.ID != 20 {
		t.Fatalf("stories = %+v", c.Stories)
	}
	if !c.CapturedAt.Equal(time.Unix(100, 0)) {
		t.Errorf("captured_at = %v", c.CapturedAt)
	}
	if len(c.Users) != 2 || c.Users["alice"].Karma != 10 || c.Users["bob"].Karma != 20 {
		t.Errorf("users = %+v", c.Users)
	}
	wantMissing := []domain.MissingUser{{ID: "ghost", Status: domain.StatusNotFound}}
	if !reflect.DeepEqual(c.MissingUsers, wantMissing) {
		t.Errorf("missing users = %+v", c.MissingUsers)
	}
	if fc.userCalls["bob"] != 1 {
		t.Errorf("bob fetched %d times, want 1", fc.userCalls["bob"])
	}
	if _, ok := fc.userCalls["zed"]; ok {
		t.Errorf("author of an uncaptured story was fetched")
	}
}

func TestCaptureTopClampsNAndRejectsNonPositive(t *testing.T) {
	svc := New(twoStoryClient(), &fakeArchive{})
	c, err := svc.CaptureTop(context.Background(), 50, ports.CaptureOptions{})
	if err != nil || len(c.Stories) != 3 {
		t.Fatalf("got %d stories, err %v", len(c.Stories), err)
	}
	if _, err := svc.CaptureTop(context.Background(), 0, ports.CaptureOptions{}); err == nil {
		t.Fatal("expected error for n=0")
	}
}

func TestCaptureTopFailsWhenTopStoriesUnavailable(t *testing.T) {
	fc := &fakeClient{topErr: errors.New("down")}
	if _, err := New(fc, &fakeArchive{}).CaptureTop(context.Background(), 5, ports.CaptureOptions{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestUserFetchErrorIsRecordedNotFatal(t *testing.T) {
	fc := twoStoryClient()
	fc.userErrs["bob"] = errors.New("timeout")
	c, err := New(fc, &fakeArchive{}).CaptureStory(context.Background(), 10, ports.CaptureOptions{Concurrency: 2})
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.MissingUser{
		{ID: "bob", Status: domain.StatusError, Reason: "timeout"},
		{ID: "ghost", Status: domain.StatusNotFound},
	}
	if !reflect.DeepEqual(c.MissingUsers, want) {
		t.Fatalf("missing users = %+v", c.MissingUsers)
	}
}

func TestConcurrencyIsBoundedAndUsed(t *testing.T) {
	// A wide tree: story with 40 kids, each with a distinct author.
	fc := &fakeClient{top: []int{1}, items: map[int]*domain.Item{}, delay: 5 * time.Millisecond}
	root := &domain.Item{ID: 1, Type: domain.TypeStory}
	for i := 2; i < 42; i++ {
		root.Kids = append(root.Kids, i)
		fc.items[i] = &domain.Item{ID: i, By: string(rune('a' + i))}
	}
	fc.items[1] = root

	if _, err := New(fc, &fakeArchive{}).CaptureTop(context.Background(), 1, ports.CaptureOptions{Concurrency: 5}); err != nil {
		t.Fatal(err)
	}
	if p := fc.peak.Load(); p > 5 || p < 2 {
		t.Fatalf("peak in-flight = %d, want 2..5", p)
	}
}

func TestGetUser(t *testing.T) {
	svc := New(twoStoryClient(), &fakeArchive{})
	u, err := svc.GetUser(context.Background(), "alice")
	if err != nil || u.Karma != 10 {
		t.Fatalf("u=%+v err=%v", u, err)
	}
	if _, err := svc.GetUser(context.Background(), "nobody"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestSaveDelegatesToArchive(t *testing.T) {
	a := &fakeArchive{}
	loc, err := New(twoStoryClient(), a).Save(context.Background(), domain.Capture{}, "out")
	if err != nil || loc != "out/file.json" || a.dest != "out" {
		t.Fatalf("loc=%q err=%v dest=%q", loc, err, a.dest)
	}
}
