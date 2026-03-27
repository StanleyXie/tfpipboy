// Package state provides terraform state tracking, change detection, and drift analysis.
package state

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"
)

// BackendType identifies which storage backend to use
type BackendType string

const (
	// BackendTypeFile uses local filesystem storage (default, for dev/testing)
	BackendTypeFile BackendType = "file"
	// BackendTypeGateway uses the HTTP/HTTPS gateway backed by Vault
	BackendTypeGateway BackendType = "gateway"
)

// GatewayAuthType identifies how to authenticate to the gateway
type GatewayAuthType string

const (
	// GatewayAuthToken uses a static bearer token
	GatewayAuthToken GatewayAuthType = "token"
	// GatewayAuthVaultToken passes a Vault token directly to the gateway
	GatewayAuthVaultToken GatewayAuthType = "vault_token"
	// GatewayAuthTLS uses mutual TLS client certificate authentication
	GatewayAuthTLS GatewayAuthType = "tls"
)

// GatewayConfig holds all configuration for connecting to the HTTP/HTTPS gateway
type GatewayConfig struct {
	// BaseURL is the gateway base URL, e.g. "https://tfstate-gateway.example.com"
	BaseURL string `yaml:"base_url"`

	// APIVersion is the gateway API version prefix, e.g. "v1" (default: "v1")
	APIVersion string `yaml:"api_version"`

	// AuthType determines how requests are authenticated (default: "token")
	AuthType GatewayAuthType `yaml:"auth_type"`

	// Token is the bearer token used when AuthType is "token" or "vault_token"
	// Reads from TFPIPBOY_GATEWAY_TOKEN env var if empty
	Token string `yaml:"token"`

	// TLSConfig holds mTLS settings when AuthType is "tls"
	TLS *GatewayTLSConfig `yaml:"tls"`

	// Timeout for HTTP requests (default: 30s)
	Timeout time.Duration `yaml:"timeout"`

	// MaxRetries for failed requests (default: 3)
	MaxRetries int `yaml:"max_retries"`

	// RetryDelay is the base delay between retries, doubles on each attempt (default: 1s)
	RetryDelay time.Duration `yaml:"retry_delay"`

	// SkipVerify disables TLS certificate verification (NOT recommended for production)
	SkipVerify bool `yaml:"skip_verify"`
}

// GatewayTLSConfig holds mutual TLS settings
type GatewayTLSConfig struct {
	// CACert is the path to the CA certificate file used to verify the gateway
	CACert string `yaml:"ca_cert"`
	// ClientCert is the path to the client certificate file
	ClientCert string `yaml:"client_cert"`
	// ClientKey is the path to the client private key file
	ClientKey string `yaml:"client_key"`
}

// DefaultGatewayConfig returns a GatewayConfig with sensible defaults
func DefaultGatewayConfig() *GatewayConfig {
	return &GatewayConfig{
		APIVersion: "v1",
		AuthType:   GatewayAuthToken,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: 1 * time.Second,
	}
}

// Validate checks that the config has required fields
func (c *GatewayConfig) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("gateway base_url is required")
	}
	if c.APIVersion == "" {
		c.APIVersion = "v1"
	}
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}
	if c.RetryDelay == 0 {
		c.RetryDelay = 1 * time.Second
	}

	// Load token from environment if not set
	if c.Token == "" && (c.AuthType == GatewayAuthToken || c.AuthType == GatewayAuthVaultToken) {
		c.Token = os.Getenv("TFPIPBOY_GATEWAY_TOKEN")
		if c.Token == "" {
			// Also try VAULT_TOKEN for vault_token auth type
			if c.AuthType == GatewayAuthVaultToken {
				c.Token = os.Getenv("VAULT_TOKEN")
			}
		}
		if c.Token == "" {
			return fmt.Errorf("gateway token is required (set via config or TFPIPBOY_GATEWAY_TOKEN env var)")
		}
	}

	if c.AuthType == GatewayAuthTLS {
		if c.TLS == nil {
			return fmt.Errorf("tls config is required when auth_type is 'tls'")
		}
		if c.TLS.ClientCert == "" || c.TLS.ClientKey == "" {
			return fmt.Errorf("tls client_cert and client_key are required")
		}
	}

	return nil
}

// BuildHTTPClient creates an *http.Client configured from this GatewayConfig
func (c *GatewayConfig) BuildHTTPClient() (*http.Client, error) {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: c.SkipVerify, //nolint:gosec // user-controlled opt-in
	}

	// Load CA cert if provided
	if c.TLS != nil && c.TLS.CACert != "" {
		caCert, err := os.ReadFile(c.TLS.CACert)
		if err != nil {
			return nil, fmt.Errorf("reading CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA cert")
		}
		tlsCfg.RootCAs = pool
	}

	// Load mTLS client certificate
	if c.TLS != nil && c.TLS.ClientCert != "" {
		cert, err := tls.LoadX509KeyPair(c.TLS.ClientCert, c.TLS.ClientKey)
		if err != nil {
			return nil, fmt.Errorf("loading client cert/key: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	transport := &http.Transport{
		TLSClientConfig: tlsCfg,
	}

	return &http.Client{
		Timeout:   c.Timeout,
		Transport: transport,
	}, nil
}
