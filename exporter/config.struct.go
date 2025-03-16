package exporter

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

// Config Collection of configs
type Config struct {
	ListenAddr              string
	EnableTLS               bool
	Certificate             string
	PrivateKey              string
	CertType                string
	Pkcs12File              string
	Pkcs12Pass              string
	ScrapeURI               string
	Username                string
	Password                string
	DefaultVpn              string
	SslVerify               bool
	useSystemProxy          bool
	Timeout                 time.Duration
	PrefetchInterval        time.Duration
	ParallelSempConnections int64
	logBrokerToSlowWarnings bool
	IsHWBroker              bool
}

const (
	CERTTYPE_PEM    = "PEM"
	CERTTYPE_PKCS12 = "PKCS12"
)

// GetListenURI returns the `listenAddr` with proper protocol (http/https),
// based on the `enableTLS` configuration parameter
func (conf *Config) GetListenURI() string {
	if conf.EnableTLS {
		return "https://" + conf.ListenAddr
	}
	return "http://" + conf.ListenAddr
}

func ParseConfig(configFile string) (map[string][]DataSource, *Config, error) {
	var cfg *ini.File
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
	conf.CertType, err = parseConfigString(cfg, "solace", "certType", "SOLACE_LISTEN_CERTTYPE")
	if conf.EnableTLS && err != nil {
		log.Println("CertType not set. Using default PEM")
		conf.CertType = CERTTYPE_PEM
	}
	conf.Certificate, err = parseConfigString(cfg, "solace", "certificate", "SOLACE_SERVER_CERT")
	if conf.EnableTLS && strings.ToUpper(conf.CertType) == CERTTYPE_PEM && err != nil {
		return nil, nil, err
	}
	conf.PrivateKey, err = parseConfigString(cfg, "solace", "privateKey", "SOLACE_PRIVATE_KEY")
	if conf.EnableTLS && strings.ToUpper(conf.CertType) == CERTTYPE_PEM && err != nil {
		return nil, nil, err
	}
	conf.Pkcs12File, err = parseConfigString(cfg, "solace", "pkcs12File", "SOLACE_PKCS12_FILE")
	if conf.EnableTLS && strings.ToUpper(conf.CertType) == CERTTYPE_PKCS12 && err != nil {
		return nil, nil, err
	}
	conf.Pkcs12Pass, err = parseConfigString(cfg, "solace", "pkcs12Pass", "SOLACE_PKCS12_PASS")
	if conf.EnableTLS && strings.ToUpper(conf.CertType) == CERTTYPE_PKCS12 && err != nil {
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
	conf.DefaultVpn, err = parseConfigString(cfg, "solace", "defaultVpn", "SOLACE_DEFAULT_VPN")
	if err != nil {
		return nil, nil, err
	}
	conf.Timeout, err = parseConfigDuration(cfg, "solace", "timeout", "SOLACE_TIMEOUT")
	if err != nil {
		return nil, nil, err
	}
	conf.PrefetchInterval, err = parseConfigDurationOptional(cfg, "solace", "prefetchInterval", "PREFETCH_INTERVAL")
	if err != nil {
		return nil, nil, err
	}
	conf.SslVerify, err = parseConfigBool(cfg, "solace", "sslVerify", "SOLACE_SSL_VERIFY")
	if err != nil {
		return nil, nil, err
	}
	conf.ParallelSempConnections, err = parseConfigIntOptional(cfg, "solace", "parallelSempConnections", "SOLACE_PARALLEL_SEMP_CONNECTIONS")
	if err != nil {
		return nil, nil, err
	}
	conf.logBrokerToSlowWarnings, err = parseConfigBoolOptional(cfg, "solace", "logBrokerToSlowWarnings", "SOLACE_LOG_BROKER_IS_SLOW_WARNING", true)
	if err != nil {
		return nil, nil, err
	}
	conf.IsHWBroker, err = parseConfigBoolOptional(cfg, "solace", "isHWBroker", "SOLACE_IS_HW_BROKER", false)
	if err != nil {
		return nil, nil, err
	}

	if conf.ParallelSempConnections < 1 {
		conf.ParallelSempConnections = 2
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
					if len(parts) < 2 {
						return nil, nil, fmt.Errorf("one or two | expected at endpoint %q. Found key %q value %q. Expected: VPN wildcard | item wildcard | Optional metric filter for v2 apis", endpointName, key.Name(), key.String())
					}
					var metricFilter []string
					if len(parts) == 3 && len(strings.TrimSpace(parts[2])) > 0 {
						metricFilter = strings.Split(parts[2], ",")
					}

					dataSource = append(dataSource, DataSource{
						Name:         scrapeTarget,
						VpnFilter:    parts[0],
						ItemFilter:   parts[1],
						MetricFilter: metricFilter,
					})
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

func parseConfigBoolOptional(cfg *ini.File, iniSection string, iniKey string, envKey string, defaultValue bool) (bool, error) {
	s, err := parseConfigString(cfg, iniSection, iniKey, envKey)
	if err != nil {
		return defaultValue, err
	}

	val, err := strconv.ParseBool(s)
	if err != nil {
		return false, fmt.Errorf("config param %q and env param %q is mandetory. Both are missing: %w", iniKey, envKey, err)
	}

	return val, nil
}

func parseConfigDurationOptional(cfg *ini.File, iniSection string, iniKey string, envKey string) (time.Duration, error) {
	s, err := parseConfigString(cfg, iniSection, iniKey, envKey)
	if err != nil {
		return time.Duration(0), err
	}

	val, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("config param %q and env param %q is mandetory. Both are missing: %w", iniKey, envKey, err)
	}

	return val, nil
}

func parseConfigIntOptional(cfg *ini.File, iniSection string, iniKey string, envKey string) (int64, error) {
	s, err := parseConfigString(cfg, iniSection, iniKey, envKey)
	if err != nil {
		return 0, err
	}

	val, err := strconv.ParseInt(s, 10, 0)
	if err != nil {
		return 0, fmt.Errorf("config param %q and env param %q is mandetory. Both are missing: %w", iniKey, envKey, err)
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

func (conf *Config) newHTTPClient() http.Client {
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
