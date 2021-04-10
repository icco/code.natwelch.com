package main

import (
	"fmt"
	"net/http"
	"os"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/monitoredresource"
	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/icco/code.natwelch.com/code"
	"github.com/icco/gutil/logging"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	project = "code"
	gcpID   = "icco-cloud"
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

	if os.Getenv("ENABLE_STACKDRIVER") != "" {
		labels := &stackdriver.Labels{}
		labels.Set("app", project, "The name of the current app.")
		sd, err := stackdriver.NewExporter(stackdriver.Options{
			ProjectID:               gcpID,
			MonitoredResource:       monitoredresource.Autodetect(),
			DefaultMonitoringLabels: labels,
			DefaultTraceAttributes:  map[string]interface{}{"app": project},
		})

		if err != nil {
			log.Fatalw("failed to create the stackdriver exporter", zap.Error(err))
		}
		defer sd.Flush()

		view.RegisterExporter(sd)
		trace.RegisterExporter(sd)
		trace.ApplyConfig(trace.Config{
			DefaultSampler: trace.AlwaysSample(),
		})
	}

	db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{})
	if err != nil {
		log.Fatalw("cannot connect to database server", zap.Error(err))
	}

	if err := db.AutoMigrate(&code.Commit{}); err != nil {
		log.Fatalw("cannot migrate Commit", zap.Error(err))
	}

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(logging.Middleware(log.Desugar(), gcpID))

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

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hi."))
	})

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hi."))
	})

  r.Post("/save", func(w http.ResponseWriter, r *http.Request) {
    payload = JSON.parse(request.body.read).symbolize_keys

    code.NewCommit(payload[:user], payload[:repo], payload[:sha], nil, true)
  }

  r.Get("/data/commit.csv" , func(w http.ResponseWriter, r *http.Request) {

    data = Commit.order(:created_on).where(user: USER).where("created_on >= ?",
                                                             Chronic.parse("2009-01-01")).group(:created_on).count

    @stats = Hash.new(0)
    data.each do |row|
      @stats[row[0].strftime("%D")] += row[1]
    end

    etag "data/commit-#{Commit.maximum(:created_on)}"
    content_type "text/csv"
    erb :"commit_data.csv"
  }

 r.Get("/data/:year/weekly.csv" , func(w http.ResponseWriter, r *http.Request) {
    @year = params[:year] || Time.now.year.to_s
    logger.info "Getting data for #{@year}."

    logger.info "USER is #{USER.inspect}."
    data = Commit.order(:created_on).where(user: USER).group(:created_on).count

    @stats = Hash.new(0)
    ("01".."52").each { |week| @stats[week] = 0 }
    data.each do |row|
      if row[0].strftime("%Y") == @year
        week = row[0].strftime("%U")
        @stats[week] += row[1] if week != "00"
      end
    end

    etag "data/weekly-#{@year}-#{Commit.maximum(:created_on)}"
    content_type "text/csv"
    erb :"weekly_data.csv"
  }

	h := &ochttp.Handler{
		Handler:     r,
		Propagation: &propagation.HTTPFormat{},
	}
	if err := view.Register([]*view.View{
		ochttp.ServerRequestCountView,
		ochttp.ServerResponseCountByStatusCode,
	}...); err != nil {
		log.Fatalw("Failed to register ochttp views", zap.Error(err))
	}

	log.Fatal(http.ListenAndServe(":"+port, h))
}
