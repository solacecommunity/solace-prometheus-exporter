package exporter

import (
	"os"
	"testing"
	"time"

	"gopkg.in/ini.v1"
)

func TestParseConfigBool(t *testing.T) {
	tests := []struct {
		name       string
		iniContent string
		iniSection string
		iniKey     string
		envKey     string
		envValue   string
		want       bool
		wantErr    bool
	}{
		{
			name:       "env true",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "enableTLS",
			envKey:     "SOLACE_LISTEN_TLS",
			envValue:   "true",
			want:       true,
			wantErr:    false,
		},
		{
			name:       "env false",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "enableTLS",
			envKey:     "SOLACE_LISTEN_TLS",
			envValue:   "false",
			want:       false,
			wantErr:    false,
		},
		{
			name:       "ini true",
			iniContent: "[solace]\nenableTLS = true\n",
			iniSection: "solace",
			iniKey:     "enableTLS",
			envKey:     "SOLACE_LISTEN_TLS",
			envValue:   "",
			want:       true,
			wantErr:    false,
		},
		{
			name:       "ini false",
			iniContent: "[solace]\nenableTLS = false\n",
			iniSection: "solace",
			iniKey:     "enableTLS",
			envKey:     "SOLACE_LISTEN_TLS",
			envValue:   "",
			want:       false,
			wantErr:    false,
		},
		{
			name:       "missing both",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "enableTLS",
			envKey:     "SOLACE_LISTEN_TLS",
			envValue:   "",
			want:       false,
			wantErr:    true,
		},
		{
			name:       "invalid env value",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "enableTLS",
			envKey:     "SOLACE_LISTEN_TLS",
			envValue:   "notabool",
			want:       false,
			wantErr:    true,
		},
		{
			name:       "invalid ini value",
			iniContent: "[solace]\nenableTLS = notabool\n",
			iniSection: "solace",
			iniKey:     "enableTLS",
			envKey:     "SOLACE_LISTEN_TLS",
			envValue:   "",
			want:       false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			} else {
				os.Unsetenv(tt.envKey)
			}

			var cfg *ini.File
			var err error
			if tt.iniContent != "" {
				cfg, err = ini.Load([]byte(tt.iniContent))
				if err != nil {
					t.Fatalf("failed to load ini: %v", err)
				}
			}

			got, err := parseConfigBool(cfg, tt.iniSection, tt.iniKey, tt.envKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseConfigBool() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("parseConfigBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseConfigBoolOptional(t *testing.T) {
	testDefault := true

	tests := []struct {
		name       string
		iniContent string
		iniSection string
		iniKey     string
		envKey     string
		envValue   string
		want       bool
		wantErr    bool
	}{
		{
			name:       "env true",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "enableTLS",
			envKey:     "SOLACE_LISTEN_TLS",
			envValue:   "true",
			want:       true,
			wantErr:    false,
		},
		{
			name:       "env false",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "enableTLS",
			envKey:     "SOLACE_LISTEN_TLS",
			envValue:   "false",
			want:       false,
			wantErr:    false,
		},
		{
			name:       "ini true",
			iniContent: "[solace]\nenableTLS = true\n",
			iniSection: "solace",
			iniKey:     "enableTLS",
			envKey:     "SOLACE_LISTEN_TLS",
			envValue:   "",
			want:       true,
			wantErr:    false,
		},
		{
			name:       "ini false",
			iniContent: "[solace]\nenableTLS = false\n",
			iniSection: "solace",
			iniKey:     "enableTLS",
			envKey:     "SOLACE_LISTEN_TLS",
			envValue:   "",
			want:       false,
			wantErr:    false,
		},
		{
			name:       "missing both",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "enableTLS",
			envKey:     "SOLACE_LISTEN_TLS",
			envValue:   "",
			want:       testDefault,
			wantErr:    true,
		},
		{
			name:       "invalid env value",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "enableTLS",
			envKey:     "SOLACE_LISTEN_TLS",
			envValue:   "notabool",
			want:       testDefault,
			wantErr:    true,
		},
		{
			name:       "invalid ini value",
			iniContent: "[solace]\nenableTLS = notabool\n",
			iniSection: "solace",
			iniKey:     "enableTLS",
			envKey:     "SOLACE_LISTEN_TLS",
			envValue:   "",
			want:       testDefault,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			} else {
				os.Unsetenv(tt.envKey)
			}

			var cfg *ini.File
			var err error
			if tt.iniContent != "" {
				cfg, err = ini.Load([]byte(tt.iniContent))
				if err != nil {
					t.Fatalf("failed to load ini: %v", err)
				}
			}

			got, err := parseConfigBoolOptional(cfg, tt.iniSection, tt.iniKey, tt.envKey, testDefault)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseConfigBoolOptional() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("parseConfigBoolOptional() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseConfigDuration(t *testing.T) {
	tests := []struct {
		name       string
		iniContent string
		iniSection string
		iniKey     string
		envKey     string
		envValue   string
		want       time.Duration
		wantErr    bool
	}{
		{
			name:       "env set",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "scrapeInterval",
			envKey:     "SOLACE_SCRAPE_INTERVAL",
			envValue:   "15s",
			want:       time.Duration(15 * time.Second),
			wantErr:    false,
		},
		{
			name:       "ini set",
			iniContent: "[solace]\nscrapeInterval = 15s\n",
			iniSection: "solace",
			iniKey:     "scrapeInterval",
			envKey:     "SOLACE_SCRAPE_INTERVAL",
			envValue:   "",
			want:       time.Duration(15 * time.Second),
			wantErr:    false,
		},
		{
			name:       "missing both",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "scrapeInterval",
			envKey:     "SOLACE_SCRAPE_INTERVAL",
			envValue:   "",
			want:       time.Duration(0 * time.Second),
			wantErr:    true,
		},
		{
			name:       "invalid env value",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "scrapeInterval",
			envKey:     "SOLACE_SCRAPE_INTERVAL",
			envValue:   "notanint",
			want:       time.Duration(0 * time.Second),
			wantErr:    true,
		},
		{
			name:       "invalid ini value",
			iniContent: "[solace]\nscrapeInterval = notanint\n",
			iniSection: "solace",
			iniKey:     "scrapeInterval",
			envKey:     "SOLACE_SCRAPE_INTERVAL",
			envValue:   "",
			want:       time.Duration(0 * time.Second),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			} else {
				os.Unsetenv(tt.envKey)
			}
			var cfg *ini.File
			var err error
			if tt.iniContent != "" {
				cfg, err = ini.Load([]byte(tt.iniContent))
				if err != nil {
					t.Fatalf("failed to load ini: %v", err)
				}
			}
			got, err := parseConfigDuration(cfg, tt.iniSection, tt.iniKey, tt.envKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseConfigDuration() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("parseConfigDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseConfigDurationOptional(t *testing.T) {
	testDefault := time.Duration(0 * time.Second)

	tests := []struct {
		name       string
		iniContent string
		iniSection string
		iniKey     string
		envKey     string
		envValue   string
		want       time.Duration
		wantErr    bool
	}{
		{
			name:       "env set",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "scrapeInterval",
			envKey:     "SOLACE_SCRAPE_INTERVAL",
			envValue:   "15s",
			want:       time.Duration(15 * time.Second),
			wantErr:    false,
		},
		{
			name:       "ini set",
			iniContent: "[solace]\nscrapeInterval = 15s\n",
			iniSection: "solace",
			iniKey:     "scrapeInterval",
			envKey:     "SOLACE_SCRAPE_INTERVAL",
			envValue:   "",
			want:       time.Duration(15 * time.Second),
			wantErr:    false,
		},
		{
			name:       "missing both",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "scrapeInterval",
			envKey:     "SOLACE_SCRAPE_INTERVAL",
			envValue:   "",
			want:       testDefault,
			wantErr:    true,
		},
		{
			name:       "invalid env value",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "scrapeInterval",
			envKey:     "SOLACE_SCRAPE_INTERVAL",
			envValue:   "notanint",
			want:       testDefault,
			wantErr:    true,
		},
		{
			name:       "invalid ini value",
			iniContent: "[solace]\nscrapeInterval = notanint\n",
			iniSection: "solace",
			iniKey:     "scrapeInterval",
			envKey:     "SOLACE_SCRAPE_INTERVAL",
			envValue:   "",
			want:       testDefault,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			} else {
				os.Unsetenv(tt.envKey)
			}
			var cfg *ini.File
			var err error
			if tt.iniContent != "" {
				cfg, err = ini.Load([]byte(tt.iniContent))
				if err != nil {
					t.Fatalf("failed to load ini: %v", err)
				}
			}
			got, err := parseConfigDurationOptional(cfg, tt.iniSection, tt.iniKey, tt.envKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseConfigDurationOptional() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("parseConfigDurationOptional() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseConfigIntOptional(t *testing.T) {
	testDefault := int64(0)

	tests := []struct {
		name       string
		iniContent string
		iniSection string
		iniKey     string
		envKey     string
		envValue   string
		want       int64
		wantErr    bool
	}{
		{
			name:       "env set",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "scrapeTimeout",
			envKey:     "SOLACE_SCRAPE_TIMEOUT",
			envValue:   "15",
			want:       15,
			wantErr:    false,
		},
		{
			name:       "ini set",
			iniContent: "[solace]\nscrapeTimeout = 15\n",
			iniSection: "solace",
			iniKey:     "scrapeTimeout",
			envKey:     "SOLACE_SCRAPE_TIMEOUT",
			envValue:   "",
			want:       15,
			wantErr:    false,
		},
		{
			name:       "missing both",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "scrapeTimeout",
			envKey:     "SOLACE_SCRAPE_TIMEOUT",
			envValue:   "",
			want:       testDefault,
			wantErr:    true,
		},
		{
			name:       "invalid env value",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "scrapeTimeout",
			envKey:     "SOLACE_SCRAPE_TIMEOUT",
			envValue:   "notanint",
			want:       testDefault,
			wantErr:    true,
		},
		{
			name:       "invalid ini value",
			iniContent: "[solace]\nscrapeTimeout = notanint\n",
			iniSection: "solace",
			iniKey:     "scrapeTimeout",
			envKey:     "SOLACE_SCRAPE_TIMEOUT",
			envValue:   "",
			want:       testDefault,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			} else {
				os.Unsetenv(tt.envKey)
			}

			var cfg *ini.File
			var err error
			if tt.iniContent != "" {
				cfg, err = ini.Load([]byte(tt.iniContent))
				if err != nil {
					t.Fatalf("failed to load ini: %v", err)
				}
			}

			got, err := parseConfigIntOptional(cfg, tt.iniSection, tt.iniKey, tt.envKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseConfigIntOptional() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("parseConfigIntOptional() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseConfigString(t *testing.T) {
	tests := []struct {
		name       string
		iniContent string
		iniSection string
		iniKey     string
		envKey     string
		envValue   string
		want       string
		wantErr    bool
	}{
		{
			name:       "env set",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "scrapeURI",
			envKey:     "SOLACE_SCRAPE_URI",
			envValue:   "http://example.com",
			want:       "http://example.com",
			wantErr:    false,
		},
		{
			name:       "ini set",
			iniContent: "[solace]\nscrapeURI = http://example.com\n",
			iniSection: "solace",
			iniKey:     "scrapeURI",
			envKey:     "SOLACE_SCRAPE_URI",
			envValue:   "",
			want:       "http://example.com",
			wantErr:    false,
		},
		{
			name:       "missing both",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "scrapeURI",
			envKey:     "SOLACE_SCRAPE_URI",
			envValue:   "",
			want:       "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			} else {
				os.Unsetenv(tt.envKey)
			}

			var cfg *ini.File
			var err error
			if tt.iniContent != "" {
				cfg, err = ini.Load([]byte(tt.iniContent))
				if err != nil {
					t.Fatalf("failed to load ini: %v", err)
				}
			}

			got, err := parseConfigString(cfg, tt.iniSection, tt.iniKey, tt.envKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseConfigString() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("parseConfigString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseConfigStringOptional(t *testing.T) {
	testDefault := "defaultValue"

	tests := []struct {
		name       string
		iniContent string
		iniSection string
		iniKey     string
		envKey     string
		envValue   string
		want       string
		wantErr    bool
	}{
		{
			name:       "env set",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "scrapeURI",
			envKey:     "SOLACE_SCRAPE_URI",
			envValue:   "http://example.com",
			want:       "http://example.com",
			wantErr:    false,
		},
		{
			name:       "ini set",
			iniContent: "[solace]\nscrapeURI = http://example.com\n",
			iniSection: "solace",
			iniKey:     "scrapeURI",
			envKey:     "SOLACE_SCRAPE_URI",
			envValue:   "",
			want:       "http://example.com",
			wantErr:    false,
		},
		{
			name:       "missing both",
			iniContent: "",
			iniSection: "solace",
			iniKey:     "scrapeURI",
			envKey:     "SOLACE_SCRAPE_URI",
			envValue:   "",
			want:       testDefault,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			} else {
				os.Unsetenv(tt.envKey)
			}

			var cfg *ini.File
			var err error
			if tt.iniContent != "" {
				cfg, err = ini.Load([]byte(tt.iniContent))
				if err != nil {
					t.Fatalf("failed to load ini: %v", err)
				}
			}

			got, err := parseConfigStringOptional(cfg, tt.iniSection, tt.iniKey, tt.envKey, testDefault)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseConfigStringOptional() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("parseConfigStringOptional() = %v, want %v", got, tt.want)
			}
		})
	}
}
