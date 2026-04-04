# go-core Commit Conventions

Extends Conventional Commits with auth-API-specific types and scopes.

## Format

```
<type>(<scope>): <description>

[optional body]
[optional footer]
```

## Types

| Type | When |
|---|---|
| `feat` | New feature (endpoint, auth method, admin page) |
| `fix` | Bug fix |
| `security` | Security patch, vulnerability fix, hardening |
| `docs` | Documentation (README, Swagger annotations, comments) |
| `refactor` | Restructure without behavior change |
| `test` | Adding or updating tests |
| `chore` | Build, deps, CI, config |
| `deps` | Dependency updates (especially security-related) |

## Scopes

Pick the most specific scope that fits. Multi-scope is OK for cross-cutting changes: `feat(auth,middleware)`.

### Domain scopes
`auth`, `user`, `social`, `twofa`, `email`, `webauthn`, `oidc`, `rbac`, `webhook`, `session`, `sessiongroup`, `bruteforce`, `geoip`, `sso`, `admin`, `health`

### Package scopes  
`app`, `middleware`, `database`, `redis`, `models`, `dto`, `jwt`, `config`, `sms`

### Infrastructure scopes
`api`, `docker`, `build`, `ci`

## Security-First Messaging

This is an auth system — always mention security implications in auth-related commits:

```
security(middleware): fix JWT validation bypass with malformed tokens

Tokens with truncated signatures were accepted due to missing length
check before base64 decoding. Now validates full signature format.

BREAKING CHANGE: Invalid tokens return 401 instead of 500
```

For dependency updates that fix vulnerabilities, reference CVEs:

```
deps(security): update golang.org/x/crypto to v0.17.0

Addresses CVE-2023-45288, CVE-2023-45289
```

## Examples

```
feat(webauthn): add discoverable credential login flow
fix(twofa): recovery code not invalidated after use
security(session): enforce session group logout on token refresh
refactor(user): extract password validation to shared helper
test(oidc): add PKCE auth code exchange coverage
chore(docker): pin postgres image to 16.2-alpine
docs(api): update Swagger for brute-force config endpoints
```

## Pre-commit

```bash
make fmt       # format
make test      # tests pass
make lint      # linter clean
make security  # gosec + nancy
make swag-init # if API routes changed
```
