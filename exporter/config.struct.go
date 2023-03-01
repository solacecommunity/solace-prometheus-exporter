package exporter

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

// Collection of configs
type Config struct {
	ListenAddr     string
	EnableTLS      bool
	Certificate    string
	PrivateKey     string
	ScrapeURI      string
	Username       string
	Password       string
	SslVerify      bool
	useSystemProxy bool
	Timeout        time.Duration
	DataSource     []DataSource
}

// getListenURI returns the `listenAddr` with proper protocol (http/https),
// based on the `enableTLS` configuration parameter
func (c *Config) GetListenURI() string {
	if c.EnableTLS {
		return "https://" + c.ListenAddr
	} else {
		return "http://" + c.ListenAddr
	}
}

func ParseConfig(configFile string) (map[string][]DataSource, *Config, error) {
	var cfg *ini.File = nil
	var err error

	conf := &Config{}

	if len(configFile) > 0 {
		opts := ini.LoadOptions{
			AllowBooleanKeys: true,
		}
		cfg, err = ini.LoadSources(opts, configFile)
		if err != nil {
			return nil, nil, fmt.Errorf("can't open config file %q: %w", configFile, err)
		}
	}

	conf.ListenAddr, err = parseConfigString(cfg, "solace", "listenAddr", "SOLACE_LISTEN_ADDR")
	if err != nil {
		return nil, nil, err
	}
	conf.EnableTLS, err = parseConfigBool(cfg, "solace", "enableTLS", "SOLACE_LISTEN_TLS")
	if err != nil {
		return nil, nil, err
	}
	conf.Certificate, err = parseConfigString(cfg, "solace", "certificate", "SOLACE_SERVER_CERT")
	if conf.EnableTLS && err != nil {
		return nil, nil, err
	}
	conf.PrivateKey, err = parseConfigString(cfg, "solace", "privateKey", "SOLACE_PRIVATE_KEY")
	if conf.EnableTLS && err != nil {
		return nil, nil, err
	}
	conf.ScrapeURI, err = parseConfigString(cfg, "solace", "scrapeUri", "SOLACE_SCRAPE_URI")
	if err != nil {
		return nil, nil, err
	}
	conf.Username, err = parseConfigString(cfg, "solace", "username", "SOLACE_USERNAME")
	if err != nil {
		return nil, nil, err
	}
	conf.Password, err = parseConfigString(cfg, "solace", "password", "SOLACE_PASSWORD")
	if err != nil {
		return nil, nil, err
	}
	conf.Timeout, err = parseConfigDuration(cfg, "solace", "timeout", "SOLACE_TIMEOUT")
	if err != nil {
		return nil, nil, err
	}
	conf.SslVerify, err = parseConfigBool(cfg, "solace", "sslVerify", "SOLACE_SSL_VERIFY")
	if err != nil {
		return nil, nil, err
	}

	endpoints := make(map[string][]DataSource)
	if cfg != nil {
		var scrapeTargetRe = regexp.MustCompile(`^(\w+)(\.\d+)?$`)
		for _, section := range cfg.Sections() {
			if strings.HasPrefix(section.Name(), "endpoint.") {
				endpointName := strings.TrimPrefix(section.Name(), "endpoint.")

				var dataSource []DataSource
				for _, key := range section.Keys() {
					scrapeTarget := scrapeTargetRe.ReplaceAllString(key.Name(), `$1`)

					parts := strings.Split(key.String(), "|")
					if len(parts) != 2 {
						return nil, nil, fmt.Errorf("exactly one %q expected at endpoint %q. Found key %q value %q. Expecected: VPN wildcard | item wildcard", "|", endpointName, key.Name(), key.String())
					} else {
						dataSource = append(dataSource, DataSource{
							Name:       scrapeTarget,
							VpnFilter:  parts[0],
							ItemFilter: parts[1],
						})
					}
				}

				endpoints[endpointName] = dataSource
			}
		}
	}

	return endpoints, conf, nil
}

func parseConfigBool(cfg *ini.File, iniSection string, iniKey string, envKey string) (bool, error) {
	s, err := parseConfigString(cfg, iniSection, iniKey, envKey)
	if err != nil {
		return false, err
	}
	val, err := strconv.ParseBool(s)
	if err != nil {
		return false, fmt.Errorf("config param %q and env param %q is mandetory. Both are missing: %w", iniKey, envKey, err)
	}

	return val, nil
}

func parseConfigDuration(cfg *ini.File, iniSection string, iniKey string, envKey string) (time.Duration, error) {
	s, err := parseConfigString(cfg, iniSection, iniKey, envKey)
	if err != nil {
		return 0, err
	}

	val, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("config param %q and env param %q is mandetory. Both are missing: %w", iniKey, envKey, err)
	}

	return val, nil
}

func parseConfigString(cfg *ini.File, iniSection string, iniKey string, envKey string) (string, error) {
	s := os.Getenv(envKey)
	if len(s) > 0 {
		return s, nil
	}

	if cfg != nil {
		s := cfg.Section(iniSection).Key(iniKey).String()
		if len(s) > 0 {
			return s, nil
		}
	}

	return "", fmt.Errorf("config param %q and env param %q is mandetory. Both are missing", iniKey, envKey)
}

func (conf *Config) newHttpClient() http.Client {
	var proxy func(req *http.Request) (*url.URL, error)
	if conf.useSystemProxy {
		proxy = http.ProxyFromEnvironment
	}
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: !conf.SslVerify}, Proxy: proxy}
	client := http.Client{
		Timeout:       conf.Timeout,
		Transport:     tr,
		CheckRedirect: conf.redirectPolicyFunc,
	}

	return client
}

// Redirect callback, re-insert basic auth string into header
func (conf *Config) redirectPolicyFunc(req *http.Request, _ []*http.Request) error {
	conf.httpVisitor()(req)
	return nil
}

func (conf *Config) httpVisitor() func(*http.Request) {
	return func(request *http.Request) {
		request.SetBasicAuth(conf.Username, conf.Password)
	}
}
