package code

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MetricLastSync is the unix time of the last completed sync window.
	MetricLastSync = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "code_last_sync_timestamp_seconds",
		Help: "Unix timestamp of the last completed sync window.",
	})
	// MetricSyncErrors counts window fetch/upsert failures.
	MetricSyncErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "code_sync_errors_total",
		Help: "Total number of sync window errors.",
	})
	// MetricContributions is the commit total from the most recent backfill.
	MetricContributions = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "code_contributions_total",
		Help: "Commit total observed during the most recent full backfill.",
	})
	// MetricPrivateVisible is 0 when private commits are being dropped. Alert on == 0.
	MetricPrivateVisible = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "code_private_commits_visible",
		Help: "0 if GitHub reports restricted contributions the itemized query missed (token scope / profile setting); else 1.",
	})
)
