
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
http://&lt;host&gt;:&lt;port&gt;/solace-det   Solace metrics for Messaging Clients and Queues
</code></pre>
The [registered](https://github.com/prometheus/prometheus/wiki/Default-port-allocations) default port for Solace is 9628<br/>

## Usage

<pre><code>solace_exporter -h
usage: solace_exporter [&lt;flags&gt;]

Flags:
  -h, --help                     Show context-sensitive help (also try --help-long and --help-man).
      --web.listen-address=":9628"
                                 Address to listen on for web interface and telemetry.
      --config-file=CONFIG-FILE  Path and name of ini file with configuration settings. See sample file
                                 solace_exporter.ini.
      --sol.uri=""               Base URI on which to scrape Solace.
      --sol.user=""              Username for http requests to Solace broker.
      --sol.timeout=5s           Timeout for trying to get stats from Solace.
      --sol.sslv                 Flag that enables SSL certificate verification for the scrape URI.
      --sol.redundancy           Flag that enables scrape of redundancy metrics. Should be used for broker HA groups.
      --log.level=info           Only log messages with the given severity or above. One of: [debug, info, warn, error]
      --log.format=logfmt        Output format of log messages. One of: [logfmt, json]</code></pre>
### Security

Please note that for security reasons the parameter "password" is not available as command line parameter.<br/>
For Docker you should prefer the environment variable configuration method (see below).<br/>
Otherwise please place your options in an config file and call the exporter with:

<pre><code>solace_exporter --config-file /path/to/config/file.ini

cat /path/to/config/file.ini
[sol]
uri=http://localhost:8080
username=admin
password=admin
timeout=6s
sslVerify=false
scrapeRedundancy=false
...</code></pre>


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
