#!/bin/sh

apk add curl

curl -vks http://prometheus_exporter:9628/solace-custom

if [ $? -ne 0 ]; then
  echo "Failed to get metrics from prometheus_exporter"
  exit 1
fi
