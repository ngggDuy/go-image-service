package temporalclient

import (
	"testing"

	"go-image-service/internal/config"
)

func TestOptions_LocalHasNoCredentials(t *testing.T) {
	opts := Options(config.Config{
		TemporalAddress:   "localhost:7233",
		TemporalNamespace: "default",
	})

	if opts.HostPort != "localhost:7233" {
		t.Errorf("HostPort = %q, want %q", opts.HostPort, "localhost:7233")
	}
	if opts.Namespace != "default" {
		t.Errorf("Namespace = %q, want %q", opts.Namespace, "default")
	}
	if opts.Credentials != nil {
		t.Error("Credentials set for a local cluster, want nil")
	}
	if opts.ConnectionOptions.TLS != nil {
		t.Error("TLS set for a local cluster, want nil")
	}
}

func TestOptions_CloudHasCredentialsAndTLS(t *testing.T) {
	opts := Options(config.Config{
		TemporalAddress:   "img.acct1.tmprl.cloud:7233",
		TemporalNamespace: "img.acct1",
		TemporalAPIKey:    "an-api-key",
	})

	if opts.Namespace != "img.acct1" {
		t.Errorf("Namespace = %q, want %q", opts.Namespace, "img.acct1")
	}
	if opts.Credentials == nil {
		t.Error("Credentials nil with an API key set, want non-nil")
	}
	if opts.ConnectionOptions.TLS == nil {
		t.Error("TLS nil with an API key set, want non-nil")
	}
}
