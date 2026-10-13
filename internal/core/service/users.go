package service

import (
	"context"
	"sort"
	"sync"

	"interview-process/internal/core/domain"
)

func (r *run) user(ctx context.Context, id string) (*domain.User, error) {
	release, err := r.acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	return r.client.User(ctx, id)
}

// authors returns the distinct, sorted authors across all threads.
func authors(threads []domain.Thread) []string {
	set := map[string]struct{}{}
	for _, th := range threads {
		th.Root.Walk(func(n domain.Node, _ int) {
			if n.Item != nil && n.Item.By != "" {
				set[n.Item.By] = struct{}{}
			}
		})
	}
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// fetchUsers fetches each profile once, concurrently. Failures are recorded,
// not returned, so one bad profile doesn't sink the capture.
func (r *run) fetchUsers(ctx context.Context, ids []string) (map[string]domain.User, []domain.MissingUser) {
	users := make(map[string]domain.User, len(ids))
	missing := []domain.MissingUser{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			u, err := r.user(ctx, id)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err != nil:
				missing = append(missing, domain.MissingUser{ID: id, Status: domain.StatusError, Reason: err.Error()})
			case u == nil:
				missing = append(missing, domain.MissingUser{ID: id, Status: domain.StatusNotFound})
			default:
				users[id] = *u
			}
		}()
	}
	wg.Wait()
	sort.Slice(missing, func(i, j int) bool { return missing[i].ID < missing[j].ID })
	return users, missing
}
