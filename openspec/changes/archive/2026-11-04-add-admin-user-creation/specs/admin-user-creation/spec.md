# Spec Delta

## Purpose

Allow authorized operators to create PENDING user accounts through REST with required idempotency, validated credentials and normalized profile fields, returning safe backend account snapshots while preserving separate public registration, activation and session flows.

## ADDED Requirements

### Requirement: Authorize creation through one current identity and exact grant

The service SHALL expose `POST /api/v1/users` for a valid bearer actor without password-change restriction and with exact `iam.user.create=true`. Eligible requests SHALL perform one Me followed by one CreateUser. Role admin alone, wildcard/prefix/read/update/status/session grants, browser/body claims or service identity SHALL NOT replace the grant. Invalid identity SHALL receive 401; restricted or insufficiently permitted actors SHALL receive 403 without creation. No Register, target read, authority lookup, grant, activation or session RPC SHALL be added.

#### Scenario: Non-admin permission-only creator

- **WHEN** an unrestricted non-admin with the exact create grant submits valid input
- **THEN** one Me precedes one CreateUser without a role-admin or read-permission gate or additional business RPC

#### Scenario: Missing or insufficient authority

- **WHEN** otherwise valid input comes from a restricted actor or one with role only, false/missing exact grant, wildcard/prefix grant, or only read/update/status/session permissions
- **THEN** 403 is returned after Me with zero CreateUser calls

#### Scenario: Authentication failure

- **WHEN** bearer is missing, Me fails or cannot establish a valid principal
- **THEN** missing bearer performs zero RPCs and failed/invalid authentication invokes no creation or fallback identity, cookie credential or retry

### Requirement: Accept only explicit creation input from its designated source

Body SHALL be a JSON object with required non-null string email, password, name, phone and country_code; address SHALL be optional. Missing/null required values SHALL fail local required validation; absent/null address SHALL be omitted and an explicit empty address SHALL undergo validation. Empty/malformed body, top-level null/array/scalar or wrong known-field type SHALL fail 400 without creation. Unknown fields, confirm_password, actor/target/profile IDs, role/application/permission claims, must_change_password, field masks and body/query technical key/version inputs SHALL be ignored without selecting identity or authority. Creation SHALL have no local no-op or field defaults derived from Me.

#### Scenario: Missing required fields and no create no-op

- **WHEN** the body is {}, contains only unknown fields, or omits/nulls required creation values
- **THEN** safe field errors are returned without CreateUser or successful changed:false output

#### Scenario: Optional address versus empty or wrong-typed input

- **WHEN** address is absent/null, explicitly empty, or a known field has an incompatible JSON type
- **THEN** absent/null address is unselected, empty address fails its minimum and incompatible types fail parsing, without destructive or coerced defaults

#### Scenario: Caller claims cannot select privilege or technical state

- **WHEN** valid creation fields accompany forged IDs/roles/permissions/app access/version/key/field mask or confirmation fields in body/query/headers
- **THEN** only allowlisted body values and the designated key affect CreateUser, while actor remains the verified caller and backend assigns the new account ID/membership

### Requirement: Preserve register credential validation and raw forwarding

Email SHALL reject original non-ASCII input, then validate a separate trimmed/lowercased view for required, 6–50-byte length and the pattern below. The original parsed email SHALL be forwarded unchanged for backend canonicalization. Password SHALL be required 6–50 bytes, every byte 0x21–0x7e, without composition requirements, trimming, case conversion or other mutation. These new-credential rules SHALL NOT change public login/current-password acceptance. All applicable credential errors SHALL aggregate with other independent body errors, retain existing localized error identities and never echo credentials.

```text
^[A-Za-z0-9](?:[A-Za-z0-9._~\-]*[A-Za-z0-9])?@[A-Za-z0-9](?:[A-Za-z0-9._~\-]*[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9._~\-]*[A-Za-z0-9])?)+$
```

#### Scenario: Register-compatible email view

- **WHEN** email is ` User.Name@Example.COM `, uses allowed internal punctuation, or reaches validation-view lengths 5/6/50/51 bytes
- **THEN** the valid view passes at 6/50 and its original email remains unchanged on the outbound request, while invalid lengths/patterns fail
- **AND** plus-addressing, missing domain dot, invalid punctuation and original non-ASCII including edge whitespace fail rather than being silently normalized away

#### Scenario: New password boundaries and raw characters

- **WHEN** password is a simple printable value of 6/50 bytes or contains spaces/controls/Unicode or has 5/51 bytes
- **THEN** valid lowercase-only/digit-only/symbol-only values pass unchanged, while invalid length/character rules produce their distinct field errors without creation

#### Scenario: Safe independent errors and auth regression

- **WHEN** email/password and selected profile/status fields fail together in English, Indonesian or Chinese requests, or public login/password-change requests execute afterward
- **THEN** creation accumulates safe localized field lists without secret values or duplicated rules, and existing register/login/current/new-password policies and raw forwarding remain unchanged

### Requirement: Normalize required profile values once and preserve optional address

Name SHALL be required ASCII letters/spaces/apostrophes/hyphens, checked before trimming/collapsing ordinary spaces and capitalizing word starts after spaces/apostrophes/hyphens without lowercasing remaining letters; normalized length SHALL be 3–100 bytes. Required country_code SHALL add plus to digits and match plus followed by 1–3 ASCII digits with nonzero first digit, without whitespace trim/inference. Required phone SHALL be 12–20 ASCII digits after removing one +62/62 prefix then at most one zero only when this request supplies valid +62 country. Optional selected address SHALL be unchanged printable ASCII 0x20–0x7e plus CR/LF, 3–200 bytes. Validation SHALL accumulate errors across fields and copy input values; outbound mapping SHALL NOT normalize phone a second time.

#### Scenario: Normalized name and phone

- **WHEN** name is `  NASA  o'neil-jean  `, country_code is 62/+62 and phone is +62812345678901, 62812345678901, 0812345678901 or 812345678901
- **THEN** the outbound values are `NASA O'Neil-Jean`, +62 and 812345678901 without changing caller data or inferring another source

#### Scenario: Profile bounds and forbidden bytes

- **WHEN** normalized names reach 2/3/100/101 bytes, phones reach 11/12/20/21 digits, selected addresses reach 2/3/200/201 bytes, or name/phone/address/country contain disallowed bytes/formats
- **THEN** only in-range permitted values pass; raw invalid bytes are not hidden by normalization and all invalid fields are reported without creation

#### Scenario: Address preservation and remaining prefixes

- **WHEN** address is omitted/null or contains allowed surrounding spaces/CR/LF, or phone normalization leaves a legitimate leading 62/0 sequence
- **THEN** no address is invented, selected address bytes remain exact, and remaining phone bytes are preserved through transport mapping rather than stripped again

### Requirement: Constrain initial account status to PENDING

Omitted/null status SHALL select PENDING. Explicit status SHALL be exactly PENDING without trimming, case folding or label translation; other/empty strings SHALL fail safe local validation and wrong types SHALL fail parsing. Every eligible CreateUser payload SHALL carry explicit PENDING presence. Caller role/app/activation/session claims SHALL NOT cause any other creation state or secondary operation. Backend SHALL own default member assignment and account restriction values.

#### Scenario: Omitted null or explicit PENDING

- **WHEN** valid input omits status, supplies null or exactly PENDING
- **THEN** exactly one CreateUser carries a present PENDING value, with no activation, role grant or login call

#### Scenario: Contradictory initial state

- **WHEN** status is ACTIVE/SUSPENDED/DEACTIVATED, empty, whitespace-bearing, lowercased, unknown or wrong-typed
- **THEN** safe 400 is returned without silently defaulting contradictory input or creating an account

#### Scenario: Backend membership and restriction

- **WHEN** the backend creates the account with its default member assignment and must_change_password value
- **THEN** the safe returned values remain authoritative and do not change caller permissions or produce REST membership/credential/session side effects

### Requirement: Require one opaque key without a target Version precondition

Exactly one case-insensitive parsed Idempotency-Key SHALL be required, nonblank printable ASCII 0x20–0x7e of 1–200 bytes. Commas/internal spaces SHALL remain opaque without application trim/generation/hash/UUID restriction. Missing/invalid/repeated values including identical case-varied duplicates SHALL fail 400 without fallback to body/query/parent key. Create SHALL NOT require a path target or Version, validate unused Version headers or infer actor version. Outgoing command metadata SHALL remove inherited version/key from a copy and contain exactly the selected key; no expected-version or caller target ID SHALL be added to creation payload.

#### Scenario: Valid opaque key and unused Version

- **WHEN** one valid key is supplied with missing, malformed or repeated Version values
- **THEN** creation remains eligible, the selected key is preserved once and no Version metadata or expected-version payload enters CreateUser

#### Scenario: Missing duplicate or invalid key

- **WHEN** key is missing, blank, empty, control/non-ASCII, longer than 200 bytes or repeated with any capitalization
- **THEN** safe 400 is returned with no CreateUser, key echo, generation or alternate-source repair

#### Scenario: Parent and payload isolation

- **WHEN** inherited metadata contains old mutation values or a dependency changes its payload pointers/outgoing metadata
- **THEN** only the selected key and present creation status/country values affect this call, and caller-owned credentials/profile/status/key/metadata remain unchanged

### Requirement: Leave creation transactions and replay under current backend authority

Backend SHALL own canonical email, duplicate-account detection, account ID/version, password hash/password-sensitive fingerprint, default member assignment, transaction/locks, audit/outbox and idempotency records. Each caller retry SHALL run fresh Me then one eligible CreateUser, with current backend authority remaining applicable before committed replay. REST SHALL NOT hash/store credentials, persist replay/account state, create grants, pre-read an account, invoke Register, automatically retry/refetch, replace key or convert a conflict into success. A replayed creation snapshot SHALL NOT claim current target state after later independent changes.

#### Scenario: Fresh creation

- **WHEN** backend accepts a valid PENDING creation request
- **THEN** REST returns the command's safe account snapshot without a target-confirmation, member-grant or activation call

#### Scenario: Saved replay or revoked authority

- **WHEN** a caller repeats the same selected key/request and backend returns a saved outcome or denies current authority after Me
- **THEN** that outcome follows a fresh Me and one CreateUser without using a local cached success or claiming a fresh target read

#### Scenario: Email idempotency or resource conflict

- **WHEN** backend returns duplicate-email, IC or RB conflict, including a changed password/profile payload under the same key
- **THEN** the safe encoded error is returned without credential disclosure, lookup, key replacement, grant or automatic resubmission

### Requirement: Return an exact safe creation snapshot

Success SHALL be HTTP 200/code SS/message `User created successfully.` with data containing exactly user. User SHALL contain exactly id/email/name/address/phone/country_code/status/version/created_at/must_change_password/roles/application_access. IDs SHALL be positive numeric PostgreSQL-range values, version a positive int64 decimal string and created_at valid UTC RFC3339 preserving fractional precision. Roles SHALL contain role_id/code/name/status/role_status and application_access application_id/code/name/status/application_status; order/text SHALL be preserved, empty arrays serialized as [], booleans including false explicit. Returned email SHALL equal the submitted email's canonical trim/lowercase view and creation snapshot status SHALL be PENDING. Nil/malformed/mismatched success SHALL fail safe 500 without partial data or fallback. Generated ID SHALL NOT be compared with actor/forged caller IDs, and version SHALL NOT be fabricated or required to equal one.

No changed, registration/session/login_required fields, profile ID, duplicated top-level version, password/dob/hash/auth_version, credentials or effective permissions SHALL be added. Safe response text SHALL not undergo write normalization or phone-prefix concatenation. Returned user/collections SHALL remain independent of dependency objects and concurrent requests.

#### Scenario: Safe precise backend account

- **WHEN** CreateUser returns a valid PENDING user with canonical email, generated ID, precise version above 2^53, valid timestamp and membership
- **THEN** only the documented snapshot is returned under data.user with exact values and precision, without actor-ID substitution, version arithmetic or another RPC

#### Scenario: Empty assignments and private generated values

- **WHEN** safe assignments are absent/empty, a restriction flag is false or generated data carries private credential/dob sentinels
- **THEN** collections are [], false remains explicit and no private/generated field is exposed or synthesized

#### Scenario: Malformed identity or reused result

- **WHEN** success lacks response/user, has invalid IDs/version/timestamp/assignment entries, mismatched canonical email, non-PENDING status or reused mutable objects
- **THEN** malformed outcomes yield safe 500 with empty error data, and valid mapped snapshots cannot mutate dependency or another request's data

### Requirement: Preserve secure transport no-store and safe errors without browser effects

Collection POST and its OPTIONS preflight requesting POST SHALL emit Cache-Control no-store on success, replay and early CORS/auth/validation/dependency failures. Allowed existing-origin preflight SHALL finish with zero RPCs. Calls SHALL reuse verified TLS1.3, one environment-owned credential, original bearer, safe locale/request metadata and bounded parent deadline/cancellation. Password SHALL travel only in the secured request payload, never response/log/metadata/replay storage in REST. This bearer operation SHALL have no cookie fallback, CSRF/cookie proof or new Origin requirement and SHALL never set/delete cookies or issue tokens, even on replay/failure.

Existing error status/public code/localized safe fields/unknown-Reason fallback/Retry-After behavior SHALL remain unchanged, including 504/499 context mapping, no raw internal-message exposure, no global user-registry expansion and no automatic retry.

#### Scenario: Early failure and preflight

- **WHEN** CORS/authentication/header/body validation fails or an allowed preflight requests POST users collection
- **THEN** no-store and no Set-Cookie apply without unauthorized creation, and allowed preflight invokes zero RPCs

#### Scenario: Secure exact method and bounded context

- **WHEN** a selected request has browser service claims, parent metadata or a deadline/cancellation
- **THEN** exact CreateUser receives the selected payload/key with original bearer and one configured service credential on the shared secure connection, without extending context, credential metadata leakage or retry

#### Scenario: Structured failures and unknown internals

- **WHEN** Me/CreateUser returns encoded 401/403/404/409/429/503, safe public fields/retry information, unknown/raw internals, timeout or cancellation
- **THEN** the existing safe status/code/field/fallback/Retry-After and 504/499 policy applies without partial account output, secret input or cookie effects

### Requirement: Preserve existing routes and independent follow-up permissions

Only versioned POST users collection SHALL be added. Public Register/Login SHALL retain existing URLs, input/credential rules, zero-Me behavior, status/envelopes and browser effects. GET ListUsers SHALL retain its role gate, validation, envelope/cache behavior including GET preflight; all self/target profile, detail/status, password and session contracts SHALL remain unchanged. No unversioned or admin-create alias SHALL be registered. Create grant SHALL NOT imply read/update/status/session grants. Follow-up detail/profile/activation SHALL be caller-controlled separately authorized requests; creation SHALL not activate or promise current state/version for a later mutation.

#### Scenario: Separate creation and follow-up operations

- **WHEN** a creator later reads/edits/activates the returned user ID or lacks the corresponding follow-up grant
- **THEN** each endpoint performs its own existing authentication/permission flow or denies its missing grant, without hidden operations inside CreateUser

#### Scenario: Collection and public route coexistence

- **WHEN** public users/create or users/login, collection POST, collection GET/preflight and target-user/session operations coexist
- **THEN** each keeps its own method/policy/envelope/call count/cache/cookie behavior without duplicate Me, role-gate bypass on GET or making public registration protected

#### Scenario: Concurrent creation and existing calls

- **WHEN** concurrent creations/replays/conflicts and public/list/profile/detail/status/session requests use distinct actors, credentials, keys, normalized profiles, metadata and outcomes
- **THEN** values, returned IDs/versions/membership, field errors and cookies remain isolated on the shared connection with no local credential/replay state or cross-request authority
