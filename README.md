
# solace_exporter, a Prometheus Exporter for Solace Message Brokers

## Disclaimer

This exporter is not developed and maintained by Solace.<br/>
It can be used as-is or as a basis for development of customer specific exporters for Solace brokers.<br/>
It is currently only tested against PubSub+ software brokers (VMRs), not appliances.<br/>

## Features

The exporter is written in go, based on the Solace Legacy SEMP protocol.<br/>
It implements the following endpoints:<br/>
<pre><code>http://&lt;host&gt;:&lt;port&gt;/             Document page showing list of endpoints
http://&lt;host&gt;:&lt;port&gt;/metrics      Golang and standard Prometheus metrics
http://&lt;host&gt;:&lt;port&gt;/solace-std   Solace metrics for System and VPN levels
http://&lt;host&gt;:&lt;port&gt;/solace-det   Solace metrics for Messaging Clients and Queues</code></pre>
The [registered](https://github.com/prometheus/prometheus/wiki/Default-port-allocations) default port for Solace is 9628<br/>

## Usage

<pre><code>solace_exporter -h
usage: solace_exporter [&lt;flags&gt;]

Flags:
  -h, --help                     Show context-sensitive help (also try --help-long and --help-man).
      --log.level=info           Only log messages with the given severity or above. One of: [debug, info, warn, error]
      --log.format=logfmt        Output format of log messages. One of: [logfmt, json]
      --config-file=CONFIG-FILE  Path and name of config file. See sample file solace_exporter.ini.</code></pre>

The configuration parameters can be placed into a config file or into a set of environment variables. For Docker you should prefer the environment variable configuration method (see below).<br/> If the exporter is started with a config file argument then the config file entries have precedence over the environment variables. If a parameter is neither found in the config file nor in the environment the exporter exits with an error.<br/>

### Config File

<pre><code>solace_exporter --config-file /path/to/config/file.ini</code></pre>

Sample config file:
<pre><code>[solace]
# Address to listen on for web interface and telemetry.
listenAddr=0.0.0.0:9628

# Base URI on which to scrape Solace broker.
scrapeUri=http://localhost:8080

# Basic Auth username for http scrape requests to Solace broker.
username=admin

# Basic Auth password for http scrape requests to Solace broker.
password=admin

# Timeout for http scrape requests to Solace broker.
timeout=5s

# Flag that enables SSL certificate verification for the scrape URI.
sslVerify=false

# Flag that enables scrape of redundancy metrics. Should be used for broker HA groups.
redundancy=false
</code></pre>

### Environment Variables
Sample environment variables:
<pre><code>SOLACE_LISTEN_ADDR=0.0.0.0:9628
SOLACE_SCRAPE_URI=http://localhost:8080
SOLACE_USERNAME=admin
SOLACE_PASSWORD=admin
SOLACE_TIMEOUT=5s
SOLACE_SSL_VERIFY=false
SOLACE_REDUNDANCY=false</code></pre>

## Build

### Default Build
<pre><code>cd &lt;some-directory&gt;/solace_exporter
go build
</code></pre>

## Docker

### Build Docker Image

A build Dockerfile is included in the repository.<br/>
This is used to automatically build and push the latest image to the Dockerhub repository [dabgmx/solace-exporter](https://hub.docker.com/r/dabgmx/solace-exporter)

### Run Docker Image

Environment variables are recommended to parameterize the exporter in Docker. Example:<br/>

<pre><code>docker run -d \
 -p 9628:9628 \
 --env SOLACE_LISTEN_ADDR=0.0.0.0:9628 \
 --env SOLACE_SCRAPE_URI=http://localhost:8080 \
 --env SOLACE_USERNAME=admin \
 --env SOLACE_PASSWORD=admin \
 --env SOLACE_TIMEOUT=5s \
 --env SOLACE_SSL_VERIFY=false \
 --env SOLACE_REDUNDANCY=false \
 --name solace-exporter \
 dabgmx/solace-exporter</code></pre>

## Bonus Material

The sub directory **testfiles** contains some sample curl commands and their outputs. This is just fyi and not needed for building.
