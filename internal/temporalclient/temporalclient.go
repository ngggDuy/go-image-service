// Package temporalclient builds Temporal client options for both a local
// cluster and Temporal Cloud, so the two binaries that dial Temporal share
// one code path.
package temporalclient

import (
	"crypto/tls"

	"go-image-service/internal/config"

	"go.temporal.io/sdk/client"
)

// Options returns client options for cfg. An empty TemporalAPIKey means a
// local, unauthenticated cluster and yields today's plain options.
func Options(cfg config.Config) client.Options {
	opts := client.Options{
		HostPort:  cfg.TemporalAddress,
		Namespace: cfg.TemporalNamespace,
	}
	if cfg.TemporalAPIKey != "" {
		// Temporal Cloud authenticates with an API key over server TLS; no
		// client certificate is involved, so an empty tls.Config is correct.
		opts.Credentials = client.NewAPIKeyStaticCredentials(cfg.TemporalAPIKey)
		opts.ConnectionOptions = client.ConnectionOptions{TLS: &tls.Config{}}
	}
	return opts
}

// Dial connects to Temporal using Options(cfg).
func Dial(cfg config.Config) (client.Client, error) {
	return client.Dial(Options(cfg))
}
