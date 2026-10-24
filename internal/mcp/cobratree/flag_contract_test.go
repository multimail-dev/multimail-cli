package cobratree

import (
	"context"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"path/filepath"
	"strings"
	"testing"

	"multimail-pp-cli/internal/cli"
)

func TestStructuredNamesCannotContainFlagSyntax(t *testing.T) {
	for _, name := range []string{"deliver=file:dummy", "config=dummy", "profile=dummy", "limit=5", "", "-deliver", "deliver value", "deliver\nvalue", "limit\x00"} {
		t.Run(name, func(t *testing.T) {
			if got, err := cliArgsFromMCP(map[string]any{name: true}); err == nil || len(got) != 0 {
				t.Fatalf("malformed structured name %q produced CLI arguments %q", name, got)
			}
		})
	}
}

// Parse the real companion tree without executing commands or pre/post hooks.
// Equals belongs to CLI syntax, never to an MCP property name.
func TestCompanionRootFlagParsingContract(t *testing.T) {
	for _, tc := range []struct {
		name    string
		argv    []string
		want    string
		wantErr bool
	}{
		{"equals", []string{"--deliver=file:dummy"}, "file:dummy", false},
		{"separated", []string{"--deliver", "file:dummy"}, "file:dummy", false},
		{"empty explicit", []string{"--deliver="}, "", false},
		{"missing", []string{"--deliver"}, "", true},
		{"no shorthand", []string{"-d", "file:dummy"}, "", true},
		{"unknown", []string{"--not-a-flag=x"}, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := cli.RootCmd()
			err := cmd.ParseFlags(tc.argv)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseFlags(%q) error=%v", tc.argv, err)
			}
			if err == nil {
				got, _ := cmd.Flags().GetString("deliver")
				if got != tc.want {
					t.Fatalf("got %q want %q", got, tc.want)
				}
			}
		})
	}
}

func TestCompanionImportInputShorthand(t *testing.T) {
	for _, argv := range [][]string{{"--input=dummy.jsonl"}, {"--input", "dummy.jsonl"}, {"-i", "dummy.jsonl"}, {"-i=dummy.jsonl"}} {
		cmd, _, err := cli.RootCmd().Find([]string{"import"})
		if err != nil {
			t.Fatal(err)
		}
		if err := cmd.ParseFlags(argv); err != nil {
			t.Fatal(err)
		}
		got, _ := cmd.Flags().GetString("input")
		if got != "dummy.jsonl" {
			t.Fatalf("%q parsed input=%q", argv, got)
		}
	}
}

func TestMalformedStructuredNameRejectedByHandler(t *testing.T) {
	handler := shellOutToCLI(func() (string, error) { return filepath.Join(t.TempDir(), "no-companion"), nil }, []string{"stale"})
	result, err := handler(context.Background(), mcplib.CallToolRequest{Params: mcplib.CallToolParams{Arguments: map[string]any{"deliver=file:dummy": true}}})
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("result=%v err=%v", result, err)
	}
	if len(result.Content) != 1 {
		t.Fatalf("unexpected content: %v", result.Content)
	}
	text, ok := result.Content[0].(mcplib.TextContent)
	if !ok || !strings.Contains(text.Text, "invalid structured parameter name") {
		t.Fatalf("missing parser rejection: %v", result.Content)
	}
}

func TestStructuredValuesRemainLiteral(t *testing.T) {
	for _, value := range []string{"", "name=value", "two words", "--deliver=file:dummy", "-i", "ordinary"} {
		argv, err := cliArgsFromMCP(map[string]any{"input": value})
		if err != nil {
			t.Fatal(err)
		}
		cmd, _, err := cli.RootCmd().Find([]string{"import"})
		if err != nil {
			t.Fatal(err)
		}
		if err := cmd.ParseFlags(argv); err != nil {
			t.Fatal(err)
		}
		got, _ := cmd.Flags().GetString("input")
		if got != value || !cmd.Flags().Changed("input") {
			t.Fatalf("%q changed to %q", value, got)
		}
		if cmd.Flags().Changed("deliver") {
			t.Fatal("literal value changed root flag")
		}
	}
}

func TestDeclaredFlagNamesMatchStructuredContract(t *testing.T) {
	root := cli.RootCmd()
	root.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		if !blockedRootFlags[flag.Name] {
			t.Errorf("root flag %q missing from exclusion", flag.Name)
		}
		if flag.Shorthand != "" {
			t.Errorf("root shorthand %q needs explicit MCP contract coverage", flag.Shorthand)
		}
	})
	var check func(*cobra.Command)
	check = func(cmd *cobra.Command) {
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			if !validStructuredFlagName(flag.Name) {
				t.Errorf("declared flag %q incompatible with MCP names", flag.Name)
			}
		})
		for _, child := range cmd.Commands() {
			check(child)
		}
	}
	check(root)
}

func TestWrongTypeBooleanValueCannotBecomeRootFlag(t *testing.T) {
	argv, err := cliArgsFromMCP(map[string]any{"full": "--deliver=file:dummy"})
	if err != nil {
		t.Fatal(err)
	}
	cmd, _, err := cli.RootCmd().Find([]string{"sync"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.ParseFlags(argv); err == nil {
		t.Fatalf("wrong-type boolean value accepted as CLI syntax: %q", argv)
	}
	if cmd.Flags().Changed("deliver") {
		t.Fatal("value changed root deliver flag")
	}
}

func TestStructuredBooleanValues(t *testing.T) {
	for _, value := range []bool{true, false} {
		argv, err := cliArgsFromMCP(map[string]any{"full": value})
		if err != nil {
			t.Fatal(err)
		}
		cmd, _, err := cli.RootCmd().Find([]string{"sync"})
		if err != nil {
			t.Fatal(err)
		}
		if err := cmd.ParseFlags(argv); err != nil {
			t.Fatal(err)
		}
		got, _ := cmd.Flags().GetBool("full")
		if got != value || !cmd.Flags().Changed("full") {
			t.Fatalf("explicit bool %v parsed as %v (argv=%q)", value, got, argv)
		}
	}
}
