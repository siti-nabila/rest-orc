# Spec Delta

## Purpose

Allow authorized operators to read a user's role assignment history and grant or revoke selected roles through REST while preserving target versions, current backend authority, atomicity, idempotency and safe provenance.

## ADDED Requirements

### Requirement: Authorize each assignment operation with one current identity and exact permission

The service SHALL expose GET `/api/v1/users/{user_id}/roles` with exact `iam.user.read=true`, POST `/api/v1/users/{user_id}/roles/grant` with exact `iam.user.role.grant=true`, and POST `/api/v1/users/{user_id}/roles/revoke` with exact `iam.user.role.revoke=true`. Each eligible bearer request SHALL perform one Me followed by exactly one ListUserRoles, GrantUserRoles or RevokeUserRoles, respectively. Missing/invalid identity SHALL receive 401 and restricted or insufficiently permitted actors SHALL receive 403 before business validation or dispatch. Transport syntax errors can fail earlier. Admin role, wildcard/prefix/false grants, another operation's permission, cookies and caller-supplied authority SHALL NOT substitute for the exact grant. Commands SHALL NOT require an additional user-read or role-read grant.

#### Scenario: Separate permission-only operators

- **WHEN** unrestricted non-admin actors possess only the exact permission for their chosen list, grant or revoke operation and submit valid input
- **THEN** each request performs one Me followed by its single business RPC without another authority/catalog/detail lookup

#### Scenario: Ineligible actor before business validation

- **WHEN** a restricted actor or an actor with missing, false, wildcard, prefix, role-only or unrelated permission submits business-invalid input
- **THEN** 403 is returned without the assignment RPC or existence disclosure

#### Scenario: Authentication cannot fall back

- **WHEN** bearer is absent, Me fails or identity is invalid, including with cookies or forged actor/service claims
- **THEN** no assignment RPC or alternate identity is used, and absent bearer invokes zero RPCs

### Requirement: Select only the numeric target path and designated inputs

Target user_id SHALL come exclusively from nonempty ASCII decimal path digits representing 1–2147483647; leading zeros SHALL be accepted numerically. Zero, signs, whitespace, fractions, hex, Unicode digits, overflow and literal me SHALL fail 400 without an assignment RPC. Read SHALL ignore bodies, including malformed JSON, and unused Version/Idempotency-Key headers. Commands SHALL accept a JSON object with required role_ids and reason; absent/malformed/non-object bodies or wrong field types SHALL fail 400. Unknown body/query claims SHALL be ignored and SHALL NOT replace target, actor, permissions, reason, selected IDs or header preconditions.

#### Scenario: Independent actor and target sources

- **WHEN** actor 17 uses path user 00023 while body/query/header claims identify user 99 or greater authority
- **THEN** only numeric target 23 and the operation's designated inputs reach the business RPC with actor 17 unchanged

#### Scenario: Target boundary and invalid forms

- **WHEN** target is 1, 2147483647, a leading-zero valid decimal or an invalid form
- **THEN** valid targets map numerically and invalid forms fail 400 with no target lookup or self alias

#### Scenario: Read input tolerance and command syntax

- **WHEN** a read contains malformed JSON and repeated invalid mutation headers, or a command body is missing, null, array, scalar, malformed or wrong-typed
- **THEN** the read ignores those unused inputs and a command fails safely without business dispatch or alternative input sources

### Requirement: Parse bounded assignment pagination and server-owned controls

List SHALL support page, limit, last_id, keyword, application_code and statuses, each with at most one decoded query value, rejecting identical duplicates. Omitted page/limit SHALL default to 1/20. Present page SHALL be ASCII decimal 1–2147483647 and limit 1–100, accepting leading zeros without sign, whitespace, empty value, fraction or overflow. Page SHALL NOT be capped by total_pages nor limit by total. Present fields, sort, search, search_fields, search_mode, created_from, created_to, updated_from, updated_to and code SHALL fail 400 even if empty; other unknown keys SHALL be ignored without granting authority.

#### Scenario: Defaults and full numeric bounds

- **WHEN** pagination is omitted or contains valid boundary/leading-zero values
- **THEN** one ListUserRoles uses defaults 1/20 or the selected numbers without a total-derived cap

#### Scenario: Duplicate or invalid supported query

- **WHEN** any supported key is repeated or a numeric control is empty, signed, non-ASCII, whitespace-bearing or out of range
- **THEN** safe localized validation fails before ListUserRoles

#### Scenario: Unsupported projection and sort

- **WHEN** caller-controlled fields/sort/search modes, dates or exact code filtering are supplied
- **THEN** 400 is returned without forwarding them or silently treating a known unsupported control as accepted

### Requirement: Filter assignment history and normalize logical keyword safely

Omitted application_code SHALL leave scope to backend authority; a present value SHALL match lowercase ASCII `[a-z][a-z0-9_]*` with 1–50 bytes without trim/case correction. Omitted statuses SHALL retain ACTIVE and REVOKED history. Present statuses SHALL be nonempty CSV of those exact assignment states, trim ordinary ASCII spaces around tokens, deduplicate and sort values; empty tokens or other states SHALL fail. Categories SHALL combine AND and selected states OR. Keyword SHALL allow at most 200 raw and normalized bytes of printable ASCII plus whitespace 0x09–0x0d, reject invalid bytes before normalization, collapse permitted whitespace to single spaces, and select no search when omitted/empty/whitespace-only. Nonempty text SHALL use logical keyword FULL_TEXT_TRIGRAM search over the backend's assignment role-name search. Independent field errors SHALL accumulate safely and localize EN/ID/ZH without echoing input.

#### Scenario: Assignment states differ from catalog states

- **WHEN** statuses is omitted, ` REVOKED,ACTIVE,ACTIVE ` or DISABLED
- **THEN** history is unrestricted, canonically filtered to ACTIVE/REVOKED, or rejected, respectively, without treating role_status as assignment status

#### Scenario: Safe keyword normalization

- **WHEN** keyword has allowed whitespace, reaches the 200-byte boundary or contains forbidden control/non-ASCII bytes
- **THEN** valid text is normalized for logical keyword search and forbidden/overlong raw text fails without hidden character removal

#### Scenario: Independent query errors

- **WHEN** page, limit, application_code, statuses and keyword independently fail
- **THEN** all applicable safe field errors are returned in the selected locale without ListUserRoles

### Requirement: Preserve stable assignment ordering and opaque cursors

List SHALL request explicit id ASC ordering and the full safe assignment projection. last_id SHALL remain a once-URL-decoded opaque string up to 2048 bytes with no trim, base64 decode, numeric interpretation, synthesis or cursor arithmetic. Empty/omitted cursor SHALL mean no cursor. Backend SHALL own validity, target/query binding and rejection; callers SHALL reset their cursor when target/filter/keyword/sort changes. REST SHALL NOT drop or repair rejected cursors, retry, aggregate multiple pages or make role/application lookups.

#### Scenario: Full stable history page

- **WHEN** list is requested with supported controls
- **THEN** one ListUserRoles receives explicit id ASC and full safe assignment fields without a user-selected projection

#### Scenario: Cursor bytes and length

- **WHEN** a cursor contains encoded punctuation/spaces, is empty or exceeds 2048 decoded bytes
- **THEN** an in-range decoded value is preserved exactly, empty disables cursor and overlong values fail locally

#### Scenario: Query-bound cursor failure

- **WHEN** backend rejects a cursor reused for another target/query or malformed cursor bytes
- **THEN** its safe error returns after one ListUserRoles with no refetch or cursor replacement

### Requirement: Return precise assignment pages with nullable provenance

List success SHALL be HTTP 200/code SS/message `User roles retrieved successfully.` with data containing exactly items, total, page, limit, total_pages, has_next, has_prev, next_cursor, user_id and version. Top-level user_id SHALL match the numeric target and version SHALL be its positive int64 entity version as an exact decimal string. Nonnegative int64 total and backend pagination/cursor/flags SHALL be preserved without narrowing, recomputation or deriving total from current items; empty items SHALL be []. Each item SHALL contain exactly the 17 fields below. Text and item order SHALL remain unchanged without write-time validation or inference of effective permissions.

| Fields | Representation |
| --- | --- |
| id, role_id, application_id | Positive numeric IDs in 1–2147483647; id identifies the assignment rather than the role |
| code, name, application_code | Backend strings; code is distinct from role_id and role_code |
| role_code | Positive int32 number |
| status | ACTIVE or REVOKED assignment state |
| role_status, application_status | ACTIVE or DISABLED definition state |
| is_system | Backend boolean including false |
| application_access_status | Explicit null or ACTIVE/REVOKED |
| granted_by, revoked_by | Explicit null or positive numeric ID in 1–2147483647 |
| granted_at, revoked_at | Explicit null or valid UTC RFC3339 timestamp preserving fractional precision |
| revoke_reason | Explicit null or unchanged backend string |

Nullable legacy provenance SHALL remain valid even for ACTIVE/REVOKED records, without fabricating actor/time/reason or requiring one state's metadata from another. A returned version SHALL be usable by a separately authorized command; the read SHALL confer no write permission or freshness guarantee.

#### Scenario: Complete safe history

- **WHEN** backend returns ACTIVE/REVOKED assignments, DISABLED role/application definitions, null legacy provenance, false system flags and historical text
- **THEN** all 17 documented item fields retain their safe values and explicit nulls without hiding inactive history or claiming effective access

#### Scenario: Large total precise version and empty page

- **WHEN** total exceeds 2147483647, version is 9007199254740993 or 9223372036854775807, or the requested page has no items with nonzero total
- **THEN** exact int64 total, string target version and unchanged pagination return with items=[] and no count/version arithmetic

#### Scenario: Caller uses returned target version

- **WHEN** a caller separately submits list/detail target version on a grant/revoke request with that command's permission
- **THEN** command dispatch uses only its explicit Version header and backend decides freshness without an internal read or implicit permission grant

### Requirement: Validate selected role IDs without silently deduplicating

Commands SHALL require role_ids as a JSON array of 1–20 string decimal IDs, checking entry count before any normalization. Each element SHALL contain only ASCII decimal digits representing 1–2147483647, accepting leading zeros. Missing/null/empty array, numeric/mixed/null elements, labels/string codes/role_codes, signs, whitespace, non-ASCII, zero and overflow SHALL fail 400. Duplicate numeric identities, including `"10"` and `"00010"`, SHALL fail rather than deduplicate to success. Valid IDs SHALL become an owned ascending numeric sequence for one atomic backend command; this normalization SHALL NOT mutate caller data or split the batch.

#### Scenario: Selected option values and count boundaries

- **WHEN** role_ids contains 1 or 20 distinct valid string IDs, including leading-zero IDs and a boundary ID
- **THEN** their numeric identities are sorted in an owned sequence and sent once without a role-catalog confirmation

#### Scenario: Duplicate identity and oversized input

- **WHEN** role_ids repeats an ID directly or numerically, or contains 21 entries even if duplicates could reduce the size
- **THEN** safe role_ids errors return 400 without silently deduplicating, truncating or issuing any command

#### Scenario: Missing or wrong identity representation

- **WHEN** role_ids is omitted/null/empty, has numeric/mixed/null elements or contains invalid decimal strings, labels or role codes
- **THEN** 400 is returned without identity lookup, coercion or fallback

### Requirement: Validate required reason and exact command preconditions

Commands SHALL require string reason whose raw bytes allow only ASCII 0x20–0x7e plus CR/LF. They SHALL reject tab/NUL/DEL/other controls/non-ASCII before trimming surrounding space/CR/LF, then require 3–500 bytes, preserving internal bytes. Independent role_ids/reason errors SHALL accumulate and localize without input echo. Exactly one case-insensitive parsed Idempotency-Key and one Version header SHALL be required. Key SHALL be nonblank printable ASCII 1–200 bytes, preserving spaces/commas as opaque content without trim/generation/hash/UUID rules. Version SHALL be ASCII decimal 1–9223372036854775807, accepting leading zeros with explicit expected_user_version presence and no float conversion. Missing/duplicate/invalid headers SHALL fail 400 without body/query/actor fallback or automatic reads.

#### Scenario: Safe reason and precise opaque headers

- **WHEN** valid reason normalizes to 3 or 500 bytes and Version exceeds 2^53 with a printable key containing spaces/commas
- **THEN** exact numeric target version, one unchanged parsed key and internally unchanged trimmed reason reach the command

#### Scenario: Invalid bytes and accumulated body fields

- **WHEN** role_ids is invalid and reason is missing/null, wrong-typed, forbidden-byte-bearing or outside its normalized length
- **THEN** safe 400 is returned, accumulating independent domain field errors where transport types parse, without command dispatch

#### Scenario: Required header multiplicity and no fallback

- **WHEN** either header is missing/repeated under any capitalization, key is blank/control/non-ASCII/overlong, or Version is zero/signed/whitespace/overflow
- **THEN** safe 400 returns even if identical values or body/query/Me alternatives exist, without generation, refetch or mutation

### Requirement: Preserve backend eligibility atomicity and safety decisions

REST SHALL dispatch each eligible full grant/revoke batch once and SHALL NOT determine eligibility from a prior list, role status, local state or caller claims. Backend SHALL own current method/IAM/actor authority, target existence/version, whole-batch atomicity, audit/outbox and policy locks. Grant SHALL leave target ACTIVE, role/application ACTIVE and non-IAM ACTIVE application-access checks to backend, including for already-ACTIVE assignments. Revoke SHALL leave self platform-admin and last-active-admin safety to backend and SHALL NOT require local grant-eligibility checks. REST SHALL NOT grant application access, activate targets/definitions, change sessions, issue credentials or implement transactions locally.

#### Scenario: Eligible grant or backend rejection

- **WHEN** a valid grant batch is accepted or backend rejects inactive target, missing/disabled role, inactive application or missing application access
- **THEN** the authoritative safe outcome is returned with no partial local success or extra activation/access/catalog RPC

#### Scenario: Revoke inactive history or protected administrator

- **WHEN** backend accepts revoke/no-op despite inactive grant prerequisites or denies self platform-admin/last-admin revocation
- **THEN** its authoritative result or safe denial is retained without applying grant restrictions or overriding safety locally

#### Scenario: Atomic multi-role failure

- **WHEN** one selected role fails backend validation or an atomic backend dependency fails
- **THEN** only the shared safe error returns with empty data, without reporting earlier roles as successful or splitting/retrying the batch

### Requirement: Leave no-op idempotency and replay to current backend authority

Each caller retry SHALL run fresh Me and one eligible command. Backend SHALL own normalized fingerprint, actor/method/target/key scope, expiry, current authority before replay, version comparison and committed outcome. Already-active grants and absent/revoked registered-role revokes SHALL still dispatch for backend eligibility/safety/no-op decisions. REST SHALL have no replay cache, automatic retry, pre-read version comparison or input-derived result-version growth rule. Conflicts SHALL NOT trigger a replacement key/version, read, resubmission or grant/revoke fallback.

#### Scenario: Eligible no-op

- **WHEN** backend accepts an already-active grant or absent/revoked assignment revoke as changed=false
- **THEN** its saved version and empty changed_role_ids are returned after one command without a local short circuit or increment

#### Scenario: Committed replay

- **WHEN** a caller repeats its key and normalized payload and backend returns a committed result
- **THEN** fresh Me and one command precede the exact safe saved outcome without requiring input+1 or refetching current assignments

#### Scenario: Current authority or conflict after Me

- **WHEN** backend rejects current authority, method/scope, version, key reuse or a busy resource after successful Me
- **THEN** the encoded safe failure is retained without cached success, extra business calls or automatic retries

### Requirement: Return exact safe command outcomes and reject malformed successes

Grant/revoke success SHALL be HTTP 200/code SS with message `User roles granted successfully.` or `User roles revoked successfully.` and exactly user_id, changed_role_ids, version and changed. user_id SHALL match the target within 1–2147483647; version SHALL be a positive int64 decimal string; changed_role_ids SHALL be [] or strictly ascending distinct positive PostgreSQL-range numeric role IDs drawn from the requested set; changed SHALL equal whether that list is nonempty, preserving false. Returned valid values SHALL NOT be synthesized, reordered or incremented.

All three operations SHALL fail safe 500 with empty data for nil responses, wrong/invalid target/version or malformed success. List SHALL also reject nil required metadata, nil entries, invalid item/optional actor IDs, role_code, enum or present timestamp, negative total/total_pages, page below 1 or limit outside 1–100. Missing nullable provenance SHALL remain valid. Collections/pointers SHALL be independent of caller/dependency objects; no partial output, fabricated metadata or recovery RPC SHALL occur. Commands SHALL NOT expose reason, key, auth_version, effective permissions, login_required, profile or credential fields.

#### Scenario: Exact changed subset or no-op

- **WHEN** backend returns a valid changed subset or changed=false with no changed IDs and a precise version
- **THEN** exactly the four safe command fields return with numeric IDs, [] and false preserved and version as an exact string

#### Scenario: Invalid command success

- **WHEN** command success is nil, has a wrong target/version or returns duplicate, unsorted, unrequested, invalid IDs or an inconsistent changed flag
- **THEN** safe 500 returns without partial data, repairing the result or another RPC

#### Scenario: Invalid list success and isolated mapped values

- **WHEN** list metadata/items are malformed or a mapped collection/nullable pointer is later modified
- **THEN** malformed success fails entirely and valid mapped data remains independent of the original and other requests

### Requirement: Reuse secure transport and shared safe error mapping

Operations SHALL reuse verified TLS1.3-or-later transport, one environment-owned service credential, original bearer, safe locale/request metadata and the earlier parent/client deadline with cancellation. Caller claims SHALL NOT select service identity or backend authority. Outgoing metadata SHALL be copied; inherited version/idempotency-key SHALL be removed. Read SHALL send neither precondition; command SHALL send only its chosen key in metadata and explicit expected target version in payload. Nil context/request/response handling and resource cleanup SHALL remain safe. Existing explicit public status/code/registered locale/structured field/unknown-Reason fallback/Retry-After mapping and 504/499 context behavior SHALL remain unchanged without global mapper-registry expansion, raw internal messages, secret output, automatic retry or alternate identity.

#### Scenario: Copied request metadata and exact methods

- **WHEN** reads/commands carry safe parent metadata, inherited mutation preconditions and forged browser service claims
- **THEN** the selected UserService method receives copied safe metadata, original bearer and one configured credential, with command-only key/payload version and no parent mutation

#### Scenario: Encoded fields retry guidance and unknown Reason

- **WHEN** backend returns explicit public 401/403/404/409/429/503, localized structured fields, RetryInfo, unregistered user/role Reason or unknown internal failure
- **THEN** shared safe status/code/fields/Retry-After and localized fallback return without registry expansion, input echo, partial success or retry

#### Scenario: Deadline cancellation and cleanup

- **WHEN** a parent is canceled/expired or has an earlier deadline during any operation
- **THEN** no deadline is extended and established 499/504 handling and cleanup apply without a replacement request or leaked transport

### Requirement: Apply no-store browser isolation and preserve existing routes

All assignment paths SHALL emit Cache-Control no-store on success, early CORS/auth/input/dependency failures, malformed IDs and OPTIONS. Allowed GET/POST preflight SHALL follow existing exact-origin CORS with zero RPCs. Neither list, command, replay, no-op, failure nor target=self SHALL issue/delete cookies, return tokens, require new Origin/CSRF proof or use cookie authentication; supplied Origin SHALL retain existing validation. Only the three versioned routes SHALL be added; standard HEAD can follow the protected GET policy, without custom self/unversioned/PUT/PATCH/DELETE aliases. Public auth, legacy ListUsers role/cache policy, detail's short role view, role catalog and all existing profile/password/session/status/creation contracts SHALL remain unchanged. Concurrent operations SHALL isolate actor, target, query/cursor, IDs/reason/key/version, nullable results, locale, errors and browser effects on the shared connection.

#### Scenario: Early failures standalone handlers and preflight

- **WHEN** a malformed target, missing bearer, denied Origin, failed Me, invalid input, standalone handler result or allowed preflight reaches an assignment path
- **THEN** no-store and no Set-Cookie apply without unauthorized business dispatch, and allowed preflight invokes zero RPCs

#### Scenario: Route coexistence and permission separation

- **WHEN** assignment routes coexist with public auth, ListUsers, detail, role options/catalog and other user operations, or unsupported aliases are requested
- **THEN** each registered request keeps its existing route/permission/call policy, public auth remains zero-Me and no assignment aliases or implicit permission grants appear

#### Scenario: Mixed concurrent requests

- **WHEN** controlled concurrent assignment reads/grants/revokes and existing requests use different actors, targets, preconditions, filters, results and contexts
- **THEN** each receives only its own metadata/payload/outcome with one Me, correct business method ordering, independent collections and no cookie effects or data race
