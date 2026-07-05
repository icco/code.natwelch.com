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
	// MetricPrivateVisible is 1 when the most recent window surfaced private
	// commits (or had no restricted activity), 0 when GitHub reported restricted
	// (private) contributions the itemized query missed — a sign the token can't
	// read private repos. Alert on == 0.
	MetricPrivateVisible = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "code_private_commits_visible",
		Help: "1 if private commits are surfaced (or no restricted activity); 0 if GitHub reports restricted contributions the itemized query missed (check token scope / profile setting).",
	})
)
