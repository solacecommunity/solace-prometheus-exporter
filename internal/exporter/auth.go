package exporter

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type AuthType int

const (
	AuthTypeBasic AuthType = iota
	AuthTypeOAuth
)

// setAuthHeader returns a visitor that applies the configured authentication to an *http.Request.
// The returned visitor is always non-nil. For OAuth, token retrieval is deferred to request time (see below), so
// this function does not fetch a token and currently never returns a non-nil error; the error is retained in the
// signature for callers to remain correct should auth setup gain a failure mode.
func (conf *Config) setAuthHeader(ctx context.Context) (func(*http.Request), error) {
	switch conf.authType {
	case AuthTypeBasic:
		return func(request *http.Request) {
			request.SetBasicAuth(conf.Username, conf.Password)
		}, nil
	case AuthTypeOAuth:
		// Fetch the token from the shared cache on EVERY request (it refreshes shortly before expiry). Capturing
		// the token string once here would make long-lived async prefetch fetchers keep sending a stale bearer
		// after it expires, causing broker-wide 401s.
		return func(request *http.Request) {
			token, err := conf.getOAuthToken(ctx)
			if err != nil {
				// Leave Authorization unset so the scrape fails visibly (solace_up=0) rather than sending a
				// stale/blank token. We never return a nil visitor, which would panic the scrape.
				return
			}
			request.Header.Set("Authorization", "Bearer "+conf.issuerPrefixedToken(token))
		}, nil
	default:
		// No auth configured.
		return func(*http.Request) {}, nil
	}
}

// getOAuthToken retrieves a new OAuth token using the client credentials flow if the current token is expired or about to expire.
// The token is cached in the shared oAuthToken cache, so that concurrent scrape requests (each on its own Config.Clone)
// reuse the same token instead of each fetching a new one.
func (conf *Config) getOAuthToken(ctx context.Context) (string, error) {
	cache := conf.oAuthToken
	if cache == nil {
		return "", errors.New("oauth token cache is not initialised; build the Config via ParseConfig")
	}

	cache.mu.RLock()
	if cache.token != "" && time.Now().Before(cache.expiry.Add(-time.Minute*5)) {
		token := cache.token
		cache.mu.RUnlock()
		return token, nil
	}
	cache.mu.RUnlock()

	cache.mu.Lock()
	defer cache.mu.Unlock()

	// Double-check after acquiring the write lock
	if cache.token != "" && time.Now().Before(cache.expiry.Add(-time.Minute*5)) {
		return cache.token, nil
	}

	client := conf.basicHTTPClient()
	reqContext := context.WithValue(ctx, oauth2.HTTPClient, &client)

	cc := &clientcredentials.Config{
		ClientID:     conf.OAuthClientID,
		ClientSecret: conf.OAuthClientSecret,
		TokenURL:     conf.OAuthTokenURL,
		Scopes:       []string{conf.OAuthClientScope},
	}

	token, err := cc.Token(reqContext)
	if err != nil {
		return "", err
	}

	cache.token = token.AccessToken
	cache.expiry = token.Expiry

	return cache.token, nil
}

func (conf *Config) issuerPrefixedToken(token string) string {
	if conf.OAuthIssuer == "" {
		return token
	}
	// Solace expects the issuer as unpadded base64 between '~' markers. Using RawStdEncoding avoids the previous
	// fragile "strip exactly one '=' " logic, which corrupted the value for issuer lengths whose base64 had zero
	// or two padding characters.
	encoded := base64.RawStdEncoding.EncodeToString([]byte(conf.OAuthIssuer))
	return "~" + encoded + "~" + token
}
