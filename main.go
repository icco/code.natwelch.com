package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/icco/code.natwelch.com/code"
	"github.com/icco/code.natwelch.com/static"
	"github.com/icco/gutil/etag"
	"github.com/icco/gutil/logging"
	"github.com/icco/gutil/otel"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"moul.io/zapgorm2"
)

const (
	project = "code"
	gcpID   = "icco-cloud"
	user    = "icco"
)

var (
	log = logging.Must(logging.NewLogger(project))
)

func main() {
	port := "8080"
	if fromEnv := os.Getenv("PORT"); fromEnv != "" {
		port = fromEnv
	}
	log.Infow("Starting up", "host", fmt.Sprintf("http://localhost:%s", port))

	if err := otel.Init(ctx, log, project, service); err != nil {
		log.Errorw("could not init opentelemetry", zap.Error(err))
	}

	zgl := zapgorm2.New(log.Desugar())
	zgl.SetAsDefault()
	db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{
		Logger: zgl,
	})
	if err != nil {
		log.Fatalw("cannot connect to database server", zap.Error(err))
	}

	if err := db.AutoMigrate(&code.Commit{}); err != nil {
		log.Fatalw("cannot migrate Commit", zap.Error(err))
	}

	r := chi.NewRouter()
	r.Use(etag.Handler(false))
	r.Use(middleware.RealIP)
	r.Use(logging.Middleware(log.Desugar(), gcpID))
	r.Use(otel.Middleware)

	crs := cors.New(cors.Options{
		AllowCredentials:   true,
		OptionsPassthrough: false,
		AllowedOrigins:     []string{"*"},
		AllowedMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:     []string{"Link"},
		MaxAge:             300, // Maximum value not ignored by any of major browsers
	})
	r.Use(crs.Handler)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hi."))
	})

	r.Mount("/", http.FileServer(http.FS(static.Assets)))

	r.Post("/save", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		cmt := &code.Commit{}

		if err := json.NewDecoder(r.Body).Decode(cmt); err != nil {
			log.Errorw("could not decode json", zap.Error(err))
			http.Error(w, "could not decode json", http.StatusInternalServerError)
			return
		}

		gh := code.GithubClient(ctx, os.Getenv("GITHUB_TOKEN"))
		if err := cmt.CheckAndSave(ctx, gh, db); err != nil {
			log.Errorw("could not save", zap.Error(err))
			http.Error(w, "could not save", http.StatusInternalServerError)
			return
		}
	})

	r.Get("/data/commit.csv", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/csv")

		data, err := code.CommitsForAllTime(r.Context(), db, user)
		if err != nil {
			log.Errorw("could not get commits", zap.Error(err))
			http.Error(w, "could not get commits", http.StatusInternalServerError)
			return
		}

		var records [][]string
		for d, v := range data {
			records = append(records, []string{d, strconv.FormatInt(v, 10)})
		}

		sort.Slice(records, func(i, j int) bool {
			return records[i][0] < records[j][0]
		})

		csvWr := csv.NewWriter(w)
		if err := csvWr.Write([]string{"date", "commits"}); err != nil {
			log.Fatalw("error writing header to csv", zap.Error(err))
		}
		if err := csvWr.WriteAll(records); err != nil {
			log.Errorw("error writing record to csv", zap.Error(err))
			http.Error(w, "error writing record to csv", http.StatusInternalServerError)
			return
		}

		if err := csvWr.Error(); err != nil {
			log.Fatalw("csv error", zap.Error(err))
			http.Error(w, "csv error", http.StatusInternalServerError)
			return
		}
	})

	r.Get("/data/{year}/weekly.csv", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/csv")

		yearStr := chi.URLParam(r, "year")
		year, err := strconv.Atoi(yearStr)
		if err != nil {
			log.Errorw("could not parse year", "year", yearStr, zap.Error(err))
			http.Error(w, "could not parse year", http.StatusInternalServerError)
			return
		}

		log.Infow("getting data", "year", year, "user", user)
		data, err := code.CommitsForYear(r.Context(), db, user, year)
		if err != nil {
			log.Errorw("could not get weekly commits", zap.Error(err))
			http.Error(w, "could not get weekly commits", http.StatusInternalServerError)
			return
		}

		var records [][]string
		for d, v := range data {
			records = append(records, []string{d, strconv.FormatInt(v, 10)})
		}

		sort.Slice(records, func(i, j int) bool {
			return records[i][0] < records[j][0]
		})

		csvWr := csv.NewWriter(w)
		if err := csvWr.Write([]string{"week", "commits"}); err != nil {
			log.Fatalw("error writing header to csv", zap.Error(err))
		}
		if err := csvWr.WriteAll(records); err != nil {
			log.Errorw("error writing record to csv", zap.Error(err))
			http.Error(w, "error writing record to csv", http.StatusInternalServerError)
			return
		}

		if err := csvWr.Error(); err != nil {
			log.Fatalw("csv error", zap.Error(err))
			http.Error(w, "csv error", http.StatusInternalServerError)
			return
		}
	})

	log.Fatal(http.ListenAndServe(":"+port, r))
}
