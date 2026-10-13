package hnapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return New(WithBaseURL(srv.URL+"/v0/"), WithRetries(2), WithBackoff(time.Millisecond))
}

func TestTopStoryIDs(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v0/topstories.json" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`[3, 1, 2]`))
	})
	ids, err := c.TopStoryIDs(context.Background())
	if err != nil || !reflect.DeepEqual(ids, []int{3, 1, 2}) {
		t.Fatalf("ids=%v err=%v", ids, err)
	}
}

func TestItemDecodesAndIgnoresUnknownFields(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v0/item/8863.json" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`{"by":"dhouston","id":8863,"kids":[8952,9224],"type":"story","brand_new_field":true}`))
	})
	it, err := c.Item(context.Background(), 8863)
	if err != nil {
		t.Fatal(err)
	}
	if it.ID != 8863 || it.By != "dhouston" || !reflect.DeepEqual(it.Kids, []int{8952, 9224}) {
		t.Fatalf("item = %+v", it)
	}
}

func TestNullBodyMeansNotFound(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("null\n")) })
	it, err := c.Item(context.Background(), 1)
	if it != nil || err != nil {
		t.Fatalf("item=%v err=%v, want nil, nil", it, err)
	}
	u, err := c.User(context.Background(), "nobody")
	if u != nil || err != nil {
		t.Fatalf("user=%v err=%v, want nil, nil", u, err)
	}
}

func TestUserPath(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v0/user/jl.json" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`{"id":"jl","karma":2937,"created":1173923446}`))
	})
	u, err := c.User(context.Background(), "jl")
	if err != nil || u.Karma != 2937 {
		t.Fatalf("user=%+v err=%v", u, err)
	}
}

func TestRetriesServerErrors(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.Write([]byte(`{"id":1}`))
	})
	it, err := c.Item(context.Background(), 1)
	if err != nil || it.ID != 1 || calls.Load() != 3 {
		t.Fatalf("item=%v err=%v calls=%d", it, err, calls.Load())
	}
}

func TestGivesUpAfterRetries(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	if _, err := c.Item(context.Background(), 1); err == nil {
		t.Fatal("expected error")
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3 (1 + 2 retries)", calls.Load())
	}
}

func TestDoesNotRetryClientErrors(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusForbidden)
	})
	if _, err := c.Item(context.Background(), 1); err == nil || calls.Load() != 1 {
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
}

func TestHonorsContextCancel(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusBadGateway) })
	c.backoff = time.Hour
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := c.Item(ctx, 1); err == nil {
		t.Fatal("expected error")
	}
	if time.Since(start) > time.Second {
		t.Fatal("did not stop on context cancel")
	}
}
