# jobtread-cli

Go-based JobTread CLI with a live JobTread Pave schema importer, runtime-generated resource commands, safe keychain-backed grant resolution, and token-optimized output defaults.

## Installation

### Download Binary

Download the latest release from [GitHub Releases](https://github.com/Robben-Media/jobtread-cli/releases).

### Build from Source

```bash
git clone https://github.com/Robben-Media/jobtread-cli.git
cd jobtread-cli
go build ./cmd/jobtread
```

## Current capabilities

- `jobtread pave query` for raw JobTread Pave queries.
- `jobtread sync import` to pull the live JobTread Pave schema into the local coverage manifest.
- Runtime-generated resource commands sourced from `internal/catalog/api_manifest.json`.
- Token-oriented output modes: `auto`, `json`, `toon`, `table`, and `raw`.
- JMESPath shaping via `--query`.
- Keychain-backed `createGrant` flow for authenticated generated commands.

## Commands

```bash
go run ./cmd/jobtread items summary
go run ./cmd/jobtread sync import
go run ./cmd/jobtread sync validate
go run ./cmd/jobtread pave query --public --body '{"version":{}}'
go run ./cmd/jobtread current-grants get --fields id
```

## Configuration

The CLI reads configuration from environment variables and matching flags.

```bash
export JOBTREAD_BASE_URL="https://api.jobtread.com"
export JOBTREAD_API_TOKEN="your-grant-key"
export JOBTREAD_AUTH_SCHEME="Bearer"
export JOBTREAD_KEYCHAIN_SERVICE="app.jobtread.com"
export JOBTREAD_MANIFEST="internal/catalog/api_manifest.json"
```

Credential resolution is fail-closed:

1. `JOBTREAD_API_TOKEN` as a grant key
2. `JOBTREAD_EMAIL` and `JOBTREAD_PASSWORD`
3. `JOBTREAD_KEYCHAIN_SERVICE`
4. Targeted macOS keychain lookup against likely JobTread service names

No broad keychain dump or enumeration is used.

If no grant key is provided, the CLI fetches `captchaToken`, signs it the same way JobTread’s app does, and calls `createGrant` with the targeted login credentials.

## Output behavior

- Default output is `--output auto`.
- `auto` chooses TOON for flat uniform arrays and compact JSON otherwise.
- Stdout prints the response body only.
- `--meta` and `--debug` send metadata to stderr.
- List commands default to one page unless `--all` is requested.
- Generated Pave commands keep default selections small; use `--fields`, `--select`, or `--query` to expand only when needed.

## Blacksmith CI

The repository workflow in [.github/workflows/ci.yml](.github/workflows/ci.yml) runs on:

```yaml
runs-on: blacksmith-2vcpu-ubuntu-2404
```
