// Package grpcauth dials gRPC services in two environments with one API.
//
// Locally (docker-compose) services are plain host:port and are dialled
// without transport security, exactly as before. On Cloud Run a service is
// addressed by its full https:// URL, which serves two purposes at once: it
// is dialled over TLS on port 443, and the URL itself is the audience of the
// OIDC ID token that proves the caller holds roles/run.invoker on the callee.
package grpcauth

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"google.golang.org/api/idtoken"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/credentials/oauth"
)

// IsCloudRun reports whether addr is a Cloud Run https:// URL rather than a
// plain host:port.
func IsCloudRun(addr string) bool {
	return strings.HasPrefix(addr, "https://")
}

// DialTarget converts a configured address into a gRPC dial target.
// A Cloud Run URL loses its scheme and gains the implicit :443; anything else
// is already a dial target and passes through untouched.
func DialTarget(addr string) string {
	if !IsCloudRun(addr) {
		return addr
	}
	return strings.TrimSuffix(strings.TrimPrefix(addr, "https://"), "/") + ":443"
}

// Dial connects to a gRPC service, adding TLS and per-RPC ID-token
// credentials when the address is a Cloud Run URL.
func Dial(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	if !IsCloudRun(addr) {
		return grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// The audience must be the callee's full URL with no trailing slash, or
	// the receiving front end rejects the token.
	audience := strings.TrimSuffix(addr, "/")
	source, err := idtoken.NewTokenSource(ctx, audience)
	if err != nil {
		return nil, fmt.Errorf("id token source for %s: %w", audience, err)
	}

	return grpc.NewClient(
		DialTarget(addr),
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
		grpc.WithPerRPCCredentials(oauth.TokenSource{TokenSource: source}),
	)
}
