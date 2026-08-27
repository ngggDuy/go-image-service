package authclient

import (
	"context"

	"go-image-service/gen/authpb"
	"go-image-service/internal/auth"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// Client is the httpserver's gRPC client for the auth service. It satisfies
// transport.Authenticator (Register, Login, Verify).
type Client struct {
	rpc  authpb.AuthServiceClient
	conn *grpc.ClientConn
}

func Dial(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{rpc: authpb.NewAuthServiceClient(conn), conn: conn}, nil
}

func (c *Client) Close() error { return c.conn.Close() }

func (c *Client) Register(ctx context.Context, email, password string) (string, error) {
	resp, err := c.rpc.Register(ctx, &authpb.RegisterRequest{Email: email, Password: password})
	if err != nil {
		return "", err
	}
	return resp.GetUserId(), nil
}

func (c *Client) Login(ctx context.Context, email, password string) (string, error) {
	resp, err := c.rpc.Login(ctx, &authpb.LoginRequest{Email: email, Password: password})
	if err != nil {
		// Anti-corruption layer: translate the gRPC status code back into the
		// domain error the rest of the app already understands.
		if status.Code(err) == codes.Unauthenticated {
			return "", auth.ErrInvalidCredentials
		}
		return "", err
	}
	return resp.GetToken(), nil
}

func (c *Client) Verify(ctx context.Context, token string) (string, error) {
	resp, err := c.rpc.Verify(ctx, &authpb.VerifyRequest{Token: token})
	if err != nil {
		return "", err // the middleware maps any error to 401
	}
	return resp.GetUserId(), nil
}
