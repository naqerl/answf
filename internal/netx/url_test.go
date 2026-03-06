package netx

import "testing"

func TestNormalizeHTTPURL(t *testing.T) {
	t.Parallel()

	got, err := NormalizeHTTPURL("example.com/a")
	if err != nil {
		t.Fatalf("NormalizeHTTPURL returned error: %v", err)
	}
	if got != "https://example.com/a" {
		t.Fatalf("unexpected URL: got %q", got)
	}
}

func TestNormalizeWSEndpoint(t *testing.T) {
	t.Parallel()

	got, err := NormalizeWSEndpoint("https://browserless.example")
	if err != nil {
		t.Fatalf("NormalizeWSEndpoint returned error: %v", err)
	}
	if got != "wss://browserless.example" {
		t.Fatalf("unexpected endpoint: got %q", got)
	}
}
