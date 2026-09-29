
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shreyaabaranwal/pr-herder/internal/githubmcp"
	"github.com/shreyaabaranwal/pr-herder/internal/store"
)


type Publisher interface {
	PublishBlocks(ctx context.Context, blocks []map[string]any) error
}

type Scheduler struct {
	store     *store.Store
	github    *githubmcp.Client
	publisher Publisher
	log       *slog.Logger

	digestHourLocal    int
	staleDaysThreshold int
	quietStart         int
	quietEnd           int


	lastDigestDate string
}

func NewScheduler(
	s *store.Store,
	gh *githubmcp.Client,
	pub Publisher,
	log *slog.Logger,
	digestHourLocal, staleDaysThreshold, quietStart, quietEnd int,
) *Scheduler {
	return &Scheduler{
		store:              s,
		github:             gh,
		publisher:          pub,
		log:                log,
		digestHourLocal:    digestHourLocal,
		staleDaysThreshold: staleDaysThreshold,
		quietStart:         quietStart,
		quietEnd:           quietEnd,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	s.maybeRunDigest(ctx) 

	for {
		select {
		case <-ctx.Done():
			s.log.Info("scheduler stopping")
			return
		case <-ticker.C:
			s.maybeRunDigest(ctx)
		}
	}
}

func (s *Scheduler) maybeRunDigest(ctx context.Context) {
	now := time.Now()
	today := now.Format("2006-01-02")

	if now.Hour() != s.digestHourLocal {
		return
	}
	if today == s.lastDigestDate {
		return
	}
	if IsQuietHours(now, s.quietStart, s.quietEnd) {
		s.log.Warn("digest hour falls inside quiet hours, skipping -- check DIGEST_HOUR_LOCAL vs QUIET_HOURS config",
			"digest_hour", s.digestHourLocal, "quiet_start", s.quietStart, "quiet_end", s.quietEnd)
		s.lastDigestDate = today
		return
	}

	s.lastDigestDate = today
	if err := s.runDigest(ctx); err != nil {
		s.log.Error("digest run failed", "err", err)
	}
}

func (s *Scheduler) runDigest(ctx context.Context) error {
	repos, err := s.store.GetDistinctRepos(ctx)
	if err != nil {
		return fmt.Errorf("get distinct repos: %w", err)
	}

	var allStale []StalePR
	for _, repo := range repos {
		prs, err := s.github.ListOpenPullRequests(ctx, repo.Owner, repo.Name)
		if err != nil {
			s.log.Warn("failed to list open PRs, skipping repo",
				"repo", repo.Owner+"/"+repo.Name, "err", err)
			continue
		}
		allStale = append(allStale, FilterStale(prs, time.Now(), s.staleDaysThreshold)...)
	}

	blocks := BuildDigestBlocks(allStale)
	if blocks == nil {
		s.log.Info("digest run: no stale PRs found, skipping publish")
		return nil
	}

	if err := s.publisher.PublishBlocks(ctx, blocks); err != nil {
		return fmt.Errorf("publish digest: %w", err)
	}

	s.log.Info("digest published", "stale_count", len(allStale))
	return nil
}
