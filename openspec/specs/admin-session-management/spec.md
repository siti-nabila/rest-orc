# admin-session-management Specification

## Purpose

Allow unrestricted callers with exact session administration permissions to inspect and revoke sessions of a selected user while the auth backend remains authoritative for current access, ownership, idempotency, and entity versions.

## Requirements

### Requirement: Authorize target-session operations from one current Me

The service SHALL expose `GET /api/v1/users/{user_id}/sessions`, `DELETE /api/v1/users/{user_id}/sessions/{session_id}`, and `POST /api/v1/users/{user_id}/sessions/revoke-all` with active bearer authentication. Each eligible request SHALL perform exactly one Me followed by exactly one ListUserSessions, RevokeUserSession, or RevokeAllUserSessions respectively. For otherwise valid requests, unrestricted callers SHALL require exact permission `iam.session.read=true` for list or `iam.session.revoke=true` for either mutation from that request's verified Me result. Restricted callers or insufficient permissions SHALL receive HTTP 403 without a session RPC. Missing bearer/invalid principal SHALL receive HTTP 401. Roles, permission prefixes/wildcards, browser permission claims, or service identity SHALL NOT substitute for the exact permission. No permission, user-detail, or session confirmation RPC SHALL be added.

#### Scenario: Non-admin with exact permission

- **WHEN** an unrestricted non-admin caller has the exact required permission and valid input
- **THEN** the request succeeds or receives the authoritative session RPC outcome after one Me, without a role-admin requirement or second lookup

#### Scenario: Missing false wildcard or role-only permission

- **WHEN** valid input is sent by a restricted caller or a caller with only an admin role, false/missing permission, wildcard, prefix, or browser-supplied grant
- **THEN** HTTP 403 is returned after one Me and zero session RPCs

#### Scenario: Authentication stops the pipeline

- **WHEN** bearer is absent or Me fails
- **THEN** missing bearer performs zero RPCs and a failed Me performs no subsequent session RPC, cookie fallback, or retry

### Requirement: Separate target identifiers from actor and browser authority

The selected user SHALL come only from the path user_id parsed as nonempty ASCII decimal digits within 1–2147483647 without sign, whitespace normalization, or overflow. Leading zeros SHALL be accepted numerically. Single revoke SHALL use the path session_id matching `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$` without trim, case folding, UUID version restriction, or all-zero exclusion. Actor SHALL remain the verified bearer owner, independent of target. Bodies SHALL be ignored, including malformed bodies, and empty bodies SHALL be accepted. Query/body/header owner, role, namespace, service, session, key, or version claims SHALL NOT replace their designated path/header sources or add caller authority.

#### Scenario: Target differs from actor

- **WHEN** actor 17 with exact permission requests sessions of path user 23 while body/query claims user 99
- **THEN** user 23 enters the business request, actor 17 remains the authenticated caller, and no confirmation lookup occurs

#### Scenario: Identifier boundaries and forms

- **WHEN** user_id is 1, 2147483647, or a leading-zero decimal in range and session_id is a canonical lowercase UUID including all-zero
- **THEN** validated numeric target and unchanged UUID reach the backend
- **AND** zero, 2147483648, signs, non-ASCII digits, whitespace, overflow, uppercase or malformed UUID fail HTTP 400 before the session RPC

### Requirement: Preserve bounded target-session pagination and backend metadata

List SHALL accept only page, limit, and last_id as list controls. Omitted page/limit SHALL default to 1/20; present values SHALL be nonempty ASCII digits within page 1–2147483647 and limit 1–100. Duplicate page/limit/last_id SHALL fail HTTP 400. Last_id omitted/empty SHALL mean no cursor; otherwise its once URL-decoded value SHALL be forwarded unchanged and limited to 2048 bytes, without trimming or interpreting its contents. Supplied sort, fields, search, keyword, statuses, application_code, created_from, or created_to SHALL fail HTTP 400 even when empty. The RPC SHALL use explicit created_at DESC then id DESC with no selected fields/search. REST SHALL preserve item order, total int64, page, limit, total_pages, has_next, has_prev, and next_cursor without narrowing/recomputation; empty items SHALL be []. Changing target or limit SHALL require the caller to reset the cursor according to backend ownership/binding rules.

#### Scenario: Default and boundary pagination

- **WHEN** pagination is omitted or uses valid boundary values and an opaque once-decoded cursor of at most 2048 bytes
- **THEN** defaults or validated values and fixed ordering reach ListUserSessions exactly once
- **AND** empty/zero/negative/fractional/non-ASCII/whitespace/overflow values, limit 101, duplicate controls, 2049-byte cursor, or unsupported controls fail locally without that RPC

#### Scenario: Large total and empty page

- **WHEN** backend total is 2147483648 on an empty page with authoritative flags and cursor
- **THEN** items is [] and all backend paginator values remain unchanged, including nonzero total and has_prev independent of the page number

#### Scenario: Cursor bound to another target

- **WHEN** the backend rejects a cursor that belongs to another target or list configuration
- **THEN** its safe error is preserved without local cursor rewriting, target fallback, or retry

### Requirement: Return safe session metadata for the selected owner

List items SHALL contain exactly id, user_id, status, created_at, last_seen_at, expires_at, revoked_at, device_label, user_agent, ip_address, and is_current. User_id SHALL remain a positive numeric ID in the target range and SHALL equal the selected target, not necessarily actor. Session UUIDs/status/timestamps SHALL be valid; status SHALL remain ACTIVE/REVOKED, required created_at/expires_at SHALL be UTC RFC3339 with fractional precision, and absent last_seen_at/revoked_at SHALL be null. Historical expiry, false is_current and empty text SHALL be preserved. Is_current SHALL remain the backend value without inferring it from target=actor. Credentials, hashes, auth_version, internal/generated fields, and partial results SHALL NOT be exposed. Invalid/nil successful dependency response/meta/row, invalid paginator, or owner mismatch SHALL fail safely as HTTP 500. Concurrent results SHALL own their data independently.

#### Scenario: Historical safe row

- **WHEN** the backend returns target-owned ACTIVE/REVOKED rows with past expiry, optional absent timestamps, false is_current and empty display metadata
- **THEN** exact safe fields are returned in backend order with null optional timestamps and no synthetic EXPIRED status or identity inferred from device/IP

#### Scenario: Owner mismatch or malformed success

- **WHEN** a successful dependency result is nil, malformed, has invalid required timestamps/counters/status/UUID, or contains a session of another owner
- **THEN** a safe HTTP 500 with no partial session data is returned without another RPC

#### Scenario: Independent results

- **WHEN** responses are reused by a dependency or multiple owners are requested concurrently
- **THEN** result slices/optional values remain isolated and cannot mutate or expose another request's data

### Requirement: Forward one validated mutation idempotency key

Both mutations SHALL require exactly one parsed HTTP Idempotency-Key header value, case-insensitive by header name, nonblank, 1–200 bytes, and printable ASCII bytes 0x20–0x7e. Spaces inside an otherwise nonblank key SHALL be allowed. Duplicate headers, even identical, invalid/non-ASCII/control values, missing/empty/all-space keys, or overlength keys SHALL fail HTTP 400 before the mutation RPC. The selected parsed value SHALL be forwarded without application trimming, rewriting, UUID enforcement, generation or hashing as exactly one idempotency-key metadata entry. A comma in a single header SHALL be opaque key content. Bodies/query SHALL NOT supply this key. List SHALL neither require nor forward a key. Inherited idempotency-key/version metadata SHALL NOT override or contaminate selected operation inputs; safe caller metadata SHALL remain unchanged.

#### Scenario: Stable opaque key

- **WHEN** a mutation supplies one printable nonblank 1–200-byte key including internal spaces or commas
- **THEN** exactly that parsed value becomes the sole idempotency-key entry and caller context metadata is not mutated

#### Scenario: Invalid duplicate or forged source

- **WHEN** a key is missing, blank, repeated, over 200 bytes, non-ASCII/control-bearing, or supplied only in body/query
- **THEN** HTTP 400 is returned without a mutation RPC and without echoing the key in errors

#### Scenario: Inherited preconditions

- **WHEN** outgoing parent metadata contains an unrelated idempotency-key/version or a list request supplies mutation headers
- **THEN** mutation uses only its valid HTTP key and designated version payload, list forwards neither precondition, and unrelated safe metadata is preserved on copied context

### Requirement: Use target entity Version only for revoke-all

Revoke-all SHALL require exactly one case-insensitive HTTP Version header containing nonempty ASCII decimal digits representing 1–9223372036854775807, without sign, whitespace normalization, floating-point conversion, or overflow. Leading zeros SHALL be accepted numerically. Its numeric value SHALL enter a present expected_user_version payload field. Missing/duplicate/invalid Version SHALL fail HTTP 400 without the mutation RPC. The service SHALL NOT use actor Me.version, auth_version, request body/query version, computed counters, or an automatic target read as the precondition. List/single revoke SHALL NOT require or forward Version. A conflict SHALL NOT trigger version refresh or retry. Callers SHALL supply a known authoritative target version; no new target-detail/version-source API is promised by this capability.

#### Scenario: Precise version beyond browser safe integer

- **WHEN** revoke-all supplies Version 9007199254740993 or 9223372036854775807
- **THEN** the exact numeric value reaches a present expected_user_version field without precision loss, even if actor version differs

#### Scenario: Missing invalid or alternate version source

- **WHEN** Version is absent, empty, zero, duplicated, signed, nonnumeric, whitespace-bearing, overflowed, or supplied only in body/query/actor data
- **THEN** HTTP 400 is returned with no revoke-all RPC or target lookup

#### Scenario: Version conflict

- **WHEN** the backend returns a version conflict
- **THEN** that safe outcome is returned without replacing Version, reading target state, or resubmitting the mutation

### Requirement: Preserve backend replay and current-authority decisions

Backend SHALL remain authoritative for ownership, exact current permission, service method/IAM namespace, idempotency scope/fingerprint/expiry/replay, version conflicts, transaction/audit/outbox and revocation. REST SHALL invoke one Me then one eligible business RPC on each caller retry, without local replay cache, fingerprint, automatic application retry, or fallback credential. A repeated key SHALL NOT bypass Me or backend authority checks. REST SHALL NOT pre-check current target version or reject a valid committed replay result merely because its version is not an input-derived increment.

#### Scenario: Backend committed replay

- **WHEN** a caller repeats the same valid mutation request/key and the backend returns its saved safe result
- **THEN** REST forwards that result after a fresh Me and one mutation RPC without extra version/session reads or local version arithmetic

#### Scenario: Authority revoked or key reused differently

- **WHEN** the backend denies current authority or returns an idempotency conflict after Me
- **THEN** the mapped error is returned without using an earlier success, changing the key/payload, or retrying automatically

### Requirement: Return exact mutation outcomes without browser session effects

Single revoke success SHALL use HTTP 200/code SS/message `User session revoked successfully.` and exactly data.changed, including false. Revoke-all success SHALL use HTTP 200/code SS/message `All user sessions revoked successfully.` and exactly data.changed plus data.version serialized as the backend's decimal string. Returned version SHALL be positive and within int64 range; invalid/nil successful responses SHALL fail safely as HTTP 500. REST SHALL NOT synthesize changed, input+1 version, user_id, auth_version, login_required, current-session state or replacement credentials. Neither operation SHALL issue/delete cookies or require CSRF proof, including success/replay, changed=false, error, and target=actor/current session. Later protected requests SHALL still depend on backend verification.

#### Scenario: False changed and exact version

- **WHEN** single revoke returns changed=false or revoke-all returns a valid changed/version pair including false and version above 2^53
- **THEN** false remains explicit, single output contains only changed, and all output contains changed/version with exact string precision

#### Scenario: Invalid global version

- **WHEN** revoke-all returns nil success, version zero, or a uint64 version above MaxInt64
- **THEN** a safe HTTP 500 is returned without partial success or cookie mutation

#### Scenario: Admin revokes its own current sessions

- **WHEN** an authorized actor targets its own user or current session through the admin endpoint
- **THEN** the same backend response contract applies with no Set-Cookie or inferred login_required, and no confirmation lookup occurs

### Requirement: Apply no-store and existing safe error and transport policies

All three target-session routes SHALL emit Cache-Control no-store on success, local validation, preflight and early CORS/auth/dependency failures, including invalid target/session identifiers. Allowed preflight SHALL use existing exact-origin CORS policy and complete without Me or business RPC. Local validation SHALL return safe localized catalog field errors without input credentials/keys. Backend status/code/fields/Retry-After and timeout/cancellation SHALL follow http-error-mapping without automatic retry. List success SHALL use HTTP 200/code SS/message `User sessions retrieved successfully.`. All RPCs SHALL reuse the existing verified TLS1.3 connection, one environment-owned service token, original bearer, locale/request metadata and bounded parent deadline/cancellation; no dependency/env/backend/protobuf/local-storage changes SHALL be introduced.

#### Scenario: Early failure or preflight

- **WHEN** a target-session request has missing bearer, denied Origin, malformed target, or an allowed preflight
- **THEN** its response remains no-store with no browser cookie mutation and no unauthorized business RPC; allowed preflight invokes zero RPCs

#### Scenario: Structured failure and context termination

- **WHEN** a business RPC returns safe 401/403/404/409/429/503 with explicit metadata/retry guidance, times out, or is canceled
- **THEN** established status/body/retry handling is preserved without leaked internals, retry, cookie mutation, or extended deadline

#### Scenario: Shared secure connection with isolated requests

- **WHEN** concurrent administration, self-session, public auth and ListUsers requests carry distinct owners, permissions, keys, versions and metadata
- **THEN** each uses only its selected payload/credential context on the reusable secure connection with no cross-request leakage or new connection per request

### Requirement: Coexist with legacy users and self-session authorization

New target-session routes SHALL belong to the users feature and SHALL authenticate exactly once without inheriting the legacy role-admin gate. Existing ListUsers SHALL retain its current role-admin gate and response contract. Existing public Register/Login SHALL remain public, and self-session /me URLs, permission eligibility, header requirements and cleanup behavior SHALL remain unchanged. Target-session routes SHALL NOT be registered as aliases of self-session or unversioned endpoints; a literal me target SHALL be invalid user_id rather than selecting self authority. Route composition SHALL not add authentication/permission middleware effects to unrelated routes.

#### Scenario: Route policies remain distinct

- **WHEN** a non-admin with iam.session.read requests target sessions, the same caller requests ListUsers, or public Register/Login is called without bearer
- **THEN** target-session exact permission is honored, ListUsers keeps its legacy role requirement, and public auth is not intercepted by protected middleware

#### Scenario: Self-session still uses its own policy

- **WHEN** an unrestricted bearer without admin session permissions/headers uses /api/v1/me/sessions
- **THEN** self-session retains its original one-Me ordering and eligibility, while /api/v1/users/me/sessions cannot become a self alias
