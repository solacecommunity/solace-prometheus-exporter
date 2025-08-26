package exporter

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/oauth2/clientcredentials"
)

type AuthType int

const (
	AuthTypeBasic AuthType = iota
	AuthTypeOAuth
)

// setAuthHeader sets the appropriate authentication header on the request based on the configured auth type.
// It returns a function that can be used to set the header on an http.Request, or an error if there was an issue obtaining an OAuth token.
func (conf *Config) setAuthHeader() (func(*http.Request), error) {
	if conf.authType == AuthTypeBasic {
		return func(request *http.Request) {
			request.SetBasicAuth(conf.Username, conf.Password)
		}, nil
	}
	if conf.authType == AuthTypeOAuth {
		token, err := conf.getOAuthToken()
		if err != nil {
			return nil, err
		}
		return func(request *http.Request) {
			request.Header.Set("Authorization", "Bearer "+token)
		}, nil
	}

	// Optionally default to no auth
	return func(request *http.Request) {}, nil
}

// getOAuthToken retrieves a new OAuth token using the client credentials flow if the current token is expired or about to expire.
func (conf *Config) getOAuthToken() (string, error) {
	if conf.oAuthAccessToken != "" && time.Now().Before(conf.oAuthTokenExpiry.Add(-time.Minute*5)) {
		return conf.oAuthAccessToken, nil
	}
	ctx := context.Background()
	cc := &clientcredentials.Config{
		ClientID:     conf.OAuthClientID,
		ClientSecret: conf.OAuthClientSecret,
		TokenURL:     conf.OAuthTokenURL,
	}

	token, err := cc.Token(ctx)
	if err != nil {
		return "", err
	}

	conf.oAuthAccessToken = token.AccessToken
	conf.oAuthTokenExpiry = token.Expiry

	return conf.oAuthAccessToken, nil
}
