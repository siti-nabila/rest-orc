# Design

## Context

See proposal.md for motivation and specs/application-catalog/spec.md for the behavioral contract. This design is required because the change crosses feature ownership, authentication, shared SDK injection, browser policy and integration transport.

Graphify routed discovery through current roles app/client/gateway nodes; source was then verified directly. Nearby role domain/query/usecase/handler/gateway files and root scenario runners establish the current pattern. `docs/grpc-auth-validation-rules.md` specifies applications code/name keyword search, id DESC, ACTIVE/DISABLED definitions and application-code options. The pinned `api-contracts` version `v0.0.0-20261023084653-d32fd5c1bab4` exposes ApplicationService.ListApplications with ApplicationItem's five scalar fields and PageMeta.total int64. There is no GetApplication RPC, description, version or timestamps in this response.

Read-only grpc-auth `internal/handler/catalog.go` and `internal/features/catalog/read.go` confirmed status mapping, IAM/method authorization, snapshot reads and current exact `iam.permission.read` checks. Catalog query normalization and keyword projection routing support the requested search/sort. Existing REST role catalog already strips mutation metadata, supports opaque cursors, preserves int64 totals and maps options locally. The reusable `pkg/pagination.Page` still has int32 total. The current latest injected auth client constructor is NewWithUserRoleManagementService, after the role catalog seam; default SDKs are initialized on the same connection in NewWithSelfSessionService.

## Goals / Non-Goals

**Goals:** Keep one understandable request path per representation, permission checks before business parsing, exact application-code selection and page fidelity, and existing transport/security/lifecycle behavior. Preserve independently evolving feature DTOs and pre-existing working-tree edits.

**Non-Goals:** No shared catalog framework, role/users refactor, cross-feature DTO import, generic pagination migration, speculative detail RPC/route or mutation interface, local database/cache/JWT state, new runtime config/dependency or generated graph changes. See the proposal for product scope exclusions.

## Decisions

### 1. Applications owns its contracts and one read usecase

Create `internal/features/applications/domain/` for an identity-only actor, raw multi-value input, typed list query, five-field application view, two-field option view and `ApplicationPage[T]` with int64 total. The page has two immediate consumers: application list and option representation; it does not widen legacy pagination or import roles-domain types. Scalar item copies and owned slices are sufficient; no clone interface or wrapper for a five-scalar item is needed.

Use one focused `ApplicationCatalogList` usecase behind a small applications-owned ListApplications gateway contract, shared by both HTTP operations. It authorizes nil/invalid principal, restriction and exact permission before query normalization, then dispatches once and returns an independent item slice. Use explicit exported handler/gateway/interface names (`ApplicationCatalogHandler`, `ApplicationCatalogGateway`, `ApplicationListUseCase`, `ApplicationCatalogClient`) to prevent the directory-basename Graphify collisions seen with generic Handler/Gateway/ListUseCase names.

Alternative: put applications inside roles/users or reuse their actor/page/handler DTOs. Rejected because definitions have their own identity/output/permission, and cross-feature coupling would obscure ownership. A separate options usecase adds no business responsibility.

### 2. Inherit catalog conventions with explicit application differences

Capture raw decoded values and duplicates before usecase validation. Support only page/limit/last_id/keyword/statuses; preserve the spec's defaults, positive decimal bounds, status canonicalization, raw ASCII search validation and localized independent errors. Typed query validation protects direct gateway calls without mutating inputs. Request full safe projection (no fields override), explicit id DESC, optional statuses and logical keyword FULL_TEXT_TRIGRAM. Backend resolves its code/name search document; never forward physical fields from the caller.

Record minor REST choices explicitly: the 2048 decoded-byte cursor budget and duplicate-key rejection match role catalog; known unsupported code/application_code/date/projection/search/sort controls fail even when empty. Other unknown authority claims are ignored. ApplicationFilter has only statuses, so no application scope field is forwarded. Omitted statuses remains unrestricted for both list/options; callers wanting ACTIVE options request it explicitly. Do not locally filter inactive/system definitions or change result order/total.

Alternative: expose PageQuery controls or silently accept application_code from role catalog. Rejected because it would promise unsupported behavior and let callers influence projection/search/sort. Avoid a broad generic query parser extraction; only reuse existing pure helpers/errors when semantics and public wording genuinely match, otherwise keep small feature-local validation.

### 3. Add one compatible ApplicationService SDK seam

Create `internal/client/authservice/application_client.go` with a one-method generated-SDK `ApplicationCatalogService` and ListApplications wrapper. Add an additive NewWithApplicationCatalogService after NewWithUserRoleManagementService, preserving every old constructor signature and guard order. Add its field/default in the current terminal NewWithSelfSessionService using `applicationv1.NewApplicationServiceClient(transport.Connection())`. Older constructors keep a functional default; no second connection, mutable setter or broadened legacy Service/role/assignment interface is required.

The wrapper retains nil context/request/response guards, existing RequestContext, deferred cancellation and original error identity. Any new internal nil-dependency/request/response errors use unused dictionary codes determined from the current registry during implementation; do not change shared public error mapping or introduce business policies in the client. The interface intentionally excludes GrantApplicationAccess/RevokeApplicationAccess.

Alternative: widen Service or add the new method to role/assignment mocks. Rejected because existing unrelated consumers do not need ApplicationService capabilities, and the default SDK already supplies the method on the current connection.

### 4. Map one backend page defensively and build options locally

Implement `ApplicationCatalogGateway` in `internal/features/applications/gateway/authservice/`, with direct request/response mapping. Copy outgoing metadata, delete inherited version/idempotency-key and keep original bearer/safe locale/request ID. No read precondition or request body is parsed. Preserve context/deadline and one configured service credential through the existing transport.

Reject nil response/meta/items, invalid positive PostgreSQL-range IDs, unsupported state or invalid page metadata with safe 500 and no partial data. Keep total int64 and every backend page/cursor/flag value, including nonzero-total empty pages. Read code/name are historical strings, not new write input: preserve bytes without trimming/casing/ASCII validation. This follows existing role text handling and avoids rejecting valid stored Unicode/whitespace. No fabricated version/description/timestamp/count/assignment/effective-access fields.

The handler's options representation loops only over this page, producing exactly `{value: app.Code, label: app.Code + " - " + app.Name}`. It preserves metadata and uses the same validated list usecase; no new RPC or extra lookup. Code identity differs from the numeric id, while the label is display-only. No empty-description fallback exists because the SDK has no description field; an empty name yields the literal concatenation.

Alternative: fetch every page, derive value from ID, enrich options through role/access queries or invent metadata from related models. Rejected because those would change identity, pagination, authority, cost or public contracts.

### 5. Compose only two exact bearer routes with early no-store

Place handler files directly under `internal/features/applications/handler/` with feature-domain HTTP DTOs. Add focused `internal/app/applications_routes.go` wiring and invoke it from current feature route composition after existing registrations; use routes.router plus one routes.authentication middleware per route. Explicitly register literal options and the collection only, leaving unsupported paths unregistered; do not add a detail placeholder or reuse the legacy users admin group.

Extend LifecycleNoStore before CORS/auth for the two exact application paths, using its existing casing/trailing-slash normalization and respecting the configured API prefix. Standalone handler entry points set no-store before principal extraction. Retain exact-origin GET preflight with zero RPCs, optional-Origin bearer requests and zero cookie/token/CSRF side effects. Existing ListUsers GET/preflight cache behavior stays intact. The shared fake-service fixture must wire the same new feature without duplicate authentication.

Alternative: no-store only at handlers or route registration under a role-admin group. Rejected because early errors would be cacheable or exact-permission-only operators would be incorrectly denied.

### 6. Verify at existing seams with controlled transports

All authored tests stay under root `test/`: applications domain/usecase/handler/gateway runners, client/app additions, fake-backed `test/integration/applications`, and existing authservice TLS scenarios. Each implementation slice lands its meaningful boundary tests and relevant README content rather than postponing all tests/docs to the end.

Add a dedicated integration ApplicationServer embedding the generated unimplemented server, implementing ListApplications only, and register ApplicationService once alongside the existing UserService/RoleService/Profile/Auth/Session servers on the current controlled TLS connection. Use fresh scalar fixtures and copied slices, verify exact wire payload/method, original bearer, one environment credential, deadlines/locale/request metadata and one connection. Controlled barriers mix list/options with role/user/assignment/public flows to expose query/result/error/cookie leakage; use synctest for timing/cancellation. Do not use live backend calls or Docker.

Alternative: a second gRPC transport or real backend integration. Rejected because neither is needed to verify this read adapter and would reduce determinism or complicate lifecycle.

## Risks / Trade-offs

- [Permission name is easy to infer incorrectly] → Pin exact iam.permission.read in spec, usecase and positive/negative HTTP/TLS tests; user/role/application-write grants alone do not qualify.
- [Options mistaken for access eligibility] → Keep ACTIVE/DISABLED/system definitions readable and document that selection does not provision or promise access.
- [ID/code confusion] → Assert options value equals application.code even when numeric ID is different, and keep exactly two item fields.
- [int64 totals narrowed through shared pagination] → Feature-owned page and boundary tests above int32 through JSON and TLS.
- [Query rules drift between list/options] → One parser/usecase/gateway query and equivalent-query/cursor tests; only item mapping differs.
- [New constructor breaks prior injection/defaults] → Add after current constructor tail, preserve old guards, test old constructors and default SDK on the existing connection.
- [No-store or data leaks during early/concurrent failures] → Exact-path middleware tests, standalone handler checks and controlled mixed-request race coverage.
- [Generic downstream Reason descriptions remain localized fallbacks] → Preserve explicit public status/code and existing shared mapper behavior; no registry expansion.

## Migration Plan

Implementation adds adapters/routes/tests/documentation without changing dependencies, configuration or backend schema. The configured service must already allow the exact ApplicationService method/IAM scope and the actor must possess the current catalog grant; this change does not provision them. No deployment/release action is included. A later normal rollback can remove the new route composition/adapters while existing routes continue; these catalog reads create no assignment or access state to undo.
