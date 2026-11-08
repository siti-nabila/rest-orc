# Proposal

## Why

Operators can now inspect a user and select role IDs from the role catalog, but REST cannot yet read assignment history or grant/revoke selected roles. The pinned auth SDK already exposes these operations. Adding their REST adapters completes this user-management flow while retaining backend ownership of current authority, application access, optimistic concurrency and idempotency.

## What Changes

- Add `GET /api/v1/users/{user_id}/roles` with exact `iam.user.read`, returning paginated assignment history and the target user's entity version.
- Add `POST /api/v1/users/{user_id}/roles/grant` and `/revoke` with exact `iam.user.role.grant` and `iam.user.role.revoke`, respectively. Commands require string-decimal `role_ids`, a reason, one target `Version` and one `Idempotency-Key`.
- Reject duplicate role IDs within a payload, including numerically equivalent strings. Preserve eligible backend no-ops and committed replays.
- Add users-owned domain/usecase/handler/gateway adapters and three generated-SDK wrappers on the existing connection, with safe metadata, no-store, nullable assignment provenance and precise versions/counts.
- Add deterministic root-level scenario, HTTP/TLS and isolation coverage plus REST documentation during implementation.
- Keep application access provisioning, activation, role-definition writes, effective permissions, frontend work and deployment outside this change. Commands make no catalog/detail/permission/session calls.

## Capabilities

### New Capabilities

- `user-role-management`: Authorized assignment list/grant/revoke, bounded input, target version/idempotency, safe assignment history and command results, transport/browser policy and regression guarantees.

### Modified Capabilities

None. Existing user detail retains its short embedded role view; role catalog retains its read-only definition/options operations. Their permissions and responses are unchanged. Existing status/session/profile and shared error/credential policies are reused without changing their requirements.

## Impact

- Users feature, app composition and early no-store route matching; additive auth client SDK interface/constructor and wrappers; local dictionary entries only where genuinely required.
- Tests under `test/features/users`, `test/client/authservice`, `test/app` and `test/integration`; README endpoint/permission/body/header examples.
- No dependency upgrades, new runtime configuration, connection, persistence, JWT/replay cache, sibling modifications or generated Graphify edits.
- Read-only discovery verified `docs/grpc-auth-validation-rules.md`, current users/status and roles/catalog flows, pinned UserService protobufs and grpc-auth assignment handlers/services. REST intentionally rejects payload duplicates even though the backend normalizes them; assignment statuses are ACTIVE/REVOKED rather than catalog ACTIVE/DISABLED.
