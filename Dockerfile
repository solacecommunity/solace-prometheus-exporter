FROM golang:1.20.7 AS builder
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

EXPOSE 9628
ENTRYPOINT [ "/solace_prometheus_exporter", "--config-file=/etc/solace/solace_prometheus_exporter.ini" ]
CMD [ ]

COPY docker/solace_prometheus_exporter.ini /etc/solace/solace_prometheus_exporter.ini

COPY --from=builder /etc/passwd /etc/passwd

COPY --from=builder /bin/solace_prometheus_exporter /solace_prometheus_exporter
