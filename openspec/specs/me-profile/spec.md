# me-profile Specification

## Purpose

Menyediakan profil dan akses pengguna login pada GET /api/v1/me melalui tepat satu snapshot Me backend, dengan response REST aman, dukungan restricted user, dan tanpa perubahan endpoint existing.

Me merupakan bagian feature `auth` untuk snapshot identitas/akses autentikasi. Pembacaan/pengelolaan profil dan perubahan data user sebagai resource tetap menjadi tanggung jawab feature `users`; pemindahan ownership Login/Register existing dari feature users ke auth berada di luar scope change ini.

## Requirements

### Requirement: Read the authenticated user's own profile

The service SHALL provide `GET /api/v1/me` using the existing user bearer and configured service credential. The endpoint SHALL NOT require an admin role or an additional permission grant. The caller's identity and access scope SHALL come from the backend's verified Me response. Request body/query fields SHALL NOT select a target user, application, namespace, or permission map; these unused inputs SHALL be ignored rather than forwarded.

#### Scenario: Non-admin user

- **WHEN** a valid authenticated non-admin user requests `/api/v1/me` and the backend returns a valid Me snapshot
- **THEN** the endpoint returns HTTP 200 without a local admin/permission gate

#### Scenario: Missing authorization

- **WHEN** Authorization is missing, empty, or whitespace-only
- **THEN** the endpoint returns the existing HTTP 401 error envelope without calling Me

#### Scenario: Browser identity or access claims

- **WHEN** the request includes user_id, application_code, namespace, permissions, or service identity claims in query/body/headers outside the existing authorization contract
- **THEN** those claims do not select another user or change the returned access scope
- **AND** the backend still receives the ordinary empty Me request and environment-owned service credential

### Requirement: Use one current backend snapshot per request

For a request that reaches downstream authentication, the service SHALL invoke exactly one Me RPC and project that successful response into REST data. It SHALL NOT perform a second Me call, permission/list aggregation RPC, local privilege inference, or cross-request snapshot caching. Existing bearer, locale, request metadata, TLS/service credential, earlier parent deadline, and cancellation behavior SHALL be preserved without introducing application retries.

#### Scenario: One Me invocation

- **WHEN** a locally eligible request successfully reads the current user's profile
- **THEN** exactly one `/user.v1.UserService/Me` RPC is called and its snapshot supplies both profile and access
- **AND** no ListUsers, permission, role, or application lookup RPC is added

#### Scenario: Fresh snapshot for each request

- **WHEN** consecutive requests for the same user receive different profile or access snapshots from the backend
- **THEN** each HTTP response reflects its own backend result without reusing the previous request's roles or permissions

#### Scenario: Metadata and cancellation

- **WHEN** a request has a user bearer, locale, existing request metadata, or a bounded/canceled parent context
- **THEN** the Me call retains the existing transport propagation and timeout/cancellation contract
- **AND** no identity fallback or retry is introduced

### Requirement: Explicit safe REST response mapping

The endpoint SHALL return HTTP 200 with `code="SS"`, `message="Profile retrieved successfully."`, and a typed `data` object containing the fields below. It SHALL serialize only the listed safe values rather than raw generated messages or authentication context. IDs SHALL remain JSON numbers, legacy role_codes SHALL remain numeric metadata, and version SHALL be the exact decimal string of the backend value. Empty collections SHALL be `[]` and an empty permissions map SHALL be `{}`.

| Data field | Representation |
| --- | --- |
| `id`, `email`, `fullname`, `address`, `phone`, `country_code` | Backend profile values; id is a JSON number and the others are strings |
| `created_at` | RFC3339 UTC string preserving timestamp precision, or null when backend timestamp is absent |
| `status` | Backend status string |
| `version` | Decimal string without floating-point conversion |
| `must_change_password` | Backend boolean restriction marker |
| `role_codes`, `role_names` | Arrays of numeric legacy codes and legacy names; not a source for new permissions |
| `roles` | Array of objects with numeric `role_id` and string `code`, `name`, `status`, `role_status` |
| `application_access` | Array of objects with numeric `application_id` and string `code`, `name`, `status`, `application_status` |
| `permissions` | Object mapping backend permission codes to booleans; an absent key does not grant permission |

#### Scenario: Complete safe snapshot

- **WHEN** Me returns profile/state, legacy role metadata, scoped roles/application access, and permissions
- **THEN** the response retains their values and collection order through the explicit DTO
- **AND** it adds no credentials, generated/internal fields, inferred grants, or application data from another source

#### Scenario: Large version and UTC timestamp

- **WHEN** a backend user version exceeds JavaScript's safe integer range and created_at contains a valid nonzero fractional timestamp
- **THEN** version is returned exactly as a decimal string and created_at is formatted in UTC without losing its provided precision
- **AND** no synthetic version or current timestamp is generated

#### Scenario: Phone is already composed by Me

- **WHEN** backend Me returns a phone value that already includes country_code, including legacy repeated-prefix data
- **THEN** REST returns that phone string unchanged and also exposes country_code separately

#### Scenario: Empty or absent optional data

- **WHEN** a successful snapshot has empty collections/maps, nil nested collection entries, or no created_at
- **THEN** arrays are `[]`, permissions is `{}`, nil entries are skipped without inventing zero-valued assignments, and created_at is null
- **AND** false permission values remain false without gaining privileges

### Requirement: Restricted users receive no privileges

The endpoint SHALL allow a successful backend Me response with `must_change_password=true`. It SHALL retain only the profile/state values and restriction marker, while all legacy and scoped privilege collections are empty in the DTO: role_codes, role_names, roles, and application_access SHALL be `[]`, and permissions SHALL be `{}`. It SHALL NOT grant access from role names/codes or preserve stale/nonempty privilege data for a restricted snapshot.

#### Scenario: Restricted minimum profile

- **WHEN** backend Me identifies a restricted user and supplies a minimum profile
- **THEN** the endpoint returns HTTP 200 with the profile, state, version, created_at representation, and must_change_password=true
- **AND** it does not reject the user for lacking roles or permissions

#### Scenario: Privilege data cannot survive restriction

- **WHEN** a snapshot marked must_change_password=true also contains privilege values, including data reused by a test dependency
- **THEN** no legacy/scoped role, application access, or permission value is exposed as a grant in that restricted response

### Requirement: Fail closed on Me errors

Downstream authentication, authorization, dependency, timeout, cancellation, and unknown failures SHALL follow the existing `http-error-mapping` contract. Me failure SHALL NOT produce HTTP 200, a partial profile, a cached prior profile, or empty permissions presented as a successful fallback. A nil successful Me response or invalid non-nil timestamp SHALL fail safely as an internal error. No response or added diagnostic SHALL expose credentials or raw internal dependency details.

#### Scenario: Authentication or service denial

- **WHEN** backend Me rejects an invalid/expired user or service credential with UNAUTHENTICATED, or denies service authority with PERMISSION_DENIED
- **THEN** the normal existing mapping returns HTTP 401 or 403 with a safe error envelope
- **AND** no profile response or fallback identity is produced

#### Scenario: Dependency or timeout failure

- **WHEN** backend Me fails while reading profile/access, returns an explicit HTTP status, times out, or receives cancellation
- **THEN** the existing status/code/locale/retry-guidance mapping is retained
- **AND** there is no second RPC, partial success, or empty-permission fallback

#### Scenario: Invalid successful response

- **WHEN** a dependency returns nil with no error or a non-nil protobuf timestamp outside its valid range
- **THEN** the response fails safely instead of constructing a default authenticated profile or normalizing invalid time silently

### Requirement: Preserve existing endpoints and isolate request data

The Me endpoint SHALL preserve its own response and authorization policy, existing Register/Login/ListUsers paths, and ListUsers success/error shapes and admin authorization. Public Register/Login SHALL adopt only the explicit response/browser-policy changes in auth-session-lifecycle while retaining existing credential validation and zero-Me behavior. Authentication middleware SHALL NOT be broadened over public or lifecycle routes, and ListUsers SHALL NOT migrate its admin gate to permissions through this change. Mutable profile/access data SHALL remain isolated from dependency objects, other requests, and callers reading the request snapshot; it SHALL NOT be stored in global mutable state.

#### Scenario: Existing public and admin flows

- **WHEN** Register/Login and ListUsers requests execute after Me and lifecycle route registration
- **THEN** public routes remain free of bearer/Me requirements with their specified lifecycle responses, and ListUsers retains its Me-before-ListUsers admin flow
- **AND** neither /users/me nor an unversioned /me alias is introduced

#### Scenario: Snapshot mutation and concurrent requests

- **WHEN** a dependency reuses mutable response collections or concurrent requests carry different snapshots
- **THEN** modifying a returned snapshot does not mutate dependency/context data or another response
- **AND** bearer, locale, role/application collections, and permissions do not leak between requests
