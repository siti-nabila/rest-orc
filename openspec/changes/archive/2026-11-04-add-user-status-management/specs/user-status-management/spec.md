# Spec Delta

## Purpose

Allow authorized operators to change a selected user's account status through REST with explicit reason, target entity version and idempotency, while the auth backend remains authoritative for transitions, current access, replay and session invalidation.

## ADDED Requirements

### Requirement: Authorize status changes from one current Me

The service SHALL expose `PATCH /api/v1/users/{user_id}/status` for active unrestricted bearer actors with exact `iam.user.status.update=true` from that request's verified identity. Each eligible request SHALL perform one Me followed by one ChangeUserStatus. Roles, wildcard/prefix grants, read/session permissions, target=self and browser claims SHALL NOT replace the exact permission. Restricted/insufficiently permitted actors SHALL receive 403 without the mutation; invalid principal SHALL receive 401.

#### Scenario: Non-admin status operator

- **WHEN** an unrestricted non-admin with the exact status permission submits valid input
- **THEN** one Me precedes one ChangeUserStatus without a role-admin gate or another authority/target/session lookup

#### Scenario: Ineligible actor

- **WHEN** otherwise valid input is submitted by a restricted actor or one with only admin role, false/missing/wildcard/prefix/read/session permissions
- **THEN** 403 is returned after one Me and no ChangeUserStatus

#### Scenario: Authentication stops dispatch

- **WHEN** bearer is missing, principal is invalid, or Me fails
- **THEN** missing bearer invokes zero RPCs and failed/invalid authentication invokes no later mutation, cookie fallback or retry

### Requirement: Keep actor target and body sources separate

Target user_id SHALL come only from nonempty ASCII decimal path digits representing 1–2147483647 without sign, trim, hex or overflow. Leading zeros SHALL be accepted numerically. Body SHALL be a JSON object with required string status and reason. Missing/null fields SHALL fail validation, and malformed/empty/non-object bodies or wrong field types SHALL fail 400 without mutation. Unknown body fields and query claims SHALL be ignored and SHALL NOT supply identity, permission, status, reason, key or version outside their designated sources.

#### Scenario: Different actor and target with forged claims

- **WHEN** actor 17 submits path user 00023 and JSON status/reason while body/query/header claims select user 99 or broader authority
- **THEN** numeric target 23 and only the body status/reason enter the mutation, with actor 17 unchanged
- **AND** neither a target confirmation lookup nor an authority claim is added

#### Scenario: Numeric target boundaries

- **WHEN** target is 1, 2147483647 or leading-zero decimal in range
- **THEN** the numeric target is preserved
- **AND** zero, signs, whitespace, Unicode digits, hex, overflow or out-of-range target fail 400 before the mutation

#### Scenario: Body presence and typing

- **WHEN** status/reason is omitted, null or wrong-typed, or the entire body is absent, malformed, null, array or scalar
- **THEN** safe 400 is returned with no ChangeUserStatus and no source fallback to query/header/actor data

### Requirement: Validate exact status codes and normalize reason safely

Status SHALL be exactly PENDING, ACTIVE, SUSPENDED or DEACTIVATED, without trimming, case folding, defaulting or UI-label translation. Raw reason SHALL allow only ASCII 0x20–0x7e plus CR/LF before normalization; tab/NUL/DEL/other controls and non-ASCII SHALL be rejected even at the edges. Surrounding permitted whitespace SHALL then be trimmed; normalized reason SHALL be 3–500 bytes, with internal text/spaces/line endings unchanged. Independent body field errors SHALL be accumulated and localized without echoing inputs.

#### Scenario: Supported status and reason boundaries

- **WHEN** an exact status code and valid raw reason normalize to 3 or 500 bytes, including internal punctuation/spaces/CR/LF
- **THEN** the exact status and trimmed reason are sent once, preserving their internal bytes

#### Scenario: Invalid bytes cannot disappear during normalization

- **WHEN** reason has non-ASCII/control bytes at the edges or internally, normalizes below 3 or above 500 bytes, or status is lowercased/whitespace-bearing/unknown
- **THEN** safe localized field errors return 400 without mutation or silent character removal

#### Scenario: Independent body errors

- **WHEN** transport inputs parse but status and reason independently fail domain rules
- **THEN** the response includes errors for both fields in the existing field-error envelope with empty data and no mutation

### Requirement: Require one caller key and one target entity Version

The mutation SHALL require exactly one case-insensitive parsed Idempotency-Key header and exactly one Version header. Key SHALL be nonblank printable ASCII 0x20–0x7e of 1–200 bytes; internal spaces and commas SHALL remain opaque content without application trim/generation/hash/UUID enforcement. Version SHALL be nonempty ASCII digits in 1–9223372036854775807 without sign, trim or float conversion; leading zeros SHALL be accepted numerically. Missing/duplicate/invalid headers SHALL fail 400 without mutation or alternate-source fallback.

#### Scenario: Precise version and opaque key

- **WHEN** Version is 9007199254740993 or 9223372036854775807 and one valid key contains spaces or commas
- **THEN** the exact numeric target version has explicit payload presence and exactly the selected parsed key reaches mutation metadata

#### Scenario: Header multiplicity and boundaries

- **WHEN** a header is missing or repeated with any capitalization, or key is blank/non-ASCII/control/over 200 bytes, or Version is empty/zero/signed/whitespace/nonnumeric/overflow
- **THEN** 400 is returned without mutation, including identical repeated values
- **AND** rejected key/version inputs are not echoed in public errors

#### Scenario: Target version has no actor fallback

- **WHEN** body/query/Me contains a version or key but the designated header is missing or invalid
- **THEN** mutation is rejected rather than using actor version, automatic detail read, generated key or browser alternative sources

### Requirement: Preserve backend transition and account safety authority

REST SHALL locally validate status syntax without deciding current-state transitions, same-status no-op, self-inactivation or last-admin safety. It SHALL dispatch eligible same-status requests and leave PENDING→ACTIVE/DEACTIVATED, ACTIVE→SUSPENDED/DEACTIVATED, SUSPENDED→ACTIVE/DEACTIVATED, DEACTIVATED→ACTIVE, and valid same-state outcomes to backend authority. No target read, local state mirror, extra revocation RPC or transaction/audit implementation SHALL be added.

#### Scenario: Activation or reactivation

- **WHEN** a valid status operator requests ACTIVE for a target and the backend accepts activation/reactivation
- **THEN** REST returns the authoritative command result without login, token issuance or a second detail/session call

#### Scenario: Same-status no-op including PENDING

- **WHEN** valid input requests the target's existing status, including PENDING
- **THEN** one ChangeUserStatus is still invoked and backend changed=false/version/status are preserved
- **AND** REST does not invent changed=true, increment a version or skip backend authority

#### Scenario: Transition self or last-admin denial

- **WHEN** backend rejects a transition, self suspend/deactivate or last-active-admin inactivation
- **THEN** encoded IST/409, SIF/403 or LAR/409 is preserved according to the existing safe error policy without local override or retry

### Requirement: Leave idempotency and replay under current backend authority

Each caller retry SHALL run fresh Me and one eligible ChangeUserStatus, with no REST replay cache/fingerprint/retry. Backend SHALL own scope/fingerprint/expiry/replay, current authority before replay, version comparison, transactional audit/outbox and session/refresh revocation. REST SHALL NOT pre-check current target version or reject a saved valid result for lacking input-derived version growth. Conflict SHALL NOT cause automatic key/version replacement, detail refetch or resubmission.

#### Scenario: Committed replay

- **WHEN** caller repeats the same key and normalized request and backend returns its committed result
- **THEN** the saved safe outcome is returned after a fresh Me and one mutation, without extra target reads or local version arithmetic

#### Scenario: Authority revoked or key conflict after Me

- **WHEN** backend denies current authority or returns IC/VC/RB conflict after a successful Me
- **THEN** the safe encoded outcome is returned without using a previous success, changing inputs or automatic retry

### Requirement: Return exact safe mutation outcomes without browser effects

Success SHALL use HTTP 200/code SS/message `User status updated successfully.` with exactly numeric user_id, status string, version as exact decimal string and changed boolean including false. Successful response SHALL be nonnil, target-matching within 1–2147483647, have an allowed status and positive int64 version; malformed success SHALL fail safe 500 without partial data. No auth_version, reason, profile, credential, login_required, restriction or inferred side-effect fields SHALL be added.

Neither success, replay, no-op, target=self nor failure SHALL issue/delete cookies or require CSRF/cookie proof. Cookie credentials SHALL NOT replace bearer. Backend session invalidation SHALL NOT cause a second REST revocation call or browser cleanup inference.

#### Scenario: False changed and precise result version

- **WHEN** backend returns changed=false or a valid version above 2^53 unrelated to an input+1 assumption
- **THEN** all four exact fields are returned unchanged with explicit false and string version precision

#### Scenario: Invalid successful result

- **WHEN** success is nil, has zero/out-of-range/wrong target ID, empty/unsupported status or zero/overflow version
- **THEN** safe 500 is returned with empty error data, no partial command output and no cookie mutation

#### Scenario: Operator targets itself

- **WHEN** an eligible actor uses its own numeric target and backend returns either valid success/no-op or denial
- **THEN** the same backend contract applies with no Set-Cookie, inferred login_required or credential issuance

### Requirement: Apply no-store secure transport and existing safe error mapping

The status path SHALL emit Cache-Control no-store on all results, early CORS/auth/validation/dependency failures and OPTIONS, including malformed IDs. Allowed preflight SHALL finish under existing exact-origin CORS with zero RPCs. Calls SHALL reuse verified TLS1.3, one env-owned service credential, original bearer, safe locale/request metadata and bounded parent deadlines/cancellation. Gateway SHALL copy metadata, remove inherited version/idempotency-key and select exactly one mutation key; version SHALL be payload only.

Existing status/code/public field/Retry-After mapping and safe localized description fallback SHALL remain unchanged. User-namespace Reason descriptions not registered in the existing mapper SHALL retain its generic localized fallback while explicit public status/code remain preserved; raw internal messages SHALL NOT be exposed. No global mapper-registry expansion or automatic retry SHALL be introduced by this capability.

#### Scenario: Early failure and preflight

- **WHEN** bearer is missing, Me fails, Origin is denied, IDs/headers/body are invalid, or an allowed preflight targets the status path
- **THEN** no-store and no Set-Cookie apply without an unauthorized business RPC
- **AND** allowed preflight invokes zero RPCs

#### Scenario: Copied metadata and optional payload presence

- **WHEN** a valid request has inherited mutation metadata, unrelated safe metadata and browser service claims
- **THEN** only selected target/status/normalized reason/present expected version and one key reach the exact ChangeUserStatus method with the original bearer and one configured service credential
- **AND** caller-owned metadata and pointer values remain unchanged

#### Scenario: Structured errors and context termination

- **WHEN** backend returns safe encoded 401/403/404/409/429/503, localized public fields/RetryInfo, an unknown failure, timeout or cancellation
- **THEN** existing public status/code/field/retry policy and bounded 504/499 behavior are retained without private input/details, partial success, cookie effects or retry

### Requirement: Use detail as a separate version source and preserve existing routes

Caller MAY obtain a target Version through the independent user-detail endpoint with its own iam.user.read permission, then submit the status mutation with iam.user.status.update. Neither permission SHALL imply the other. Mutation SHALL NOT internally invoke detail or promise a prior read remains current. Existing public auth, ListUsers role gate, detail, Me, profile/password and lifecycle/self/admin-session URLs/contracts/policies SHALL remain unchanged. Only the versioned PATCH route SHALL be added; literal me SHALL not become a self alias and no POST/unversioned alias SHALL be registered.

#### Scenario: Independent detail then status requests

- **WHEN** a caller reads target 23 and submits its exact version on a separate valid status request for target 23
- **THEN** the reads and mutation execute Me→GetUser and Me→ChangeUserStatus independently with their own permissions and no hidden extra RPC
- **AND** stale-version conflict is returned without automatic refetch or retry

#### Scenario: Distinct permissions and routing

- **WHEN** a read-only caller attempts status, a status-only caller attempts detail, or public/Register/Login/ListUsers/session requests coexist
- **THEN** exact endpoint permissions remain separate, public auth stays zero-Me, ListUsers retains its legacy role gate and all existing call ordering/browser policies remain intact

#### Scenario: Concurrent isolation and aliases

- **WHEN** concurrent status/detail/session/public/ListUsers requests use distinct actors, targets, reason/key/version/permission/context/outcomes, or literal me/unversioned/POST aliases are attempted
- **THEN** each registered operation uses isolated inputs/results/metadata/cookies without duplicate Me and the status endpoint does not acquire those aliases
