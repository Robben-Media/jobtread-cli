# jobtread-cli

Go-based JobTread CLI bootstrap published as a private GitHub repository with Blacksmith-backed CI.

## Scope

- Go CLI foundation for a complete JobTread API surface area.
- Direct request command for immediate access to every documented endpoint.
- API item catalog that can be expanded or generated once JobTread's full developer inventory is exported.
- GitHub Actions workflow configured to run on Blacksmith runners.

## Commands

```bash
go run ./cmd/jobtread version
go run ./cmd/jobtread items list
go run ./cmd/jobtread request GET /v1/example --query limit=10
```

## Configuration

The CLI reads configuration from environment variables.

```bash
export JOBTREAD_BASE_URL="https://api.jobtread.com"
export JOBTREAD_API_TOKEN="your-api-token"
export JOBTREAD_AUTH_SCHEME="Bearer"
```

## Blacksmith CI

The repository workflow in [.github/workflows/ci.yml](.github/workflows/ci.yml) is set to:

```yaml
runs-on: blacksmith-2vcpu-ubuntu-2404
```

Blacksmith's current quickstart requires:

- A GitHub organization repository, not a personal repository.
- The Blacksmith GitHub App installed for the organization or repo.
- Any required GitHub IP allowlists updated if the organization restricts inbound access.

## Next build step

To move from bootstrap to full resource coverage, export the official JobTread API inventory from their developer docs or sandbox and generate resource-specific Cobra commands from that catalog.
