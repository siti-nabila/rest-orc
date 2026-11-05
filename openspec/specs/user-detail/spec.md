# user-detail Specification

## Purpose

Provide authorized callers with a safe administrative view of a selected user and its authoritative entity version through REST, while preserving independent actor authentication, backend authority, and existing endpoint policies.

## Requirements

### Requirement: Authorize detail with one current Me and exact user-read permission

The service SHALL expose `GET /api/v1/users/{user_id}` for active bearer callers. An eligible request SHALL perform exactly one Me followed by exactly one GetUser. An unrestricted actor SHALL require exact `iam.user.read=true` from that request's verified identity. Role-admin, wildcard/prefix permissions, session permissions, target=actor, browser claims or service identity SHALL NOT substitute for this grant. No permission, profile, session or list aggregation lookup SHALL be added.

#### Scenario: Permission-only reader

- **WHEN** a valid non-admin unrestricted bearer has iam.user.read=true and requests a valid target
- **THEN** one Me precedes one GetUser and the authoritative target detail or backend failure is returned without a role-admin gate

#### Scenario: Ineligible actor

- **WHEN** an otherwise valid request is made by a restricted actor or one with missing/false/wildcard-only/prefix-only permissions or only the admin role or a session permission
- **THEN** HTTP 403 is returned after one Me with zero GetUser calls

#### Scenario: Authentication failure

- **WHEN** bearer is absent or Me fails or cannot establish a valid principal
- **THEN** missing bearer performs zero RPCs, and failed/invalid Me performs no later GetUser or identity fallback
- **AND** failures follow the existing safe authentication/error mapping

### Requirement: Select only the numeric path target and ignore unused inputs

The target SHALL come exclusively from path user_id, consisting of nonempty ASCII decimal digits representing 1–2147483647 without sign, trimming, hex or overflow. Leading zeros SHALL be accepted numerically. Query/body fields and unrelated headers SHALL NOT select a target or authority. Bodies, including empty and malformed bodies, SHALL be ignored. Version/Idempotency-Key SHALL NOT be required, validated or forwarded as read preconditions, including duplicate or malformed values.

#### Scenario: Independent actor and target

- **WHEN** actor 17 reads path user 00023 while query/body/header fields claim user 99, permissions, namespace or service identity
- **THEN** GetUser selects numeric user 23, actor remains 17, and claims do not change authority or produce extra RPCs

#### Scenario: Target boundary forms

- **WHEN** user_id is 1, 2147483647 or a leading-zero decimal in range
- **THEN** the exact numeric ID reaches the target read
- **AND** empty/zero/signed/whitespace/non-ASCII/hex/overflow/out-of-range IDs fail safely as HTTP 400 without GetUser

#### Scenario: Ignored bodies and mutation headers

- **WHEN** a valid detail request includes malformed JSON, unused query controls, or invalid/repeated Version or Idempotency-Key headers
- **THEN** the read still uses only the path ID without parsing the body or requiring any mutation precondition

### Requirement: Return an explicit safe target detail envelope

Success SHALL return HTTP 200, code SS and message `User retrieved successfully.` with a direct typed data object containing exactly the fields below. Stored profile text SHALL be preserved without write-time normalization or validation. Raw generated/internal fields, password, dob, token/credential/hash/auth_version, legacy role_codes/role_names and invented effective permissions SHALL NOT be exposed or aggregated.

| Field | Representation |
| --- | --- |
| id | Positive numeric target ID in 1–2147483647 |
| email, name, address, country_code | Backend strings |
| phone | Backend phone value unchanged |
| status | Backend PENDING/ACTIVE/SUSPENDED/DEACTIVATED |
| version | Exact positive int64 entity version as decimal string |
| created_at | Required valid UTC RFC3339 timestamp preserving fractional precision |
| must_change_password | Backend target boolean, including false |
| roles | Array of role_id, code, name, status, role_status |
| application_access | Array of application_id, code, name, status, application_status |

Assignment IDs SHALL be positive numeric IDs in 1–2147483647. Collections SHALL preserve backend order/text/status values and serialize empty collections as []. These assignments describe the target and SHALL NOT grant the actor any permission.

#### Scenario: Safe complete target

- **WHEN** GetUser returns a valid target with profile, state and assignment collections, even if legacy generated fields carry credential or dob sentinels
- **THEN** only the documented fields are returned in their documented shape and no private or generated field is exposed
- **AND** stored name/email/phone/address/country strings remain unchanged, without composing a phone prefix

#### Scenario: Empty assignments and false flag

- **WHEN** backend assignments are absent/empty and must_change_password=false
- **THEN** roles and application_access are [] and the restriction boolean remains explicitly false

#### Scenario: Restricted or inactive target is information

- **WHEN** an eligible reader receives a target marked must_change_password=true or with a valid non-ACTIVE account status
- **THEN** target state and safe backend assignment values are returned unchanged
- **AND** target restriction does not replace actor eligibility or trigger Me-style privilege redaction, cookie cleanup or credential issuance

### Requirement: Preserve precise target versions and reject malformed successes

Version SHALL be the backend target entity value within 1–9223372036854775807, formatted without float conversion, incrementing or substituting actor data. Success SHALL be rejected safely as HTTP 500 if the response/user is nil, ID is invalid or differs from the selected target, version is invalid, status is unsupported, created_at is absent/invalid, or an assignment entry is nil or has an invalid ID. No partial detail or fallback SHALL be returned.

#### Scenario: Version precision and fractional timestamp

- **WHEN** the backend returns version 9007199254740993 or 9223372036854775807, different from actor Me.version, with a fractional valid creation time
- **THEN** data.version contains the exact target decimal string and created_at preserves timestamp precision in UTC

#### Scenario: Malformed or mismatched success

- **WHEN** a successful dependency result is nil or contains zero/out-of-range/wrong target ID, zero/overflow version, unsupported status, absent/invalid timestamp or malformed assignment entries
- **THEN** safe HTTP 500 is returned with no partial profile or target confirmation/list retry

### Requirement: Keep detail reads fresh and request data independent

Each detail request SHALL authenticate and read its own current backend result without cross-request caching. Returned arrays and optional data SHALL be independent of dependency objects and other responses. Concurrent actor/target/profile/assignment/version values SHALL remain isolated without local mutable user/session state.

#### Scenario: Consecutive target changes

- **WHEN** consecutive detail requests receive different target profile/assignment/version results
- **THEN** each response reflects its own result after its own Me and GetUser calls rather than a previous snapshot or actor profile

#### Scenario: Reused objects and concurrent targets

- **WHEN** a dependency reuses response objects or multiple actors/targets are read concurrently
- **THEN** modifying one mapped response cannot mutate dependency data or another response and no target/permission/metadata value crosses requests

### Requirement: Make detail a separate caller-controlled version source

Callers SHALL be able to use data.version as Version on a separately authorized revoke-all request for the same target. The read SHALL confer no session permission. Session operations SHALL retain their existing one-Me/one-business-RPC behavior and SHALL NOT automatically read detail, substitute actor version, refresh a stale version or retry a mutation. A detail read SHALL NOT promise the version will remain current until a later mutation.

#### Scenario: Read then revoke all

- **WHEN** a caller with both required permissions reads target 23 and submits that exact version plus a mutation key on the separate revoke-all endpoint for target 23
- **THEN** the detail request performs Me→GetUser and the mutation request independently performs Me→RevokeAllUserSessions with the supplied target/version/key
- **AND** neither request adds a hidden detail/permission/session lookup

#### Scenario: Separate permission boundaries

- **WHEN** a user-read-only caller attempts a session operation, or a session-revoke-only caller attempts detail
- **THEN** each endpoint enforces its own exact permission and does not derive eligibility from the other grant

#### Scenario: Concurrent change after detail

- **WHEN** a target changes after a successful read and the backend rejects the later mutation's supplied version
- **THEN** the established safe conflict is returned without a detail refetch, version replacement or automatic mutation retry

### Requirement: Emit no-store without browser side effects

Detail responses SHALL use Cache-Control no-store on success, local validation, authentication/backend failure and preflight, including malformed target IDs and failures before handler execution. Allowed preflight SHALL use existing exact-origin CORS and finish with zero RPCs. Detail SHALL neither require CSRF/cookie proof nor issue/delete cookies on any result; cookies SHALL NOT replace bearer authentication.

#### Scenario: Early errors and preflight

- **WHEN** bearer is missing, Me fails, Origin is denied, target is malformed, or an allowed preflight targets a detail path
- **THEN** the response is no-store with no Set-Cookie and no unauthorized GetUser
- **AND** allowed preflight finishes before authentication with zero RPCs

#### Scenario: Success or failure targeting self

- **WHEN** a permitted actor reads its own numeric target through the detail route or that read fails
- **THEN** the same no-store/permission/response policy applies without cookie mutation or inferred login-required behavior

### Requirement: Preserve secure transport and safe error authority

The read SHALL reuse existing verified TLS1.3 transport, exactly one environment-owned service credential, original bearer, locale/request metadata and bounded parent deadline/cancellation. Inherited version/idempotency-key SHALL be removed from copied outgoing detail metadata without modifying the parent. Backend method/IAM/current-user authority SHALL remain authoritative. Existing safe status/code/field/Retry-After mapping SHALL be retained without retry or leaked internals.

#### Scenario: Secure exact target dispatch

- **WHEN** an eligible request reaches GetUser with unrelated parent mutation metadata and a browser-supplied service credential claim
- **THEN** the selected target, original bearer, safe metadata and one configured service credential reach the exact GetUser method on the reused connection
- **AND** mutation preconditions/browser service claims do not enter that RPC and the parent metadata remains unchanged

#### Scenario: Backend denial or dependency failure

- **WHEN** Me succeeds but GetUser denies current permission or method/IAM scope, returns not found, explicit structured status/fields/retry guidance, or an unknown failure
- **THEN** the shared safe error envelope/status is returned with no partial detail, fallback identity, retry or credential disclosure

#### Scenario: Timeout and cancellation

- **WHEN** the request has an earlier deadline, is canceled during GetUser, or is already canceled/expired before dispatch
- **THEN** the deadline is not extended, established 504/499 behavior is preserved, and no retry or browser side effect occurs

### Requirement: Preserve route ownership and existing endpoint policies

The detail route SHALL belong to users and authenticate once without inheriting legacy ListUsers role-admin middleware or adding effects to public auth. Existing ListUsers, Me, profile/password, lifecycle, self-session and admin-session URLs, contracts and authorization SHALL remain unchanged. GET `/api/v1/users/me` SHALL be an invalid numeric detail target rather than a self alias. No unversioned detail alias SHALL be introduced.

#### Scenario: Public and protected route coexistence

- **WHEN** public Register/Login, non-admin permission-only detail, ListUsers, and both session families run after route registration
- **THEN** public auth remains zero-Me, detail uses its own exact permission, ListUsers retains its legacy role gate, and each session endpoint retains its existing policy/order

#### Scenario: No self alias or implicit migration

- **WHEN** a caller requests `/api/v1/users/me`, an unversioned user detail path, or `/api/v1/me`
- **THEN** literal me fails numeric detail validation after authentication, the unversioned path remains unavailable, and Me retains its independent response and restricted-user policy
