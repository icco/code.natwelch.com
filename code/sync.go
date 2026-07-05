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

// RunSync fills any fully-missing history years, refreshes recent data, then
// refreshes a trailing window on Interval until ctx is cancelled. Intended to
// run as a goroutine.
func RunSync(ctx context.Context, log *zap.SugaredLogger, db *gorm.DB, opts SyncOptions) {
	if opts.Token == "" {
		log.Warnw("GITHUB_TOKEN is empty; contribution sync disabled")
		return
	}

	log.Infow("starting contribution sync", "user", opts.User, "interval", opts.Interval, "start_year", opts.StartYear)
	backfill(ctx, log, db, opts)
	refresh(ctx, log, db, opts) // ensure recent data is current on boot

	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Infow("contribution sync shutting down")
			return
		case <-ticker.C:
			refresh(ctx, log, db, opts)
		}
	}
}

// backfill fetches only calendar years that have no data yet, so a restart
// doesn't re-fetch all of history (~200 GraphQL calls) every boot. The
// periodic refresh keeps recent years current, and a fully-missing year
// (e.g. one abandoned by an earlier rate-limited backfill) still gets healed.
func backfill(ctx context.Context, log *zap.SugaredLogger, db *gorm.DB, opts SyncOptions) {
	now := time.Now().UTC()
	have, err := YearsWithData(ctx, db, opts.User)
	if err != nil {
		log.Errorw("could not read existing years; backfilling all", zap.Error(err))
		have = nil
	}

	years := missingYears(have, opts.StartYear, now.Year())
	if len(years) == 0 {
		log.Infow("backfill: all years present, skipping")
		return
	}

	log.Infow("backfill: fetching missing years", "years", years)
	for _, y := range years {
		select {
		case <-ctx.Done():
			return
		default:
		}
		from := time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC)
		to := from.AddDate(1, 0, 0)
		if to.After(now) {
			to = now
		}
		syncRange(ctx, log, db, opts, from, to)
	}
	updateTotalMetric(ctx, log, db, opts.User)
}

// refresh re-syncs the trailing 13 months to catch new, backdated, and late commits.
func refresh(ctx context.Context, log *zap.SugaredLogger, db *gorm.DB, opts SyncOptions) {
	from := time.Now().UTC().AddDate(-1, -1, 0)
	syncRange(ctx, log, db, opts, from, time.Now().UTC())
	updateTotalMetric(ctx, log, db, opts.User)
}

// missingYears returns the [start, end] years absent from have, ascending.
func missingYears(have map[int]bool, start, end int) []int {
	var out []int
	for y := start; y <= end; y++ {
		if !have[y] {
			out = append(out, y)
		}
	}
	return out
}

func updateTotalMetric(ctx context.Context, log *zap.SugaredLogger, db *gorm.DB, user string) {
	total, err := TotalCount(ctx, db, user)
	if err != nil {
		log.Errorw("could not read contribution total", zap.Error(err))
		return
	}
	MetricContributions.Set(float64(total))
}

// syncRange fetches each monthly window in [from,to) and upserts it, retrying
// transient GitHub failures with backoff. It returns early on ctx cancellation.
func syncRange(ctx context.Context, log *zap.SugaredLogger, db *gorm.DB, opts SyncOptions, from, to time.Time) {
	for _, w := range monthlyWindows(from, to) {
		select {
		case <-ctx.Done():
			return
		default:
		}

		days, err := fetchWithRetry(ctx, log, opts, w[0], w[1])
		if err != nil {
			MetricSyncErrors.Inc()
			log.Errorw("fetch window failed after retries", "from", w[0], "to", w[1], zap.Error(err))
			continue
		}

		rows := make([]Contribution, 0, len(days))
		for d, c := range days {
			dt, perr := time.Parse("2006-01-02", d)
			if perr != nil {
				continue
			}
			rows = append(rows, Contribution{User: opts.User, Date: dt, Count: c})
		}

		if err := Upsert(ctx, db, rows); err != nil {
			MetricSyncErrors.Inc()
			log.Errorw("upsert window failed", "from", w[0], "to", w[1], zap.Error(err))
			continue
		}
		MetricLastSync.SetToCurrentTime()
	}
	log.Infow("sync range complete", "from", from, "to", to)
}

// fetchWithRetry retries a window fetch up to 3 times with linear backoff so a
// transient error or rate-limit doesn't leave a permanent hole in the data.
func fetchWithRetry(ctx context.Context, log *zap.SugaredLogger, opts SyncOptions, from, to time.Time) (map[string]int, error) {
	var lastErr error
	for attempt := range 3 {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 5 * time.Second):
			}
		}
		days, err := FetchCommitContributions(ctx, log, opts.Token, opts.User, from, to)
		if err == nil {
			return days, nil
		}
		lastErr = err
		log.Warnw("fetch window failed; retrying", "from", from, "to", to, "attempt", attempt+1, zap.Error(err))
	}
	return nil, lastErr
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
