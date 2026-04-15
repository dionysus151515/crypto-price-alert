package binance

import "testing"

func TestParseProxyURLAddsDefaultScheme(t *testing.T) {
	t.Parallel()

	u, err := parseProxyURL("127.0.0.1:7890")
	if err != nil {
		t.Fatalf("parseProxyURL returned error: %v", err)
	}
	if got := u.String(); got != "http://127.0.0.1:7890" {
		t.Fatalf("proxy URL = %q, want http://127.0.0.1:7890", got)
	}
}

func TestParseProxyURLRejectsMissingHost(t *testing.T) {
	t.Parallel()

	if _, err := parseProxyURL("http://"); err == nil {
		t.Fatal("expected error for invalid proxy url, got nil")
	}
}
