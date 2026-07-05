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
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/icco/code.natwelch.com/code"
	"github.com/icco/code.natwelch.com/static"
	"github.com/icco/gutil/etag"
	"github.com/icco/gutil/logging"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"moul.io/zapgorm2"
)

const (
	service = "code"
	project = "icco-cloud"
)

var log = logging.Must(logging.NewLogger(service))

func main() {
	port := envOr("PORT", "8080")
	user := envOr("GITHUB_USER", "icco")

	interval, err := time.ParseDuration(envOr("SYNC_INTERVAL", "6h"))
	if err != nil {
		log.Fatalw("invalid SYNC_INTERVAL", zap.Error(err))
	}
	startYear, err := strconv.Atoi(envOr("SYNC_START_YEAR", "2008"))
	if err != nil {
		log.Fatalw("invalid SYNC_START_YEAR", zap.Error(err))
	}

	log.Infow("Starting up", "host", fmt.Sprintf("http://localhost:%s", port))

	zgl := zapgorm2.New(log.Desugar())
	zgl.SetAsDefault()
	db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{Logger: zgl})
	if err != nil {
		log.Fatalw("cannot connect to database server", zap.Error(err))
	}
	if err := db.AutoMigrate(&code.Contribution{}); err != nil {
		log.Fatalw("cannot migrate Contribution", zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go code.RunSync(ctx, log, db, code.SyncOptions{
		User:      user,
		Token:     os.Getenv("GITHUB_TOKEN"),
		Interval:  interval,
		StartYear: startYear,
	})

	srv := &http.Server{Addr: ":" + port, Handler: router(db, user)}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalw("server error", zap.Error(err))
		}
	}()

	<-ctx.Done()
	stop()
	log.Infow("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Errorw("graceful shutdown failed", zap.Error(err))
	}
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
	r.Use(middleware.RealIP)
	r.Use(logging.Middleware(log.Desugar(), project))

	crs := cors.New(cors.Options{
		AllowCredentials: true,
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
	return r
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
