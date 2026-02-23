package checker

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// ShadowsocksConfig holds parsed Shadowsocks connection parameters.
type ShadowsocksConfig struct {
	Host     string
	Port     string
	Method   string
	Password string
}

// ParseShadowsocksURL parses a ss:// URI into its components.
// Supported formats:
//   - ss://BASE64(method:password)@host:port
//   - ss://BASE64(method:password@host:port)  (legacy SIP002)
func ParseShadowsocksURL(rawURL string) (ShadowsocksConfig, error) {
	var cfg ShadowsocksConfig

	u, err := url.Parse(rawURL)
	if err != nil {
		return cfg, fmt.Errorf("parse url: %w", err)
	}

	// Try modern format: ss://BASE64(method:password)@host:port
	if u.User != nil {
		userInfo := u.User.Username()
		decoded, err := base64.RawURLEncoding.DecodeString(userInfo)
		if err != nil {
			// Try standard base64.
			decoded, err = base64.StdEncoding.DecodeString(userInfo)
			if err != nil {
				return cfg, fmt.Errorf("base64 decode userinfo: %w", err)
			}
		}
		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return cfg, fmt.Errorf("invalid method:password in userinfo")
		}
		cfg.Method = parts[0]
		cfg.Password = parts[1]
		cfg.Host, cfg.Port, err = net.SplitHostPort(u.Host)
		if err != nil {
			return cfg, fmt.Errorf("host:port: %w", err)
		}
		return cfg, nil
	}

	// Legacy format: ss://BASE64(method:password@host:port)
	fragment := strings.TrimPrefix(rawURL, "ss://")
	// Strip fragment / plugin opts.
	if idx := strings.IndexByte(fragment, '#'); idx != -1 {
		fragment = fragment[:idx]
	}
	decoded, err := base64.StdEncoding.DecodeString(fragment)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(fragment)
		if err != nil {
			return cfg, fmt.Errorf("base64 decode legacy: %w", err)
		}
	}
	// decoded = method:password@host:port
	atIdx := strings.LastIndexByte(string(decoded), '@')
	if atIdx == -1 {
		return cfg, fmt.Errorf("missing @ in legacy ss URI")
	}
	methodPass := string(decoded[:atIdx])
	hostPort := string(decoded[atIdx+1:])
	parts := strings.SplitN(methodPass, ":", 2)
	if len(parts) != 2 {
		return cfg, fmt.Errorf("invalid method:password")
	}
	cfg.Method = parts[0]
	cfg.Password = parts[1]
	cfg.Host, cfg.Port, err = net.SplitHostPort(hostPort)
	if err != nil {
		return cfg, fmt.Errorf("host:port legacy: %w", err)
	}
	return cfg, nil
}

// CheckShadowsocks performs a TCP connectivity check against a Shadowsocks server.
// Full protocol handshake is not performed (that would require an encryption
// library), but a successful TCP connection indicates the server is reachable.
// The function also sends a minimal probe to confirm the port is accepting data.
func CheckShadowsocks(address string, opts Options) Result {
	result := Result{Address: address, Protocol: ProtocolShadowsocks}

	cfg, err := ParseShadowsocksURL(address)
	if err != nil {
		result.Error = fmt.Sprintf("parse: %v", err)
		return result
	}

	hostPort := net.JoinHostPort(cfg.Host, cfg.Port)
	start := time.Now()

	conn, err := net.DialTimeout("tcp", hostPort, opts.Timeout)
	if err != nil {
		result.Error = fmt.Sprintf("tcp: %v", err)
		return result
	}
	defer conn.Close()

	// Send a few random bytes — a healthy SS server will keep the connection
	// open waiting for the encrypted handshake rather than immediately closing.
	conn.SetDeadline(time.Now().Add(2 * time.Second))
	_, _ = conn.Write([]byte{0x05, 0x01, 0x00}) // SOCKS5 greeting (will be garbage to SS)
	buf := make([]byte, 16)
	conn.Read(buf) //nolint:errcheck — ignore read result; we care about liveness

	result.Alive = true
	result.Latency = time.Since(start)
	return result
}
