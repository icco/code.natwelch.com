FROM golang:1.19-alpine as builder

ENV GOPROXY="https://proxy.golang.org"
ENV GO111MODULE="on"
ENV NAT_ENV="production"
ENV PRANA_LOG_FORMAT="json"

RUN apk add --no-cache git make g++ ca-certificates

WORKDIR /go/src/github.com/icco/code.natwelch.com
COPY . .

RUN go build -o /go/bin/code .
CMD /go/bin/code
