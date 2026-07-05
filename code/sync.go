package code

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SyncOptions configures the background sync loop.
type SyncOptions struct {
	User      string
	Token     string
	Interval  time.Duration
	StartYear int
}

// RunSync backfills all history once, then refreshes a trailing window on
// Interval until ctx is cancelled. Intended to run as a goroutine.
func RunSync(ctx context.Context, log *zap.SugaredLogger, db *gorm.DB, opts SyncOptions) {
	if opts.Token == "" {
		log.Warnw("GITHUB_TOKEN is empty; contribution sync disabled")
		return
	}

	log.Infow("starting contribution sync", "user", opts.User, "interval", opts.Interval, "start_year", opts.StartYear)
	backfillFrom := time.Date(opts.StartYear, 1, 1, 0, 0, 0, 0, time.UTC)
	if total := syncRange(ctx, log, db, opts, backfillFrom, time.Now().UTC()); total >= 0 {
		MetricContributions.Set(float64(total))
	}

	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Infow("contribution sync shutting down")
			return
		case <-ticker.C:
			// Trailing 13 months catches backdated / late commits.
			from := time.Now().UTC().AddDate(-1, -1, 0)
			syncRange(ctx, log, db, opts, from, time.Now().UTC())
		}
	}
}

// syncRange fetches each monthly window in [from,to) and upserts it. Returns
// the summed commit count, or -1 if the range was aborted before any window.
func syncRange(ctx context.Context, log *zap.SugaredLogger, db *gorm.DB, opts SyncOptions, from, to time.Time) int {
	total := 0
	for _, w := range monthlyWindows(from, to) {
		select {
		case <-ctx.Done():
			return -1
		default:
		}

		days, err := FetchCommitContributions(ctx, log, opts.Token, opts.User, w[0], w[1])
		if err != nil {
			MetricSyncErrors.Inc()
			log.Errorw("fetch window failed", "from", w[0], "to", w[1], zap.Error(err))
			continue
		}

		rows := make([]Contribution, 0, len(days))
		for d, c := range days {
			dt, perr := time.Parse("2006-01-02", d)
			if perr != nil {
				continue
			}
			rows = append(rows, Contribution{User: opts.User, Date: dt, Count: c})
			total += c
		}

		if err := Upsert(ctx, db, rows); err != nil {
			MetricSyncErrors.Inc()
			log.Errorw("upsert window failed", "from", w[0], "to", w[1], zap.Error(err))
			continue
		}
		MetricLastSync.SetToCurrentTime()
	}
	log.Infow("sync range complete", "from", from, "to", to, "commits", total)
	return total
}

// monthlyWindows splits [from,to) into <=1-month [start,end) pairs so the
// GraphQL contributions connection never needs pagination. Windows are
// aligned to calendar-month boundaries (rather than "start+1 month"), so a
// range spanning parts of N distinct calendar months yields N windows.
func monthlyWindows(from, to time.Time) [][2]time.Time {
	var out [][2]time.Time
	for start := from; start.Before(to); {
		end := time.Date(start.Year(), start.Month()+1, 1, 0, 0, 0, 0, start.Location())
		if end.After(to) {
			end = to
		}
		out = append(out, [2]time.Time{start, end})
		start = end
	}
	return out
}
