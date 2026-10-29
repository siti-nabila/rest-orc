# my-session-management Specification

## Purpose

Let authenticated unrestricted users inspect and revoke their own sessions through REST while the downstream auth service remains authoritative for ownership, eligibility, revocation, and session validity.

## Requirements

### Requirement: Authenticate self-session operations without an admin gate

The service SHALL expose `GET /api/v1/me/sessions`, `DELETE /api/v1/me/sessions/{session_id}`, and `POST /api/v1/me/sessions/revoke-all` for the bearer owner. Each eligible request SHALL execute exactly one Me followed by exactly one corresponding session RPC, without a second authentication/permission/session lookup. Non-admin unrestricted users SHALL be eligible; restricted users SHALL receive HTTP 403 without a session RPC. Missing/invalid principal SHALL receive HTTP 401. Cookies SHALL NOT substitute for bearer credentials or become RPC proof for these operations.

#### Scenario: Eligible owner

- **WHEN** a non-admin unrestricted bearer owner submits a valid list, single-revoke, or revoke-all request
- **THEN** one Me precedes one ListMySessions, RevokeMySession, or RevokeAllMySessions respectively without another lookup or local admin permission gate

#### Scenario: Missing bearer or failed authentication

- **WHEN** bearer is absent, or Me rejects/fails the request
- **THEN** absent bearer performs zero RPCs and Me failure performs no later session RPC
- **AND** no cookie credential fallback or browser cleanup occurs

#### Scenario: Restricted owner

- **WHEN** Me identifies must_change_password=true on an otherwise valid self-session request
- **THEN** HTTP 403 is returned with one Me and zero session RPCs, regardless of supplied roles or permissions
- **AND** existing restricted Me/password/logout eligibility remains unchanged

### Requirement: Keep owner and caller authority server verified

Session owner SHALL come from verified authentication context, never browser user_id/session claims, role/permission maps, service identity, or namespace fields. Single revoke SHALL use only its validated path session_id as target. Bodies SHALL not supply business fields for these three operations and empty bodies SHALL be accepted. Version, Idempotency-Key, refresh credentials, and CSRF proof SHALL not become required or be mapped from browser input into RPC preconditions/credentials. Backend ownership and current-session checks SHALL remain authoritative after Me.

#### Scenario: Forged identity and preconditions

- **WHEN** requests contain body/query/header claims for another owner, session target, service authority, version, or idempotency
- **THEN** the operation still uses verified bearer ownership and only the specified pagination/path inputs enter the business RPC
- **AND** no browser claim changes method or namespace grants

#### Scenario: State changes after Me

- **WHEN** Me succeeds but the session RPC rejects ownership, account restriction, or current-session validity
- **THEN** the safe backend error is returned without retry, fallback identity, partial success, or cookie deletion

### Requirement: Validate a bounded pagination contract and preserve opaque cursors

List SHALL accept page, limit, and last_id. Omitted page/limit SHALL default to 1/20. Present page and limit SHALL be nonempty ASCII decimal digits within page 1–2147483647 and limit 1–100, without overflow or whitespace normalization. Duplicate known pagination keys SHALL return HTTP 400. Last_id omitted/empty SHALL select no cursor; otherwise its value after one URL decoding SHALL be at most 2048 bytes and SHALL be forwarded unchanged, without trimming, decoding its contents, or inventing an ID. Unsupported list controls sort, fields, search, keyword, statuses, application_code, created_from, and created_to SHALL return HTTP 400 when supplied. Invalid local input SHALL not invoke ListMySessions.

#### Scenario: Defaults and numeric boundaries

- **WHEN** pagination is omitted or valid explicit values use page 1/2147483647 and limit 1/100
- **THEN** defaults 1/20 or the explicit values are used without clamping to totals
- **AND** present empty, zero, negative, fractional, non-ASCII, whitespace, overflow, page 2147483648, or limit 101 values fail safely as HTTP 400

#### Scenario: Cursor pass-through

- **WHEN** last_id contains a URL-encoded opaque cursor within the 2048-byte boundary
- **THEN** its once-decoded value reaches the backend exactly and no internal cursor parsing or normalization occurs
- **AND** 2049 bytes or duplicate pagination keys are rejected before the list RPC

#### Scenario: Unsupported list controls

- **WHEN** a request supplies a listed unsupported sort/search/field/filter control, even with an empty value
- **THEN** HTTP 400 is returned without silently enabling or ignoring that control or invoking ListMySessions

### Requirement: Use deterministic backend-owned session ordering and metadata

The list RPC SHALL receive page/limit/last_id with explicit created_at DESC followed by id DESC, no selected fields or search, and no target-user override. REST SHALL preserve returned item order and all authoritative pagination fields: items, total, page, limit, total_pages, has_next, has_prev, next_cursor. Total SHALL preserve the backend nonnegative int64 value without int32 narrowing or recomputation. Empty collections SHALL be [] and an empty/out-of-range page SHALL not erase nonzero total or other metadata. Returned cursors SHALL be passed through unchanged rather than generated locally.

#### Scenario: Total larger than int32 and empty page

- **WHEN** the backend returns total=2147483648 with no items on the requested page
- **THEN** items is [] while total remains exactly 2147483648 and the backend page/limit/total_pages/flags/cursor remain unchanged

#### Scenario: Cursor traversal

- **WHEN** the backend supplies a nonempty next_cursor and has_prev on a cursor-based page
- **THEN** both are returned as supplied without deriving has_prev from page or recomputing an offset
- **AND** changing limit requires the caller to reset its cursor according to the documented backend contract

### Requirement: Expose only safe owned session metadata

Each list item SHALL contain exactly id, user_id, status, created_at, last_seen_at, expires_at, revoked_at, device_label, user_agent, ip_address, and is_current. Session IDs SHALL be canonical UUID strings; user_id SHALL be a positive JSON number equal to the verified owner. Status SHALL retain backend ACTIVE/REVOKED and expiry SHALL remain a separate timestamp, without an invented EXPIRED status. Is_current SHALL preserve the backend boolean including false. Timestamps SHALL be RFC3339 UTC preserving fractional precision; required created_at/expires_at SHALL be present and valid, optional last_seen_at/revoked_at SHALL be null when absent. Text metadata SHALL be display-only and SHALL NOT grant identity or authority. Credentials, hashes, auth_version, and generated/internal fields SHALL never be exposed.

#### Scenario: Safe historical session

- **WHEN** the backend returns owned ACTIVE and REVOKED records including past expiry, optional nil timestamps, empty text, and false is_current
- **THEN** values/order are faithfully returned with null optional timestamps and explicit false, without rejecting valid historical timestamps or inferring authentication from device/IP

#### Scenario: Invalid or mismatched dependency result

- **WHEN** list success is nil, lacks valid paginator/required timestamps, contains a nil row, invalid UUID/status/timestamp/owner, negative counters, or another user's session
- **THEN** a safe HTTP 500 with no partial session data is returned without a fallback or confirmation RPC

#### Scenario: Per-request isolation

- **WHEN** dependency response objects are reused or requests run concurrently for distinct owners
- **THEN** each response owns its collections and cannot expose or mutate another request's data

### Requirement: Revoke one canonical session without inventing current-session state

Single revoke SHALL validate session_id against `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$` without trimming, case conversion, UUID-version restrictions, or implicit all-zero exclusion. A valid target SHALL be sent unchanged in exactly one RevokeMySession RPC after Me. Success SHALL return HTTP 200, code SS, message `Session revoked successfully.`, and exactly data.changed as the backend boolean including false. The service SHALL NOT add login_required, perform a session list/get to infer the current session, issue tokens, or delete cookies on single revoke. FE SHALL use previously obtained is_current when deciding whether to clear its access token memory.

#### Scenario: Canonical target and invalid forms

- **WHEN** an eligible owner supplies a lowercase canonical UUID, including a format-valid all-zero value
- **THEN** it is forwarded unchanged for backend ownership/existence verification
- **AND** uppercase, malformed, unhyphenated, or whitespace-bearing values return HTTP 400 with no revoke RPC

#### Scenario: Changed and unchanged backend outcomes

- **WHEN** RevokeMySession returns changed=true or changed=false
- **THEN** that boolean is returned explicitly with no cookie mutation or additional field/lookup
- **AND** backend not-found/forbidden failures remain mapped errors rather than local successes

#### Scenario: Target is actually the current session

- **WHEN** a successful single revoke targets the caller's current backend session
- **THEN** REST still returns only changed and performs no automatic cookie cleanup or current-session inference
- **AND** later protected requests rely on backend verification rather than local cached session state

### Requirement: Revoke all sessions with authoritative flags and conditional cleanup

Revoke-all SHALL invoke exactly one RevokeAllMySessions RPC with an empty business request after Me. Success SHALL return HTTP 200, code SS, message `All sessions revoked successfully.`, and exactly changed/login_required as backend booleans including false. Only successful login_required=true SHALL expire both __Secure-refresh and __Secure-csrf at their documented Path=/api/v1/auth with host-only Secure/SameSite=Lax removal semantics; only refresh SHALL be HttpOnly. Cleanup SHALL work without incoming cookies, without reading them or requiring CSRF. Errors, nil results, and login_required=false SHALL leave cookies unchanged. No version, replacement credential, automatic login, or local revocation state SHALL be invented.

#### Scenario: Confirmed login-required success

- **WHEN** RevokeAllMySessions succeeds with login_required=true regardless of the changed flag or incoming cookies
- **THEN** both booleans are returned unchanged and both cookie deletions are emitted with their original names/path/attributes
- **AND** FE can clear its access token memory and require login again

#### Scenario: False flags and failure

- **WHEN** backend success has login_required=false, or the operation fails/returns nil without an error
- **THEN** false flags remain explicit on success, a nil result fails safely as HTTP 500, and neither case deletes or replaces cookies

### Requirement: Do not cache session responses and preserve safe error authority

All three self-session routes SHALL emit Cache-Control no-store on success, local rejection, authentication/dependency failure, and preflight, including checks before handler execution. Local malformed pagination/UUID SHALL use HTTP 400 and safe catalog messages without credentials. Backend status/code/localized field errors/Retry-After and timeout/cancellation SHALL follow http-error-mapping without retries or success-only browser effects on failure. List success SHALL use HTTP 200, code SS, message `Sessions retrieved successfully.`, and the documented typed page data.

#### Scenario: Early denial and preflight

- **WHEN** bearer is missing, CORS denies Origin, or an allowed preflight targets a self-session URL
- **THEN** the response remains no-store with zero business RPCs and no browser cookie mutation
- **AND** allowed preflight completes before Me using the existing exact-origin CORS policy

#### Scenario: Backend structured error and cancellation

- **WHEN** a session RPC returns explicit 409/429/503 with safe field errors/RetryInfo, times out, or is canceled
- **THEN** established HTTP status/body/retry-guidance mapping is preserved with no-store and no retry, identity fallback, or cookie mutation

### Requirement: Reuse secure request transport and retain existing behavior

The three session RPCs SHALL use the existing reusable auth connection with verified TLS1.3, exactly one environment-owned service token, original bearer, locale/request metadata, and bounded parent deadline/cancellation. Method and IAM namespace grants SHALL remain backend/operator-owned. No browser authority claims, per-request connection, local JWT/session storage, dependency/env change, or backend/protobuf modification SHALL be introduced. Existing auth lifecycle, Me, password, profile and ListUsers URLs, responses, credential policies, and RPC ordering SHALL remain unchanged. No admin/user-target session route or unversioned/legacy-users alias SHALL be added.

#### Scenario: Secure ordered dispatch and denial

- **WHEN** eligible self-session requests run or the backend denies method/IAM namespace authority
- **THEN** one Me and at most one session RPC use the same secure connection and expected credentials/metadata, without expanded privileges or retry

#### Scenario: Concurrent existing and new requests

- **WHEN** list/revoke-one/revoke-all requests run concurrently with distinct owners, metadata and flag outcomes alongside existing endpoints
- **THEN** arguments, results, cookies, and metadata remain isolated and existing endpoint policies/order remain intact

#### Scenario: Route ownership and compatibility

- **WHEN** self-session routes are registered
- **THEN** they belong to auth on the existing shared authenticated /me group without duplicate Me middleware
- **AND** no /me/sessions, /api/v1/users/me/sessions, or /api/v1/users/{user_id}/sessions alias is introduced
