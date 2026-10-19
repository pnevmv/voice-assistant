// Package platform holds the plumbing shared by every service binary:
// env-based config, slog setup and graceful HTTP lifecycle.
package platform

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Base is the config every service needs. Service-specific settings live in
// each service's own package and are read with the helpers below.
type Base struct {
	Service      string
	HTTPAddr     string
	LogLevel     string // debug | info | warn | error
	LogFormat    string // json | text
	DatabaseURL  string
	OTLPEndpoint string // empty disables tracing export
	ShutdownWait time.Duration
}

var defaultAddr = map[string]string{
	"gateway": ":8081",
	"rag":     ":8082",
	"voice":   ":8083",
	"crawler": ":8084", // admin only: health, metrics, manual triggers
}

// LoadBase reads configuration from the environment. Per-service overrides use
// the prefix <SERVICE>_ (e.g. GATEWAY_HTTP_ADDR); shared keys are unprefixed.
func LoadBase(service string) Base {
	prefix := strings.ToUpper(service) + "_"
	return Base{
		Service:      service,
		HTTPAddr:     String(prefix+"HTTP_ADDR", defaultAddr[service]),
		LogLevel:     String("LOG_LEVEL", "info"),
		LogFormat:    String("LOG_FORMAT", "json"),
		DatabaseURL:  String("DATABASE_URL", ""),
		OTLPEndpoint: String("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		ShutdownWait: Duration("SHUTDOWN_TIMEOUT", 20*time.Second),
	}
}

// Validate reports a config error early, before anything starts listening.
func (b Base) Validate() error {
	if b.HTTPAddr == "" {
		return fmt.Errorf("%s: HTTP address is empty", b.Service)
	}
	return nil
}

func String(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func Int(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// Bool parses the strconv.ParseBool forms (true/false/1/0/t/f/T/F/TRUE/FALSE). Anything else,
// including an empty value, yields def, so a typo can never switch on a
// dev-only feature.
func Bool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func Duration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

// Required returns the value of key or an error naming the missing variable.
// Use it for secrets; never give secrets defaults.
func Required(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return v, nil
}
