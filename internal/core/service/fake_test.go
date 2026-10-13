package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"interview-process/internal/core/domain"
)

// fakeClient is an in-memory ports.HNClient. Ids absent from items return
// (nil, nil), mirroring the API's null response.
type fakeClient struct {
	top       []int
	topErr    error
	items     map[int]*domain.Item
	itemErrs  map[int]error
	users     map[string]*domain.User
	userErrs  map[string]error
	delay     time.Duration
	inFlight  atomic.Int32
	peak      atomic.Int32
	mu        sync.Mutex
	userCalls map[string]int
}

func (f *fakeClient) track() func() {
	n := f.inFlight.Add(1)
	for {
		p := f.peak.Load()
		if n <= p || f.peak.CompareAndSwap(p, n) {
			break
		}
	}
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	return func() { f.inFlight.Add(-1) }
}

func (f *fakeClient) TopStoryIDs(ctx context.Context) ([]int, error) {
	defer f.track()()
	return f.top, f.topErr
}

func (f *fakeClient) Item(ctx context.Context, id int) (*domain.Item, error) {
	defer f.track()()
	if err := f.itemErrs[id]; err != nil {
		return nil, err
	}
	return f.items[id], nil
}

func (f *fakeClient) User(ctx context.Context, id string) (*domain.User, error) {
	defer f.track()()
	f.mu.Lock()
	if f.userCalls == nil {
		f.userCalls = map[string]int{}
	}
	f.userCalls[id]++
	f.mu.Unlock()
	if err := f.userErrs[id]; err != nil {
		return nil, err
	}
	return f.users[id], nil
}
