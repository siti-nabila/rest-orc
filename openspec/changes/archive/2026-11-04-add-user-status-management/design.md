# Design

## Context

See proposal.md for motivation. Current integration points and constraints:

- Users composition registers per-route session/detail authentication before the broad legacy role-admin `/users` group. Public auth and the shared `/me` group are established earlier. Dedicated detail/session handlers avoid widening the ListUsers/profile constructor.
- Principal exposes cloned permissions, actor ID and restriction marker. Existing users-domain validation already provides decimal target parsing, byte-digit checks, session key/version rules and safe localized catalogs.
- `internal/client/authservice/user_client.go` wraps GetUser behind a one-method interface; existing session interfaces are separate. Constructors default SDKs on one reusable connection and offer additive full-dependency injection points. All old interfaces/signatures must remain valid.
- The pinned SDK provides ChangeUserStatusRequest with UserId, optional Status, Reason, and optional ExpectedUserVersion; response has UserId, Status, Version uint64 and Changed. No dependency/protobuf update is needed.
- Read-only grpc-auth inspection of handler/user_management.go, features/user/{change_status,management_transaction}.go and repositories/domain/user_management.go confirms mandatory key/version, reason trim, status presence, backend transitions and same-status no-op. Current permission is checked inside the transaction before replay; replay precedes current target version comparison. Status change owns revocation/audit/outbox there.
- docs/grpc-auth-validation-rules.md explicitly selects PATCH `/users/{user_id}/status` and raw ASCII printable plus CR/LF for reason. docs/user-management-spec.md has an older POST reference. The route proposed to and selected by the user is PATCH; reconcile the status reference during implementation documentation without introducing a POST alias.
- Shared mapper uses auth/common registries. Published `user` errors carry explicit public code/http_status through the SDK encoder; unknown Reason descriptions currently use the safe localized fallback, and public field messages are retained. This change does not broaden global registry behavior or replace that algorithm.
- Graphify revision f824b74379b1eb2a6eed01a2204c2c5d6ec0a054 was consulted for module routing; current uncommitted feature source was verified directly. No generated graph edits are needed.

## Goals / Non-Goals

**Goals:** A junior-readable users handler → usecase → gateway → shared SDK flow that separates actor/target, validates only REST syntax and body policy locally, and preserves authoritative command/replay results.

**Non-Goals:** A local transition engine, local self/last-admin policy, user/session cache, automatic target reads, generic permission framework or broad constructor/response-mapper refactor. Detail read and status mutation are separate requests, not an atomic workflow.

## Decisions

### 1. Separate users status operation and per-route authentication

Add UserStatusActor{UserID, MustChangePassword, Permissions}, a small ChangeUserStatusGateway, focused usecase, and dedicated UserStatusHandler directly under users/handler. Derive actor only from Principal. Usecase guards nil context/zero actor as 401, then restricted/exact iam.user.status.update missing or false as 403 before domain body validation/business gateway. Roles, wildcards, read/session permissions and target=self are not bypasses. HTTP syntax failures may return 400 before usecase eligibility; valid unauthorized requests never dispatch the mutation.

Composition in internal/app/user_status_routes.go registers PATCH `/users/:user_id/status` with route-specific authentication, after public auth and before the broad role-admin group. Mirror this in test/shared/fixtures/app.go. Keep ListUsers, detail and existing session composition intact. A new prefix auth group or registration in routes.admin is rejected because it risks duplicate Me and an incorrect role gate. Do not expand ListUsers/profile/detail handler constructors.

### 2. Explicit JSON inputs with presence and small domain validation

Use users-owned ChangeUserStatusBody with pointer string Status/Reason so missing and null are required errors, while wrong JSON types remain parsing failures. Require a JSON object; empty body, malformed JSON, top-level null/array/scalar fail 400. Decode only body fields: do not bind query/path/header claims into this DTO. Unknown fields are ignored; no extra duplicate-JSON-member framework is introduced beyond the ordinary JSON decoder. Existing CORS policy is sufficient; this bearer mutation does not add an Origin/CSRF requirement.

Construct ChangeUserStatusRequest with path UserID, pointer body fields, selected Version and IdempotencyKey. Reuse ParseUserDetailID/ValidateUserDetailID for the same ID contract without renaming old validators. Apply the following rules in a focused NormalizeAndValidate method returning a copied normalized request:

- Status must be present and exactly PENDING/ACTIVE/SUSPENDED/DEACTIVATED; no trim, uppercase conversion, default or UI-label mapping.
- Validate every raw reason byte before trimming: allow 0x20–0x7e plus CR/LF, reject tab/NUL/DEL/other controls and all non-ASCII, even at the ends.
- Trim permitted surrounding whitespace, require 3–500 bytes afterward, preserve internal spaces/CR/LF/punctuation unchanged. Required/min/max/invalid-character messages use the existing catalog/locale and accumulate independent status/reason field errors. Do not echo reason or key.
- Validate numeric target/version/key for direct usecase/gateway consumers as well. Transport parse errors may fail before body validation; aggregation is required for independent body rules once transport inputs parse.

Avoid a generic validation engine or cross-feature DTO imports. Key/version rules may be reused where semantics match; do not change current session validators/error identities just to rename them. Any new status/reason field definitions belong to the existing localized registry.

### 3. Header preconditions belong only to the selected mutation

Read Idempotency-Key and Version case-insensitively with multiplicity checks. Each must have exactly one parsed value; identical duplicates also fail 400. Key is nonblank 1–200 bytes printable ASCII 0x20–0x7e, allowing spaces inside a nonblank value and commas as opaque content. Preserve the parsed key without trim/generation/hash/UUID restriction. Version is nonempty ASCII digits 1–MaxInt64 without sign/trim/float conversion; leading zeros are accepted numerically.

The caller supplies target entity version, commonly from GET detail, independently of actor Me.version. Body/query/header alternate identity/version/key sources are ignored. Missing/invalid/repeated headers cannot be repaired from detail, Me, query or body.

Gateway copies outgoing metadata, removes inherited version/idempotency-key, then sets exactly one selected idempotency-key. Version is a present ExpectedUserVersion pointer in the protobuf payload, not metadata; Status also has explicit presence and Reason contains the normalized body value. Copy pointer values so SDK/dependency mutation cannot alter caller-owned request state. Preserve safe unrelated metadata, original bearer, locale/request ID, earlier deadlines/cancellation, and the existing env-owned service token policy including fail-closed conflicts.

### 4. Additive SDK adapter and focused gateway

Add UserStatusService with only the generated ChangeUserStatus signature, a client field, wrapper in existing user_client.go and a default generated UserService SDK on the existing connection after constructor nil guards. Add NewWithUserStatusService accepting the current explicit dependencies plus status SDK, building through NewWithUserDetailService and checking the new nil dependency afterward. Retain all old signatures and do not widen Service/UserDetailService/session interfaces. The wrapper owns nil context/request/response guards and RequestContext/deferred cleanup. No type assertion, mutable setter, options framework or new transport layer.

Users gateway change_status.go owns domain↔protobuf mapping and safe response checks. It never invokes GetUser, EvaluateAccess, ListUsers or a session RPC. Gateway and usecase preserve error identity, independent request values and zero/one dispatch behavior. Use the shared mapper for all HTTP errors; no handler/business HTTP status decisions beyond parsing failures and existing auth guards.

### 5. Backend owns transitions, safety and replay

Syntactic status validation accepts all four codes because backend accepts a same-status no-op, including PENDING→PENDING. Document backend transitions for context, but do not implement them in REST:

| Current | Different allowed next states |
| --- | --- |
| PENDING | ACTIVE, DEACTIVATED |
| ACTIVE | SUSPENDED, DEACTIVATED |
| SUSPENDED | ACTIVE, DEACTIVATED |
| DEACTIVATED | ACTIVE |

All valid same-state requests are dispatched once; backend decides no-op/version/replay outcomes. Returning to PENDING from another state is backend conflict, not a local transition decision. Self suspend/deactivate and last-active-admin protection remain backend checks, including concurrency and current authority. REST does not infer policy from Me actor status, target=actor or prior detail.

Each caller retry still performs Me then one ChangeUserStatus. Backend performs current authority revalidation before committed replay, and owns fingerprint, transaction, locks, audit/outbox and session/refresh revocation for SUSPENDED/DEACTIVATED. No REST second revocation call, replay cache, transaction emulation or retry. Map backend VC/IC/IST/RB/LAR conflicts and SIF denial using encoded public status/code plus existing safe fallback/field policy, without adding ad hoc public descriptions or changing global registry behavior.

### 6. Exact safe output without browser recovery inference

HTTP 200/code SS/message `User status updated successfully.` returns a direct data object with exactly numeric user_id, status string, version decimal string and changed boolean. Validate successful response is nonnil, user_id is positive PostgreSQL-range and matches target, status is one of the four codes, and version is 1–MaxInt64. Reject malformed success with safe 500 and no partial outcome. Keep false changed explicit and preserve backend status/version; do not compute input+1, require version growth or compare committed replay to current target state.

No login_required, must_change_password, auth_version, credential, reason, full profile or synthesized side-effect fields. Do not inject BrowserPolicy: success/no-op/replay/errors/target=self never set or clear cookies or issue tokens. This also applies to activation/reactivation; the caller must perform ordinary login separately when needed, and protected requests continue to rely on backend verification. A backend revocation is not undone by a REST rollback.

### 7. Cache/error boundaries and compatibility

Extend LifecycleNoStore matcher for `/users/<segment>/status` under the API prefix before CORS/auth, including malformed IDs and OPTIONS; standalone handler also sets no-store. Existing exact CORS allowlist/headers already support PATCH, Version and Idempotency-Key. No environment changes or broadened authentication.

Shared http-error-mapping remains unchanged: preserve explicit backend status/public code/localized fields/Retry-After, native timeout 504/cancel 499, generic localized fallback for unknown Reason/raw internals, and no automatic retry. Add SDK-encoded user-catalog error fixtures rather than asserting raw private status messages or assuming the user registry is globally configured. Existing detail/session errors retain their prior mapping.

README will show independent read/status permissions, GET target→PATCH with exact Version/key, non-atomic/stale-conflict behavior, no automatic refetch, and correct PATCH-only route reference. Existing public Register/Login (including PENDING registration), ListUsers role gate, detail, Me restricted policy, password/profile and all lifecycle/self/admin-session contracts remain unchanged. Literal me target is invalid numeric input, not a self alias; no unversioned or POST alias is registered.

## Risks / Trade-offs

- [Reason normalization hides invalid bytes] → raw ASCII/CR/LF check before trim; tests include Unicode/control at edges and 2/3/500/501-byte normalized bounds.
- [Actor version substituted for target] → separate path/header/body inputs and distinct-version HTTP/TLS cases, no detail lookup within mutation.
- [Frontend interprets activation as login or suspension as cookie cleanup] → exact four-field result, no BrowserPolicy, no credential issuance on all outcomes.
- [A same-status no-op is dropped or replay rejected by local version arithmetic] → one RPC for every eligible request, raw result precision and replay tests.
- [Last-admin/self/transition policy reimplemented inconsistently] → backend-only policy with encoded IST/SIF/LAR outcomes and controlled concurrency; fake tests do not claim real backend transaction verification.
- [Global user-catalog registration changes unrelated error descriptions] → preserve existing mapper/registries and explicit public-code/status/fallback policy.
- [Broad users middleware causes duplicate authentication] → per-route authentication before legacy group, production route inventory and shared HTTP/TLS ordering regressions.

## Migration Plan

1. Implement additive users/client/routes with root tests and README per layer; no backend/schema/dependency/env migration.
2. Verify fake SDK HTTP and existing generated UserService TLS bufconn registration, exact optional payload presence, selected metadata, deadline/cancellation and backend no-op/replay/safety errors. Test separate detail→status calls without read permission inheritance or hidden refetch.
3. Run mixed concurrent status/detail/session/public/ListUsers cases, targeted race, build/full test/vet, strict OpenSpec and scoped diff/hash audit. Keep main specs/archive/generated artifacts unchanged during apply.
4. For separately authorized rollout, operator provisions `/user.v1.UserService/ChangeUserStatus`, IAM namespace and exact user status permission. Real backend/FE smoke tests and deployment are separate. Rollback removes the additive route but cannot restore already revoked sessions or reverse committed backend status/audit/idempotency effects.
