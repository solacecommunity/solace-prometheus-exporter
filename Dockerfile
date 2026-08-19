FROM golang:1.27rc3-alpine AS builder
LABEL builder=true

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./... \
 && go install -v ./... \
 && go test -short ./...

# Declared after the dependency/test layer on purpose: COMMIT changes on every commit, and an ARG referenced by a
# RUN is part of that layer's cache key -- declaring it earlier would re-run the whole test suite for a metadata-only
# rebuild. The defaults keep a plain `docker build` honest; only the pipeline passes real values.
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# internal/version backs the solace_exporter_build_info metric; prometheus/common/version backs the "Build context"
# startup log line. Both are fed from the same three build args so they can never disagree.
RUN LDFLAGS="-s -w -extldflags '-static'"; \
    LDFLAGS="$LDFLAGS -X solace_exporter/internal/version.Version=${VERSION}"; \
    LDFLAGS="$LDFLAGS -X solace_exporter/internal/version.Commit=${COMMIT}"; \
    LDFLAGS="$LDFLAGS -X solace_exporter/internal/version.BuildDate=${BUILD_DATE}"; \
    LDFLAGS="$LDFLAGS -X github.com/prometheus/common/version.Version=${VERSION}"; \
    LDFLAGS="$LDFLAGS -X github.com/prometheus/common/version.Revision=${COMMIT}"; \
    LDFLAGS="$LDFLAGS -X github.com/prometheus/common/version.BuildDate=${BUILD_DATE}"; \
    go build \
    -a \
    -ldflags "$LDFLAGS" \
    -o /bin/solace_prometheus_exporter \
    ./cmd/solace-prometheus-exporter

# Install ca-certificates
RUN apk add --no-cache ca-certificates


FROM scratch
LABEL maintainer="https://github.com/solacecommunity/solace-prometheus-exporter"

EXPOSE 9628
ENTRYPOINT [ "/solace_prometheus_exporter", "--config-file=/etc/solace/solace_prometheus_exporter.ini" ]
CMD [ ]

COPY configs/solace_prometheus_exporter.ini /etc/solace/solace_prometheus_exporter.ini

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --from=builder /bin/solace_prometheus_exporter /solace_prometheus_exporter