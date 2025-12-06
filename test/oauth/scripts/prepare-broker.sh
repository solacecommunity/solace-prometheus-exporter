#!/bin/bash
# SEMP/v2/config/
echo "Creating OAuth profile in Solace broker for SEMP access"
curl -X POST -u admin:admin \
  -H "Content-type: application/json" \
  http://localhost:8081/SEMP/v2/config/oauthProfiles \
  -d '{
        "oauthProfileName": "keycloaksemp",
        "displayName": "keycloak-semp",
        "enabled": true,
        "issuer": "http://keycloak:8080/realms/test-realm",
        "endpointAuthorization": "http://keycloak:8080/realms/test-realm/protocol/openid-connect/auth",
        "endpointToken": "http://keycloak:8080/realms/test-realm/protocol/openid-connect/token",
        "endpointUserinfo": "http://keycloak:8080/realms/test-realm/protocol/openid-connect/userinfo",
        "endpointIntrospection": "http://keycloak:8080/realms/test-realm/protocol/openid-connect/token/introspect",
        "endpointJwks": "http://keycloak:8080/realms/test-realm/protocol/openid-connect/certs"
  }'

echo "Creating management user 'oauth-admin' in Solace broker"
curl -X POST -u admin:admin \
  -H "Content-type: application/json" \
  http://localhost:8081/SEMP/v2/config/managementUsers \
  -d '{
        "username": "prometheus-exporter",
        "enabled": true,
        "oauthProfileName": "keycloaksemp",
        "authorizationGroup": "admin"
      }'

echo "Get token for prometheus-exporter oauth user"
export TOKEN=$(curl -s -X POST \
  "http://localhost:8080/realms/test-realm/protocol/openid-connect/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=prometheus-exporter" \
  -d "scope=solace" \
  -d "client_secret=my-secret" | jq -r .access_token)

curl -v \
  -H "Authorization: Bearer $TOKEN" \
  http://localhost:8081/SEMP/v2/config/about
