# Proposal

## Why

REST can browse application definitions and manage individual user-role assignments, but cannot yet inspect or provision a user's application access. The pinned SDK already provides compound access commands, allowing operators to grant access with selected roles or revoke access and all active roles in one backend-owned operation.

## What Changes

- Add `GET /api/v1/users/{user_id}/applications` using `UserService/ListUserApplicationAccess` and exact `iam.user.read` permission, returning stored ACTIVE/REVOKED history, precise pagination and the target user's entity version.
- Add `POST /api/v1/users/{user_id}/applications/{application_code}/grant` using `ApplicationService/GrantApplicationAccess`, requiring BOTH `iam.user.application.grant` and `iam.user.role.grant`, string-decimal `role_ids`, reason, target `Version` and `Idempotency-Key`.
- Add the corresponding `/revoke` operation using `ApplicationService/RevokeApplicationAccess`, requiring BOTH `iam.user.application.revoke` and `iam.user.role.revoke`, reason and the same required headers. Revoke targets the whole application, not a caller-selected role subset.
- Preserve backend atomicity, eligibility, system-application protection, current authority, optimistic concurrency, no-op and committed replay. Support access-only changes with empty changed-role lists and revocations affecting more than 20 roles.
- Add users-owned adapters, compatible shared SDK wrappers, no-store/browser isolation, deterministic root-level scenarios, HTTP/TLS and concurrency coverage, and REST documentation during implementation.
- Bound list controls to page/limit/last_id/keyword/application_code/statuses with server-owned `application_id ASC` ordering. Access keyword uses the existing backend's literal case-insensitive CONTAINS over application_code/name; logical FULL_TEXT_TRIGRAM is not supported for this RPC and will not be sent or silently substituted after failure.

## Capabilities

### New Capabilities

- `user-application-access`: Independently authorized access history and compound application grant/revoke, including safe inputs/results, target versions, idempotency, transport/browser boundaries and regression guarantees.

### Modified Capabilities

None. Application catalog remains informational, user-role management retains selected-role-only commands, and user detail keeps its short embedded access view. The new capability does not change their permissions, search behavior or responses.

## Impact

- `internal/features/users/{domain,usecase,handler,gateway/authservice}`, app composition and early no-store matching; read wrapper in `user_client.go`, compound wrappers in `application_client.go` on the existing auth connection.
- Tests under `test/features/users`, `test/client/authservice`, `test/app`, `test/integration` and shared fixtures; README routes, permission matrix, request/response examples, search distinction and compound revoke guidance.
- No dependency, protobuf, environment, persistence, local JWT/replay storage, generic catalog framework, sibling implementation, Git history, frontend or deployment changes. No permission catalog/effective-access endpoint, application/role definition mutation or implicit user activation.
- Discovery verified pinned `api-contracts` structs/services, current grpc-auth access handlers/services/validation/search support, local validation decisions and existing user-role/catalog adapters. HTTP fakes and TLS bufconn will verify orchestration and wire contracts; real backend database transactions/search/cursor snapshots/audit/outbox and grant provisioning remain rollout checks.
