# Proposal

## Why

REST can select roles and manage user-role assignments, but callers cannot yet browse application definitions or obtain application codes for FE selection. Exposing the existing ListApplications RPC as a catalog and options page provides that foundation for later application-access management without provisioning access.

## What Changes

- Add `GET /api/v1/applications` for paginated application definitions and `GET /api/v1/applications/options` for FE choices.
- Require an unrestricted bearer actor with exact `iam.permission.read=true`, matching the pinned backend contract. Each eligible request performs one Me and one ListApplications.
- Support bounded page/limit, normalized keyword, ACTIVE/DISABLED status CSV and an opaque cursor, with explicit id DESC order and server-owned search/projection/sort.
- Return the five safe application fields id/code/name/status/is_system; options contain value=`application.code` and label=`code - name`, preserving one page's int64 totals and pagination.
- Add applications-owned adapters, one generated-SDK wrapper on the existing auth connection, early no-store, safe errors and deterministic root-level tests/documentation during implementation.
- Keep application detail/mutations, user application-access list/grant/revoke, permission catalog, effective permissions, statuses options, frontend and deployment outside this change. No SDK upgrade or new RPC is needed.

## Capabilities

### New Capabilities

- `application-catalog`: Authorized application definition list and FE options with bounded query rules, exact safe response shapes, secure request transport, browser policy and compatibility guarantees.

### Modified Capabilities

None. Role catalog and user-role management keep their existing inputs, outputs, permissions and RPC ordering. Existing auth/user/session/error/credential policies remain intact; application definitions do not imply assignments or effective access.

## Impact

- New `internal/features/applications/{domain,usecase,handler,gateway/authservice}` feature and app route composition; additive authservice client seam/default on the shared transport and no-store path matching.
- Root `test/features/applications`, `test/integration/applications`, client/app/TLS scenarios and README examples. The controlled TLS harness will register one ApplicationService alongside its existing services.
- No dependency/configuration/storage changes, generic catalog framework, broad pagination refactor, sibling writes, generated Graphify edits or changes to existing public contracts.
- Discovery verified current role catalog source/specs, users assignment client constructor tail, query/options decisions in `docs/grpc-auth-validation-rules.md`, pinned ApplicationService/ApplicationItem protobufs and read-only grpc-auth catalog authorization/query mapping. The current SDK has no GetApplication method; this change deliberately exposes only list and options.
