FROM golang:1.13 AS builder
LABEL builder=true

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./... \
 && go install -v ./... \
 && go test -short ./... \
 && go build \
    -a \
    -ldflags '-s -w -extldflags "-static"' \
    -o /bin/solace_prometheus_exporter



FROM scratch
LABEL maintainer="https://github.com/solacecommunity/solace-prometheus-exporter"

ENV SOLACE_LISTEN_ADDR="0.0.0.0:9628"
ENV SOLACE_SCRAPE_URI=http://localhost:8080
ENV SOLACE_USERNAME="admin"
ENV SOLACE_PASSWORD="admin"
ENV SOLACE_TIMEOUT="5s"
ENV SOLACE_SSL_VERIFY="false"
ENV SOLACE_REDUNDANCY="false"

EXPOSE 9628
ENTRYPOINT [ "/solace_prometheus_exporter" ]
CMD [ ]

COPY --from=builder /etc/passwd /etc/passwd

COPY --from=builder /bin/solace_prometheus_exporter /solace_prometheus_exporter
