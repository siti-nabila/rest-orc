# auth-input-validation Specification

## Purpose

Menolak input register/login yang tidak sesuai aturan sebelum pemanggilan service, dengan field errors yang konsisten, tanpa mengubah credential login lama atau contract route/response existing.

## Requirements

### Requirement: Email validation without changing forwarded input

For both register and login, the service SHALL reject email containing non-ASCII input. It SHALL validate a separate trimmed, lowercased view of the email against required, 6–50 byte length, and the backend-compatible pattern below. A valid request SHALL forward the original email unchanged. Non-ASCII input SHALL NOT become valid by being removed during trimming or normalization.

```text
^[A-Za-z0-9](?:[A-Za-z0-9._~\-]*[A-Za-z0-9])?@[A-Za-z0-9](?:[A-Za-z0-9._~\-]*[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9._~\-]*[A-Za-z0-9])?)+$
```

#### Scenario: Email boundaries

- **WHEN** the validation view is a valid email of exactly 6 or 50 bytes
- **THEN** email validation passes for both endpoints
- **AND** a view of 5 or 51 bytes fails with the applicable length error

#### Scenario: Surrounding whitespace and capitalization

- **WHEN** the original email is ` User.Name@Example.COM ` and its validation view passes
- **THEN** the service forwards ` User.Name@Example.COM ` exactly to the downstream

#### Scenario: Email pattern compatibility

- **WHEN** an email uses `.`, `_`, `~`, or `-` in positions allowed by the pattern
- **THEN** format validation passes
- **AND** `user+tag@example.com`, missing domain dots, or disallowed leading/trailing punctuation fail format validation

#### Scenario: Non-ASCII email

- **WHEN** email contains a non-ASCII character, including surrounding Unicode whitespace
- **THEN** validation rejects the email rather than removing that character and accepting the request

### Requirement: Register password validation

The register endpoint SHALL require a nonempty password of 6–50 bytes whose every byte lies in `0x21–0x7E`. It SHALL NOT require a combination of uppercase, lowercase, numeric, or symbol characters. It SHALL NOT trim, case-convert, or otherwise mutate the password.

#### Scenario: Register length boundaries

- **WHEN** a printable, non-space ASCII password has exactly 6 or 50 bytes
- **THEN** register password validation passes
- **AND** a 5 or 51 byte password fails the applicable length rule

#### Scenario: Simple password is allowed

- **WHEN** a password is `aaaaaa`, `123456`, or `!!!!!!`
- **THEN** local register validation accepts it without a composition error

#### Scenario: Invalid register characters

- **WHEN** a nonempty register password contains space, tab, CR/LF, NUL, DEL, or non-ASCII characters
- **THEN** the password field receives an invalid-character error
- **AND** the service does not silently strip, trim, or replace the offending input

### Requirement: Login preserves existing credentials

The login endpoint SHALL validate only required and 6–50 byte length rules for its password. It SHALL NOT apply the register password character restriction or composition rules. For locally valid requests, both email and password SHALL be forwarded exactly as received after transport parsing.

#### Scenario: Existing password with spaces or non-ASCII characters

- **WHEN** a login password of 6–50 bytes contains spaces, non-ASCII characters, or control characters representable in the request
- **THEN** local password validation passes and the exact password is forwarded for authoritative downstream verification

#### Scenario: Byte count rather than rune count

- **WHEN** a login password contains three `é` characters, totaling 6 UTF-8 bytes
- **THEN** local password length validation passes
- **AND** the same value fails register character validation

#### Scenario: Login password boundaries

- **WHEN** a login password is empty, 5 bytes, or 51 bytes long
- **THEN** local validation fails with the applicable required/length errors
- **AND** passwords of exactly 6 or 50 bytes are not rejected by length validation

### Requirement: Aggregate local field errors before business RPC

The service SHALL collect all applicable email and password validation errors before invoking Register or Login downstream. Invalid input SHALL return HTTP 400 using the existing bad-request `code/errors/data` envelope, with localized message lists per field and empty `data`. Required, minimum, maximum, and format/character errors SHALL remain distinct; each rule SHALL contribute at most one message per field. No error SHALL include the submitted password or other credential values.

#### Scenario: Both fields invalid

- **WHEN** email and password both fail validation
- **THEN** the response contains errors for both fields and neither business RPC is called

#### Scenario: Empty input aggregates required and minimum rules

- **WHEN** email's validation view and the raw password are both empty
- **THEN** each field contains its required and minimum-length messages
- **AND** the empty email does not additionally receive a format error, and the empty password does not receive a character error

#### Scenario: Multiple failures on one field

- **WHEN** a nonempty register password is too short and contains an invalid character
- **THEN** both minimum-length and invalid-character messages are returned for password

#### Scenario: Locale and safe messages

- **WHEN** invalid input is submitted with supported English, Indonesian, or Chinese request locales
- **THEN** each field message uses the existing locale resolution/fallback behavior without duplicating the field name or echoing credential values

### Requirement: Preserve public routes and existing responses

The service SHALL preserve the existing public routes `POST /api/v1/users/create` and `POST /api/v1/users/login`, without a bearer requirement or `Me` call. Successful responses SHALL retain HTTP 200 and existing success code/message, while adopting the explicitly revised `auth-session-lifecycle` contract: Register returns lowercase token with an empty value plus pending registration identity/status without cookie mutation; Login returns safe lowercase session data and secure refresh/CSRF cookies. The legacy `data.Token` shape SHALL be replaced, not duplicated as an alias. Login SHALL require allowed Origin and JSON content type as defined by auth-session-lifecycle. Unknown fields SHALL retain existing parsing behavior; `confirm_password` SHALL NOT become required or gain new validation. Downstream errors for locally valid requests SHALL continue to follow the existing `http-error-mapping` contract. All email/password validation requirements above SHALL remain unchanged.

#### Scenario: Locally valid success

- **WHEN** locally valid input passes the applicable transport/origin policy and the relevant business RPC succeeds
- **THEN** exactly that RPC is called once, original request values and the selected safe context metadata are forwarded, and its lifecycle success response is returned
- **AND** Register issues no cookie while Login issues refresh/CSRF cookies without exposing refresh in JSON

#### Scenario: Downstream conflict or authentication failure

- **WHEN** locally valid input produces a downstream email conflict 409 or login failure 401
- **THEN** its HTTP status, public code, and safe error body are preserved by the shared mapper without retry or success cookie mutation

#### Scenario: Malformed transport payload

- **WHEN** malformed JSON or incompatible known-field types fail transport parsing after applicable transport/origin checks pass
- **THEN** existing HTTP 400 behavior is retained and the business use case/RPC is not invoked

#### Scenario: Existing request shape

- **WHEN** a request supplies only email/password or includes unknown fields such as confirm_password
- **THEN** no confirmation field is required or forwarded and unknown fields retain existing parser behavior
- **AND** no new password composition/character restriction is applied to Login

#### Scenario: Browser policy rejects before credential dispatch

- **WHEN** Login is missing an allowed Origin or JSON content type
- **THEN** the lifecycle 403/415 policy applies without Login or Me calls, independently of unchanged credential validation rules
