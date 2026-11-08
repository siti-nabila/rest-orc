# Design

## Context

See `proposal.md` for motivation. Current users adapters already isolate actor/target, normalize raw input, forward one business RPC, sanitize copied metadata and return safe domain DTOs. `registerUsersRoutes` registers permission-based operations before the broad legacy admin group. `LifecycleNoStore` runs before CORS/authentication. Application catalog and compound access operations share a generated ApplicationService, but serve different consumers.

Read-only discovery on 2026-11-06 verified:

- `go.mod` pins `api-contracts v0.0.0-20261023084653-d32fd5c1bab4`; its cached generated UserService includes ListUserApplicationAccess, and ApplicationService includes GrantApplicationAccess/RevokeApplicationAccess with explicit optional expected_user_version and access_status. No SDK upgrade is needed.
- `internal/features/users/{domain,usecase,handler,gateway/authservice}/user_roles*`, `internal/client/authservice/{user_client,application_client,self_session_client}.go`, `internal/app/{users_routes,user_role_routes}.go` and existing root tests establish the implementation pattern.
- `docs/grpc-auth-validation-rules.md` sections 7/10/12/15 specify these routes, string role IDs, duplicate rejection, required reason/version/key and exact permissions. `docs/user-management-spec.md` also separates application access from role assignment.
- Sibling grpc-auth `internal/handler/application_access.go`, `internal/features/application/{list,grant,revoke,mutation}.go`, `internal/repositories/domain/{application_access,catalog,catalog_search}.go` and `docs/application-access-api.md` verify compound semantics and cursor/search support. The sibling is evidence only and remains unchanged. Its older API document is checked against current source and the pinned structs.
- Graphify at `graphify-out/graph.json` routes to existing app/users/catalog modules and reports commit `0754d27228e23085f1e96a8b390e7eedf4174998`. The dirty tree includes newer catalog/assignment implementations, so current source controls decisions; graph output is not regenerated.
- Main specs `application-catalog`, `user-role-management` and `user-detail` cover informational definitions, selected assignments and short embedded access respectively. This addition changes none of those contracts. Existing uncommitted code/spec/archive content is preserved.

## Goals / Non-Goals

**Goals:** Keep user application access in the users feature, express each endpoint's exact authority and typed inputs, reuse one verified connection, and adapt compound results without inheriting selected-role-only assumptions. Make external behavior testable at domain, usecase, HTTP and wire boundaries.

**Non-Goals:** No local eligibility lookup, transaction, audit/outbox write, replay cache, application activation, JWT/session storage, frontend selection UI, generic catalog abstraction or backend search extension. No implementation is authorized by this planning workflow.

## Decisions

### 1. Users ownership with one backend operation per request

Add focused `user_application_access.go` plus responsibility-based query/validation files in users/domain, users/usecase and users/gateway/authservice; add `user_application_access_handler.go` directly under users/handler. Use explicit names such as `UserApplicationAccessGateway` and operation-specific consumer interfaces. Domain DTOs have no Fiber, protobuf or concrete client imports. A single focused gateway can use all three RPC wrappers while each usecase consumes only its operation.

Compose `internal/app/user_application_access_routes.go` from `registerUsersRoutes` before `routes.admin(...)`. Each route uses the existing authentication middleware exactly once. User-level relationships belong here even though mutations use ApplicationService. Alternatives: putting the feature in applications would mix definitions with user relationships; implementing compound access through sequential existing role/access calls would lose atomicity.

| REST operation | Full RPC | Exact actor grants |
| --- | --- | --- |
| GET `/api/v1/users/{user_id}/applications` | `/user.v1.UserService/ListUserApplicationAccess` | `iam.user.read` |
| POST `/api/v1/users/{user_id}/applications/{application_code}/grant` | `/application.v1.ApplicationService/GrantApplicationAccess` | `iam.user.application.grant` AND `iam.user.role.grant` |
| POST `/api/v1/users/{user_id}/applications/{application_code}/revoke` | `/application.v1.ApplicationService/RevokeApplicationAccess` | `iam.user.application.revoke` AND `iam.user.role.revoke` |

An owned actor view is mapped only from request Principal; usecases reject nil context/zero actor as 401 and restriction/missing exact grants as 403 before domain validation. Existing handler-level malformed JSON/header syntax can fail earlier. No implicit user-read, permission-read, role-read or admin-role prerequisite is introduced for commands. Backend still checks current authority/method/IAM scope after Me.

### 2. Reuse validators where semantics match and keep commands distinct

Reuse existing users numeric target, reason, key and version primitives. Reuse or narrowly extract role selection/application-code parsing where cohesive without changing old behavior; do not route commands through the old UserRoleCommand type, which requires selected roles for both operations.

- user_id: path ASCII digits in 1–2147483647, accepting leading zeros. Literal `me` is invalid.
- application_code: path on commands, query filter on list; exact lowercase ASCII `[a-z][a-z0-9_]*`, 1–50 bytes, without trim/case correction. This preserves the local REST decision even though backend normalizes raw codes. Syntactically valid `iam` reaches backend system protection; do not locally label applications as system or resolve a numeric ID.
- Grant body: required `role_ids` array of 1–20 string-decimal IDs and required reason. Check submitted count first, reject numeric duplicates including `10`/`00010`, sort a copied numeric set; same-application/existence/status decisions stay remote.
- Revoke body: required reason only. Unknown fields, including role_ids, are ignored and never passed to the revoke RPC. Document and test that role_ids cannot narrow a whole-application revoke. Separate grant/revoke body types prevent accidental selected-role dispatch.
- reason: reject raw bytes outside printable ASCII plus CR/LF, trim surrounding space/CR/LF, require 3–500 bytes and preserve internal bytes. Aggregate independently invalid grant role_ids/reason when transport parsing succeeds.
- Commands require exactly one parsed Idempotency-Key and one Version, case-insensitively counted. Key is nonblank printable ASCII 1–200 bytes and opaque, including spaces/commas; version is ASCII decimal 1–MaxInt64 with leading zeros accepted, passed by explicit pointer without float conversion. Do not use body/query/Me version or generate keys.

Alternatives: backend-only duplicate normalization violates local REST decisions; accepting a revoke subset changes the backend contract; rejecting unknown revoke role_ids would depart from nearby unknown-field handling. Ignoring it is explicit and documented, never interpreted as a filter.

### 3. Fixed access-history query and an explicit search exception

Expose six list controls: page/limit/last_id/keyword/application_code/statuses, each at most once including identical duplicates. Follow existing defaults 1/20, page up to MaxInt32, limit up to 100, exact application-code validation, ACTIVE/REVOKED CSV canonicalization, safe ASCII keyword normalization and 2048-decoded-byte opaque cursor budget. Reject known fields/sort/search/search_fields/search_mode/date/code controls even when empty; ignore unrelated unknown claims. Request full projection by leaving fields unset and explicitly sort `application_id ASC`; never copy assignment `id ASC` or definition `id DESC`.

For nonempty normalized keyword send `Search{Fields: ["application_code", "name"], Mode: CONTAINS, Keyword: ...}`. Empty/whitespace-only keyword means nil search. Current grpc-auth catalog search explicitly excludes UserApplicationAccessCatalog from logical keyword FULL_TEXT/FULL_TEXT_TRIGRAM; its supported explicit-field path accepts CONTAINS/PREFIX and defaults to application_code/name. Select CONTAINS with fixed fields so FE still uses a single keyword and `%`, `_`, backslash remain literal backend input. No rank, full-text or prefix control is offered.

This is an explicit compatibility exception to the general keyword/full-text guidance in `docs/grpc-auth-validation-rules.md` (qualified there by RPC support). README must distinguish access history search from application catalog and role-assignment search. Alternatives: copying logical keyword/full-text produces BAD_REQUEST; changing grpc-auth search requires separately authorized sibling work; hidden fallback after a failed search creates extra calls and unpredictable cursor semantics. Existing catalog/assignment specs remain unchanged.

### 4. Safe list schema with target version and lossless pagination

Return 200/SS/`User application access retrieved successfully.` with exactly ten data fields: items, total, page, limit, total_pages, has_next, has_prev, next_cursor, user_id, version. Reuse the established users relationship-page approach without widening legacy shared int32 pagination. total is nonnegative int64; version is positive int64 encoded as a decimal string. Preserve backend page metadata, text, order and empty-page totals, with items=[] and no recalculation.

Each item has exactly eleven fields: application_id, application_code, name, application_status, is_system, status, granted_by, granted_at, revoked_by, revoked_at, revoke_reason. Application ID and present actor IDs are positive PostgreSQL INTEGER values. Definition state is ACTIVE/DISABLED, access state ACTIVE/REVOKED. `granted_at` is required for full-projection access history, matching current backend reader/DDL; map valid timestamp to UTC RFC3339Nano. Other four provenance fields are explicitly nullable, with present timestamps checked and backend strings unchanged. Do not impose request reason/name normalization on historical values or invent cross-field provenance. No assignment ID, definition version, permissions or credentials are added.

Reject nil response/metadata or nil item entries, invalid target/version/IDs/enums/timestamps, absent required granted_at or invalid page metadata as a whole safe 500. An absent or empty item collection remains a valid empty array. Optional pointers and collections are cloned. No catalog supplementation for never-granted applications or user-status eligibility filtering occurs. Alternatives: embedding the five-field user-detail view loses provenance/paging; serializing generated types risks omitempty/unsafe schema and integer precision; validating every historical string with write rules breaks legacy reads.

### 5. Compound command results need their own mapper

Return 200/SS and `Application access granted successfully.` or `Application access revoked successfully.` with exactly user_id, application_id, access_status, changed_role_ids, version, changed. user_id must match target; application_id must be positive PostgreSQL INTEGER. The SDK response contains numeric application ID but no code: validate its range and trust backend resolution, without a forbidden catalog lookup to prove the code/ID pair. version is positive int64 as a string; changed_role_ids is an owned numeric array, empty as [], strictly ascending/distinct/in-range.

Grant requires ACTIVE access_status and changed IDs within requested roles. Revoke permits REVOKED or explicit null access_status and all actual changed role IDs, without input subset or 20-role output cap. Do not call the existing `safeUserRoleCommand`, which enforces changed iff nonempty changed IDs and a requested subset for both operations. Here changed=false requires an empty array, while changed=true may have an empty array for access-only changes. Revoke may clean roles without a stored access row and return null status. Do not infer a result version from input+1 or repair inconsistent ordering/IDs. Validate nil/wrong-target/invalid-version/invalid-state/malformed outcomes before exposing data.

Backend alone owns grant eligibility (ACTIVE target, ACTIVE nonsystem non-IAM application, ACTIVE same-application roles), additive activation, pair identity/provenance preservation, whole-application revoke even for inactive targets/definitions, system protection and all audit/outbox/atomicity. No local pre-read or multi-RPC loop. Backend owns no-op, current-authority-before-replay, fingerprint/key scope and version conflict; every caller retry gets fresh Me and one eligible business RPC. Do not cache/retry uncertain outcomes.

### 6. Compatible SDK seams on the existing transport

Add a one-method UserApplicationAccessService read seam alongside user_client.go and a two-method ApplicationAccessService seam alongside application_client.go. These mirror two real generated services and keep ApplicationCatalogService and UserRoleManagementService unchanged. Add injectable constructor support after the existing NewWithApplicationCatalogService chain, preserving every prior signature/guard and supplying generated SDK defaults on the same transport in old constructors. Do not create a new connection or expand unrelated old interfaces to force test mocks to implement access operations.

Wrappers follow nil context/request/response checks, original error identity and RequestContext/deferred cancellation. Gateway uses the existing safe metadata copy primitive: remove inherited version/idempotency-key for reads, set only the selected command key for mutations and put Version only in payload. One environment-owned service token, original bearer, locale and request ID remain transport-owned. No env/dependency changes. TLS tests extend existing UserService/ApplicationService mocks, each already registered once; no duplicate server registrations.

### 7. Browser policy, safe errors and verification

Extend early no-store matching for normalized users/{target}/applications and users/{target}/applications/{code}/{grant|revoke}, including malformed numeric targets/codes and standard framework normalization. Standalone handlers also set no-store. Keep optional supplied Origin validation, allowed exact-origin GET/POST preflight with zero RPCs, bearer-only auth and zero cookie/token effects, including self/no-op/replay/error. Preserve legacy users GET/preflight cache behavior.

Use the shared response/error boundary unchanged. Assert encoded application Reason AASP/HTTP409 and unknown-Reason locale fallback as well as current-authority, not-found, validation, version/idempotency/resource-busy, dependency, RetryInfo, 504 and 499 cases. New infrastructure/validation errors use existing registries where possible and add only necessary local dictionary entries; do not widen the global downstream reason registry or return raw backend messages.

Follow root test runners/scenarios, strict argument/call/order assertions and controlled TLS bufconn. Cover differing actors/targets, both missing halves of each AND permission, 11/10/6-field response schemas, access-only changes, revoke >20 roles, absent-row cleanup, no-op/replay, metadata cloning, malformed results and mixed concurrent catalog/assignment/public flows. Assert original constructor defaults and unchanged catalog RPC payloads. Focused tests then full GOWORK=off tests/build/vet and targeted race are implementation acceptance, not evidence claimed by this proposal.

## Risks / Trade-offs

- [Accidentally reusing selected-role semantics] → Dedicated compound response mapping and scenarios for changed=true/empty IDs, null status and >20 revocations.
- [Unknown revoke role_ids can mislead callers] → Separate body DTO, explicit whole-application documentation and tests demonstrating that submitted subset claims never affect the command.
- [Search preference differs from this RPC's support] → Explicit CONTAINS design/spec/README contract and exact-wire tests; no silent fallback or sibling change.
- [Authorization changes after Me] → Local exact AND checks reduce unnecessary calls; current backend remains authoritative and can deny every replay.
- [Numeric application ID cannot be locally correlated with path code] → Check ID bounds and exact payload code, rely on backend resolution; no extra read or invented identity map.
- [A valid version becomes stale immediately] → Accept only caller Version and propagate conflict; no internal refresh or automatic mutation retry.
- [Dirty tree contains completed prior features] → Preserve baseline file hashes and scope all planning edits to this change; implementation review must include coexistence tests.
- [Mock success is weaker than live transactional evidence] → Document test limits and separate rollout validation of method/IAM/user grants, backend literal search/cursor snapshots, compound transactions/audit/outbox and retained replay. Docker/live systems are not required here.

## Migration Plan

Implementation is additive with no schema, config, dependency or data migration. After explicit apply authorization, implement and verify locally, then document deployment prerequisites for the existing configured service identity: Me plus the three exact method allowlist entries, IAM scope and the corresponding current actor grants. Existing ListApplications permission and method stay independent.

Deployment is outside this request. A later authorized rollout can expose the three routes after those prerequisites pass. Rolling REST back removes these additive routes and preserves committed backend access/role changes; rollback must never issue compensating grants/revokes automatically. Archive/main-spec synchronization and Git operations are separate actions.
