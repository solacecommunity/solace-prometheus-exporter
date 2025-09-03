#!/bin/sh

apk add curl jq

echo "Get token for prometheus-exporter oauth user"
export TOKEN=$(curl -vks -X POST \
  "https://keycloak:8443/realms/test-realm/protocol/openid-connect/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=prometheus-exporter" \
  -d "scope=solace" \
  -d "client_secret=my-secret" | jq -r .access_token)

curl -vks \
  -H "Authorization: Bearer $TOKEN" \
  https://solbroker:1943/SEMP/v2/config/about
