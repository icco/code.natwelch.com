module github.com/icco/code.natwelch.com

go 1.16

require (
	contrib.go.opencensus.io/exporter/stackdriver v0.13.8
	github.com/aws/aws-sdk-go v1.40.7 // indirect
	github.com/go-chi/chi/v5 v5.0.7
	github.com/go-chi/cors v1.2.0
	github.com/google/go-github/v37 v37.0.0
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/icco/gutil v0.0.0-20220115163816-b7b82159b0b6
	github.com/jackc/pgx/v4 v4.13.0 // indirect
	go.opencensus.io v0.23.0
	go.opentelemetry.io/contrib v0.21.0 // indirect
	go.opentelemetry.io/otel/oteltest v1.0.0-RC1 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/zap v1.20.0
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	gorm.io/driver/postgres v1.1.0
	gorm.io/gorm v1.21.12
	moul.io/zapgorm2 v1.1.0
)
