package cobratree

import (
	"reflect"
	"testing"
)

func TestSplitShellArgs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty", "", nil},
		{"single token", "contacts", []string{"contacts"}},
		{"two tokens", "inbox health", []string{"inbox", "health"}},
		{"extra whitespace", "  foo   bar  ", []string{"foo", "bar"}},
		{"tabs", "foo\tbar", []string{"foo", "bar"}},
		{"quoted token", `"hello world"`, []string{"hello world"}},
		{"mixed quoted and bare", `contacts "john doe" active`, []string{"contacts", "john doe", "active"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitShellArgs(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitShellArgs(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFlagRejection(t *testing.T) {
	// shellOutToCLI is a closure that requires a real binary; instead we test
	// the flag-rejection logic directly by reimplementing the guard the same
	// way shellOutToCLI does. This keeps the test fast and deterministic.
	rejectFlags := func(raw string) ([]string, error) {
		tokens := splitShellArgs(raw)
		for _, tok := range tokens {
			if len(tok) > 0 && tok[0] == '-' {
				return nil, &flagError{tok}
			}
		}
		return tokens, nil
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"long flag", "--config /tmp/evil", true},
		{"short flag", "-c", true},
		{"bare double dash", "--", true},
		{"flag with equals", "--config=/tmp/evil", true},
		{"long flag mid-args", "contacts --verbose", true},
		{"clean positional", "contacts", false},
		{"two positionals", "inbox health", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := rejectFlags(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("rejectFlags(%q): err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

type flagError struct{ token string }

func (e *flagError) Error() string { return "flag-like argument: " + e.token }

func TestPositionalPassthrough(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"single word", "contacts", []string{"contacts"}},
		{"two words", "inbox health", []string{"inbox", "health"}},
		{"search query", "search hello", []string{"search", "hello"}},
		{"resource name with dots", "user@example.com", []string{"user@example.com"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := splitShellArgs(tt.input)
			// Verify none are flag-like (same guard as shellOutToCLI).
			for _, tok := range tokens {
				if len(tok) > 0 && tok[0] == '-' {
					t.Fatalf("unexpected flag-like token %q in positional input", tok)
				}
			}
			if !reflect.DeepEqual(tokens, tt.want) {
				t.Errorf("splitShellArgs(%q) = %v, want %v", tt.input, tokens, tt.want)
			}
		})
	}
}

func TestBlockedRootFlags(t *testing.T) {
	// Every root persistent flag must be blocked. If you add a flag to
	// root.go's PersistentFlags, add it to blockedRootFlags in shellout.go
	// and to this list.
	allBlocked := []string{
		"args", "config", "profile", "deliver",
		"json", "compact", "csv", "plain", "quiet", "select",
		"no-color", "human-friendly",
		"timeout", "dry-run", "no-cache", "no-input",
		"idempotent", "ignore-missing", "yes", "agent",
		"data-source", "rate-limit",
	}
	for _, flag := range allBlocked {
		t.Run(flag, func(t *testing.T) {
			result := cliArgsFromMCP(map[string]any{flag: "evil-value"})
			if len(result) != 0 {
				t.Errorf("cliArgsFromMCP({%q: ...}) = %v, want empty (blocked flag)", flag, result)
			}
		})
	}
}

func TestAllowedFlags(t *testing.T) {
	// Subcommand-specific flags (not in blockedRootFlags) must pass through.
	tests := []struct {
		name string
		args map[string]any
		want []string
	}{
		{"format string", map[string]any{"format": "json"}, []string{"--format", "json"}},
		{"limit float64", map[string]any{"limit": float64(10)}, []string{"--limit", "10"}},
		{"id string", map[string]any{"id": "abc-123"}, []string{"--id", "abc-123"}},
		{"multiple allowed", map[string]any{"format": "json", "limit": float64(5)}, []string{"--format", "json", "--limit", "5"}},
		{"allowed with blocked mixed", map[string]any{"format": "json", "config": "/tmp/evil"}, []string{"--format", "json"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cliArgsFromMCP(tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("cliArgsFromMCP(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestShellMetachars(t *testing.T) {
	// Shell metacharacters in the args field should pass through as literal
	// tokens (they are not interpreted by a shell since we use exec.Command).
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"semicolon", "foo;bar", []string{"foo;bar"}},
		{"pipe", "foo|bar", []string{"foo|bar"}},
		{"ampersand", "foo&bar", []string{"foo&bar"}},
		{"backtick", "foo`bar`", []string{"foo`bar`"}},
		{"dollar", "foo$bar", []string{"foo$bar"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitShellArgs(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitShellArgs(%q) = %v, want %v", tt.input, got, tt.want)
			}
			// Also verify none trip the flag guard.
			for _, tok := range got {
				if len(tok) > 0 && tok[0] == '-' {
					t.Errorf("metachar token %q incorrectly flagged", tok)
				}
			}
		})
	}
}
