#!/bin/bash

export REALM="test-realm"

export TOKEN=$(curl -s -X POST \
  "http://localhost:8080/realms/master/protocol/openid-connect/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=admin" \
  -d "password=admin" \
  -d "grant_type=password" \
  -d "client_id=admin-cli" | jq -r .access_token)

echo "Creating realm: $REALM"
curl -s -X POST "http://localhost:8080/admin/realms" \
-H "Authorization: Bearer $TOKEN" \
-H "Content-Type: application/json" \
-d '{
  "realm": "test-realm",
  "enabled": true
}'

echo "Creating client 'prometheus-exporter' in realm: $REALM"
curl -s -X POST "http://localhost:8080/admin/realms/${REALM}/clients" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "clientId": "prometheus-exporter",
    "enabled": true,
    "protocol": "openid-connect",
    "publicClient": false,
    "serviceAccountsEnabled": true,
    "redirectUris": ["http://localhost:3000/*"],
    "secret": "my-secret"
  }'

echo "Creating client 'solace-broker' in realm: $REALM"
curl -s -X POST "http://localhost:8080/admin/realms/${REALM}/clients" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "clientId": "solace-broker",
    "enabled": true,
    "protocol": "openid-connect",
    "publicClient": false,
    "serviceAccountsEnabled": true,
    "redirectUris": ["http://localhost:3000/*"],
    "secret": "my-secret"
  }'

echo "Creating client scope 'solace' in realm: $REALM"
curl -s -X POST "http://localhost:8080/admin/realms/${REALM}/client-scopes" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "solace",
    "protocol": "openid-connect"
  }'

SCOPE_ID=$(curl -s \
  -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/admin/realms/${REALM}/client-scopes?name=solace" \
  | jq -r '.[] | select(.name=="solace") | .id')

CLIENT_ID=$(curl -s \
  -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/admin/realms/${REALM}/clients?clientId=prometheus-exporter" \
  | jq -r '.[0].id')

echo "Adding audience mapper to client scope 'solace' in realm: $REALM"
curl -s -X POST "http://localhost:8080/admin/realms/${REALM}/client-scopes/$SCOPE_ID/protocol-mappers/models" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "audience",
    "protocol": "openid-connect",
    "protocolMapper": "oidc-audience-mapper",
    "config": {
      "included.client.audience": "solace",
      "id.token.claim": "true",
      "access.token.claim": "true"
    }
  }'

echo "Assigning optional client scope 'solace' to client 'prometheus-exporter' in realm: $REALM"
curl -s -X PUT "http://localhost:8080/admin/realms/${REALM}/clients/$CLIENT_ID/optional-client-scopes/$SCOPE_ID" \
  -H "Authorization: Bearer $TOKEN"


CLIENT_ID=$(curl -s \
  -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/admin/realms/${REALM}/clients?clientId=solace-broker" \
  | jq -r '.[0].id')


echo "Assigning optional client scope 'solace' to client 'solace-broker' in realm: $REALM"
curl -s -X PUT "http://localhost:8080/admin/realms/${REALM}/clients/$CLIENT_ID/optional-client-scopes/$SCOPE_ID" \
  -H "Authorization: Bearer $TOKEN"
