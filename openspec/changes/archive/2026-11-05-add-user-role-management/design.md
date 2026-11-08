# Design

## Context

See proposal.md for motivation and specs/user-role-management/spec.md for the observable contract. This change spans authentication, users adapters, shared transport, routing and browser policy, so a design is needed.

Discovery consulted Graphify's users route/status/client paths, then current source, analogous root-level runners/scenarios, `docs/grpc-auth-validation-rules.md`, and the pinned `api-contracts` version `v0.0.0-20261023084653-d32fd5c1bab4`. The existing UserService already has ListUserRoles, GrantUserRoles and RevokeUserRoles. List returns 17-field assignment items, int64 total and top-level user_id/version. Commands return user_id/changed_role_ids/version/changed with expected_user_version presence and a metadata key.

Read-only grpc-auth inspection confirmed assignment catalog normalization, the role-name-only logical keyword projection (`docs/sql/catalog-keyword-search/04_verify.sql`), snapshot reads, current permission checks before replay, canonical sorted role IDs, atomic audit/outbox, grant eligibility and self/last-admin revoke guards. Its input normalizer deduplicates IDs; the REST document explicitly requires rejecting duplicates. The REST boundary will enforce the stricter rule without changing the downstream service.

Existing status adapters establish header/reason/version/metadata conventions. Role catalog establishes int64-safe page mapping and optional timestamps but belongs to another feature. `pkg/pagination.Page` uses int32 total and cannot represent assignment list totals safely. Existing user-detail embedded roles are intentionally smaller than assignment history.

## Goals / Non-Goals

**Goals:** Keep identity authorization before business validation, make every request traceable through the existing handler/usecase/gateway/client boundaries, preserve nullable history and exact versions/counts, and keep each batch one backend command on the shared transport.

**Non-Goals:** No generic management/catalog framework, cross-feature DTO import, broad pagination migration, new connection/config/dependency, local authority/transaction/replay state or refactor of existing feature behavior. Scope exclusions remain those in the proposal.

## Decisions

### 1. Users owns assignment flows and independently checked permissions

Add users-owned actor, raw query, typed query, command input, list/item and result types under `internal/features/users/domain/`. DTOs remain free of Fiber/protobuf/concrete client dependencies. Use three focused usecases (ListUserRoles, GrantUserRoles, RevokeUserRoles) with small consumer-owned gateway contracts. An identity-only actor contains verified user ID, restriction and permissions. Nil context/invalid actor fails authentication; restricted/missing exact grant fails before normalization and dispatch. Commands do not depend on read permissions. Transport syntax/header parsing can fail earlier, matching existing status behavior.

Add an explicitly named `UserRoleManagementHandler` directly under `internal/features/users/handler/` and `UserRoleManagementGateway` under `gateway/authservice/`. Name consumer interfaces explicitly, such as `UserRoleListUseCase`, to avoid the directory-basename Graphify collisions previously encountered with generic Handler/Gateway/ListUseCase names. Reuse pure users-owned ID/reason/header primitives only where semantics match; do not couple these commands to status DTOs. A small private command validator shared by grant/revoke has two immediate consumers; a generic strategy/factory is unnecessary.

Alternative: put assignments in roles or widen legacy users handlers/interfaces. Rejected because user targets, user versions and permissions own these operations and existing consumers should not inherit new methods.

### 2. Make list parsing explicit and keep backend query semantics intact

Capture repeated query values into a users-owned raw input map; parse business controls only after authorization. Match the spec's six supported keys, unsupported control rejection, bounds and independent localized errors. Typed query validation also protects direct gateway callers. Send full projection (no caller field selection), explicit id ASC, canonical assignment statuses, application_code and opaque cursor. Nonempty search uses the logical `keyword` field and FULL_TEXT_TRIGRAM; do not send raw `role.name` as a protobuf field. Backend's assignment document contains role name, distinct from the role catalog's name/code document.

Use a specific list data type containing the eight conventional page fields plus user_id/version, with total int64. Do not import role-catalog Page or change the existing shared pagination type. Preserve empty out-of-range pages and all backend metadata. Cursor budget 2048 decoded bytes and page/limit 1/20 defaults inherit the role-catalog REST convention; these are explicit new assignment decisions, not inferred backend maximums.

Alternative: reuse legacy ListUsers parsing/page or expose backend PageQuery controls. Rejected because totals would narrow and the supported filters, search, order and security boundary differ.

### 3. Normalize command inputs without concealing invalid input

Use presence-aware JSON fields: role_ids is a required string array and reason a required string. Reject non-object/wrong types without coercion, then accumulate independent role_ids/reason domain errors. Check the original role count 1–20; parse each positive PostgreSQL-range decimal, reject numeric duplicates before sorting an owned numeric copy. Thus `"10"` and `"00010"` are invalid together, while an isolated leading-zero value is accepted. No labels/code/role_code resolution or automatic catalog request.

Reason follows the status command's raw ASCII printable plus CR/LF allowlist, then trims only those permitted surrounding spaces/line breaks and checks 3–500 bytes. Check bytes before trim so invalid edge controls cannot disappear. Read exactly one parsed Version and Idempotency-Key using case-insensitive header multiplicity; reuse existing low-level version/key primitives. Keep key opaque and version exact uint64 constrained to positive int64. Unknown fields are ignored consistently with existing commands, including spoofed body/query preconditions.

Alternative: pass duplicates through for backend deduplication or use numeric JSON IDs. Rejected by the documented REST selection contract. Sorting an owned list agrees with backend fingerprint normalization while leaving input ownership intact.

### 4. Add a three-method SDK seam without widening existing consumers

Add `UserRoleManagementService` containing only the three generated UserService methods and an additive `NewWithUserRoleManagementService` after the current `NewWithRoleCatalogService` constructor chain. Preserve all existing signatures and dependency-guard precedence. Add a field/default SDK initialization in the existing terminal `NewWithSelfSessionService`, using its existing transport connection, so old constructors remain fully functional. Add wrappers in `internal/client/authservice/user_client.go`, retaining nil context/request/response guards, RequestContext, deferred cancellation and original error identity. Use dictionary-backed internal errors, choosing unused codes at implementation time.

The gateway maps exactly one SDK request/response per operation. Copy outgoing metadata before deleting inherited version/idempotency-key. List forwards neither; commands set only one selected key, with an owned expected_user_version pointer in payload. Preserve original bearer, safe locale/request ID, TLS, one configured credential and bounded request context. Do not introduce retries or alternate identity.

Alternative: widen Service, replace constructors, create another transport or use a mutable setter. Rejected because it burdens existing mocks/consumers or risks lifecycle/request isolation.

### 5. Map history and command results defensively without inventing policy

Map all 17 safe assignment fields directly. Preserve explicit nulls for optional access state, actor IDs, timestamps and revoke reason; clone pointers/collections. Validate required IDs, positive role_code, enum values and valid present timestamps, but do not apply write-time text rules to historical strings or require optional provenance based on assignment state. Format valid times in UTC with fractional precision. Map top-level target/version and int64 page totals without narrowing or floating-point conversion.

For commands validate target, positive int64 version, ascending unique changed IDs in the submitted set and changed/list consistency. Backend already enforces sorted changed IDs; rejecting malformed success is preferable to silently repairing it. Preserve the valid order/result version, including no-op/replay, without an expected+1 rule. Return safe 500 with empty data for malformed success and never partially map a page/batch.

Backend remains authoritative for ACTIVE target/role/application/application-access grant prerequisites; IAM's access exception; whole-batch transaction/audit/outbox; no-ops; current permissions before replay; version/idempotency conflicts; self platform-admin and last-admin revocation guards. Revoke does not inherit grant prerequisites. REST does not pre-read state, provision application access or infer login/session effects.

Alternative: infer eligibility/effective permissions or repair dependency output locally. Rejected because stale reads and local policy duplicate backend authority and can misrepresent committed results.

### 6. Wire precise bearer routes and early no-store policy

Add a focused `internal/app/user_role_routes.go` composition function, invoked from users route composition before its broad legacy admin group. Register the three exact routes with one existing authentication middleware per request; keep public auth registration/order intact. Direct handler methods set no-store before extracting context so standalone use has the same browser policy.

Extend `LifecycleNoStore`'s users-path segment matching for target/roles and target/roles/grant|revoke, including malformed target IDs, casing/trailing-slash normalization consistent with existing matcher behavior and OPTIONS. It already runs before CORS/auth. Preserve GET ListUsers cache behavior, exact-origin preflight and optional-Origin bearer semantics. No cookies, tokens, new CSRF proof or inferred browser cleanup.

Alternative: attach these routes under the legacy role-admin group or set no-store only inside handlers. Rejected because permission-only operators would be denied or early errors would remain cacheable.

### 7. Test at existing seams and document with each implementation slice

Use existing root `test/` runners with ordinary `.go` scenarios/mocks. Domain coverage owns parsing, boundary/error localization, duplicates/ownership and precise JSON shapes. Usecase tests own exact permissions/precedence and one dispatch. Client/gateway scenarios own constructor compatibility, mapping, metadata copying, malformed success, no-op/replay, error identity and cancellation. Handler/app/HTTP scenarios own exact routes/envelopes, preflight/no-store, wrong types, forged sources and zero cookie effects.

Extend the existing `test/integration/authservice/mocks.Server` with the three methods and register UserService once on the controlled TLS server. Keep RoleService registration unchanged. Exercise original bearer, exactly one environment credential, payload presence, deadline/locale/request metadata and controlled concurrent mixed requests; no live backend or Docker. Document endpoint permissions, assignment-vs-definition states, all safe fields, caller-owned Version/key, option-value strings, duplicate rejection, backend eligibility and no-op/conflict behavior in README as each slice lands.

Alternative: new real-service integration or a second UserService registration. Rejected because tests should be deterministic and duplicate registration would conflict.

## Risks / Trade-offs

- [Read history mistaken for effective access] → Keep assignment/role/application/access states distinct and document nullable history as information only.
- [REST duplicate policy is stricter than backend] → Explicit rejection tests for repeated/numerically equivalent values and 21-entry input before normalization.
- [Version can become stale after list/detail] → Caller supplies the exact target Version; preserve backend conflict with no refetch/retry.
- [Backend denies authority after successful Me] → Preserve current backend denial, including before replay; no cached outcome in REST.
- [Long constructor chain and existing uncommitted role-catalog work] → Add a compatible seam after the current tail, test old guards/defaults and preserve unrelated edits.
- [Graphify directory-basename node collisions] → Use explicit new type/interface names, inspect current source for every dependency, and leave generated files untouched.
- [int64 totals/versions or null provenance accidentally lost] → Dedicated typed page/result mapping plus large-boundary/null/empty tests and mutation-isolation checks.
- [Unregistered downstream user/role descriptions use generic locale fallback] → Retain shared safe mapper behavior and explicit status/code; no global registry expansion.

## Migration Plan

Implementation is additive: land domain/client/gateway/usecase/HTTP composition with scoped tests and documentation, then run focused and full verification. No database/configuration/SDK migration is needed. Release/deployment is outside this change. The configured service and operators must already have the required backend method/IAM/permission grants; REST does not provision them. If later rolled back through the normal release process, remove the new route composition/adapters while preserving existing routes; any backend-committed assignments remain authoritative and are not undone automatically.
