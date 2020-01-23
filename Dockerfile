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
    -o /bin/solace_exporter



FROM scratch
LABEL maintainer="Daniel Brunold <dab@gmx.ch>"

ENV SOLACE_WEB_LISTEN_ADDRESS="0.0.0.0:9628"
ENV SOLACE_SCRAPE_TIMEOUT="5s"
ENV SOLACE_SSL_VERIFY="false"
ENV SOLACE_INCLUDE_REDUNDANCY="true"
ENV SOLACE_USER="admin"
ENV SOLACE_PASSWORD="admin"

EXPOSE 9628
ENTRYPOINT [ "/solace_exporter" ]
CMD [ ]

COPY --from=builder /etc/passwd /etc/passwd

COPY --from=builder /bin/solace_exporter /solace_exporter
