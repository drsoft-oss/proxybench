package checker

import (
	"testing"
)

func TestParseShadowsocksURL(t *testing.T) {
	// Modern format: ss://BASE64(method:password)@host:port
	// base64("chacha20-ietf-poly1305:mypassword") = "Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpteXBhc3N3b3Jk"
	modern := "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpteXBhc3N3b3Jk@192.168.1.1:8388"
	cfg, err := ParseShadowsocksURL(modern)
	if err != nil {
		t.Fatalf("ParseShadowsocksURL modern: %v", err)
	}
	if cfg.Method != "chacha20-ietf-poly1305" {
		t.Errorf("method = %q, want chacha20-ietf-poly1305", cfg.Method)
	}
	if cfg.Password != "mypassword" {
		t.Errorf("password = %q, want mypassword", cfg.Password)
	}
	if cfg.Host != "192.168.1.1" {
		t.Errorf("host = %q, want 192.168.1.1", cfg.Host)
	}
	if cfg.Port != "8388" {
		t.Errorf("port = %q, want 8388", cfg.Port)
	}

	// Legacy format: ss://BASE64(method:password@host:port)
	// base64("aes-256-gcm:secret@10.0.0.1:8389") = "YWVzLTI1Ni1nY206c2VjcmV0QDEwLjAuMC4xOjgzODk="
	legacy := "ss://YWVzLTI1Ni1nY206c2VjcmV0QDEwLjAuMC4xOjgzODk="
	cfg2, err := ParseShadowsocksURL(legacy)
	if err != nil {
		t.Fatalf("ParseShadowsocksURL legacy: %v", err)
	}
	if cfg2.Method != "aes-256-gcm" {
		t.Errorf("legacy method = %q, want aes-256-gcm", cfg2.Method)
	}
	if cfg2.Password != "secret" {
		t.Errorf("legacy password = %q, want secret", cfg2.Password)
	}
	if cfg2.Host != "10.0.0.1" {
		t.Errorf("legacy host = %q, want 10.0.0.1", cfg2.Host)
	}
	if cfg2.Port != "8389" {
		t.Errorf("legacy port = %q, want 8389", cfg2.Port)
	}
}

func TestParseShadowsocksURL_invalid(t *testing.T) {
	cases := []string{
		"ss://",
		"ss://notbase64!!!@host:port",
		"http://1.2.3.4:8080",
	}
	for _, c := range cases {
		_, err := ParseShadowsocksURL(c)
		if err == nil {
			t.Errorf("expected error for %q, got nil", c)
		}
	}
}
