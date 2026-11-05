# user-profile-update Specification

## Purpose

Allow authorized operators to partially update a selected user's profile through REST with explicit target entity version, optional idempotency, safe normalization and distinct local/backend no-op outcomes, while preserving existing self-profile and user-management contracts.

## Requirements

### Requirement: Authorize target profile updates with current exact permission

The service SHALL expose `PATCH /api/v1/users/{user_id}/profile` for bearer actors with a valid current identity, no password-change restriction and exact `iam.user.update=true`. Eligible requests SHALL perform one Me and, only with effective fields, one UpdateUser. Roles, wildcards/prefixes, read/status/session permissions, target=self, body/browser claims or service identity SHALL NOT substitute for this grant. Invalid principal SHALL receive 401; restricted or insufficiently permitted actor SHALL receive 403 before local success or business mutation. No target/authority/session lookup SHALL be added.

#### Scenario: Permission-only operator

- **WHEN** a valid unrestricted non-admin with exact update permission submits valid selected fields for another target or itself
- **THEN** one Me precedes one UpdateUser without a role-admin gate, read permission or additional lookup

#### Scenario: Ineligible actor including empty selection

- **WHEN** otherwise valid selected or empty input comes from a restricted actor, admin-role-only actor, missing/false grant, wildcard/prefix grant, or read/status/session-only grant
- **THEN** 403 is returned after Me without UpdateUser or successful local no-op

#### Scenario: Missing or failed authentication

- **WHEN** bearer is absent, Me fails or cannot establish a valid principal
- **THEN** missing bearer performs zero RPCs and failed/invalid authentication performs no mutation or successful no-op, without cookie fallback or retry

### Requirement: Keep target and technical preconditions caller-controlled

Target SHALL come only from nonempty ASCII decimal path digits representing 1–2147483647. Version SHALL be exactly one case-insensitive parsed header containing ASCII decimal digits representing 1–9223372036854775807. Leading zeros SHALL be accepted numerically; signs, trimming, Unicode digits, hex, floats, zero and overflow SHALL fail 400. Version SHALL remain mandatory even without effective fields and SHALL represent the target entity rather than Me actor version. Body/query/alternate header claims SHALL NOT replace target, version, permission or idempotency sources.

#### Scenario: Independent actor target and precise version

- **WHEN** actor 17 submits target 00023 and Version 9007199254740993 or 9223372036854775807 with forged body/query target/version/permission claims
- **THEN** target 23 and the exact caller target version are selected independently of actor 17 and its version

#### Scenario: Invalid target or Version on any selection

- **WHEN** target is malformed/out of range or Version is missing, repeated including identical/case-varied values, or invalid, with selected fields or an empty selection
- **THEN** safe 400 is returned without UpdateUser or a fabricated successful no-op

#### Scenario: No fallback precondition

- **WHEN** the designated Version header is unusable but actor/body/query/prior detail contains a version
- **THEN** the request fails rather than reading detail, substituting actor data or repairing the caller input

### Requirement: Preserve partial body presence and editable allowlist

The body SHALL accept a JSON object or top-level null and select only non-null string name, address, phone and country_code. Omitted/null fields SHALL remain unselected. Unknown fields, caller field masks and immutable/authority fields such as email/password/status/roles/permissions SHALL be ignored and SHALL NOT overwrite data or authorize the actor. Empty selected strings SHALL undergo validation rather than clear fields or become absent. Missing body, malformed JSON, arrays/scalars and incompatible known-field types SHALL fail 400 without mutation.

#### Scenario: Mixed selected null and forged fields

- **WHEN** input selects a name, null phone and unknown email/status/user_id/field_mask claims
- **THEN** only name is eligible for the command and unselected fields remain unchanged

#### Scenario: Empty and incompatible values

- **WHEN** a selected editable value is empty or a known field has a number/boolean/object/array value
- **THEN** empty strings fail field validation and incompatible types fail parsing, without mutation or destructive clearing

#### Scenario: Empty effective selection

- **WHEN** parsed body is null, {}, only unknown fields or only null editable fields
- **THEN** it has no effective update fields and follows the authenticated local no-op rule after all required validations

### Requirement: Normalize selected profile values with existing rules

Selected name SHALL contain only ASCII letters, ordinary spaces, apostrophes and hyphens, checked before normalization; surrounding/repeated spaces SHALL collapse and word starts after spaces/apostrophes/hyphens SHALL capitalize without lowercasing remaining letters. Normalized name SHALL be 3–100 bytes. Selected country_code SHALL add plus to digit-only input then match plus followed by 1–3 ASCII digits with nonzero first digit, without trim/inference. Selected phone SHALL be 12–20 ASCII digits after removing one +62/62 prefix then at most one leading zero only when this request explicitly supplies a valid country code normalizing to +62. Selected address SHALL remain unchanged printable ASCII 0x20–0x7e plus CR/LF, 3–200 bytes. Applicable errors SHALL accumulate across selected fields, be localized and never echo personal input. Normalization SHALL copy caller values.

#### Scenario: Name capitalization and normalized bounds

- **WHEN** name is `  NASA  o'neil-jean  ` or reaches normalized length boundaries 2/3/100/101
- **THEN** the valid example becomes `NASA O'Neil-Jean`, lengths 3/100 pass and 2/101 fail
- **AND** raw digits/tabs/CR/LF/Unicode/control characters fail before normalization could hide them

#### Scenario: Explicit phone and country normalization

- **WHEN** country_code is 62/+62 and phone is +62812345678901, 62812345678901, 0812345678901 or 812345678901
- **THEN** normalized values are +62 and 812345678901
- **AND** phone-only or other-country input does not remove a country prefix/zero or infer target country; 11/21 final digits fail while 12/20 pass

#### Scenario: Country-only address and several field failures

- **WHEN** country code alone is selected, address contains permitted spaces/CR/LF, or several selected fields are invalid
- **THEN** phone is not synthesized, permitted address bytes remain exact at 3/200-byte bounds, and all invalid fields have safe localized errors without mutation
- **AND** invalid country formats, address lengths outside 3–200, tabs/controls/Unicode and caller-pointer mutation are rejected or prevented

### Requirement: Treat an optional key as absent or exactly one valid value

Idempotency-Key SHALL be optional. If present, exactly one case-insensitive parsed header value SHALL be required: nonblank printable ASCII 0x20–0x7e of 1–200 bytes, with opaque internal spaces/commas and no application trim/generation/hash/UUID restriction. Empty/blank/invalid/repeated values SHALL fail 400 even for local no-op. Absent key SHALL NOT acquire a parent/body/query key or a generated replacement. Selected target version SHALL be explicit payload presence, not outgoing version metadata; only the selected optional key SHALL enter command key metadata. Caller-owned metadata/pointers SHALL remain unchanged.

#### Scenario: Absent versus selected key

- **WHEN** valid selected input omits the key or supplies one valid opaque key
- **THEN** UpdateUser receives zero key metadata values when absent or exactly that selected value when present, with inherited version/key removed from a copy

#### Scenario: Invalid supplied key on an empty request

- **WHEN** no effective fields exist but a supplied key is empty, blank, control/non-ASCII, over 200 bytes or repeated including identical case-varied values
- **THEN** 400 is returned rather than treating it as absent or succeeding locally

#### Scenario: Optional field and parent isolation

- **WHEN** a dependency changes its payload pointer/mask or outgoing metadata values
- **THEN** caller profile fields, target version, optional key and inherited metadata remain independent and unchanged

### Requirement: Distinguish local no-op from authoritative backend outcomes

After valid actor, target, technical headers and body validation, a request without effective fields SHALL return HTTP 200/code SS/message `User profile updated successfully.` with data exactly `{"changed":false}` and zero UpdateUser/GetUser calls. It SHALL NOT return a user/version, check target existence/current version, reserve/replay a key, write backend audit or claim an authoritative transaction. Any valid selected field SHALL dispatch once even when equal to prior/actor data. Backend SHALL remain authoritative for current command permission, target existence, version conflicts, changed/no-op, audit/outbox and optional-key replay. No REST cache/fingerprint, automatic retry/refetch or input replacement SHALL be introduced.

#### Scenario: Local no-op with syntactically valid stale or nonexistent target

- **WHEN** an eligible request has no effective fields and valid target/header syntax, even if an external observer knows the target does not exist or the version is stale
- **THEN** one Me and zero target/mutation RPCs produce only changed:false without an existence/version verification claim

#### Scenario: Selected equal value and backend no-op

- **WHEN** valid fields happen to match actor/prior detail data or backend returns changed=false
- **THEN** one UpdateUser still occurs and a valid backend user/version accompanies its authoritative changed result

#### Scenario: Keyed retry current denial or conflict

- **WHEN** a caller retries a selected request and backend replays a safe stored result, denies current access or returns VC/IC/RB conflict
- **THEN** fresh Me and one eligible UpdateUser return that outcome without a REST replay cache, current-target comparison, version-growth assumption, key replacement or retry
- **AND** an unkeyed call has no invented idempotency/replay guarantee

### Requirement: Return a safe command user snapshot with exact precision

A dispatched success SHALL use HTTP 200/code SS/message `User profile updated successfully.` and data containing exactly changed boolean and user object. User SHALL contain exactly id, email, name, address, phone, country_code, status, version, created_at, must_change_password, roles and application_access. Version SHALL be a decimal string in 1–9223372036854775807; ID and assignment IDs SHALL be numeric 1–2147483647; status SHALL be PENDING/ACTIVE/SUSPENDED/DEACTIVATED; created_at SHALL be valid UTC RFC3339 preserving fractional precision. Roles SHALL contain role_id/code/name/status/role_status; application_access SHALL contain application_id/code/name/status/application_status. Arrays SHALL preserve order and serialize empty as []; booleans including false SHALL remain explicit. Safe backend text SHALL be preserved without write normalization or phone-prefix concatenation. No raw generated/private fields, credential/dob/hash/auth_version, invented profile ID, duplicated top-level version, login_required or effective permissions SHALL be added.

#### Scenario: Safe authoritative command result

- **WHEN** UpdateUser returns a safe target, changed=false or true, precise version above 2^53 and stored values different from input normalization
- **THEN** only changed and the documented user snapshot are returned, preserving backend values and exact data.user.version without input+1 arithmetic or another read

#### Scenario: Empty arrays false flag and target state

- **WHEN** target assignments are empty, must_change_password=false, or target is restricted/inactive under a supported status
- **THEN** arrays are [], false remains explicit and target state is returned as information without substituting actor eligibility or adding a read-permission gate

#### Scenario: Invalid or shared successful result

- **WHEN** response/user is nil, target ID mismatches/is invalid, version is zero/overflow, status/timestamp is invalid/missing, assignment entries are nil or have invalid IDs, or dependency objects are reused
- **THEN** malformed success yields safe 500 without partial data/fallback, and valid mapped responses remain independent of dependency objects and other requests

### Requirement: Preserve secure transport cache browser and error boundaries

All outcomes for the profile path SHALL emit Cache-Control no-store, including local/backend no-op, early CORS/auth/validation/dependency errors and OPTIONS on malformed IDs. Allowed existing-origin preflight SHALL finish with zero RPCs. The operation SHALL require bearer without cookie fallback, CSRF/cookie proof or new Origin requirement and SHALL never set/delete cookies, issue credentials or infer logout/login_required, including target=self. Calls SHALL reuse verified TLS1.3 and one environment-owned credential with original bearer, safe locale/request metadata and bounded parent deadline/cancellation. Existing safe status/public code/field/localized fallback/Retry-After handling SHALL remain unchanged, including native timeout 504/cancel 499 and no global user-error registry expansion or automatic retry.

#### Scenario: Early failure preflight and self-target

- **WHEN** CORS/authentication/validation fails, preflight targets a malformed profile path, or an eligible operation targets itself with any result
- **THEN** no-store and no Set-Cookie apply, allowed preflight makes zero RPCs, and no unauthorized mutation or browser recovery action occurs

#### Scenario: Secure method and metadata

- **WHEN** selected input includes parent mutation metadata and browser service-credential claims
- **THEN** the exact UpdateUser method receives selected optional fields/mask/target version and optional key on the reused secure connection with original bearer, one configured credential and safe context metadata

#### Scenario: Backend failure or terminated context

- **WHEN** backend returns encoded 401/403/404/409/429/503, localized fields/retry information, unknown/raw internals, timeout or cancellation
- **THEN** the existing safe status/code/field/fallback/retry guidance and 504/499 policy apply without private input, partial user data, extended deadlines, retry or cookie effects

### Requirement: Keep detail permissions and existing routes independent

Callers SHALL be able to obtain target Version through separately authorized user detail, then submit a distinct profile PATCH with iam.user.update; neither read nor update grant SHALL imply the other. Mutation SHALL NOT read detail internally or promise an earlier version remains current. Existing self-profile, detail/status, public Register/Login, Me/password, ListUsers role gate and all session contracts SHALL remain unchanged. Only the versioned profile PATCH SHALL be introduced; literal me SHALL remain an invalid numeric target, with no PATCH `/users/{user_id}`, POST or unversioned alias.

#### Scenario: Separate detail and update permissions

- **WHEN** a caller reads a target then patches it, or a read-only/update-only caller tries the other endpoint
- **THEN** Me→GetUser and Me→UpdateUser are independent authorized requests, or the attempted endpoint denies its missing exact grant without inherited permission
- **AND** stale version conflict causes no hidden refetch/resubmission

#### Scenario: Existing policies and route aliases

- **WHEN** self-profile, detail/status, session, public auth, ListUsers and attempted profile aliases coexist
- **THEN** their established eligibility/methods/results/call counts remain intact, public auth stays zero-Me, and only the intended profile PATCH adds a route

#### Scenario: Concurrent isolation

- **WHEN** concurrent target-profile/self-profile/detail/status/session/public/ListUsers calls use distinct actors, targets, fields, keys, versions, permissions, contexts and outcomes
- **THEN** request values, result snapshots, collections, metadata and cookies remain isolated on the shared connection without duplicate Me or cross-request caching
