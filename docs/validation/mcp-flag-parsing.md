# MCP companion flag parsing hardening

The companion adapter previously compared raw MCP property names with its root
flag exclusion map, then prefixed accepted names with `--`. A property such as
`deliver=file:dummy` therefore became a CLI root option. A separate value token
could also become another option when the named option was boolean.

This is parser correctness and hardening. It does not establish a distinct
unauthorized file-write or other permission boundary.

Structured property names now accept canonical ASCII letters/digits followed by
letters/digits, hyphens or underscores. Empty names, embedded equals, leading
hyphens, whitespace and control characters return an MCP error before companion
execution. Root options remain excluded. Values are emitted as one
`--name=value` token so the companion parser cannot reinterpret a value as another
option. Explicit false and empty string values remain explicit. CLI users can
still use ordinary equals/separated forms and declared shorthands; MCP properties
use canonical names.

The regression was run before the fix on main
`6f2d84b0497aa18d342dfbc25d2ddbcd5de3db47`. It failed because malformed keys
produced argv options, including `--deliver=file:dummy`. A second regression failed
before value binding: `full: "--deliver=file:dummy"` became two options, and an
explicit false value was omitted.

Tests call the real Cobra tree's flag parser without command execution or hooks.
They cover root equals/separated forms, missing/empty values, unknown root flags,
the absence of a deliver shorthand, import's `-i` shorthand, malformed names,
literal string values, boolean type mismatch, explicit booleans, all declared flag
names, root exclusions and the adapter's error result. No real configuration,
credentials, delivery target, MCP server or external service is used.

Validation commands (cached modules, `GOPROXY=off`, `GOSUMDB=off`,
`GOENV=off`, `GOTOOLCHAIN=local`, `GOMAXPROCS=2`, empty environment, private
HOME/TMP/build cache):

```sh
go test -mod=readonly -count=1 -timeout=120s ./...
go test -race -mod=readonly -count=1 -timeout=120s ./...
go vet -mod=readonly ./...
go build -mod=readonly -o /private/output/ ./cmd/...
```

The ordinary sandbox initially refused an existing `httptest` loopback bind with
OS EPERM. The normal approved local-test invocation permits only dummy test
listeners and uses no real API inputs. GitHub Actions is disabled; local gate
results are recorded in the PR without fabricating hosted statuses or changing
protection. The only tracked workflow closes PRs on PR events; no main-push
release or deployment workflow and no repository webhooks were present.

These adapter files are generated upstream. Preserve the name/value boundary and
its real-parser regressions if regenerating them; this change does not modify an
external generator.

Final local gate: full tests, race tests, vet and both command builds passed on
Go 1.27.3 darwin/arm64, with unchanged go.mod/go.sum. Final parser tests also
require empty strings to mark the option as explicitly set, preventing a default
empty value from hiding an omitted argument.
