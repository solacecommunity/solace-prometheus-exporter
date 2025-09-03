package exporter

import (
	"crypto/tls"
	"net/http"
	"net/url"
)

func (conf *Config) basicHTTPClient() http.Client {
	var client http.Client
	var proxy func(req *http.Request) (*url.URL, error)

	if conf.useSystemProxy {
		proxy = http.ProxyFromEnvironment
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !conf.SslVerify}, //nolint:gosec
		Proxy:           proxy,
	}
	client = http.Client{
		Timeout:   conf.Timeout,
		Transport: tr,
	}
	return client
}

func (conf *Config) newHTTPClient() http.Client {
	// ctx := context.Background()
	client := conf.basicHTTPClient()

	client.CheckRedirect = conf.redirectPolicyFunc

	// if conf.authType == AuthTypeOAuth {
	// 	cc := &clientcredentials.Config{
	// 		ClientID:     conf.OAuthClientID,
	// 		ClientSecret: conf.OAuthClientSecret,
	// 		TokenURL:     conf.OAuthTokenURL,
	// 	}
	// 	client = *oauth2.NewClient(ctx, cc.TokenSource(ctx))
	// } else {
	// 	client.CheckRedirect = conf.redirectPolicyFunc
	// }

	return client
}

// Redirect callback, re-insert basic auth string into header.
func (conf *Config) redirectPolicyFunc(req *http.Request, _ []*http.Request) error {
	f, _ := conf.httpVisitor()
	f(req)
	return nil
}

func (conf *Config) httpVisitor() (func(*http.Request), error) {
	return conf.setAuthHeader()
}
