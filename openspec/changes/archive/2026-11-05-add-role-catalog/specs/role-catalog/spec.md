# Spec Delta

## Purpose

Provide authorized operators with paginated role definitions, safe role and permission metadata, and FE selection options while preserving backend authority, caller identity, precision and existing REST contracts.

## ADDED Requirements

### Requirement: Authorize each role catalog read with one current identity and exact grant

The service SHALL expose GET `/api/v1/roles`, `/api/v1/roles/{role_id}` and `/api/v1/roles/options` to an unrestricted bearer actor with exact `iam.role.read=true`. Each eligible request SHALL run one Me followed by exactly one ListRoles for list/options or one GetRole for detail. Missing/invalid identity SHALL receive 401; restricted or insufficiently permitted actors SHALL receive 403 before business validation or dispatch. Syntax parsing MAY reject malformed input earlier. Role admin alone, wildcard/prefix grants, user-read or role-write grants, cookies and caller-supplied identity/permission/service claims SHALL NOT substitute for the grant. No additional authority, application, permission, assignment or mutation RPC SHALL be invoked.

#### Scenario: Non-admin role reader

- **WHEN** an unrestricted non-admin with the exact read grant requests list, detail or options
- **THEN** one Me precedes the operation's single business RPC without a role-admin gate or duplicate authentication

#### Scenario: Missing false or unrelated grant

- **WHEN** a restricted actor or an actor with role-only, missing/false exact grant, wildcard/prefix, user-read or role-write permission requests the catalog
- **THEN** 403 is returned after Me with no ListRoles/GetRole, including when business input is invalid

#### Scenario: Authentication failure and forged sources

- **WHEN** bearer is absent, Me fails or yields invalid identity, or HTTP/body/query/cookie claims try to select another identity or privilege
- **THEN** no role business RPC or alternate identity fallback occurs, and missing bearer invokes zero RPCs

### Requirement: Parse bounded pagination without accepting client-owned controls

List/options SHALL accept only the supported semantic controls page, limit, last_id, keyword, application_code and statuses. Each supported key SHALL have at most one decoded query value; identical duplicates SHALL fail. Omitted page/limit SHALL default to 1/20. Present page SHALL contain ASCII decimal digits representing 1–2147483647 and limit 1–100; signs, whitespace, empty values, Unicode digits, fractions and overflow SHALL fail 400. Leading zeros SHALL be accepted and mapped to the numeric value. Page SHALL NOT be capped to total_pages and limit SHALL NOT be capped to total. Unknown query keys SHALL be ignored without granting authority. Present fields, sort, search, search_fields, search_mode, created_from, created_to, updated_from, updated_to and code SHALL fail as unsupported controls even if empty; dates and additional exact-code filters remain deferred.

#### Scenario: Defaults boundaries and leading zeros

- **WHEN** page/limit are omitted or contain valid values at their boundaries, including leading zeros
- **THEN** one catalog page uses defaults 1/20 or the selected numeric values without a local upper bound derived from total

#### Scenario: Invalid or repeated controls

- **WHEN** a supported key is repeated or page/limit is empty, signed, whitespace-bearing, nondecimal or outside its range
- **THEN** safe localized field errors are returned with no role business RPC

#### Scenario: Server-owned and deferred controls

- **WHEN** fields/sort/search controls, role dates or exact code filtering are supplied alongside otherwise valid input
- **THEN** 400 is returned rather than forwarding caller-controlled projection/search/sort or silently ignoring a known unsupported filter

### Requirement: Normalize independent filters and keyword safely

Omitted application_code SHALL leave application scope to the backend; an explicit value SHALL be nonempty lowercase ASCII matching `[a-z][a-z0-9_]*` and at most 50 bytes, without trimming or lowercasing invalid input. Omitted statuses SHALL select no status restriction, including on options. Present statuses SHALL be nonempty CSV of exact ACTIVE/DISABLED tokens, with ordinary ASCII spaces around tokens removed, duplicates deduplicated and values sorted canonically; empty tokens or any other value SHALL fail. Categories SHALL combine AND and selected statuses OR. Keyword SHALL accept at most 200 raw and normalized bytes, only printable ASCII plus ASCII whitespace bytes 0x09–0x0d. It SHALL reject NUL/DEL/other controls/non-ASCII before normalizing whitespace to single spaces. Omitted/empty/whitespace-only keyword SHALL select no search; nonempty keyword SHALL use the server's logical keyword search and FULL_TEXT_TRIGRAM. Independent local field errors SHALL accumulate and localize EN/ID/ZH without echoing input.

#### Scenario: Scoped active and disabled reads

- **WHEN** application_code is clm and statuses is ` DISABLED,ACTIVE,ACTIVE `
- **THEN** scope clm and canonical ACTIVE/DISABLED OR filtering are used, while an omitted statuses filter retains both states rather than inventing ACTIVE-only options

#### Scenario: Logical keyword normalization and bounds

- **WHEN** keyword contains permitted whitespace, is whitespace-only, reaches 200/201 bytes or contains forbidden bytes
- **THEN** valid search text is normalized for logical keyword search, whitespace-only text disables search and invalid text fails before catalog dispatch

#### Scenario: Independent filter errors

- **WHEN** page, limit, application_code, statuses and keyword fail independently in the same request
- **THEN** all applicable field errors are returned safely in the selected locale without business RPC or normalization that hides invalid bytes

### Requirement: Preserve deterministic ordering and opaque query-bound cursors

List/options SHALL request explicit created_at DESC, updated_at DESC, id DESC ordering with a unique ID tie-breaker. They SHALL read the full safe role metadata without a caller-selected projection. last_id SHALL remain an opaque once-URL-decoded string up to 2048 bytes, without trim, base64 decoding, ID parsing, synthesis or cursor arithmetic in REST; an empty/omitted cursor means no cursor. Cursor validity and query binding SHALL remain backend-owned. A changed filter/keyword/sort SHALL require the caller to reset its cursor; backend cursor errors SHALL retain shared safe mapping without retry, refetch or silently dropping the cursor. List/options with identical supported controls SHALL use the same underlying query and may reuse its cursor.

#### Scenario: Explicit stable ordering

- **WHEN** list or options is requested with no caller sort
- **THEN** the single ListRoles call selects created_at DESC, updated_at DESC, id DESC and the same full role view for both representations

#### Scenario: Opaque cursor transport

- **WHEN** a caller supplies an encoded cursor, an empty cursor, or a value exceeding 2048 decoded bytes
- **THEN** an in-range decoded value is preserved exactly, empty means no cursor and overlong input fails locally without cursor rewriting or decoding

#### Scenario: Query-bound cursor rejection

- **WHEN** the backend rejects a malformed cursor or a cursor used with changed filters/keyword
- **THEN** its safe error is returned after one ListRoles with no retry, pagination loop or cursor repair

### Requirement: Resolve detail identity exclusively from a numeric role path

Detail SHALL take role_id only from its path as ASCII decimal digits in range 1–2147483647, accepting leading zeros and forwarding the parsed numeric ID. Zero, signs, whitespace, fractions, Unicode digits, overflow and nonnumeric labels/codes SHALL fail 400 without GetRole. Body/query identity, Me actor ID, numeric role_code and string code SHALL NOT replace target ID. `/roles/options` SHALL resolve to the options operation rather than detail parsing. A valid GetRole response SHALL match the requested ID; REST SHALL NOT make ListRoles/permission/assignment lookups to construct detail or disclose existence before authorization.

#### Scenario: Independent actor target and role code

- **WHEN** actor ID, path role ID, numeric role_code and string code are different values
- **THEN** only the parsed path ID selects GetRole, with no actor-ID substitution or role-code interpretation

#### Scenario: Invalid detail target

- **WHEN** role_id is zero, malformed, non-ASCII or outside the PostgreSQL integer range
- **THEN** 400 is returned after applicable authentication without GetRole or list fallback

#### Scenario: Literal options route

- **WHEN** GET `/api/v1/roles/options` is requested
- **THEN** the request performs Me then ListRoles and does not attempt to parse options as a role ID

### Requirement: Return safe role list metadata without narrowing counts or losing nulls

List success SHALL be HTTP 200/code SS/message `Roles retrieved successfully.` with data containing exactly items, total, page, limit, total_pages, has_next, has_prev and next_cursor. Each item SHALL contain exactly id, code, name, description, application_id, application_code, status, is_system, version, role_code, created_at, updated_at and affected_user_count. IDs SHALL be positive numeric PostgreSQL-range values; role_code SHALL remain a positive int32 number, distinct from ID/code. Version SHALL be a positive int64 decimal string with no float conversion. Description and created_at/updated_at SHALL be explicit null when absent; present valid timestamps SHALL be UTC RFC3339 preserving fractional precision. Text/order, false flags and zero affected_user_count SHALL be preserved without applying new-input validation or re-normalization. affected_user_count SHALL represent the backend's distinct users with ACTIVE assignments, including users/roles/applications/access in other states; it SHALL NOT claim effective or online users. The exact nonnegative int64 total and all pagination/cursor metadata SHALL be retained without int32 narrowing, recalculation or deriving totals from this page's items. Empty arrays SHALL serialize as [].

#### Scenario: Safe precise role metadata

- **WHEN** the backend returns ACTIVE or DISABLED roles, system flags, legacy text, nullable metadata and a version above 2^53
- **THEN** the documented 13 fields retain exact values, precision, nulls and false/zero values without privilege inference or extra count RPC

#### Scenario: Large total and out-of-range page

- **WHEN** backend total exceeds 2147483647 or an otherwise valid page has no items while total remains nonzero
- **THEN** the int64 total, page metadata and cursor remain unchanged with items=[], without overflow or setting total to zero

#### Scenario: Actually empty catalog

- **WHEN** backend reports no matching records with zero total/total_pages and no items
- **THEN** the same complete pagination envelope is returned with items=[], not null, and no fabricated records

### Requirement: Return role detail and permission mappings as information only

Detail success SHALL be HTTP 200/code SS/message `Role retrieved successfully.` with data containing exactly role and permissions. Role SHALL use the same 13-field safe role view as list. Each permission SHALL contain exactly id, application_id, application_code, code and description; permission IDs/application IDs SHALL be positive PostgreSQL-range numeric values and each permission SHALL belong to the returned role's application ID/code. Permissions SHALL be [] when empty, otherwise preserve backend order/text without adding permission status, inheritance, assignment identities, effective permissions or user data. System/IAM roles and DISABLED roles SHALL remain readable for an eligible actor. Returned permission mappings and affected counts SHALL NOT change the caller's permissions or promise grant eligibility.

#### Scenario: Read system or disabled role

- **WHEN** an eligible actor requests a system/IAM or DISABLED role
- **THEN** its safe role and permission metadata is returned without applying role-mutation restrictions to the read

#### Scenario: Empty or ordered permission mappings

- **WHEN** GetRole returns no permissions or a valid ordered list of same-application permissions
- **THEN** permissions is [] or the exact safe ordered mapping, without a separate ListPermissions call

#### Scenario: Informational mapping boundary

- **WHEN** detail reports permissions and affected users the caller does not possess
- **THEN** those values are returned only as role metadata and do not grant permissions, expose user records or trigger assignment/session changes

### Requirement: Adapt role options from one catalog page without conflating identities

Options success SHALL be HTTP 200/code SS/message `Role options retrieved successfully.` with the same eight pagination fields as list. Each item SHALL contain exactly value, label, code and role_code. value SHALL be the role ID as a canonical decimal string; role_code SHALL be the separate positive numeric role code as a decimal string; code SHALL retain the immutable string role code. Label SHALL be `code - description` when description is non-null/nonempty, otherwise `code - name`; selected response text SHALL remain unchanged, including whitespace, punctuation or line breaks. Label SHALL NOT select identity/authority or become a backend input. Options SHALL preserve page order, total/cursor/flags, filtering and errors from one ListRoles, with no GetRole per item, backend label RPC, multi-page aggregation or ACTIVE-only default.

#### Scenario: Description and name fallback labels

- **WHEN** description is present, absent or an empty string
- **THEN** label uses the exact description or name fallback while value/code/role_code remain separate fields and strings

#### Scenario: Paginated options and cursor reuse

- **WHEN** a filtered catalog page contains a next cursor or no items with a nonzero total
- **THEN** options changes only item representation and preserves all pagination values, using one ListRoles without fetching more pages

#### Scenario: Distinct identity values

- **WHEN** role ID is 10, numeric role_code is 3 and code is clm.legal_reviewer
- **THEN** options returns value="10", role_code="3" and code="clm.legal_reviewer", allowing the existing ListUsers numeric role filter to remain unchanged

### Requirement: Reject malformed backend success without partial data

Nil response, nil required page metadata, nil list items/role/permission entries, invalid IDs/role_code/version/status, invalid present timestamps, negative total/total_pages, page below 1, limit outside 1–100, wrong detail ID or cross-application permission entries SHALL result in safe HTTP 500 with empty error data and no partial role/options list. Absent nullable description/timestamps SHALL remain valid. A valid mapped result, nullable pointers and collections SHALL be independent of dependency/caller objects and concurrent requests. REST SHALL NOT fabricate missing metadata, downgrade detail errors to list output or retry malformed success.

#### Scenario: Invalid list or options success

- **WHEN** list metadata or any role item is missing/malformed, including an invalid ID/version/status or timestamp
- **THEN** both representations fail safely without returning preceding valid items or default pagination

#### Scenario: Invalid detail success

- **WHEN** detail lacks a role, returns another ID, contains a nil/invalid permission or mismatched application mapping
- **THEN** safe 500 is returned without partial role/permission output or another RPC

#### Scenario: Mutable dependency result isolation

- **WHEN** a caller changes mapped collections or nullable values, or simultaneous calls reuse dependencies
- **THEN** dependency data and another request's result remain independent without cross-request state

### Requirement: Preserve secure request transport and shared error mapping

All role reads SHALL reuse the existing verified TLS1.3-or-later auth connection, one environment-owned service credential and original bearer with safe locale/request ID metadata, parent cancellation and the earlier of parent/client deadlines. Caller claims SHALL NOT select service identity, IAM namespace or method authority. No Version/Idempotency-Key precondition SHALL apply; unused malformed/repeated HTTP values SHALL be ignored and inherited mutation metadata SHALL be removed from a copy before role business dispatch. Nil context/request/response guards and cleanup SHALL remain safe. Service errors SHALL retain the shared explicit status/public code/registered locale/structured fields/unknown-Reason fallback/Retry-After contract, including 504/499 for context termination, without raw internal messages, secret output, global registry expansion, automatic retry or alternate identity.

#### Scenario: Exact method and metadata

- **WHEN** a role read carries bearer, locale/request metadata, old parent mutation metadata and unused Version/key headers
- **THEN** its selected RoleService method receives copied safe metadata and one configured credential, with no business version/key metadata and no parent mutation

#### Scenario: Current method IAM or permission denial

- **WHEN** the backend denies current authority, exact method or IAM scope after a successful Me
- **THEN** the safe denial is returned without fallback identity, additional lookup or repeated business RPC

#### Scenario: Structured failures and context termination

- **WHEN** Me or a role RPC returns encoded catalog/public field errors, retry guidance, unknown internals, timeout or cancellation
- **THEN** existing status/code/locale/Retry-After and 504/499 mapping remains intact without partial data or application retry

### Requirement: Emit no-store and preserve the bearer browser boundary

Role list/detail/options SHALL emit Cache-Control no-store on success, local input errors and early CORS/auth/dependency errors, including malformed targets and allowed GET preflight. Existing allowed-origin CORS SHALL answer preflight with zero RPCs. These bearer reads SHALL not require a new Origin or CSRF/cookie proof, use cookie authentication, issue/delete cookies or return access/refresh tokens. Supplied Origin SHALL remain subject to existing CORS validation. GET ListUsers and its preflight/cache policy SHALL remain unchanged.

#### Scenario: Early errors and allowed preflight

- **WHEN** missing bearer, denied Origin, invalid query/ID or allowed GET preflight reaches a catalog path
- **THEN** no-store and no Set-Cookie apply, while allowed preflight invokes zero RPCs

#### Scenario: Standalone handler and nullable success

- **WHEN** a standalone catalog handler or the production pipeline returns success/failure without BrowserPolicy
- **THEN** the same no-store and zero cookie/token effects apply without inferring login-required behavior

#### Scenario: Cookie-only and existing collection behavior

- **WHEN** only auth cookies are supplied to role reads or a legacy users GET/preflight is requested
- **THEN** cookies do not replace a role bearer, and existing users authentication/cache behavior is retained

### Requirement: Preserve existing routes and isolate concurrent catalog reads

Only the three versioned role GET operations SHALL be added, with options matched before detail. Framework-standard HEAD behavior MAY follow the same protected read policy; no custom mutation, unversioned or user-role assignment alias SHALL be added. Public auth, Me, self/target profile/password/session, user creation/status/detail and legacy ListUsers contracts SHALL remain unchanged. Concurrent list/detail/options and existing requests SHALL keep actor, filters/search/cursor, identities, nullable metadata, version/counts, labels, locale, errors and cookies isolated on the shared connection. Role definitions SHALL not imply user assignments, application access or effective permission grants.

#### Scenario: Route coexistence and regression

- **WHEN** options/detail/list coexist with current public/user/session routes
- **THEN** options is not captured as an ID, each protected request has one Me and public auth retains zero-Me behavior without changed routes/envelopes

#### Scenario: Mixed simultaneous requests

- **WHEN** controlled concurrent catalog reads and existing public/user/session operations use distinct credentials, query/cursor values, targets, nullability and outcomes
- **THEN** each receives only its own payload/metadata/result/errors/cookies with correct method order and no data race

#### Scenario: Catalog is not access provisioning

- **WHEN** an operator lists or inspects a role or selects an option
- **THEN** no role creation/update, permission replacement, user assignment, application grant or activation operation occurs
