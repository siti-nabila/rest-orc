# Implementation verification

Verified on 2026-11-04. Go commands used `GOWORK=off GOCACHE=/tmp/rest-orc-go-cache` and the RTK CLI.

- Domain/usecase focused suites: 82 passed.
- Shared auth client suite: 52 passed, including old/new constructor guards, request presence, deadlines, cancellation and cleanup.
- Users gateway suite: 14 passed.
- Users handler/app/auth handler suites: 130 passed.
- Auth/users HTTP integration suites: 367 passed before the additional six retry-authority cases; the final cases are included in the full and race suites below.
- Generated-SDK TLS bufconn suite: 238 passed, including mixed concurrent admin/self/public auth/ListUsers requests on one reused connection.
- `go test -race ./test/features/users/... ./test/client/authservice/... ./test/integration/users/... ./test/integration/authservice/... ./test/integration/auth/... ./test/app/... ./test/features/auth/handler/...`: 890 passed across 33 packages, no races.
- `go build ./...`: passed.
- `go test ./...`: 1283 passed across 95 packages.
- `go vet ./...`: passed.
- `openspec validate add-admin-session-management --strict`: passed.
- `git diff --check`, formatting, added-file whitespace/layout and protected-file hash checks: passed.

Reviewed production DTO/validation/usecase/gateway/client/handler/composition and affected root tests. The three users-owned routes use per-route authentication before the legacy role-admin users group. Each eligible request performs one Me then one exact target-session RPC, requires the exact read/revoke permission, separates actor/target/entity version, and preserves safe backend outcomes without cookie effects. Metadata copies remove inherited mutation preconditions, select exactly one mutation key, retain caller context, and do not mutate the parent. Pagination remains int64 and backend entity versions remain exact decimal strings. Backend replay/current-authority/conflict decisions are forwarded without local replay or retries. No-store precedes CORS/auth and also applies to standalone handlers.

The modified self-session delta retains the existing three scenarios and adds separate target administration without changing self-session URLs, authorization or cleanup behavior. No main-spec sync/archive was performed.

Hashes of go.mod, go.sum, .env.example, existing main specs, archived changes and generated Graphify artifacts match the pre-implementation baseline. All authored tests are under root test/. No sibling repository, branch, commit, PR, release or deployment was changed, and Docker was not run.

Limits: verification uses controlled SDK fakes and generated SDK over TLS bufconn, not a real backend or browser. Operator method/IAM grants and exact user permissions remain prerequisites for rollout. Callers must obtain an authoritative target entity version through a source available to them; this change adds no GetUser/detail/version-source API and never substitutes actor Me.version. Revocation and idempotency persistence remain backend-owned.
