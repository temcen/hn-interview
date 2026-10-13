// Package service implements the capture use cases behind ports.Facade.
package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"interview-process/internal/core/domain"
	"interview-process/internal/core/ports"
)

// Service implements ports.Facade.
type Service struct {
	client  ports.HNClient
	archive ports.ArchiveWriter
	now     func() time.Time
}

var _ ports.Facade = (*Service)(nil)

func New(client ports.HNClient, archive ports.ArchiveWriter) *Service {
	return &Service{client: client, archive: archive, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) CaptureTop(ctx context.Context, n int, opts ports.CaptureOptions) (domain.Capture, error) {
	if n < 1 {
		return domain.Capture{}, errors.New("number of stories must be at least 1")
	}
	r := newRun(s.client, opts.Concurrency)
	ids, err := r.topStoryIDs(ctx)
	if err != nil {
		return domain.Capture{}, fmt.Errorf("fetch top stories: %w", err)
	}
	return s.capture(ctx, r, ids[:min(n, len(ids))], opts.FullProfiles), nil
}

func (s *Service) CaptureStory(ctx context.Context, id int, opts ports.CaptureOptions) (domain.Capture, error) {
	return s.capture(ctx, newRun(s.client, opts.Concurrency), []int{id}, opts.FullProfiles), nil
}

func (s *Service) GetUser(ctx context.Context, id string) (domain.User, error) {
	u, err := s.client.User(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	if u == nil {
		return domain.User{}, fmt.Errorf("user %q: %w", id, domain.ErrNotFound)
	}
	u.SubmittedCount = len(u.Submitted)
	return *u, nil
}

func (s *Service) Save(ctx context.Context, c domain.Capture, dest string) (string, error) {
	return s.archive.Write(ctx, c, dest)
}

// capture builds every story thread concurrently, then fetches the profile
// of every author found in them.
func (s *Service) capture(ctx context.Context, r *run, ids []int, fullProfiles bool) domain.Capture {
	c := domain.Capture{CapturedAt: s.now(), Stories: make([]domain.Thread, len(ids))}
	var wg sync.WaitGroup
	for i, id := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Stories[i] = r.buildThread(ctx, id)
		}()
	}
	wg.Wait()
	c.Users, c.MissingUsers = r.fetchUsers(ctx, authors(c.Stories), fullProfiles)
	return c
}

func (r *run) topStoryIDs(ctx context.Context) ([]int, error) {
	release, err := r.acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	return r.client.TopStoryIDs(ctx)
}
