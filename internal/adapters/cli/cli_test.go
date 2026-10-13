package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"interview-process/internal/core/domain"
	"interview-process/internal/core/ports"
)

type call struct {
	method      string
	n, id       int
	user, dest  string
	concurrency int
	full        bool
}

type fakeFacade struct {
	calls   []call
	capture domain.Capture
	err     error
	user    domain.User
	userErr error
}

func (f *fakeFacade) CaptureTop(_ context.Context, n int, o ports.CaptureOptions) (domain.Capture, error) {
	f.calls = append(f.calls, call{method: "top", n: n, concurrency: o.Concurrency, full: o.FullProfiles})
	return f.capture, f.err
}

func (f *fakeFacade) CaptureStory(_ context.Context, id int, o ports.CaptureOptions) (domain.Capture, error) {
	f.calls = append(f.calls, call{method: "story", id: id, concurrency: o.Concurrency})
	return f.capture, f.err
}

func (f *fakeFacade) GetUser(_ context.Context, id string) (domain.User, error) {
	f.calls = append(f.calls, call{method: "user", user: id})
	return f.user, f.userErr
}

func (f *fakeFacade) Save(_ context.Context, _ domain.Capture, dest string) (string, error) {
	f.calls = append(f.calls, call{method: "save", dest: dest})
	return dest + "/capture.json", nil
}

func run(f *fakeFacade, args ...string) (code int, stdout, stderr string) {
	var out, errb bytes.Buffer
	code = CLI{Facade: f, Stdout: &out, Stderr: &errb}.Run(context.Background(), args)
	return code, out.String(), errb.String()
}

func TestDefaultIsCaptureTop5AndSaves(t *testing.T) {
	f := &fakeFacade{}
	code, _, stderr := run(f)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	want := []call{{method: "top", n: 5, concurrency: 16}, {method: "save", dest: "output"}}
	if fmt.Sprint(f.calls) != fmt.Sprint(want) {
		t.Fatalf("calls = %v, want %v", f.calls, want)
	}
	if !strings.Contains(stderr, "output/capture.json") {
		t.Errorf("stderr should mention archive path: %s", stderr)
	}
}

func TestCaptureFlags(t *testing.T) {
	f := &fakeFacade{}
	if code, _, e := run(f, "capture", "--top", "3", "--concurrency", "4", "--full-profiles", "--out", ""); code != 0 {
		t.Fatalf("code = %d: %s", code, e)
	}
	want := []call{{method: "top", n: 3, concurrency: 4, full: true}}
	if fmt.Sprint(f.calls) != fmt.Sprint(want) {
		t.Fatalf("calls = %v (empty --out must skip saving)", f.calls)
	}
}

func TestStoryAcceptsFlagsAfterID(t *testing.T) {
	f := &fakeFacade{}
	if code, _, e := run(f, "story", "8863", "--out", "arch"); code != 0 {
		t.Fatalf("code = %d: %s", code, e)
	}
	want := []call{{method: "story", id: 8863, concurrency: 16}, {method: "save", dest: "arch"}}
	if fmt.Sprint(f.calls) != fmt.Sprint(want) {
		t.Fatalf("calls = %v", f.calls)
	}
}

func TestJSONFormatPrintsCapture(t *testing.T) {
	f := &fakeFacade{capture: domain.Capture{Users: map[string]domain.User{"a": {ID: "a"}}}}
	_, stdout, _ := run(f, "--format", "json", "--out", "")
	var got domain.Capture
	if err := json.Unmarshal([]byte(stdout), &got); err != nil || got.Users["a"].ID != "a" {
		t.Fatalf("stdout not a capture: %v\n%s", err, stdout)
	}
}

func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{"bogus"},
		{"story"},
		{"story", "abc"},
		{"user"},
		{"capture", "--format", "xml"},
		{"capture", "--top", "0"},
	} {
		if code, _, _ := run(&fakeFacade{}, args...); code != 2 {
			t.Errorf("%v: code = %d, want 2", args, code)
		}
	}
}

func TestFatalErrorExits1(t *testing.T) {
	code, _, stderr := run(&fakeFacade{err: errors.New("down")})
	if code != 1 || !strings.Contains(stderr, "down") {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
}

func TestUser(t *testing.T) {
	f := &fakeFacade{user: domain.User{ID: "jl", Karma: 2937, Created: 1173923446}}
	code, stdout, _ := run(f, "user", "jl")
	if code != 0 || !strings.Contains(stdout, "jl") || !strings.Contains(stdout, "2937") {
		t.Fatalf("code=%d stdout=%s", code, stdout)
	}
	f.userErr = domain.ErrNotFound
	if code, _, _ := run(f, "user", "nobody"); code != 1 {
		t.Fatalf("not found code = %d, want 1", code)
	}
}
