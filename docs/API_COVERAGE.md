# API Coverage Strategy

This repository now treats JobTread's live Pave schema at `https://api.jobtread.com/pave` as the coverage authority.

## Coverage pipeline

1. Fetch the live JobTread Pave schema with `jobtread sync import`.
2. Normalize the root operation inventory into `internal/catalog/api_manifest.json`.
3. Generate the runtime CLI surface from that manifest on the next invocation.
4. Enforce completeness with `jobtread sync validate`.

## Coverage rules

- Every root Pave operation from `schema.root.type.object` must appear in the manifest.
- Every manifest operation must have a generated CLI mapping.
- Empty manifests fail validation.
- Raw `jobtread pave query` remains the escape hatch, not the coverage system.

## Auth model

- Environment grant key first.
- Explicit `JOBTREAD_EMAIL` and `JOBTREAD_PASSWORD` second.
- Exact keychain service override third.
- Targeted likely-name keychain lookups last.
- No broad keychain enumeration.
- If only login credentials are available, the CLI mints a grant through `captchaToken` + `createGrant`.

## Output model

- Body-only stdout.
- Optional stderr metadata.
- `auto` output selection between compact JSON and TOON.
- JMESPath shaping before formatting.
- Generated Pave commands use minimal default selections to reduce token use.
