# API Coverage Strategy

This repository is structured to cover the complete JobTread API once the authoritative endpoint inventory is available.

## Current state

- `jobtread request METHOD PATH` can call any JobTread endpoint directly.
- `internal/catalog/api_items.yaml` seeds the coverage catalog that future generation can expand.
- CI validates the CLI bootstrap on each push and pull request.

## Why the inventory is not generated yet

JobTread's public documentation entrypoint is visible, but the machine-readable endpoint inventory was not exposed anonymously during bootstrap. Generating named commands without the authoritative docs would force endpoint guessing.

## Recommended next action

1. Export or capture the full JobTread API item list from the authenticated developer docs or sandbox.
2. Normalize the inventory into `internal/catalog/api_items.yaml`.
3. Add a generator that creates resource-specific Cobra commands from that catalog.
4. Add contract tests against a sandbox account for each API item.
