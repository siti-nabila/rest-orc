# Spec Delta

## Purpose

Provide authorized callers with paginated application definitions and FE selection options while preserving backend authority, application-code identity, safe metadata, exact page totals and existing REST behavior.

## ADDED Requirements

### Requirement: Authorize application reads with one current identity and exact permission

The service SHALL expose GET `/api/v1/applications` and `/api/v1/applications/options` to an unrestricted bearer actor with exact `iam.permission.read=true`. Each eligible request SHALL perform one Me followed by exactly one ListApplications. Missing/invalid identity SHALL receive 401; restricted or insufficiently permitted actors SHALL receive 403 before business validation or dispatch. Transport syntax errors can fail earlier. Admin role, wildcard/prefix/false grants, user-read, role-read or application grant/revoke permission and caller-supplied authority SHALL NOT substitute for the exact grant. No additional identity, permission, role, assignment, detail or mutation lookup SHALL occur.

#### Scenario: Permission-only application reader

- **WHEN** an unrestricted non-admin has only the exact catalog permission and submits valid list/options input
- **THEN** one Me precedes one ListApplications with no role-admin gate, duplicate authentication or additional lookup

#### Scenario: Missing grant or restricted actor

- **WHEN** a restricted actor or actor with missing/false/wildcard/prefix/unrelated grants submits business-invalid controls
- **THEN** 403 is returned after Me and before ListApplications or business-validation disclosure

#### Scenario: Authentication cannot fall back to claims or cookies

- **WHEN** bearer is absent, Me fails or identity is invalid, including with forged actor/service/permission claims or auth cookies
- **THEN** no business RPC or alternate identity is used, and absent bearer invokes zero RPCs

### Requirement: Parse bounded catalog pagination and reject unsupported controls

List/options SHALL support page, limit, last_id, keyword and statuses, each with at most one decoded query value, rejecting identical duplicates. Omitted page/limit SHALL default to 1/20. Present page SHALL contain ASCII decimal digits representing 1–2147483647 and limit 1–100, accepting leading zeros without signs, whitespace, empty values, fractions, non-ASCII digits or overflow. Page SHALL NOT be capped by total_pages nor limit by total. Present fields, sort, search, search_fields, search_mode, application_code, code, created_from, created_to, updated_from and updated_to SHALL fail 400 even if empty. Other unknown query keys SHALL be ignored without granting authority; application scope/exact-code/date filters are not supported by this REST catalog.

#### Scenario: Defaults and full page range

- **WHEN** page/limit are omitted or contain valid boundary/leading-zero values
- **THEN** one catalog page uses defaults 1/20 or the selected numeric values without a total-derived cap

#### Scenario: Repeated or malformed supported input

- **WHEN** a supported key is repeated or page/limit is empty, zero, signed, whitespace-bearing, nondecimal or outside its range
- **THEN** safe localized field errors return without ListApplications, including identical duplicate values

#### Scenario: Unsupported controls and unknown claims

- **WHEN** a known unsupported control is supplied or unrelated user/namespace claims accompany otherwise valid controls
- **THEN** unsupported controls fail 400, while unrelated unknown claims are ignored and do not select projection, scope, sort or authority

### Requirement: Normalize application statuses and logical keyword safely

Omitted statuses SHALL leave status unrestricted, including for options, preserving ACTIVE and DISABLED applications. Present statuses SHALL be nonempty CSV of those exact codes; ordinary ASCII spaces around tokens SHALL be trimmed, duplicates deduplicated and values sorted canonically. Empty tokens, unknown/case-folded values or REVOKED SHALL fail. Selected statuses SHALL combine OR and status/keyword categories AND. Keyword SHALL allow at most 200 raw and normalized bytes of printable ASCII plus ASCII whitespace 0x09–0x0d, reject forbidden bytes before normalization, and collapse allowed whitespace to single spaces. Omitted/empty/whitespace-only keyword SHALL disable search; nonempty text SHALL use server-owned logical keyword FULL_TEXT_TRIGRAM search over application code/name. Independent field errors SHALL accumulate and localize EN/ID/ZH without echoing inputs.

#### Scenario: Definitions retain both application states

- **WHEN** statuses is omitted or equals ` DISABLED,ACTIVE,ACTIVE `
- **THEN** no status restriction or canonical ACTIVE/DISABLED filtering is sent, without an ACTIVE-only options default or assignment-state interpretation

#### Scenario: Keyword normalization and raw byte limits

- **WHEN** keyword contains allowed whitespace, is whitespace-only, reaches 200/201 raw bytes or contains NUL/DEL/other forbidden controls/non-ASCII
- **THEN** valid nonempty text is normalized for logical search, whitespace-only disables search and invalid raw text fails before dispatch without hidden character removal

#### Scenario: Independent invalid fields

- **WHEN** page, limit, statuses and keyword independently fail in one request
- **THEN** all applicable safe field errors return in the selected locale with no ListApplications

### Requirement: Use deterministic application ordering and opaque cursors

Both representations SHALL request explicit id DESC order and the full safe application projection, with no caller-selected fields or search/sort mode. last_id SHALL remain a once-URL-decoded opaque string up to 2048 bytes, without trim, base64 decoding, ID parsing, synthesis or cursor arithmetic. Empty/omitted cursor SHALL mean no cursor. Backend SHALL own cursor validity and query binding; callers SHALL reset the cursor when filters/keyword/sort change. List/options with identical supported controls SHALL use the same query and allow its cursor to be reused. Backend cursor failure SHALL retain safe mapping without silently dropping/repairing the cursor, retrying, refetching or aggregating pages.

#### Scenario: Explicit stable query

- **WHEN** an eligible list/options request reaches the backend
- **THEN** the single ListApplications requests full safe fields, explicit id DESC and the same normalized query for both representations

#### Scenario: Opaque cursor bytes and budget

- **WHEN** cursor includes encoded punctuation/spaces, is empty or exceeds 2048 decoded bytes
- **THEN** an in-range decoded value is preserved exactly, empty selects no cursor and overlong input fails locally

#### Scenario: Query-bound cursor rejection

- **WHEN** backend rejects a malformed cursor or a cursor reused with changed status/keyword controls
- **THEN** its safe failure returns after one business RPC without cursor replacement, fallback paging or retry

### Requirement: Return exact safe application definitions and page totals

List success SHALL be HTTP 200/code SS/message `Applications retrieved successfully.` with data containing exactly items, total, page, limit, total_pages, has_next, has_prev and next_cursor. Each item SHALL contain exactly id, code, name, status and is_system. ID SHALL be a positive numeric value in 1–2147483647; status SHALL be ACTIVE or DISABLED. Backend code/name text, item order and system boolean including false SHALL remain unchanged without write-time normalization or validation of stored text. System/IAM and DISABLED applications SHALL remain readable for eligible callers. Nonnegative int64 total and backend page/cursor/flags SHALL be retained exactly without int32 narrowing, float conversion, recomputation or deriving total from current items. Empty items SHALL be []. No version, description, timestamp, affected-user count, assignment/access state, permission mapping, user or credential fields SHALL be invented.

#### Scenario: Safe five-field definitions

- **WHEN** backend returns system/IAM or DISABLED applications with stored text and false flags
- **THEN** only the five documented fields return unchanged without hiding system/disabled records or applying new-input text rules

#### Scenario: Large total and empty out-of-range page

- **WHEN** backend total exceeds 2147483647 or a valid page has no items while total remains nonzero
- **THEN** the exact int64 total and page metadata remain unchanged with items=[], rather than overflow, zeroed totals or fabricated rows

#### Scenario: Empty application catalog

- **WHEN** backend reports zero total/total_pages and no matching applications
- **THEN** the complete eight-field page envelope returns with items=[] and preserved false flags/cursor

### Requirement: Build application-code options from one catalog page

Options success SHALL be HTTP 200/code SS/message `Application options retrieved successfully.` with the same eight pagination fields as list. Each item SHALL contain exactly value and label. Value SHALL be the unchanged application.code string, distinct from the numeric application ID; label SHALL be the exact backend code plus ` - ` plus the unchanged name. There is no description fallback or additional option ID/code/status/version field. Historical whitespace, punctuation, line breaks and text SHALL remain unchanged. Options SHALL preserve page order, total, cursor, flags, filters and errors from one ListApplications, without detail/label RPCs, cross-page aggregation, local deduplication or eligibility filtering. Labels SHALL NOT become input identity or confer access.

#### Scenario: Code identity differs from numeric ID

- **WHEN** an application has numeric ID 5, code clm and name Contract Lifecycle Management
- **THEN** the option is value="clm" and label="clm - Contract Lifecycle Management", with no numeric-ID substitution or extra item fields

#### Scenario: Stored text and exact labels

- **WHEN** application names/code contain historical whitespace, punctuation or an empty name
- **THEN** value preserves code and label uses the exact `code - name` concatenation without trim, title casing or role-description fallback

#### Scenario: Filtered paginated options and cursor reuse

- **WHEN** options is requested with supported controls for a populated or empty page, including a cursor previously returned by the equivalent list query
- **THEN** only item representation changes and all pagination/filter/error behavior comes from one ListApplications without fetching other pages

### Requirement: Reject malformed successes and isolate mapped collections

Nil response, nil required page metadata, nil application entries, invalid numeric IDs, unsupported status, negative total/total_pages, page below 1 or limit outside 1–100 SHALL fail safe HTTP 500 with empty error data and no partial list/options. REST SHALL NOT fabricate missing metadata, repair invalid success, skip individual bad items or retry. Mapped arrays and request filters SHALL be independent of dependency/caller collections and other requests. Stored code/name text SHALL not acquire write-time character rules through malformed-success checks.

#### Scenario: Invalid page metadata

- **WHEN** success lacks response/metadata or contains invalid page/limit/total/total_pages
- **THEN** both representations return safe 500 with no partial data, invented defaults or recovery RPC

#### Scenario: Invalid entry after a valid row

- **WHEN** a page contains a valid application followed by a nil entry, zero/out-of-range ID or unknown state
- **THEN** the whole response fails safely without returning preceding rows or omitting the invalid entry

#### Scenario: Request and response ownership

- **WHEN** a dependency mutates its received filter or a caller modifies mapped items while other requests run
- **THEN** caller input, dependency response collections and other request results remain independent

### Requirement: Preserve secure read transport and shared safe errors

Reads SHALL reuse verified TLS1.3-or-later transport, one environment-owned service credential, original bearer, safe locale/request ID metadata and the earlier of parent/client deadlines with cancellation. Backend current actor/method/IAM scope SHALL remain authoritative after Me; caller claims SHALL NOT select service identity or permissions. Request bodies, including malformed JSON, and unused/repeated/malformed HTTP Version/Idempotency-Key SHALL be ignored rather than parsed as read preconditions. Inherited version/idempotency-key SHALL be removed from copied outgoing business metadata without parent mutation. Nil context/request/response handling and resource cleanup SHALL stay safe. Existing explicit public status/code, registered locale, structured fields, unknown-Reason localized fallback, Retry-After and 504/499 context behavior SHALL remain unchanged without raw internals/secrets, global mapper-registry expansion, automatic retry or alternate identity.

#### Scenario: Exact ApplicationService dispatch and copied metadata

- **WHEN** a read carries safe parent metadata, old mutation preconditions, malformed body/HTTP preconditions and browser-supplied service claims
- **THEN** the exact ListApplications method receives copied safe metadata, original bearer and one configured credential, with no business mutation precondition and no parent mutation

#### Scenario: Current authority and encoded failures

- **WHEN** backend denies current permission/method/IAM scope or returns explicit public errors, structured fields, RetryInfo or an unregistered Reason after Me
- **THEN** shared safe status/code/locale/field/retry/fallback behavior is retained without extra lookups, partial data, raw details or retry

#### Scenario: Deadline cancellation and cleanup

- **WHEN** parent context is canceled/expired or has an earlier deadline during a read
- **THEN** no deadline is extended and established 499/504 mapping and cleanup apply without replacing the request or leaking a transport

### Requirement: Emit no-store and retain the bearer browser boundary

Both application read paths SHALL emit Cache-Control no-store on success, local input errors and early CORS/auth/dependency failures, including allowed GET preflight. Existing exact-origin CORS SHALL finish allowed preflight with zero RPCs. Neither endpoint SHALL require a new mandatory Origin/CSRF proof, use cookie authentication, issue/delete cookies, return access/refresh tokens or infer login-required behavior. Supplied Origin SHALL remain subject to existing validation. Standalone handlers SHALL have the same no-store and cookie policy. Existing GET ListUsers and its preflight/cache policy SHALL remain unchanged.

#### Scenario: Early failure and allowed preflight

- **WHEN** missing bearer, failed Me, denied Origin, invalid query or allowed GET OPTIONS reaches either application path
- **THEN** no-store and no Set-Cookie apply without unauthorized business dispatch, and allowed preflight invokes zero RPCs

#### Scenario: Standalone handler and optional Origin

- **WHEN** a standalone handler returns success/failure or an otherwise valid bearer request omits Origin and CSRF/cookie proof
- **THEN** the documented no-store/bearer contract applies without adding browser credential effects or new required proof

#### Scenario: Cookie-only request and legacy collection cache

- **WHEN** only auth cookies are supplied to an application read or legacy users GET/preflight is requested
- **THEN** cookies do not replace the application bearer and existing users authentication/cache behavior remains unchanged

### Requirement: Preserve existing routes and keep catalog reads informational

Only the two versioned application GET operations SHALL be added, with literal options routed explicitly. Framework-standard HEAD can follow the same protected GET policy; no application detail, mutation, user-access, self or unversioned alias SHALL be added. Public auth, Me, legacy users role/cache behavior, role catalog, user-role management and existing user/profile/password/status/creation/session contracts SHALL remain unchanged. Concurrent list/options and existing requests SHALL isolate actors, queries/cursors, IDs/codes/text/system flags, totals, locale, errors and browser effects on the shared connection. Reading/selecting application definitions SHALL NOT grant access, assign roles, activate anything or represent effective permissions.

#### Scenario: Route coexistence and absent aliases

- **WHEN** application list/options coexist with current routes or detail/mutation/unversioned aliases are attempted
- **THEN** only the two catalog operations are added, existing requests preserve their routing/contracts and public auth remains zero-Me

#### Scenario: Mixed simultaneous requests

- **WHEN** controlled concurrent application list/options and role/user/assignment/public operations carry distinct credentials, query/cursor values and results
- **THEN** each receives its own metadata/payload/output/errors/cookies with one Me where required, correct method order and no data race

#### Scenario: Selection does not provision access

- **WHEN** an eligible caller reads or selects an ACTIVE/DISABLED/system application option
- **THEN** no access grant/revoke, user-role assignment, application activation or permission-evaluation operation occurs, and command eligibility remains a separate backend decision
