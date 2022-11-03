package grpc

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"golang.org/x/net/http/httpproxy"
	"golang.org/x/net/proxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

const DefaultGRPCTimeout = 1 * time.Minute

func ConnectWithTimeout(ctx context.Context, address string, timeout time.Duration) (conn *grpc.ClientConn, err error) {
	conn, err = grpcConnection(address)
	if err != nil {
		return nil, fmt.Errorf("GRPC connection failed: %v", err)
	}

	if timeout == 0 {
		timeout = DefaultGRPCTimeout
	}

	if err := waitForGRPCWithTimeout(ctx, conn, timeout); err != nil {
		return conn, fmt.Errorf("GRPC timeout: %v", err)
	}

	return conn, nil
}

func waitForGRPCWithTimeout(ctx context.Context, conn *grpc.ClientConn, timeout time.Duration) error {
	if conn == nil {
		return fmt.Errorf("nil connection")
	}
	state := conn.GetState()
	if state == connectivity.Ready {
		return nil
	}
	ctx2, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		select {
		case <-ctx2.Done():
			return fmt.Errorf("timed out waiting for ready state")
		default:
			state := conn.GetState()
			if state == connectivity.Ready {
				return nil
			}
		}
	}
}

func grpcConnection(address string) (*grpc.ClientConn, error) {
	dialOptions := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	proxyURL, err := grpcProxyURL(address)
	if err != nil {
		return nil, err
	}

	if proxyURL != nil {
		dialOptions = append(dialOptions, grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			dialer, err := proxy.FromURL(proxyURL, &net.Dialer{})
			if err != nil {
				return nil, err
			}
			return dialer.Dial("tcp", addr)
		}))
	}

	return grpc.Dial(address, dialOptions...)
}

func grpcProxyURL(addr string) (*url.URL, error) {
	// Handle ip addresses
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	url, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	// Hardcode fields required for proxy resolution
	url.Host = addr
	url.Scheme = "http"

	// Override HTTPS_PROXY and HTTP_PROXY with GRPC_PROXY
	proxyConfig := &httpproxy.Config{
		HTTPProxy:  getGRPCProxyEnv(),
		HTTPSProxy: getGRPCProxyEnv(),
		NoProxy:    getEnvAny("NO_PROXY", "no_proxy"),
		CGI:        os.Getenv("REQUEST_METHOD") != "",
	}

	// Check if a proxy should be used based on environment variables
	return proxyConfig.ProxyFunc()(url)
}

func getGRPCProxyEnv() string {
	return getEnvAny("GRPC_PROXY", "grpc_proxy")
}

func getEnvAny(names ...string) string {
	for _, n := range names {
		if val := os.Getenv(n); val != "" {
			return val
		}
	}
	return ""
}
