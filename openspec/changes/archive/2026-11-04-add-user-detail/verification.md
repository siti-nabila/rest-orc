# Implementation verification

Verified on 2026-11-04 using RTK. Go commands used `GOWORK=off GOCACHE=/tmp/rest-orc-go-cache`.

- Users domain suite: 66 passed.
- Users usecase suite: 20 passed, including existing ListUsers/profile/session regressions.
- Shared auth client suite: 60 passed, including additive constructor guard ordering, nil request/context/response, exact SDK request, metadata, bounded deadlines and cleanup.
- Users gateway suite: 31 passed, including explicit safe mapping, int64 version boundaries, profile text preservation, malformed successes and metadata/result isolation.
- Users handler/app/auth handler suites: 142 passed.
- Auth/users HTTP integration suites: 433 passed, including fresh reads, permission boundaries, detailed error mapping and caller-controlled detail→revoke-all/conflict without hidden reads or retry.
- Generated SDK TLS bufconn suite: 260 passed, including exact GetUser payload/method, TLS1.3, one reused connection, one service credential, request metadata, authority denial, conflicting service credentials and deadline/cancellation boundaries.
- `go test -race ./test/features/users/... ./test/client/authservice/... ./test/integration/users/... ./test/integration/authservice/... ./test/integration/auth/... ./test/app/... ./test/features/auth/handler/...`: 1012 passed across 33 packages with no races, including controlled mixed detail/admin-session/self-session/public auth/ListUsers scenarios.
- `go build ./...`: passed.
- `go test ./...`: 1405 passed across 95 packages.
- `go vet ./...`: passed.
- `openspec validate add-user-detail --strict`: passed.
- `git diff --check`, changed-file gofmt/whitespace/test-layout audit and protected-file hash verification: passed.

Reviewed the users-owned DTO/validation/usecase/gateway/handler/composition and shared generated SDK wrapper. The single new detail route authenticates once before the legacy role-admin users group, requires exact iam.user.read, selects only the path target, and returns safe explicit fields with target entity version formatted as a decimal string. Restricted actor eligibility is separate from target restriction/state. Nil/malformed or mismatched successful results fail closed without partial detail. Metadata copies remove read preconditions while preserving caller context; detail never mutates cookies or introduces local user/session state. No-store applies before CORS/auth as well as in the standalone handler.

Existing auth, ListUsers, profile/password and both session families retain their policies and call ordering. The caller's detail read followed by revoke-all consists of two independent requests with separate permissions; session mutations still perform no automatic GetUser/refetch/version arithmetic/retry. A concurrent target change may produce the existing backend conflict.

Compared work against a pre-implementation hash baseline to preserve prior uncommitted admin-session work. go.mod, go.sum, .env.example, existing main specs/archive and generated Graphify artifacts are unchanged. All authored tests are under root test/. No dependency upgrade, sibling/backend/protobuf change, Docker, branch/commit/PR/release/deployment, main-spec sync or archive was performed.

Limits: SDK fakes and generated SDK over TLS bufconn were used; no real backend/browser smoke test or deployment was performed. Operator GetUser method/IAM namespace grants and exact iam.user.read permissions remain rollout prerequisites. Detail is a source for a known target entity version, not a guarantee that later mutations will observe the same version.
