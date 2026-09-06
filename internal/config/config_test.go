package config

import "testing"

func TestLoad_Defaults(t *testing.T) {
	// Empty values make getenv fall back to its defaults. t.Setenv restores
	// the environment automatically when the test ends.
	for _, k := range []string{"HTTP_ADDR", "IMAGE_SERVICE_ADDR", "DATABASE_URL", "TEMPORAL_ADDRESS", "TASK_QUEUE"} {
		t.Setenv(k, "")
	}

	cfg := Load()

	tests := []struct {
		name, got, want string
	}{
		{"HTTPAddr", cfg.HTTPAddr, ":8080"},
		{"ImageServiceAddr", cfg.ImageServiceAddr, "localhost:50051"},
		{"TemporalAddress", cfg.TemporalAddress, "localhost:7233"},
		{"TaskQueue", cfg.TaskQueue, "image-resize"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("TEMPORAL_ADDRESS", "temporal:7233")
	t.Setenv("TASK_QUEUE", "custom-queue")

	cfg := Load()

	tests := []struct {
		name, got, want string
	}{
		{"HTTPAddr", cfg.HTTPAddr, ":9090"},
		{"TemporalAddress", cfg.TemporalAddress, "temporal:7233"},
		{"TaskQueue", cfg.TaskQueue, "custom-queue"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestListenAddr(t *testing.T) {
	tests := []struct {
		name, port, fallback, want string
	}{
		{"PORT unset falls back", "", "50051", ":50051"},
		{"PORT set wins", "8080", "50051", ":8080"},
		{"PORT set wins over other fallback", "9090", "50052", ":9090"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PORT", tc.port)
			if got := ListenAddr(tc.fallback); got != tc.want {
				t.Errorf("ListenAddr(%q) = %q, want %q", tc.fallback, got, tc.want)
			}
		})
	}
}

func TestLoad_CloudDefaults(t *testing.T) {
	for _, k := range []string{"TEMPORAL_NAMESPACE", "TEMPORAL_API_KEY", "OBJECT_STORE_USE_SSL", "OBJECT_STORE_ENSURE_BUCKET"} {
		t.Setenv(k, "")
	}

	cfg := Load()

	if cfg.TemporalNamespace != "default" {
		t.Errorf("TemporalNamespace = %q, want %q", cfg.TemporalNamespace, "default")
	}
	if cfg.TemporalAPIKey != "" {
		t.Errorf("TemporalAPIKey = %q, want empty", cfg.TemporalAPIKey)
	}
	if cfg.ObjectStoreUseSSL {
		t.Error("ObjectStoreUseSSL = true, want false by default")
	}
	if !cfg.ObjectStoreEnsureBucket {
		t.Error("ObjectStoreEnsureBucket = false, want true by default")
	}
}

func TestLoad_CloudOverrides(t *testing.T) {
	t.Setenv("TEMPORAL_NAMESPACE", "img.acct1")
	t.Setenv("TEMPORAL_API_KEY", "secret-key")
	t.Setenv("OBJECT_STORE_USE_SSL", "true")
	t.Setenv("OBJECT_STORE_ENSURE_BUCKET", "false")

	cfg := Load()

	if cfg.TemporalNamespace != "img.acct1" {
		t.Errorf("TemporalNamespace = %q, want %q", cfg.TemporalNamespace, "img.acct1")
	}
	if cfg.TemporalAPIKey != "secret-key" {
		t.Errorf("TemporalAPIKey = %q, want %q", cfg.TemporalAPIKey, "secret-key")
	}
	if !cfg.ObjectStoreUseSSL {
		t.Error("ObjectStoreUseSSL = false, want true")
	}
	if cfg.ObjectStoreEnsureBucket {
		t.Error("ObjectStoreEnsureBucket = true, want false")
	}
}

func TestLoad_CloudUnparseableFallback(t *testing.T) {
	tests := []struct {
		name, envVar, envVal string
		fieldBool            bool
		field                string
	}{
		{
			"OBJECT_STORE_USE_SSL unparseable falls back to false",
			"OBJECT_STORE_USE_SSL",
			"not-a-bool",
			false,
			"ObjectStoreUseSSL",
		},
		{
			"OBJECT_STORE_ENSURE_BUCKET unparseable falls back to true",
			"OBJECT_STORE_ENSURE_BUCKET",
			"garbage",
			true,
			"ObjectStoreEnsureBucket",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(tc.envVar, tc.envVal)
			cfg := Load()

			var got bool
			switch tc.field {
			case "ObjectStoreUseSSL":
				got = cfg.ObjectStoreUseSSL
			case "ObjectStoreEnsureBucket":
				got = cfg.ObjectStoreEnsureBucket
			}

			if got != tc.fieldBool {
				t.Errorf("%s = %v, want %v (from unparseable %q)", tc.field, got, tc.fieldBool, tc.envVal)
			}
		})
	}
}
