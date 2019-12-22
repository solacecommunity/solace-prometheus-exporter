# solace_exporter
Prometheus Exporter for Solace PubSub+<br/>
Experimental Status<br/>

## Features
The exporter is written in go, based on the Solace Legacy SEMP protocol. It has implemented the following endpoints:
<pre><code>
.../            HTML page with endpoints
.../metrics     Golang and standard Prometheus stuff
.../solace_std  Solace metrics for System and VPN levels
.../solace_det  Solace metrics for all individual Clients and Queues
                (Can degrade system performance, test before use it in prod)
</code></pre>
## Usage
<pre><code>
./solace_exporter -h
usage: solace_exporter [&lt;flags&gt;]

Flags:
  -h, --help               Show context-sensitive help.
      --web.listen-address=":9101"
                           Address to listen on for web interface and telemetry.
      --sol.uri="http://localhost:8080"
                           Base URI on which to scrape Solace.
      --sol.user="admin"   Username for http requests to Solace broker.
      --sol.pass="admin"   Password for http requests to Solace broker.
      --sol.timeout=5s     Timeout for trying to get stats from Solace.
      --sol.sslv           Flag that enables SSL certificate verification for the scrape URI
      --sol.reset          Flag that enables resetting system/vpn/client/queue stats in Solace
      --sol.rates          Flag that enables scrape of rate metrics
      --log.level=info     Only log messages with the given severity or above. One of: [debug, info, warn, error]
      --log.format=logfmt  Output format of log messages. One of: [logfmt, json]
      </code></pre>
## Build
### Default Build
<pre><code>cd &lt;solace-exporter-directory&gt;
go build
</code></pre>
### Static Build for Linux amd64 to run in Docker
<pre><code>cd &lt;solace-exporter-directory&gt;
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-extldflags "-static"'
</code></pre>
### Create Docker Image
A sample Dockerfile based on amd64/busybox is included in the repository.
<pre><code>cd &lt;solace-exporter-directory&gt;
docker build --tag solace_exporter .
</code></pre>
### Create Docker Container
The exporter can be configured by environment variables to facilitate running in Docker.
<pre><code>cd &lt;solace-exporter-directory&gt;
docker create \
 -p 9101:9101 \
 --env SOLACE_WEB_LISTEN_ADDRESS=":9101" \
 --env SOLACE_SCRAPE_URI="http://192.168.110.100:8080" \
 --env SOLACE_USER="admin" \
 --env SOLACE_PASSWORD="admin" \
 --env SOLACE_SCRAPE_TIMEOUT="5s" \
 --env SOLACE_SSL_VERIFY="false" \
 --env SOLACE_RESET_STATS="false" \
 --env SOLACE_INCLUDE_RATES="true" \
 --name solace_exporter \
 solace_exporter
</code></pre>

## Bonus Material
The sub directory **testfiles** contains some sample curl commands and their outputs. This is just fyi and not needed for building.

