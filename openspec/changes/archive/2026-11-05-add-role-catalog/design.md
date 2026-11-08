# Design

## Context

See proposal.md for motivation and specs/role-catalog/spec.md for the behavior contract. Current observations:

- The repository has auth/users features and completed user creation/detail/profile/status/session flows. No roles feature or RoleService wrapper exists. Public auth is composed before protected users routes; legacy GET users keeps its numeric role-admin policy.
- Pinned api-contracts d32fd5c1bab4 already provides role.v1.RoleService.ListRoles/GetRole. ListRoles returns items, PageMeta and next_cursor; GetRole returns role and permissions. RoleSummary has 13 safe fields, optional description/timestamps, positive entity version, numeric role_code and affected_user_count. PermissionItem has five metadata fields, no status.
- Read-only inspection of grpc-auth role/read.go, handler/role_read.go and repositories/domain/catalog*.go confirms current iam.role.read authorization in the read snapshot, readable system/disabled roles, all-field role projection, default page 1/limit 20, exact filters, logical keyword/FULL_TEXT_TRIGRAM and query-bound composite cursors. Backend default sort would be ID ascending; the REST decision in docs/grpc-auth-validation-rules.md instead selects created_at DESC, updated_at DESC, id DESC, so REST must send it explicitly.
- That document selects role options value=id string, role_code as a separate string, and code - description/name labels. It keeps sort/projection/search mode server-owned and defers role date filters. Backend accepts wider Unicode text; new REST query text stays ASCII by the agreed input policy. Responses are historical metadata, not newly submitted names/descriptions.
- Shared pkg/pagination.Page uses Total int32, while paginator.PageMeta.total is int64. Existing pagination mapping must not be reused through a narrowing conversion or changed globally. Existing response Writer returns success HTTP 200; server/test registries are auth/common, and unregistered role Reasons use the shared safe fallback.
- Graphify routed composition/client/list mapping investigation; actual current source was verified. Root AGENTS.md is the layout authority: handler directly under roles/handler and DTOs in roles/domain. Generated graph stays untouched by this plan. Existing root layer runners, shared HTTP fixture and one TLS bufconn harness provide reusable verification patterns; the harness currently registers User/Profile/Auth/Session services, not RoleService.

## Goals / Non-Goals

**Goals:** One traceable read path per route, typed safe output, exact grant before business validation, lossless metadata, and a display-only options adapter using the same catalog page.

**Non-Goals:** A generic catalog framework, shared pagination migration, generic SDK options/builder, role writes, permission/application catalog APIs, user assignments, statuses options, local access/replay storage, FE implementation or grant/deployment automation.

## Decisions

### 1. A roles feature with two read usecases

Add internal/features/roles/{domain,usecase,handler,gateway/authservice}. Domain owns RoleActor, raw list input, normalized ListQuery, Role, Permission, Detail, Option and a small feature-owned Page[T] with int64 Total. It has no Fiber/generated/concrete-client imports. Page[T] has two present consumers, role list and options; it is not a cross-feature paging framework. Keep existing pkg/pagination and all old DTOs unchanged. Domain includes focused list/detail gateway contracts; one concrete roles gateway implements both.

List and Get usecases guard nil context/zero actor as unauthorized and restriction/missing-or-false exact iam.role.read as forbidden. Only then normalize/validate business input and dispatch once. Handler collects query values and path text; the list input is a copied map[string][]string and Get input is path text so business validation need not run before the grant. Typed normalized ListQuery reaches the gateway. Direct usecase tests verify invalid queries/targets cannot bypass guard ordering. Handler syntax/principal failures remain safe transport errors.

Use one List usecase for list and options; option shaping is an HTTP adapter concern, without a second options usecase or per-item GetRole. Clone returned slices/nullable values at the owning boundaries, following current user detail/profile isolation. No role-name gate, EvaluateAccess or permission/application lookup is needed. Backend still checks current authority after Me; REST's snapshot does not bypass it.

Alternative rejected: reuse the users list handler/group or domain DTOs. That would inherit legacy admin policy, couple feature ownership and risk narrowing metadata.

### 2. Fixed query policy and explicit normalization

List/options accept one value each for page/limit/last_id/keyword/application_code/statuses. Collect duplicates before conversion. Normalize into owned values with defaults page=1/limit=20, ASCII-digit positive int32 page and 1–100 limit, leading zeros permitted. Explicit blank numeric inputs are invalid. Unknown keys are ignored; the known unsupported controls listed in the spec fail even when empty, including field/sort/search overrides and role dates/exact code. This follows the documented deferred-filter rule while unknown authority claims remain ineffective.

Optional application scope is absent or a canonical lowercase [a-z][a-z0-9_]* value of 1–50 bytes; explicit blank/mixed case/whitespace does not silently broaden the query. Status CSV accepts ACTIVE/DISABLED, trims ordinary spaces per token, rejects empty/unknown/non-ASCII tokens, deduplicates and sorts as backend normalization does. Omitted status has no filter for both views, without inventing grant-eligibility policy.

Keyword validates original ASCII printable/whitespace bytes and the 200-byte budget before normalization; ASCII tab/LF/VT/FF/CR are whitespace, other controls/NUL/DEL and non-ASCII fail. Collapse permitted whitespace like backend; blank means nil search. Nonblank produces PageQuery.Search{Fields:[keyword], Mode:FULL_TEXT_TRIGRAM}; do not send physical name/code search columns or accept a caller mode. Backend owns document lexemes, separators, literal search and ranking. Aggregate independent field errors in existing dictionary.FieldErrors with focused localized catalog definitions; retain EN/ID/ZH and safe input-free messages.

Send PageQuery.Sort=[created_at DESC,updated_at DESC,id DESC] and Fields empty/nil (backend returns all safe fields). Do not accept a client-defined sort grammar in this change. Copy every slice/pointer before SDK dispatch.

The only new bounded opaque-input convention is last_id <=2048 decoded bytes, consistent with existing session cursors. It may be empty, is not trimmed/decoded/parsed locally and is forwarded once-decoded exactly. Backend owns canonical cursor/query binding; a caller must reset its cursor when filters/keyword/sort change. List/options use the same query/projection/method, so the same cursor can be used between representations. No page loop, cursor fallback, query repair or refetch.

### 3. An additive SDK on the existing auth transport

Add internal/client/authservice/role_client.go with RoleCatalogService containing exactly generated ListRoles/GetRole signatures and guarded Client.ListRoles/GetRole methods. Default RoleService SDK is initialized on the same transport connection in the existing terminal constructor; retain every old dependency guard and signature. Add NewWithRoleCatalogService after NewWithUserCreationService with existing explicit dependencies plus the role SDK, preserving guard ordering. Add focused internal constructor/request/response catalog errors using available codes; do not widen Service or any existing user/session interface.

Wrappers own nil context/request/response, RequestContext and deferred cancellation, preserving dependency error identity. No type assertions, setters, generic options refactor, new connection, dependency/env change or mutable global client. Alternative rejected: a second role transport duplicates lifecycle/credential ownership even though this backend uses the configured auth connection.

The feature gateway validates its direct context/query/ID boundary without rerunning non-idempotent or raw normalization, constructs copied protobuf payloads, and removes inherited version/idempotency-key from copied outgoing metadata for ListRoles/GetRole. Unused HTTP Version/Idempotency-Key, including repeated/malformed values, are not parsed or forwarded. Existing bearer/locale/request ID/deadline/cancellation and environment service-token conflict policy remain intact. Do not derive service namespace or method authority from query/header/body; IAM/method grants remain backend/operator-owned.

### 4. Safe full role and detail mappings

One focused private role mapper is genuinely shared by list and detail. Map exactly the 13 documented fields. IDs/application IDs must be 1..2147483647, role_code a positive int32, version 1..MaxInt64 rendered with integer decimal formatting, status ACTIVE/DISABLED, and each present timestamp valid before UTC RFC3339Nano formatting. Absent description and timestamps become explicit JSON null; present empty description stays empty. Preserve response strings without applying new-input ASCII/name/code normalization, positive IDs distinct from numeric role codes, is_system=false, counts=0 and original order. System/IAM/disabled metadata is not blocked by mutation policy.

AffectedUserCount copies backend uint64 as a JSON number; it counts distinct ACTIVE assignments regardless of user/role/app/access state. It is not online/effective access and requires no extra RPC. Page[T] copies nonnegative int64 total as a JSON number without float/int32 conversion; version is the precision-sensitive string per the existing REST decision. Require page Meta, page>=1, limit 1..100, total>=0 and total_pages>=0. Preserve returned flags/cursor/page totals; do not recalculate, clamp an out-of-range page or equate an empty page with empty dataset. Empty items becomes []. Malformed success fails the whole response, never returns earlier valid rows or synthetic meta.

Detail data is {role:<safe Role>,permissions:[...safe five-field items]}. GetRole must match parsed target ID. Reject nil/invalid permission entries and application ID/code inconsistent with the owning role; preserve valid order/text without sorting/deduplicating into a new meaning. No permission status, users, assignments, auth_version, credentials or effective permission map. Detail clones nullable Role values and permission collections. Alternative rejected: return protobuf structs directly or reuse the user's role-assignment view; both have different public fields/meaning.

### 5. Options are display mapping of the same page

Handler maps each safe Role into exactly {value,label,code,role_code}; use strconv integer formatting for ID and role_code strings, never float conversion or a label as identity. Label uses exact code + " - " + description when pointer is nonnil and value != ""; otherwise name. Do not trim legacy response whitespace (including a whitespace-only nonempty description), translate names/codes or normalize punctuation/line breaks. Preserve pagination without another service call. HTTP messages are Roles retrieved successfully., Role retrieved successfully. and Role options retrieved successfully., all code SS/200 via shared Writer.

Document concrete examples where ID=10, role_code=3 and code=clm.legal_reviewer differ, and description nil/empty fallback to name. Options does not filter disabled/system roles or claim a target can be granted; FE may request statuses=ACTIVE, and later backend mutations independently enforce eligibility.

### 6. Composition, cache and verification boundary

Add internal/app/roles_routes.go and register after the current auth/users composition with route-specific authentication on each role GET. Register /roles/options before /roles/:role_id and /roles list without the legacy admin group. Mirror wiring in test/shared/fixtures/app.go. Dedicated roles handler has List/Options/Get and validated usecase/writer constructor dependencies, no BrowserPolicy. Keep auth public route ordering and legacy users gate unchanged; framework GET/HEAD policy follows the same authentication/read behavior, no custom aliases/writes.

Extend existing LifecycleNoStore before CORS/auth for the normalized roles collection and one-segment child paths (including options and malformed IDs); standalone handlers set no-store too. This follows existing target-user read policy without a new cache abstraction and leaves users collection GET/preflight unchanged. Cover missing bearer, CORS denial, Me/business failures, invalid input, GET preflight and cookies. This bearer read does not add mandatory Origin, CSRF or browser lifecycle effects.

Preserve response mapper/registry scope. Test role RNF and other SDK-encoded Reasons retaining public status/code and the current safe unregistered-Reason locale fallback; do not globally register role to change existing errors. Structured field errors and ceil Retry-After remain independent of HTTP status. Fakes verify behavior/order/mapping; extend the TLS harness with one RoleService registration and the existing call recorder to verify actual generated methods/payload/TLS metadata, connection reuse, earlier deadlines/cancellation/ended contexts, method/IAM/current authority denials and no retry.

Use controlled barriers for mixed list/options/detail and existing public/user/session calls with distinct credentials, filters/search/cursors, target IDs/numeric codes, nullable metadata, large totals/version, label fallback and independent errors/cookies. Root test runners/scenarios only. Add README examples and actual verification commands/results during apply; do not claim database search/cursor implementation, real grants or frontend/rollout verified by bufconn.

## Risks / Trade-offs

- [Backend default order and REST selected order differ] → send the documented three explicit descending fields, assert exact wire payload and reject caller overrides.
- [Shared int32 paging narrows role totals] → feature-owned Page[T] with int64 Total for list/options; regression old pagination contracts unchanged.
- [Optional metadata becomes malformed success] → distinguish valid null from invalid present timestamps and preserve nullable pointers/legacy text with isolation tests.
- [Options ID confused with role_code/code or grant eligibility] → four-field output, independent values, no ACTIVE-only default or extra grant/permission call.
- [User/admin group or parameter route captures role catalog] → standalone exact read routes/options first, exact iam.role.read and one-Me inventory tests.
- [Different query normalization invalidates a cursor] → use one query policy for both views and backend-owned binding, no local repair.
- [Backend supports more than REST query text/controls] → follow the explicit REST decisions, document 2048-byte cursor and explicit-empty rules; do not widen existing endpoints or mutate read text.

## Migration Plan

1. Apply roles domain/usecases, additive client and feature gateway, handlers/app/cache matcher/shared fixture, root unit/HTTP/TLS/concurrent tests and README. No dependency/schema/env/backend migration.
2. Run focused runners, then full test/build/vet, targeted race and strict OpenSpec plus scoped hash/whitespace review. Keep main specs/archive/generated artifacts and unrelated edits unchanged during apply.
3. Rollout is separate authorization: provision existing environment credential for exact RoleService ListRoles/GetRole methods and IAM namespace, grant eligible users iam.role.read, then smoke backend full-text/cursor behavior and FE selection. Rollback removes the three read routes/wiring; it writes no role/assignment/session state to undo. Main-spec sync/archive remains an explicit later workflow.
