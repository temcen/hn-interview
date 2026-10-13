package service

import (
	"context"
	"sync"

	"interview-process/internal/core/domain"
	"interview-process/internal/core/ports"
)

// run holds the per-capture state: the client and a semaphore that caps
// in-flight upstream calls.
type run struct {
	client ports.HNClient
	sem    chan struct{}
}

func newRun(client ports.HNClient, concurrency int) *run {
	if concurrency < 1 {
		concurrency = 1
	}
	return &run{client: client, sem: make(chan struct{}, concurrency)}
}

// acquire blocks for a slot. The slot is held only for the duration of one
// upstream call, never while waiting on children, so recursion can't deadlock.
func (r *run) acquire(ctx context.Context) (release func(), err error) {
	select {
	case r.sem <- struct{}{}:
		return func() { <-r.sem }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (r *run) item(ctx context.Context, id int) (*domain.Item, error) {
	release, err := r.acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	return r.client.Item(ctx, id)
}

func (r *run) buildThread(ctx context.Context, id int) domain.Thread {
	root := r.buildNode(ctx, id, 0)
	return summarize(root)
}

func (r *run) buildNode(ctx context.Context, id, depth int) domain.Node {
	n := domain.Node{ID: id, Depth: depth}
	item, err := r.item(ctx, id)
	switch {
	case err != nil:
		n.Status, n.Reason = domain.StatusError, err.Error()
		return n
	case item == nil:
		n.Status = domain.StatusNotFound
		return n
	case item.Deleted:
		n.Status = domain.StatusDeleted
	case item.Dead:
		n.Status = domain.StatusDead
	default:
		n.Status = domain.StatusOK
	}
	n.Item = item

	if len(item.Kids) > 0 {
		n.Children = make([]domain.Node, len(item.Kids))
		var wg sync.WaitGroup
		for i, kid := range item.Kids {
			wg.Add(1)
			go func() {
				defer wg.Done()
				n.Children[i] = r.buildNode(ctx, kid, depth+1)
			}()
		}
		wg.Wait()
	}
	return n
}

// summarize derives the missing list and stats from a finished tree.
func summarize(root domain.Node) domain.Thread {
	th := domain.Thread{Root: root, Missing: []domain.MissingItem{}}
	if root.Item != nil {
		th.Stats.ExpectedDescendants = root.Item.Descendants
	}
	authors := map[string]struct{}{}
	root.Walk(func(n domain.Node, parentID int) {
		th.Stats.MaxDepth = max(th.Stats.MaxDepth, n.Depth)
		if n.Item != nil && n.Item.By != "" {
			authors[n.Item.By] = struct{}{}
		}
		if n.Status != domain.StatusOK {
			th.Missing = append(th.Missing, domain.MissingItem{
				ID: n.ID, ParentID: parentID, Depth: n.Depth, Status: n.Status, Reason: n.Reason,
			})
		} else if n.Depth > 0 {
			th.Stats.CapturedComments++
		}
	})
	th.Stats.MissingCount = len(th.Missing)
	th.Stats.Authors = len(authors)
	return th
}
