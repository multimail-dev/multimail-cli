package client

import (
	"testing"
)

// minimalClient returns a Client with only the fields cacheKey needs (none).
func minimalClient() *Client {
	return &Client{}
}

func TestCacheKeyDeterministic(t *testing.T) {
	c := minimalClient()
	params := map[string]string{"b": "2", "a": "1"}

	first := c.cacheKey("/test", params)
	for i := 0; i < 100; i++ {
		got := c.cacheKey("/test", params)
		if got != first {
			t.Fatalf("iteration %d: got %s, want %s", i, got, first)
		}
	}
}

func TestCacheKeyOrderIndependent(t *testing.T) {
	c := minimalClient()

	k1 := c.cacheKey("/test", map[string]string{"b": "2", "a": "1"})
	k2 := c.cacheKey("/test", map[string]string{"a": "1", "b": "2"})

	if k1 != k2 {
		t.Fatalf("different insertion order produced different keys: %s vs %s", k1, k2)
	}
}

func TestCacheKeyEmptyParams(t *testing.T) {
	c := minimalClient()

	first := c.cacheKey("/empty", map[string]string{})
	for i := 0; i < 100; i++ {
		got := c.cacheKey("/empty", map[string]string{})
		if got != first {
			t.Fatalf("iteration %d: got %s, want %s", i, got, first)
		}
	}
}

func TestCacheKeyExpectedValue(t *testing.T) {
	c := minimalClient()

	// sha256("/testa=1b=2") first 8 bytes hex-encoded = "4287d6c3d5103936"
	got := c.cacheKey("/test", map[string]string{"b": "2", "a": "1"})
	want := "4287d6c3d5103936"
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
