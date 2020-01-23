
# solace_exporter

Prometheus Exporter for Solace PubSub+<br/>
Status: Prototype, tested only against PubSub+ software broker (VMR)<br/>

## Features

The exporter is written in go, based on the Solace Legacy SEMP protocol.<br/>
It implements the following endpoints:
<pre><code>
.../            HTML page with endpoints
.../metrics     Golang and standard Prometheus stuff
.../solace-std  Solace metrics for System and VPN levels
.../solace-det  Solace metrics for all individual Clients and Queues
                (Can degrade system performance, test before use it in prod)
</code></pre>

## Usage

<pre><code>
./solace_exporter -h
usage: solace_exporter [&lt;flags&gt;]

Flags:
  -h, --help               Show context-sensitive help.
      --config-file=./solace_exporter.ini
                           All options can be set via .ini file. See solace_exporter.ini for sample.
      --web.listen-address=":9628"
                           Address to listen on for web interface and telemetry.
      --sol.uri="http://localhost:8080"
                           Base URI on which to scrape Solace.
      --sol.user="admin"   Username for http requests to Solace broker.
      --sol.pass="admin"   Security: DOEST NOT EXIST, can only be set via config or env.
      --sol.timeout=5s     Timeout for trying to get stats from Solace.
      --sol.sslv           Flag that enables SSL certificate verification for the scrape URI
      --sol.redundancy     Flag that enables scrape of redundancy metrics. Should be used for HA tripples.
      --log.level=info     Only log messages with the given severity or above. One of: [debug, info, warn, error]
      --log.format=logfmt  Output format of log messages. One of: [logfmt, json]
      </code></pre>

### Security

For Docker you should prefer the ENV (see below)
Else please place your options in an config file and call the exporter with:

<pre><code>
cat /path/to/config/file.ini
[sol]
uri=http://localhost:8080
username=admin
password=admin
timeout=6s
sslVerify=false
scrapeRedundancy=false
...

solace_exporter @/path/to/config/file
</code></pre>

## Build

### Default Build to run without Docker
<pre><code>cd &lt;some-directory&gt;/solace_exporter
go build
</code></pre>

## Docker

### Build Docker Image

A build Dockerfile is included in the repository.<br/>
This is used to automatically build and push the latest image to the Dockerhub repository **dabgmx/solace-exporter**

### Run Docker Image

The command line arguments can be overridden by environment variables to facilitate deployment in Docker. Example:<br/>

<pre><code>docker run -d \
 -p 9628:9628 \
 --env SOLACE_WEB_LISTEN_ADDRESS=":9628" \
 --env SOLACE_SCRAPE_URI="http://192.168.100.100:8080" \
 --env SOLACE_USER="admin" \
 --env SOLACE_PASSWORD="admin" \
 --env SOLACE_SCRAPE_TIMEOUT="5s" \
 --env SOLACE_SSL_VERIFY="false" \
 --env SOLACE_INCLUDE_REDUNDANCY="true" \
 --name solace-exporter \
 dabgmx/solace-exporter
</code></pre>

## Bonus Material

The sub directory **testfiles** contains some sample curl commands and their outputs. This is just fyi and not needed for building.
