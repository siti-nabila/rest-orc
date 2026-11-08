# Design

## Context

See proposal.md for motivation. Current source provides the following integration points:

- `internal/app/users_routes.go` registers users self-profile, then per-route admin-session authentication, then the broad legacy `/users` role-admin group. Auth public Register/Login are registered first by `auth_routes.go`.
- `internal/features/users/handler/admin_sessions_handler.go` is the nearby example for a separate handler with per-route authentication and actor data copied from the current Principal. Principal already contains ID, MustChangePassword and cloned Permissions.
- `internal/client/authservice` has narrow generated-SDK interfaces for separate capabilities. Existing constructors end at `NewWithSelfSessionService`, which also defaults the generated admin-session client; `NewWithAdminSessionService` is an additive injection point. The base Service interface intentionally stays small for existing consumers/mocks.
- The pinned api-contracts version includes GetUserRequest{Id} and GetUserResponse{User}. User contains safe profile/state/assignment fields and legacy Password/Dob fields. Raw protobuf serialization is therefore unsuitable.
- Read-only inspection of grpc-auth `handler/user_detail.go`, `features/user/detail.go`, `features/auth_feature/read_permission.go`, `sessions/principal.go`, and `repositories/reader/user_detail.go` confirms exact `iam.user.read`, IAM namespace/method authority, unrestricted actor eligibility, a primary read snapshot, positive int64 entity version, required creation time and four management user statuses. The backend checks current authority again during GetUser. It returns safe role/application assignments, without aggregating effective permissions.
- Existing user-detail support is absent in REST. Me is actor-owned and its version may differ from target; ListUsers DTO omits version. The admin-session spec intentionally leaves the version source outside its capability and forbids adding a detail lookup within a session request. A separate read endpoint is compatible with that contract.
- Graphify at f824b74379b1eb2a6eed01a2204c2c5d6ec0a054 was consulted for routing. It predates uncommitted admin-session changes, so every relevant route/client decision above was verified in current source. Generated graph artifacts remain untouched.

## Goals / Non-Goals

**Goals:** A direct users handler → usecase → users gateway → shared generated client flow, with independently verified actor and selected target, precise version output, and a narrowly defined safe response.

**Non-Goals:** Generic permission middleware, cross-feature DTO sharing, ListUsers authorization migration, automatic detail aggregation, a local user store, or changes to mutation execution. The caller's separate detail-read and mutation requests are not one atomic transaction.

## Decisions

### 1. Users ownership and route-specific authentication

Add a dedicated `UserDetailHandler` directly under users/handler, a GetUser usecase and a consumer-owned GetUserGateway in users/domain. UserDetailActor carries UserID, MustChangePassword and Permissions. Guard nil context/zero actor as unauthorized; reject restricted actor or exact permission missing/false as forbidden before business dispatch. Role names/codes, wildcards, session permissions and target=actor confer no bypass. Backend remains authoritative when permissions change after Me.

Add focused composition in `internal/app/user_detail_routes.go` and register GET `/users/:user_id` with the existing authentication handler on that route, after public auth and before the broad legacy admin prefix. Keep the admin-session routes and existing shared `/me` group intact. Mirror the order in test/shared/fixtures/app.go. A new authenticated `/users` group or registration inside routes.admin is rejected: either may duplicate Me or incorrectly require an admin role for permission-only readers. Extending the ListUsers/profile handler constructor is unnecessary and would disturb unrelated consumers.

### 2. Read input and target validation

The sole business input is numeric path user_id in 1–2147483647. Accept leading zero decimal representations, reject signs/whitespace/non-ASCII/hex/overflow without normalization. Use focused users-domain parse/validation with safe localized `user_id` field errors; reuse the existing byte-digit predicate and catalog conventions, without renaming session validators or changing their error identities. Do not create a generic ID framework for one read flow.

Bodies, including malformed JSON, and query fields are ignored. Neither HTTP Version nor Idempotency-Key is required or validated, even when repeated/malformed. GetUserRequest sets only Id from the path. Gateway copies outgoing metadata and removes inherited version/idempotency-key, retaining original bearer, locale, request ID, deadline/cancellation and safe unrelated metadata. Browser role/namespace/service-token claims never become RPC authority. No CSRF/cookie proof, target fallback, or extra identity lookup.

### 3. Additive generated client without widening old interfaces

Add a one-method UserDetailService using the published GetUser signature, a client field, and `user_client.go` wrapper for RequestContext construction/deferred cleanup, nil context/request/response guards and unchanged errors. Initialize the default generated UserService SDK on the same existing connection after existing constructor nil guards. Do not extend Service, ProfileService or any session interface.

Follow the adjacent explicit full-dependency injection pattern with `NewWithUserDetailService(transport, service, profile, password, session, self, admin, detail)`, building through NewWithAdminSessionService and retaining its guard ordering before checking the new dependency. All old constructor signatures/semantics remain valid; no type assertions, mutable setters, options framework, connection factory or transport per request. This explicit injection supports shared fake HTTP tests combining detail and session operations; generated TLS tests use ordinary New.

### 4. Safe DTO and strict successful-result checks

Return HTTP 200/code SS/message `User retrieved successfully.` with a direct data object, not data.user. Users owns UserDetail, UserDetailRole and UserDetailApplicationAccess types; never import auth/domain or reuse Principal as a response DTO.

| Field | Mapping |
| --- | --- |
| id | numeric User.Id, positive PostgreSQL INTEGER, equal to selected target |
| email, name, address, country_code | strings copied unchanged |
| phone | User.PhoneNo copied unchanged; no country prefix composition/normalization |
| status | PENDING, ACTIVE, SUSPENDED or DEACTIVATED |
| version | positive backend Version within int64, serialized with decimal integer formatting |
| created_at | required valid timestamp formatted UTC RFC3339Nano, preserving fractional precision |
| must_change_password | backend target boolean, including false |
| roles | nonnil array; each item role_id/code/name/status/role_status |
| application_access | nonnil array; each item application_id/code/name/status/application_status |

Assignment IDs remain positive numeric PostgreSQL IDs. Preserve assignment order, text, statuses and empty collections. Do not impose write-time text/email/phone validation on stored profile strings. Reject nil nested assignment entries or invalid IDs; preserve backend assignment status strings without using them to grant actor authority. Target restriction is information about the selected user, unlike actor restriction: keep returned safe assignments even when target.must_change_password=true rather than applying Me's actor-privilege redaction to target data.

Gateway rejects nil response/user, invalid user ID/version/status/timestamp, nil assignment entries and invalid assignment IDs with a safe internal error; usecase additionally rejects an otherwise valid wrong target without returning partial data. Explicit DTO mapping excludes Password, Dob, auth_version, tokens, internal protobuf fields, legacy role_codes/role_names and invented effective permissions. Build new collections and optional data at boundaries so a reused dependency object cannot alter a response or another request. Nil-created_at is an invalid success here: backend detail requires creation time, whereas existing Me's null convention remains unchanged.

### 5. Version source remains a caller-controlled two-request flow

README will show GET detail for target 23, then pass its exact data.version in the Version header on `/users/23/sessions/revoke-all` with its own Idempotency-Key. iam.user.read grants detail only; session list/mutation still need their own exact permissions. An operator with only session.revoke may use another authoritative version source but cannot use this detail endpoint without user.read. No session route gains an internal GetUser request or changes expected payloads.

Each GET is fresh and does not accept an input version or guarantee a later mutation will succeed: concurrent changes may make the value stale, producing the existing backend 409. Do not substitute actor version, increment/refresh version locally, retry mutation, or automatically refetch after conflict. The mutation may also return a saved valid result version under its existing replay contract.

### 6. No-store, failures and compatibility

Extend LifecycleNoStore to match `/users/<single-segment>` under configured API prefix before CORS/auth, including malformed IDs/literal me and OPTIONS. Do not broaden authentication. Standalone detail handler also emits no-store. Existing exact-origin CORS is sufficient; no header/origin/env change. No Set-Cookie on detail success/error/preflight; no BrowserPolicy injection into detail.

Shared response mapper keeps backend explicit status/code/locale/field errors/Retry-After and standard 401/403/404/409/429/503/504/499 behavior. Missing or failed target read never falls back to Me/ListUsers/cached data. Real method/namespace grants remain operator-owned. Auth public routes, ListUsers role gate, Me restricted behavior, profile/password and all session routes are regression targets. GET `/users/me` is an invalid numeric target, not a self alias; unversioned `/users/{id}` is not introduced.

## Risks / Trade-offs

- [Actor data mistaken for target version/authority] → separate actor input, target ID equality checks, distinct versions and permission tests, no local fallback.
- [Broad users middleware intercepts permission-only detail or duplicates Me] → per-route authentication before legacy group, actual production route/preflight checks plus shared HTTP/TLS ordering tests.
- [Legacy protobuf fields expose credentials] → fixed DTO field set with password/dob/token sentinels in mapping and integration tests.
- [Read version becomes stale before mutation] → document non-atomic requests and preserve backend conflict, no automatic lookup/retry.
- [Existing SDK mocks break] → separate one-method interface and additive constructor; retain all old signatures and regression suites.
- [No-store failures appear before handler] → matcher before CORS/auth with invalid target/preflight/failed Me coverage.
- [Safe assignment information is confused with effective access] → return only backend assignment fields, no computed permissions or actor grant changes.

## Migration Plan

1. Implement additive users feature/client/route changes with root tests and README. No schema/env/dependency or sibling migrations.
2. Verify focused domain/usecase/client/gateway/handler/app, fake HTTP and generated SDK TLS bufconn suites; extend the already registered UserService fake, not a second registration. Cover caller-controlled detail→revoke-all and conflict without extra RPCs.
3. Run mixed concurrent detail/admin-session/self/public/ListUsers scenarios, targeted race, build/full test/vet, strict OpenSpec and final scope/diff audit.
4. For separately authorized rollout, operator grants `/user.v1.UserService/GetUser`, IAM namespace and exact `iam.user.read`. Backend/FE smoke tests and deployment are separate. Rollback removes the additive detail route; existing mutations keep accepting caller-supplied authoritative versions without requiring this endpoint.
