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
