package grpcauth

import "testing"

func TestIsCloudRun(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"https://imageservice-abc123-as.a.run.app", true},
		{"imageservice:50051", false},
		{"localhost:50051", false},
		{"", false},
	}
	for _, tc := range tests {
		t.Run(tc.addr, func(t *testing.T) {
			if got := IsCloudRun(tc.addr); got != tc.want {
				t.Errorf("IsCloudRun(%q) = %v, want %v", tc.addr, got, tc.want)
			}
		})
	}
}

func TestDialTarget(t *testing.T) {
	tests := []struct {
		name, addr, want string
	}{
		{"cloud run url gains port 443", "https://svc-abc-as.a.run.app", "svc-abc-as.a.run.app:443"},
		{"trailing slash is trimmed", "https://svc-abc-as.a.run.app/", "svc-abc-as.a.run.app:443"},
		{"host:port passes through", "imageservice:50051", "imageservice:50051"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := DialTarget(tc.addr); got != tc.want {
				t.Errorf("DialTarget(%q) = %q, want %q", tc.addr, got, tc.want)
			}
		})
	}
}

func TestDial_LocalAddrSucceedsWithoutCredentials(t *testing.T) {
	// grpc.NewClient is lazy: it must construct a client for a plain host:port
	// without contacting anything and without needing Google credentials.
	conn, err := Dial(t.Context(), "localhost:50051")
	if err != nil {
		t.Fatalf("Dial(local) returned error: %v", err)
	}
	defer conn.Close()
	if conn.Target() != "localhost:50051" {
		t.Errorf("Target() = %q, want %q", conn.Target(), "localhost:50051")
	}
}
