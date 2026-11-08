# user-application-access Specification

## Purpose

Allow authorized operators to inspect a user's stored application access and perform atomic application-access grants or whole-application revocations through REST, preserving current backend authority, precise target versions, safe history and compound command outcomes.

## Requirements

### Requirement: Authorize each access operation with exact independent grants

The service SHALL expose GET `/api/v1/users/{user_id}/applications` requiring exact `iam.user.read=true`, POST `/api/v1/users/{user_id}/applications/{application_code}/grant` requiring BOTH exact `iam.user.application.grant=true` AND `iam.user.role.grant=true`, and POST `/api/v1/users/{user_id}/applications/{application_code}/revoke` requiring BOTH exact `iam.user.application.revoke=true` AND `iam.user.role.revoke=true`. Each eligible bearer request SHALL perform exactly one Me followed by exactly one ListUserApplicationAccess on UserService, GrantApplicationAccess on ApplicationService or RevokeApplicationAccess on ApplicationService, respectively. Missing/invalid identity SHALL receive 401; restricted or insufficiently permitted actors SHALL receive 403 before business validation or dispatch. Transport syntax errors can fail earlier. Admin role, false/wildcard/prefix/unrelated grants, target=self, cookies and caller-supplied claims SHALL NOT substitute for exact grants. Commands SHALL NOT additionally require user-read, role-read or permission-read.

#### Scenario: Permission-only reader or compound operator

- **WHEN** an unrestricted non-admin actor has exactly the read grant or both grants for the selected command and submits valid input
- **THEN** one Me precedes the single selected business RPC without an admin gate or extra user/catalog/permission/assignment lookup

#### Scenario: Either half of the command authority is absent

- **WHEN** either required command grant is missing or false, or only wildcard/prefix/another operation's grants or an admin role exists
- **THEN** 403 follows Me with no business RPC, including when domain input is invalid

#### Scenario: Restricted actor or invalid identity

- **WHEN** identity is missing/invalid, Me fails, or an actor is restricted despite holding both grants
- **THEN** authentication/restriction errors return safely without business dispatch; missing bearer invokes zero RPCs and forged claims or cookies provide no fallback

### Requirement: Select the path target and keep read grant and revoke inputs distinct

Target user_id SHALL come only from nonempty path ASCII decimal digits representing 1–2147483647, accepting leading zeros. Empty/zero/signed/whitespace/hex/fraction/non-ASCII/overflow values and literal me SHALL fail 400 without a business RPC. Command application_code SHALL come only from the path and match exact lowercase ASCII `[a-z][a-z0-9_]*`, 1–50 bytes, without trimming or case correction. A syntactically valid code including iam SHALL be passed to backend eligibility/system protection without a local catalog lookup. Grant SHALL accept a JSON object with required role_ids and reason; revoke SHALL accept a JSON object with required reason. Empty/malformed/non-object bodies or wrong types for recognized fields SHALL fail 400. Unknown body/query/header claims SHALL be ignored and SHALL NOT replace target, code, actor, grants, recognized input or preconditions. In particular, revoke role_ids SHALL be ignored, SHALL NOT be type-validated as a recognized field, and SHALL NOT narrow the application-wide command. Reads SHALL ignore bodies including malformed JSON and unused/repeated/malformed Version/Idempotency-Key headers.

#### Scenario: Actor target and selected application remain independent

- **WHEN** actor 17 commands path user 00023/application clm while other sources claim user 99, application iam or different permissions
- **THEN** only numeric user 23 and path code clm reach the selected command with verified actor authority

#### Scenario: Path syntax and bounds

- **WHEN** user_id reaches valid boundaries or application_code is 1/50 bytes, empty, 51 bytes, uppercase, padded, numeric-first or contains forbidden bytes
- **THEN** valid path identities are preserved and invalid ones receive safe 400 without identity resolution or a business call

#### Scenario: Revoke cannot be narrowed by an extra role selection

- **WHEN** a valid revoke body also supplies role_ids as any JSON type or other unknown selected-role claims
- **THEN** one RevokeApplicationAccess receives user_id, application_code, reason and expected version only, with no selected-role field or separate RevokeUserRoles call

#### Scenario: Bodies and unused read headers

- **WHEN** a read includes malformed JSON/unused mutation headers or a command includes absent/malformed/non-object JSON or wrong recognized field types
- **THEN** the read ignores those unused inputs and commands fail safe 400 for invalid transport syntax without mutation dispatch

### Requirement: Parse bounded access history filters and pagination

List SHALL support page, limit, last_id, keyword, application_code and statuses, each with at most one decoded query value, rejecting identical duplicates. Omitted page/limit SHALL default to 1/20. Present page SHALL be ASCII decimal 1–2147483647 and limit 1–100, accepting leading zeros but rejecting empty, signed, padded, nondecimal or overflowing values. Page SHALL NOT be capped by total_pages nor limit by total. Present application_code SHALL use the same exact code grammar as command paths; omitted code SHALL leave scope to backend. Omitted statuses SHALL include stored ACTIVE and REVOKED access history. Present statuses SHALL be nonempty CSV of those exact codes, trimming ordinary ASCII spaces around tokens, deduplicating and sorting canonically; empty tokens, case-folded values and DISABLED SHALL fail. Selected statuses SHALL combine OR and filter/search categories AND. Present fields, sort, search, search_fields, search_mode, created_from, created_to, updated_from, updated_to and code SHALL fail 400 even when empty; other unknown query keys SHALL be ignored without granting authority. Independent invalid fields SHALL accumulate safely in EN/ID/ZH without echoing input.

#### Scenario: Defaults and full numeric range

- **WHEN** pagination is omitted, uses valid boundary/leading-zero values or selects a page beyond total_pages
- **THEN** one list dispatch uses the selected/default values without a total-based cap or hidden page aggregation

#### Scenario: Duplicate unsupported and invalid controls

- **WHEN** any supported query is repeated or numeric/code/status controls are invalid or a known unsupported control is supplied
- **THEN** safe localized 400 returns without ListUserApplicationAccess, accumulating independent field errors where applicable

#### Scenario: Access status differs from definition status

- **WHEN** statuses is omitted, equals ` REVOKED,ACTIVE,ACTIVE ` or equals DISABLED
- **THEN** access history is unrestricted, canonically filtered to ACTIVE/REVOKED or rejected respectively, without confusing application_status with access status

### Requirement: Use supported literal access search and stable opaque pagination

Keyword SHALL allow at most 200 raw and normalized bytes of printable ASCII plus whitespace 0x09–0x0d, reject forbidden bytes before normalization and collapse permitted whitespace to single spaces. Omitted/empty/whitespace-only keyword SHALL disable search. Nonempty keyword SHALL select explicit application_code/name fields with case-insensitive literal CONTAINS, preserving `%`, `_` and backslash as literal search input; REST SHALL NOT send logical keyword FULL_TEXT_TRIGRAM, offer caller-selected modes, promise ranking or retry with another search mode. List SHALL request full safe projection and explicit application_id ASC ordering. last_id SHALL be a once-URL-decoded opaque string of at most 2048 bytes, preserved without trim, base64 decode, numeric interpretation, synthesis or arithmetic. Empty/omitted last_id SHALL mean no cursor. Backend SHALL own cursor validity and target/filter/search/sort binding; callers SHALL reset cursor when those controls change. REST SHALL NOT repair/drop rejected cursors, refetch or loop over pages.

#### Scenario: Supported literal keyword payload

- **WHEN** an eligible list contains normalized nonempty keyword with literal percent/underscore/backslash characters
- **THEN** one ListUserApplicationAccess receives explicit application_code/name CONTAINS search and application_id ASC without logical full-text search or fallback calls

#### Scenario: Safe raw keyword validation

- **WHEN** keyword contains permitted whitespace, reaches 200/201 raw bytes or includes forbidden control/non-ASCII bytes
- **THEN** valid text is normalized, whitespace-only disables search and invalid/overlong raw text fails safely without hidden byte removal

#### Scenario: Cursor preservation and rejection

- **WHEN** a cursor includes encoded punctuation/spaces, reaches 2048/2049 decoded bytes or backend rejects target/query binding
- **THEN** in-range decoded bytes pass unchanged, overlong input fails locally and authoritative cursor errors return without replacement or retry

### Requirement: Return only stored safe access history with a precise target version

List success SHALL be HTTP 200/code SS/message `User application access retrieved successfully.` with exactly items, total, page, limit, total_pages, has_next, has_prev, next_cursor, user_id and version in data. user_id SHALL match target; version SHALL be the backend target's positive int64 entity value encoded as an exact decimal string. total SHALL preserve nonnegative int64 without narrowing/float conversion; backend pagination/flags/cursor SHALL be preserved without recomputation or deriving total from items. Empty items SHALL serialize as []. Each item SHALL contain exactly the eleven fields below. Backend order/text SHALL be preserved without write-time string normalization or inferred effective access. Stored ACTIVE/REVOKED records for nondeleted targets in any account status, including system/IAM or DISABLED application definitions, SHALL remain readable for eligible actors. Never-granted applications SHALL NOT be synthesized from a catalog.

| Fields | Representation |
| --- | --- |
| application_id | Positive numeric ID in 1–2147483647; no invented access-row ID |
| application_code, name | Unchanged backend strings |
| application_status | ACTIVE or DISABLED definition state |
| is_system | Explicit backend boolean including false |
| status | ACTIVE or REVOKED stored access state |
| granted_by, revoked_by | Explicit null or positive numeric actor ID in 1–2147483647 |
| granted_at | Required valid UTC RFC3339 timestamp retaining fractional precision |
| revoked_at | Explicit null or valid UTC RFC3339 timestamp retaining fractional precision |
| revoke_reason | Explicit null or unchanged backend string |

Optional provenance SHALL remain nullable without fabricated actors/times/reasons or write-time validation of historical reason text. No definition version, roles, profile, credentials or effective permissions SHALL be added. A returned target version SHALL be available for a separately authorized command, without guaranteeing freshness or write permission.

#### Scenario: Complete access history and inactive definitions

- **WHEN** backend returns ACTIVE/REVOKED stored access, system/disabled applications, nullable provenance, false flags and historical text
- **THEN** exactly eleven item fields preserve the safe backend values with required fractional granted_at and explicit optional nulls

#### Scenario: Large version total and empty page

- **WHEN** version is 9007199254740993 or MaxInt64, total exceeds MaxInt32 or a valid page has no items with a nonzero total
- **THEN** exact string version/int64 total and backend page metadata return unchanged with items=[] where appropriate

#### Scenario: No stored access or target not found

- **WHEN** backend reports an existing nondeleted target with no stored access or rejects a missing/deleted target
- **THEN** its empty versioned page or safe not-found error returns without fabricating catalog rows or issuing GetUser/ListApplications

### Requirement: Validate the additive grant role selection without silent deduplication

Grant role_ids SHALL be a JSON array with 1–20 submitted entries, checked before normalization. Each entry SHALL be a string of ASCII decimal digits representing 1–2147483647, accepting leading zeros. Missing/null/empty arrays, numeric/mixed/null elements, labels/codes/objects, signs, whitespace, non-ASCII, zero and overflow SHALL fail 400. Duplicate numeric identities, including `"10"` and `"00010"`, SHALL fail rather than silently deduplicate. Valid roles SHALL be copied and sorted numerically for one compound command, without mutating inputs, splitting batches or checking the role catalog. Backend SHALL own role existence, application ownership, active state and additive behavior. Revoke SHALL have no role selection, batch input cap or caller-selected subset.

#### Scenario: Valid option values and count boundaries

- **WHEN** grant contains 1 or 20 distinct valid string-decimal roles, including leading-zero/boundary IDs
- **THEN** an owned ascending numeric set reaches one GrantApplicationAccess without a catalog check

#### Scenario: Duplicate or oversized raw input

- **WHEN** grant repeats numeric identity or supplies 21 entries even if deduplication could reduce them
- **THEN** safe role_ids errors return without truncation, deduplication to success or any command dispatch

#### Scenario: Wrong selected-role representation

- **WHEN** grant has missing/null/empty/wrong-typed role selection or invalid elements
- **THEN** safe 400 returns without ID coercion, label lookup or alternate input source

### Requirement: Require bounded reasons and exact target command preconditions

Both commands SHALL require string reason whose raw bytes are printable ASCII 0x20–0x7e plus CR/LF; tab/NUL/DEL/other controls/non-ASCII SHALL fail before trimming surrounding space/CR/LF. The trimmed reason SHALL be 3–500 bytes with internal bytes preserved. Independent grant role_ids/reason domain errors SHALL accumulate and localize EN/ID/ZH without input echo. Exactly one case-insensitively parsed Idempotency-Key and one Version header SHALL be required. Key SHALL be nonblank printable ASCII 1–200 bytes, remaining opaque including spaces/commas without trim, key generation or UUID requirements. Version SHALL be ASCII decimal 1–9223372036854775807, accepting leading zeros, passed as explicitly present expected_user_version without float conversion. Missing/duplicate/invalid headers SHALL fail 400 without body/query/Me fallback, target pre-read or generated replacement.

#### Scenario: Normalized reason and precise opaque preconditions

- **WHEN** valid reason trims to 3 or 500 bytes and Version exceeds 2^53 with a printable key containing spaces/commas
- **THEN** one command receives the exact numeric expected target version, unchanged selected key and internally unchanged trimmed reason

#### Scenario: Invalid or independently invalid reason and roles

- **WHEN** reason is absent/null/wrong-typed, contains forbidden bytes or exceeds bounds and grant roles independently fail
- **THEN** safe transport/domain 400 returns, accumulating independent domain fields where types parse, without a compound command

#### Scenario: Header multiplicity and source isolation

- **WHEN** either header is missing/repeated under any capitalization, key is blank/control/non-ASCII/overlong or Version is zero/signed/padded/nondecimal/overflow
- **THEN** safe 400 returns even for identical duplicates or available body/query/actor alternatives, without automatic reads or mutation

### Requirement: Preserve backend compound grant atomicity and eligibility

Eligible REST grant SHALL dispatch GrantApplicationAccess once with all selected roles, target, path application, reason and expected user version. Backend SHALL own current authority, nondeleted ACTIVE target, ACTIVE nonsystem non-IAM application, registered ACTIVE same-application roles, atomic access creation/reactivation and additive selected-role activation. Existing ACTIVE provenance and unselected assignments SHALL remain backend-owned. REST SHALL NOT pre-read eligibility, activate user/application/role definitions, call individual GrantUserRoles, run local transactions or expose partial success. Unknown/disabled/mismatched roles, inactive target/application, system application or any atomic dependency failure SHALL retain their authoritative safe error.

#### Scenario: Access creation or reactivation with selected roles

- **WHEN** backend accepts the entire selected set and returns changed access/roles or an eligible no-op
- **THEN** the exact safe compound result returns after one command without local assignment/access writes or extra RPCs

#### Scenario: One selected role or eligibility check fails

- **WHEN** backend rejects a missing/disabled/wrong-application role, inactive target/application or a compound write dependency
- **THEN** only the safe error returns with no partial data, split batch, user activation or compensating role/access command

### Requirement: Revoke whole-application access and all active application roles

Eligible REST revoke SHALL dispatch RevokeApplicationAccess once with no role selection. Backend SHALL own revocation of ACTIVE stored access and every ACTIVE role assignment for the selected application, including more than 20 roles, in one atomic command. Revoke SHALL permit backend-authorized cleanup of inactive nondeleted targets or disabled definitions without applying grant eligibility locally. Absent access SHALL remain absent; already-revoked provenance and other applications/credentials SHALL remain backend-owned. Cleanup of roles without a stored access row SHALL be representable by null access_status. No local row creation, role-list loop, per-role RevokeUserRoles or grant fallback SHALL occur.

#### Scenario: More than twenty active roles

- **WHEN** backend revokes access and more than 20 role assignments
- **THEN** all actual changed role IDs return safely without truncation, a 20-role result cap or additional business calls

#### Scenario: Absent access with active roles

- **WHEN** backend cleans active roles while no access pair exists
- **THEN** changed=true, explicit null access_status and the exact changed-role list return without inventing an access row or interpreting null as an error

#### Scenario: Inactive target disabled definitions or revoked history

- **WHEN** backend accepts cleanup/no-op for an inactive nondeleted target, disabled application/roles or already-revoked history
- **THEN** its authoritative outcome is preserved without local grant restrictions, status activation or rewriting provenance

### Requirement: Leave system protection concurrency and retained replay to current backend authority

Both commands SHALL leave method/IAM/current-user authority, system/IAM application protection, target existence/version, atomicity, locks, audit/outbox, idempotency scope/fingerprint/expiry, no-op and committed replay to backend. Syntactically valid system-application requests SHALL retain backend APPLICATION_ACCESS_SYSTEM_PROTECTED/AASP/HTTP409 handling. Every caller retry SHALL perform fresh Me and one eligible selected command; REST SHALL NOT cache results, compare versions locally, assume result=input+1, short-circuit no-op or retry after conflict/busy/uncertain outcome. Backend denial after Me or on replay SHALL remain authoritative. A stale version/key conflict SHALL NOT generate a new key/version or trigger a read/resubmission/alternate command.

#### Scenario: System-application protection

- **WHEN** backend denies a syntactically valid command for iam or another system application with encoded AASP/HTTP409
- **THEN** the existing safe error mapping returns without local lookup, activation or fallback

#### Scenario: No-op and committed replay

- **WHEN** backend returns changed=false/no changed IDs or a retained original command result on caller retry
- **THEN** its saved version and outcome return after fresh Me and one command without incrementing versions or repeating local effects

#### Scenario: Current authority and concurrency conflicts

- **WHEN** backend denies changed authority/method/IAM scope or reports stale version, idempotency conflict or busy resource after Me
- **THEN** the safe authoritative error returns without replay cache success, hidden reads, key replacement or automatic retry

### Requirement: Return exact safe compound command outcomes

Grant/revoke success SHALL be HTTP 200/code SS/message `Application access granted successfully.` or `Application access revoked successfully.` with exactly user_id, application_id, access_status, changed_role_ids, version and changed. user_id SHALL match the target; numeric user/application IDs SHALL be 1–2147483647. version SHALL be a positive int64 decimal string without synthesis/arithmetic/float conversion. Grant access_status SHALL be ACTIVE; revoke access_status SHALL be REVOKED or explicit null for absent stored access. changed_role_ids SHALL serialize as [] when empty and otherwise be strictly ascending distinct positive PostgreSQL-range numeric IDs. Grant changed IDs SHALL be a subset of requested roles; revoke changed IDs SHALL have no requested-subset or 20-ID constraint. changed=false SHALL require an empty changed-role array; changed=true SHALL permit an empty array for access-only changes and SHALL NOT be derived solely from role count. Backend results SHALL NOT be repaired, sorted or truncated. A valid numeric application ID SHALL be accepted without a catalog lookup to correlate it to path code, which is resolved authoritatively by backend. No reason/key/auth_version/permissions/profile/credentials or inferred login_required SHALL be exposed.

#### Scenario: Access-only change or full no-op

- **WHEN** backend returns changed=true with no changed roles, or changed=false with no changed roles
- **THEN** exactly six safe fields preserve the explicit changed flag, [] array, status and precise version without role-count-derived inference

#### Scenario: Grant subset and unlimited revoke result

- **WHEN** grant returns a valid selected subset or revoke returns all actual sorted changed IDs including more than 20
- **THEN** the exact owned numeric list returns without output truncation or a revoke subset check

#### Scenario: Malformed command outcome

- **WHEN** success is nil or has invalid/wrong target, invalid application ID/version, invalid operation status, duplicate/unsorted/out-of-range roles, unrequested grant roles or changed=false with changed roles
- **THEN** safe HTTP 500 returns with empty error data and no partial output, repair or recovery RPC

### Requirement: Reject malformed list successes and isolate all mapped data

List SHALL fail safe HTTP 500 with empty error data for nil response/required metadata, wrong or invalid target/version, nil items, invalid application/optional actor IDs, unsupported access/definition states, absent/invalid granted_at, invalid present revoked_at, negative total/total_pages, page below 1 or limit outside 1–100. Missing nullable provenance SHALL remain valid without deriving state-specific values or applying new-input text rules to stored strings. Valid request filters/role sets/outgoing metadata and response collections/optional pointers SHALL be independent of caller/dependency objects and other requests. Malformed entries SHALL fail the whole response rather than be skipped; metadata and empty outcomes SHALL NOT be fabricated.

#### Scenario: Invalid metadata target or history entry

- **WHEN** an otherwise successful page has malformed metadata/target/version or a valid row followed by an invalid entry/timestamp
- **THEN** safe 500 returns without preceding rows, fabricated defaults, a catalog supplement or another RPC

#### Scenario: Nullable provenance and ownership

- **WHEN** optional provenance is absent or mapped items/filters/role IDs/metadata are modified after mapping
- **THEN** documented nullability is retained and valid mapped objects remain independent without changing caller/dependency data or another result

### Requirement: Reuse secure transport and shared safe error mapping

Operations SHALL reuse verified TLS1.3-or-later transport and one configured environment-owned service credential, original bearer, safe locale/request ID metadata and the earlier parent/client deadline with cancellation/cleanup. Caller claims SHALL NOT select service identity. Copied outgoing business metadata SHALL remove inherited version/idempotency-key; reads SHALL send neither precondition, and commands SHALL send only the selected key in metadata and explicit expected_user_version in payload. Nil context/request/response handling SHALL remain safe. Original dependency error identity and shared explicit public status/code/registered locale/structured fields/unknown-Reason localized fallback/Retry-After and established 504/499 behavior SHALL remain unchanged without global reason-registry expansion, raw private messages, secret output, alternate identity or retries.

#### Scenario: Exact generated services and copied metadata

- **WHEN** an eligible operation carries safe parent metadata, old mutation preconditions and forged browser service claims
- **THEN** the exact UserService read or ApplicationService command receives owned safe payload/metadata, one configured credential and original bearer on the same secure connection without parent mutation

#### Scenario: Encoded errors and retry guidance

- **WHEN** backend returns known application/user/role errors, explicit public 401/403/404/409/429/503, structured localized fields, RetryInfo or an unregistered Reason
- **THEN** existing safe status/code/fields/locale/fallback and ceil Retry-After behavior returns without input echo, registry expansion or partial success

#### Scenario: Earlier deadline cancellation and cleanup

- **WHEN** parent context is already ended, canceled or has an earlier deadline during any operation
- **THEN** no deadline is extended, existing 499/504 handling and cleanup apply, and no replacement request or automatic retry occurs

### Requirement: Apply no-store browser isolation and preserve route coexistence

All access paths SHALL emit Cache-Control no-store on success, early CORS/auth/input/backend failure, malformed target/code and allowed GET/POST preflight, including standalone handlers. Exact-origin CORS SHALL complete allowed preflight with zero RPCs. Supplied Origin SHALL retain existing validation; no new mandatory Origin/CSRF proof, cookie authentication, cookie issuance/deletion or token response SHALL be introduced for list/commands/self/no-op/replay/failure. Only the three versioned operations SHALL be added; standard framework HEAD can follow GET policy, with no self/unversioned/PUT/PATCH/DELETE aliases. Existing application/role catalogs, selected-user-role commands, user-detail short access view, public auth, legacy ListUsers role/cache behavior and other profile/password/session/status/creation contracts SHALL remain unchanged. Concurrent requests SHALL isolate actor, target, code, filters/search/cursors, role sets, key/version/reason, nullable results, locale, errors and browser effects on the shared connection.

#### Scenario: Early failure malformed identities and preflight

- **WHEN** missing bearer, denied Origin, failed Me, malformed target/code, invalid input, standalone failure or allowed preflight reaches an access path
- **THEN** no-store and no Set-Cookie apply without unauthorized business dispatch, while allowed preflight invokes zero RPCs

#### Scenario: Existing routes and independent permission sources

- **WHEN** new access routes coexist with catalog options, role assignments, detail, legacy users and public auth or unsupported aliases are attempted
- **THEN** each existing route retains its contract/authority/cache policy, catalog reads stay informational, public auth stays zero-Me and no extra access aliases or implicit grants appear

#### Scenario: Mixed simultaneous operations

- **WHEN** controlled access list/grant/revoke and existing user/catalog/assignment/public requests use distinct actors, targets, applications, inputs, contexts and results
- **THEN** each receives its own method/payload/metadata/output/error with correct one-Me/business ordering, independent collections, no cookies and no data race
