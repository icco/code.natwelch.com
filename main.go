package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.icco.me/code.natwelch.com/code"
	"go.icco.me/code.natwelch.com/static"
	"go.icco.me/gutil/etag"
	"go.icco.me/gutil/logging"
	"go.icco.me/gutil/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormtracing "gorm.io/plugin/opentelemetry/tracing"
	"moul.io/zapgorm2"
)

const service = "code"

var log = logging.Must(logging.NewLogger(service))

func main() {
	if err := run(); err != nil {
		log.Fatalw("fatal", zap.Error(err))
	}
}

func run() error {
	port := envOr("PORT", "8080")
	user := envOr("GITHUB_USER", "icco")

	interval, err := time.ParseDuration(envOr("SYNC_INTERVAL", "6h"))
	if err != nil {
		return fmt.Errorf("invalid SYNC_INTERVAL: %w", err)
	}
	startYear, err := strconv.Atoi(envOr("SYNC_START_YEAR", "2008"))
	if err != nil {
		return fmt.Errorf("invalid SYNC_START_YEAR: %w", err)
	}

	log.Infow("Starting up", "host", fmt.Sprintf("http://localhost:%s", port))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdown, err := tracing.Init(ctx, service)
	if err != nil {
		return fmt.Errorf("tracing init: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(shutdownCtx); err != nil {
			log.Warnw("tracing shutdown", zap.Error(err))
		}
	}()

	zgl := zapgorm2.New(log.Desugar())
	zgl.SetAsDefault()
	db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{Logger: zgl})
	if err != nil {
		return fmt.Errorf("cannot connect to database server: %w", err)
	}
	if err := db.Use(gormtracing.NewPlugin(gormtracing.WithoutMetrics())); err != nil {
		return fmt.Errorf("cannot register tracing plugin: %w", err)
	}
	if err := db.AutoMigrate(&code.Contribution{}); err != nil {
		return fmt.Errorf("cannot migrate Contribution: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		code.RunSync(ctx, log, db, code.SyncOptions{
			User:      user,
			Token:     os.Getenv("GITHUB_TOKEN"),
			Interval:  interval,
			StartYear: startYear,
		})
	}()

	srv := &http.Server{Addr: ":" + port, Handler: router(db, user)}
	serveErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serveErr <- err
		}
		close(serveErr)
	}()

	select {
	case <-ctx.Done():
		stop()
		log.Infow("shutdown signal received")
	case err := <-serveErr:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Errorw("graceful shutdown failed", zap.Error(err))
	}
	wg.Wait() // let the sync goroutine observe cancellation and finish cleanly
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	return nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func router(db *gorm.DB, user string) http.Handler {
	r := chi.NewRouter()
	r.Use(etag.Handler(false))
	r.Use(logging.Middleware(log.Desugar()))

	crs := cors.New(cors.Options{
		// Public, unauthenticated data: wildcard origin is only valid without credentials.
		AllowCredentials: false,
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		MaxAge:           300,
	})
	r.Use(crs.Handler)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hi."))
	})
	r.Handle("/metrics", promhttp.Handler())

	r.Get("/data/contributions.csv", func(w http.ResponseWriter, r *http.Request) {
		data, err := code.ForAllTime(r.Context(), db, user)
		if err != nil {
			log.Errorw("could not get contributions", zap.Error(err))
			http.Error(w, "could not get contributions", http.StatusInternalServerError)
			return
		}
		writeCSV(w, "date", data)
	})

	r.Get("/data/{year}/weekly.csv", func(w http.ResponseWriter, r *http.Request) {
		year, err := strconv.Atoi(chi.URLParam(r, "year"))
		if err != nil {
			http.Error(w, "could not parse year", http.StatusBadRequest)
			return
		}
		data, err := code.ForYear(r.Context(), db, user, year)
		if err != nil {
			log.Errorw("could not get weekly contributions", zap.Error(err))
			http.Error(w, "could not get weekly contributions", http.StatusInternalServerError)
			return
		}
		writeCSV(w, "week", data)
	})

	r.Mount("/", http.FileServer(http.FS(static.Assets)))
	return otelhttp.NewHandler(r, service,
		otelhttp.WithFilter(func(req *http.Request) bool {
			return req.URL.Path != "/metrics" && req.URL.Path != "/healthz"
		}),
	)
}

// writeCSV emits a sorted "<keyCol>,count" CSV.
func writeCSV(w http.ResponseWriter, keyCol string, data map[string]int64) {
	w.Header().Set("content-type", "text/csv")
	records := make([][]string, 0, len(data))
	for k, v := range data {
		records = append(records, []string{k, strconv.FormatInt(v, 10)})
	}
	sort.Slice(records, func(i, j int) bool { return records[i][0] < records[j][0] })

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{keyCol, "count"})
	if err := cw.WriteAll(records); err != nil {
		log.Errorw("error writing csv", zap.Error(err))
	}
}
