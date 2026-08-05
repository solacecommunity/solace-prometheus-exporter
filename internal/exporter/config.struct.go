package exporter

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"solace_exporter/internal/secret"

	"gopkg.in/ini.v1"
)

// secretResolveTimeout bounds how long ResolveSecrets waits on the secret backend at startup. Kept separate from
// Config.Timeout (per-scrape SEMP calls) since this only delays startup, not a live scrape.
const secretResolveTimeout = 30 * time.Second

// defaultSecretCacheTTL is how long a resolved *static* (non-leased) Vault secret is cached before being
// re-read, unless overridden via secretCacheTTL / SECRET_CACHE_TTL. Set to 0 to disable caching for those.
const defaultSecretCacheTTL = 60 * time.Second

type ExporterAuthConfig struct {
	Scheme   string
	Username string
	Password string `json:"-"`
}

// oAuthTokenCache holds the shared, mutable OAuth client-credentials token so it can be reused across scrape
// requests instead of being fetched for every request. It is referenced by pointer from Config so that a per-request
// Config copy (see Config.Clone) shares the SAME cache (keeping the token warm) without copying the mutex.
type oAuthTokenCache struct {
	mu     sync.RWMutex
	token  string
	expiry time.Time
}

// Config Collection of configs. Per-request scrape fields (ScrapeURI, Username, Password, Timeout) are
// overridden on a Config.Clone() per request, so concurrent scrapes never clobber each other's credentials.
// oAuthToken is the one intentionally shared (pointer) field, keeping the OAuth token cache warm across requests.
type Config struct {
	ListenAddr              string
	EnableTLS               bool
	Certificate             string `json:"-"`
	PrivateKey              string `json:"-"`
	CertType                string
	Pkcs12File              string `json:"-"`
	Pkcs12Pass              string `json:"-"`
	ScrapeURI               string
	Username                string
	Password                string `json:"-"`
	DefaultVpn              string
	SslVerify               bool
	Timeout                 time.Duration
	PrefetchInterval        time.Duration
	ParallelSempConnections int64
	logBrokerToSlowWarnings bool
	IsHWBroker              bool
	SempPageSize            int64
	OAuthTokenURL           string
	OAuthClientID           string
	OAuthClientSecret       string
	OAuthClientScope        string
	OAuthIssuer             string
	oAuthToken              *oAuthTokenCache
	authType                AuthType
	ExporterAuth            ExporterAuthConfig
	SecretBackend           string
	SecretCacheTTL          time.Duration
}

// Clone returns a shallow copy of Config safe to mutate per request. Scalar fields are copied by value; oAuthToken
// is shared by pointer on purpose so the cached OAuth token is reused across requests.
func (conf *Config) Clone() *Config {
	c := *conf
	return &c
}

// ResolveSecrets resolves any "vault:<path>#<field>" references among the static credential fields in place;
// non-vault values pass through unchanged. Call once at startup right after ParseConfig -- not safe to call
// concurrently with reads of these fields. ctx is bounded internally to secretResolveTimeout.
func (conf *Config) ResolveSecrets(ctx context.Context, resolver *secret.Resolver) error {
	ctx, cancel := context.WithTimeout(ctx, secretResolveTimeout)
	defer cancel()

	fields := []struct {
		name string
		val  *string
	}{
		{"username", &conf.Username},
		{"password", &conf.Password},
		{"exporterAuthUsername", &conf.ExporterAuth.Username},
		{"exporterAuthPassword", &conf.ExporterAuth.Password},
		{"oAuthClientSecret", &conf.OAuthClientSecret},
		{"pkcs12Pass", &conf.Pkcs12Pass},
	}

	for _, f := range fields {
		resolved, err := resolver.Resolve(ctx, *f.val)
		if err != nil {
			return fmt.Errorf("resolving %s: %w", f.name, err)
		}
		*f.val = resolved
	}

	// Determine auth type AFTER vault resolution so that vault-backed
	// username/password/oAuthClientSecret are checked against their
	// actual values, not the raw "vault:..." references.
	return conf.DetermineAuthType()
}

// DetermineAuthType sets conf.authType based on the resolved credential
// fields. It must be called after ResolveSecrets (or after manually
// setting the credential fields) so that vault refs have been resolved.
func (conf *Config) DetermineAuthType() error {
	anyOAuth := len(conf.OAuthClientID) > 0 || len(conf.OAuthClientSecret) > 0 || len(conf.OAuthTokenURL) > 0 || len(conf.OAuthClientScope) > 0
	allOAuth := len(conf.OAuthClientID) > 0 && len(conf.OAuthClientSecret) > 0 && len(conf.OAuthTokenURL) > 0 && len(conf.OAuthClientScope) > 0

	if anyOAuth && !allOAuth {
		return errors.New("incomplete OAuth configuration: oAuthClientID, oAuthClientSecret, oAuthTokenURL and oAuthClientScope must all be set")
	}

	switch {
	case allOAuth:
		conf.authType = AuthTypeOAuth
	case len(conf.Username) > 0 && len(conf.Password) > 0:
		conf.authType = AuthTypeBasic
	default:
		return errors.New("either basic auth (username+password) or OAuth (oAuthClientID+oAuthClientSecret+oAuthTokenURL+oAuthClientScope) must be configured")
	}

	return nil
}

const (
	CertTypePEM    = "PEM"
	CertTypePKCS12 = "PKCS12"
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

	conf := &Config{oAuthToken: &oAuthTokenCache{}}

	if len(configFile) > 0 {
		opts := ini.LoadOptions{
			AllowBooleanKeys: true,
		}
		cfg, err = ini.LoadSources(opts, configFile)

		if err != nil {
			return nil, nil, fmt.Errorf("can't open config file %q: %w", configFile, err)
		}
	}

	conf.ExporterAuth.Scheme = parseConfigStringOptional(cfg, "solace", "exporterAuthScheme", "SOLACE_EXPORTER_AUTH_SCHEME", "none")
	conf.ExporterAuth.Username = parseConfigStringOptional(cfg, "solace", "exporterAuthUsername", "SOLACE_EXPORTER_AUTH_USERNAME", "")
	conf.ExporterAuth.Password = parseConfigStringOptional(cfg, "solace", "exporterAuthPassword", "SOLACE_EXPORTER_AUTH_PASSWORD", "")
	conf.ListenAddr = parseConfigStringOptional(cfg, "solace", "listenAddr", "SOLACE_LISTEN_ADDR", "0.0.0.0:9628")
	conf.EnableTLS, err = parseConfigBoolOptional(cfg, "solace", "enableTLS", "SOLACE_LISTEN_TLS", false)
	if err != nil {
		return nil, nil, err
	}
	conf.CertType, err = parseConfigString(cfg, "solace", "certType", "SOLACE_LISTEN_CERTTYPE")
	if conf.EnableTLS && err != nil {
		log.Println("CertType not set. Using default PEM")
		conf.CertType = CertTypePEM
	}
	conf.Certificate, err = parseConfigString(cfg, "solace", "certificate", "SOLACE_SERVER_CERT")
	if conf.EnableTLS && strings.ToUpper(conf.CertType) == CertTypePEM && err != nil {
		return nil, nil, err
	}
	conf.PrivateKey, err = parseConfigString(cfg, "solace", "privateKey", "SOLACE_PRIVATE_KEY")
	if conf.EnableTLS && strings.ToUpper(conf.CertType) == CertTypePEM && err != nil {
		return nil, nil, err
	}
	conf.Pkcs12File, err = parseConfigString(cfg, "solace", "pkcs12File", "SOLACE_PKCS12_FILE")
	if conf.EnableTLS && strings.ToUpper(conf.CertType) == CertTypePKCS12 && err != nil {
		return nil, nil, err
	}
	conf.Pkcs12Pass, err = parseConfigString(cfg, "solace", "pkcs12Pass", "SOLACE_PKCS12_PASS")
	if conf.EnableTLS && strings.ToUpper(conf.CertType) == CertTypePKCS12 && err != nil {
		return nil, nil, err
	}
	conf.ScrapeURI, err = parseConfigString(cfg, "solace", "scrapeUri", "SOLACE_SCRAPE_URI")
	if err != nil {
		return nil, nil, err
	}
	conf.DefaultVpn = parseConfigStringOptional(cfg, "solace", "defaultVpn", "SOLACE_DEFAULT_VPN", "default")
	conf.Timeout, err = parseConfigDurationOptional(cfg, "solace", "timeout", "SOLACE_TIMEOUT", 5*time.Second)
	if err != nil {
		return nil, nil, err
	}
	conf.PrefetchInterval, err = parseConfigDurationOptional(cfg, "solace", "prefetchInterval", "PREFETCH_INTERVAL", 0*time.Second)
	if err != nil {
		return nil, nil, err
	}
	conf.SslVerify, err = parseConfigBoolOptional(cfg, "solace", "sslVerify", "SOLACE_SSL_VERIFY", false)
	if err != nil {
		return nil, nil, err
	}
	conf.ParallelSempConnections, err = parseConfigIntOptional(cfg, "solace", "parallelSempConnections", "SOLACE_PARALLEL_SEMP_CONNECTIONS", 1)
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
	conf.SempPageSize, err = parseConfigIntOptional(cfg, "solace", "sempPageSize", "SOLACE_SEMP_PAGE_SIZE", 100)
	if err != nil {
		return nil, nil, err
	}

	conf.OAuthTokenURL = parseConfigStringOptional(cfg, "solace", "oAuthTokenURL", "SOLACE_OAUTH_TOKEN_URL", "")
	conf.OAuthClientID = parseConfigStringOptional(cfg, "solace", "oAuthClientID", "SOLACE_OAUTH_CLIENT_ID", "")
	conf.OAuthClientSecret = parseConfigStringOptional(cfg, "solace", "oAuthClientSecret", "SOLACE_OAUTH_CLIENT_SECRET", "")
	conf.OAuthClientScope = parseConfigStringOptional(cfg, "solace", "oAuthClientScope", "SOLACE_OAUTH_CLIENT_SCOPE", "")
	conf.OAuthIssuer = parseConfigStringOptional(cfg, "solace", "oAuthIssuer", "SOLACE_OAUTH_ISSUER", "")
	conf.Username = parseConfigStringOptional(cfg, "solace", "username", "SOLACE_USERNAME", "admin")
	conf.Password = parseConfigStringOptional(cfg, "solace", "password", "SOLACE_PASSWORD", "admin")
	conf.SecretBackend = parseConfigStringOptional(cfg, "solace", "secretBackend", "SECRET_BACKEND", "")
	conf.SecretCacheTTL, err = parseConfigDurationOptional(cfg, "solace", "secretCacheTTL", "SECRET_CACHE_TTL", defaultSecretCacheTTL)
	if err != nil {
		return nil, nil, err
	}

	// Fails fast on missing/incomplete credentials, same as before vault support existed -- this only checks
	// presence/shape, so it works on raw "vault:..." refs too. ResolveSecrets calls DetermineAuthType again after
	// resolution, since a ref that resolves to an empty value would pass this check but must still be caught.
	if err := conf.DetermineAuthType(); err != nil {
		return nil, nil, err
	}

	if conf.ParallelSempConnections < 1 {
		conf.ParallelSempConnections = 2
	}

	if conf.SempPageSize < 1 {
		conf.SempPageSize = 100
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
	s := parseConfigStringOptional(cfg, iniSection, iniKey, envKey, "")
	if s == "" {
		return defaultValue, nil
	}

	val, err := strconv.ParseBool(s)
	if err != nil {
		return false, fmt.Errorf("config param %q and env param %q is invalid: %w", iniKey, envKey, err)
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
		return 0, fmt.Errorf("config param %q and env param %q is invalid: %w", iniKey, envKey, err)
	}

	return val, nil
}

func parseConfigDurationOptional(cfg *ini.File, iniSection string, iniKey string, envKey string, defaultValue time.Duration) (time.Duration, error) {
	s := parseConfigStringOptional(cfg, iniSection, iniKey, envKey, "")
	if s == "" {
		return defaultValue, nil
	}

	val, err := time.ParseDuration(s)
	if err != nil {
		return defaultValue, fmt.Errorf("config param %q and env param %q is invalid: %w", iniKey, envKey, err)
	}

	return val, nil
}

func parseConfigIntOptional(cfg *ini.File, iniSection string, iniKey string, envKey string, defaultValue int64) (int64, error) {
	s := parseConfigStringOptional(cfg, iniSection, iniKey, envKey, "")

	if s == "" {
		return defaultValue, nil
	}

	val, err := strconv.ParseInt(s, 10, 0)
	if err != nil {
		return 0, fmt.Errorf("config param %q and env param %q is invalid: %w", iniKey, envKey, err)
	}

	return val, nil
}

func parseConfigString(cfg *ini.File, iniSection string, iniKey string, envKey string) (string, error) {
	s := os.Getenv(envKey)
	if len(s) > 0 {
		return s, nil
	}

	if cfg != nil {
		s := iniKeyValue(cfg, iniSection, iniKey)
		if len(s) > 0 {
			return s, nil
		}
	}

	return "", fmt.Errorf("config param %q and env param %q is mandetory. Both are missing", iniKey, envKey)
}

// parseConfigStringOptional tries to find the config value from environment variable first, then from ini file.
func parseConfigStringOptional(cfg *ini.File, iniSection string, iniKey string, envKey string, defaultValue string) string {
	s := os.Getenv(envKey)
	if len(s) > 0 {
		return s
	}

	if cfg != nil {
		s := iniKeyValue(cfg, iniSection, iniKey)
		if len(s) > 0 {
			return s
		}
	}

	return defaultValue
}

// iniKeyValue returns the value of iniKey in iniSection, matching the key name case-insensitively. This keeps
// historically diverging spellings interchangeable (for example scrapeUri and scrapeURI) without breaking existing
// config files. It returns the value of the first non-empty matching key.
func iniKeyValue(cfg *ini.File, iniSection string, iniKey string) string {
	section := cfg.Section(iniSection)
	if section.HasKey(iniKey) {
		if v := section.Key(iniKey).String(); len(v) > 0 {
			return v
		}
	}

	for _, key := range section.Keys() {
		if strings.EqualFold(key.Name(), iniKey) {
			if v := key.String(); len(v) > 0 {
				return v
			}
		}
	}

	return ""
}
