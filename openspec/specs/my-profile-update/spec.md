# my-profile-update Specification

## Purpose

Memungkinkan pengguna terautentikasi memperbarui sebagian profilnya sendiri melalui PATCH /api/v1/me/profile, dengan normalisasi terkontrol, validasi lokal, ownership terverifikasi, dan response aman tanpa mengubah endpoint existing.

## Requirements

### Requirement: Authenticate and authorize the current profile owner

The service SHALL provide `PATCH /api/v1/me/profile` for the verified current user without an admin-role or additional permission requirement. It SHALL authenticate once through Me and reject a missing/invalid principal or a principal with `must_change_password=true` before returning a local no-op or dispatching a mutation. The backend SHALL remain authoritative for ownership and current session eligibility.

#### Scenario: Non-admin owner

- **WHEN** an authenticated nonrestricted non-admin submits a valid selected profile field
- **THEN** the request is eligible for its own profile update without an admin gate
- **AND** exactly one Me precedes exactly one UpdateProfile invocation

#### Scenario: Missing bearer or failed authentication

- **WHEN** bearer authorization is missing or Me fails
- **THEN** the existing safe authentication/error mapping is returned and UpdateProfile is not called
- **AND** missing authorization does not call Me

#### Scenario: Restricted user including no-op input

- **WHEN** Me identifies a restricted user and the PATCH contains selected fields or no effective fields
- **THEN** HTTP 403 is returned without an UpdateProfile call or successful no-op

#### Scenario: Browser ownership claims

- **WHEN** a request supplies user ID, profile ID, roles, permissions, namespace, or service credential claims outside the editable field allowlist
- **THEN** those claims cannot select a target or grant authority
- **AND** the mutation remains tied to the verified bearer without another user lookup

### Requirement: Preserve partial-update field presence

Only non-null `name`, `phone`, `country_code`, and `address` values SHALL select fields for update. Omitted/null fields SHALL be ignored; unknown fields and client-supplied field masks SHALL be ignored. A selected empty string SHALL undergo validation rather than clear a field or become omitted. Malformed JSON or incompatible types for known fields SHALL return HTTP 400 without a mutation.

#### Scenario: Mixed selected and ignored fields

- **WHEN** the body contains a valid name, null phone, omitted address, and unknown fields
- **THEN** only name is selected and the other profile values are not overwritten

#### Scenario: Empty selected values

- **WHEN** name, phone, country_code, or address is explicitly supplied as an empty string
- **THEN** that field fails its applicable local validation and is not treated as absent

#### Scenario: Browser field mask cannot select data

- **WHEN** the body includes a field_mask, user_id, or other unknown field without a non-null editable value
- **THEN** it does not select a mutation field and the authenticated no-op rule applies

#### Scenario: Invalid known-field type

- **WHEN** JSON is malformed or a known field contains a number, boolean, object, or array
- **THEN** the response is HTTP 400 with the existing bad-request envelope and no UpdateProfile call

### Requirement: Normalize and validate profile names

A selected name SHALL contain only ASCII letters, ordinary spaces, apostrophes, and hyphens. The service SHALL reject disallowed input before normalization can remove it, trim surrounding spaces, collapse repeated spaces, and uppercase the first letter of each word without lowercasing remaining letters. Spaces, apostrophes, and hyphens delimit word starts. The normalized name SHALL be 3–100 bytes long.

#### Scenario: Readable capitalization without destroying acronyms

- **WHEN** name is `  NASA  o'neil-jean  `
- **THEN** the forwarded name is `NASA O'Neil-Jean`

#### Scenario: Normalized length boundaries

- **WHEN** normalized names have lengths 2, 3, 100, and 101 bytes
- **THEN** 3 and 100 pass length validation while 2 and 101 fail

#### Scenario: Disallowed name characters

- **WHEN** name contains digits, tabs, CR/LF, control characters, non-ASCII letters, or Unicode whitespace
- **THEN** name is rejected without stripping those characters to make it valid

### Requirement: Normalize country codes and phone numbers deterministically

A selected country_code SHALL normalize digit-only input by adding `+` and otherwise SHALL match `^\+[1-9][0-9]{0,2}$` without whitespace trimming or country inference. A selected phone SHALL contain 12–20 ASCII digits after normalization. Only when this request explicitly supplies a valid country_code normalizing to `+62`, phone normalization SHALL remove one leading `+62` or `62`, then at most one leading `0`. Other prefixes SHALL NOT be removed.

#### Scenario: Indonesian prefix and leading zero

- **WHEN** country_code is `62` or `+62` and phone is `+62812345678901`, `62812345678901`, `0812345678901`, or `812345678901`
- **THEN** country_code becomes `+62` and phone becomes `812345678901`

#### Scenario: Length checked after normalization

- **WHEN** country_code is `+62` and phone is `081234567890`
- **THEN** phone normalizes to 11 digits and fails the minimum-length rule
- **AND** normalized phones of exactly 12 and 20 digits pass while 11 and 21 fail

#### Scenario: No inference from the existing profile

- **WHEN** phone alone is selected, or a country_code other than `+62` is supplied
- **THEN** no prefix or leading zero is removed by the Indonesian normalization rule
- **AND** a valid 12–20 digit phone passes without reading or guessing an existing country code

#### Scenario: Country-code boundaries and invalid format

- **WHEN** country_code is `1`, `62`, or `123`
- **THEN** it normalizes to a valid plus-prefixed code
- **AND** empty input, zero-leading codes, four digits, Unicode digits, spaces, or malformed plus signs fail

#### Scenario: Country-code-only update and invalid phone characters

- **WHEN** only country_code is selected
- **THEN** the stored phone is not synthesized, re-normalized, or selected
- **AND** a selected phone containing spaces, punctuation, Unicode digits, or a residual plus sign fails rather than being cleaned to digits

### Requirement: Validate addresses without rewriting them

A selected address SHALL contain only printable ASCII bytes `0x20–0x7E` and CR/LF and SHALL be 3–200 bytes long. The service SHALL preserve its exact content, including surrounding spaces and line breaks, without trimming or case conversion.

#### Scenario: Address content and length

- **WHEN** an allowed address contains surrounding spaces or CR/LF and has exactly 3 or 200 bytes
- **THEN** the exact parsed address is forwarded
- **AND** lengths below 3 or above 200 fail

#### Scenario: Invalid address characters

- **WHEN** address contains tabs, NUL, DEL, another forbidden control, or non-ASCII text
- **THEN** local address validation fails without an UpdateProfile call

### Requirement: Aggregate local errors before the mutation

The service SHALL collect applicable field errors across all selected fields and return HTTP 400 using the existing localized `code/errors/data` envelope with empty data. Unselected fields SHALL not receive required errors. Validation SHALL not expose submitted personal data or credentials, and normalizing a request SHALL not mutate another request's data.

#### Scenario: Several invalid fields

- **WHEN** an eligible authenticated request has invalid name, phone, country_code, and address
- **THEN** the response contains localized error lists for all invalid fields and no UpdateProfile invocation
- **AND** the authentication Me call remains accounted for

#### Scenario: Locale and request isolation

- **WHEN** invalid requests use supported English, Indonesian, or Chinese locales, or request values are reused across isolated calls
- **THEN** messages follow existing locale behavior without input values and normalization does not alter the original input or another request

### Requirement: Explicit mutation selection and authenticated local no-op

The service SHALL construct an explicit field mask from valid selected fields and send only their normalized values to UpdateProfile. An eligible request with no effective fields SHALL return HTTP 200, code SS, message `Profile updated successfully.`, and data exactly `{"changed":false}` without mutation, profile fallback, or fabricated version. This local no-op SHALL not claim that a backend mutation transaction, latest version check, or audit write occurred.

#### Scenario: Partial mask without destructive replacement

- **WHEN** an eligible request selects any nonempty subset of the four editable fields
- **THEN** exactly those fields appear in an explicit nonempty mask and unselected fields are not reset by a legacy full replacement
- **AND** caller-supplied target IDs, field masks, versions, or other data do not enter the mutation payload

#### Scenario: No effective field

- **WHEN** eligible JSON input is `{}`, null, only unknown fields, or only null editable fields
- **THEN** exactly one Me and zero UpdateProfile calls occur and data is exactly `{"changed":false}`

#### Scenario: Same value is still an effective selection

- **WHEN** a selected valid value appears equal to a value in the Me response
- **THEN** UpdateProfile is still invoked once and backend changed/profile/version values determine the result

#### Scenario: No unsupported technical preconditions

- **WHEN** an eligible request omits Version and Idempotency-Key headers
- **THEN** those omissions do not reject a valid update or no-op
- **AND** the service does not invent an expected version or promise optimistic concurrency unsupported by the self-profile RPC

### Requirement: Map authoritative safe mutation results

A successful mutation response SHALL use HTTP 200, code SS, and message `Profile updated successfully.`. Its typed data SHALL contain backend changed, exact decimal-string version, and a profile object with numeric id/user_id and string name/address/phone/country_code. A nil response, missing profile, zero profile/owner IDs, zero version, or a returned owner different from the verified principal SHALL fail safely as HTTP 500 rather than expose a default or another user's profile.

#### Scenario: Successful update and precise version

- **WHEN** the backend returns a valid owned profile and a version larger than JavaScript's safe integer range
- **THEN** safe profile values and changed are retained and version is serialized exactly as a decimal string without floating-point conversion
- **AND** credentials, raw generated fields, inferred privilege fields, or concatenated phone prefixes are absent

#### Scenario: Authoritative backend no-op

- **WHEN** a nonempty mask reaches the backend and it returns changed=false with a valid profile and version
- **THEN** those authoritative profile/version fields are included instead of being replaced by the shorter local no-op response

#### Scenario: Invalid or mismatched successful response

- **WHEN** a successful dependency result is nil, incomplete, zero-versioned, or belongs to another user
- **THEN** a safe internal-error envelope is returned without profile fallback or a second RPC

### Requirement: Preserve secure transport and existing REST contracts

UpdateProfile SHALL reuse the configured authenticated connection and preserve user bearer, one environment-owned service credential, verified TLS, locale, existing metadata, earlier parent deadline, and cancellation. It SHALL follow the existing http-error-mapping contract without application retries, credential leakage, new identity claims, or per-request connections. Existing public auth, GET Me, and admin ListUsers behavior SHALL remain unchanged.

#### Scenario: Secure ordered calls over one connection

- **WHEN** a valid update traverses the configured transport
- **THEN** one Me precedes one UpdateProfile on the reused connection with the correct full method, original user metadata, one configured service token, and verified TLS

#### Scenario: Downstream failure and cancellation

- **WHEN** Me or UpdateProfile returns a structured field error with explicit status, authorization denial, dependency failure, timeout, or cancellation
- **THEN** existing safe status/code/locale/retry-guidance mapping is preserved, including explicit non-400 field errors
- **AND** no retry, fallback identity/profile, or later mutation follows a failed authentication

#### Scenario: Regression and concurrent isolation

- **WHEN** existing Register/Login/GET Me/ListUsers requests or concurrent profile requests execute after route registration
- **THEN** their route shapes, public/admin policies, RPC counts, context propagation, and request-data isolation remain intact
- **AND** neither duplicate authentication nor a new unversioned/profile route alias is introduced
