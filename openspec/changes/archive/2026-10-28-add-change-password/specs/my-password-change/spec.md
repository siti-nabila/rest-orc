# Spec Delta

## Purpose

Menyediakan perubahan password diri sendiri untuk pengguna terverifikasi, termasuk restricted user, dengan validasi credential yang kompatibel, hasil backend yang aman, dan arahan login ulang tanpa mengubah endpoint existing.

## ADDED Requirements

### Requirement: Authenticate the current user without excluding restricted sessions

The service SHALL provide `POST /api/v1/me/password` for the verified bearer owner without an admin-role or additional user-permission gate. A valid request SHALL invoke exactly one Me before exactly one `/auth.v1.AuthService/ChangePassword`. Restricted users SHALL remain eligible. The backend SHALL independently verify session, ownership, and current password; browser claims SHALL not select another owner or caller authority.

#### Scenario: Eligible non-admin user

- **WHEN** an authenticated nonrestricted non-admin submits locally valid credentials
- **THEN** one Me precedes one ChangePassword without another user or permission lookup

#### Scenario: Restricted user

- **WHEN** Me identifies a user with must_change_password=true and the credential input is locally valid
- **THEN** ChangePassword is invoked once despite empty roles or permissions
- **AND** the restriction still applies to unrelated protected mutations

#### Scenario: Missing authorization or invalid principal

- **WHEN** authorization is missing or the authenticated principal is absent or invalid
- **THEN** the existing safe unauthorized response is returned without ChangePassword
- **AND** missing authorization invokes neither Me nor ChangePassword

#### Scenario: Failed authentication

- **WHEN** Me rejects a credential or fails with a dependency error
- **THEN** its existing mapped error is returned without a later password RPC or fallback identity

#### Scenario: Forged ownership or service claims

- **WHEN** body, query, or headers supply user_id, email, roles, permissions, namespace, service credentials, version, or field masks outside the password contract
- **THEN** those claims cannot target another user, add authority, or enter the password RPC payload

### Requirement: Parse only the required credential fields

The JSON request SHALL accept current_password and new_password as strings, ignoring unknown fields without forwarding them. Omitted, null, or empty credential fields SHALL fail required and minimum-length validation. Malformed JSON or incompatible types for known fields SHALL return HTTP 400 before ChangePassword. No confirmation field, Version header, field mask, or Idempotency-Key SHALL become required.

#### Scenario: Omitted or null credentials

- **WHEN** an authenticated request contains {}, JSON null, or null/omitted current_password or new_password
- **THEN** applicable required and minimum-length errors are returned for the missing credentials
- **AND** exactly one Me and zero ChangePassword calls occur

#### Scenario: Malformed payload or known-field type

- **WHEN** JSON is malformed, the body is empty, or either credential contains a number, boolean, array, or object
- **THEN** the existing HTTP 400 transport-error envelope is returned with no password use case or password RPC call

#### Scenario: Unknown fields and unsupported preconditions

- **WHEN** valid password fields are accompanied by unknown fields or no Version/Idempotency-Key headers
- **THEN** the request is not rejected for those extra fields or absent headers and only its two passwords reach ChangePassword

### Requirement: Preserve current password compatibility

The service SHALL validate current_password only for required and 6–50 byte length, using the parsed string unchanged. It SHALL NOT trim, normalize, case-convert, apply new-password character rules, or verify/hash the current password locally.

#### Scenario: Current password boundaries

- **WHEN** current_password has exactly 6 or 50 bytes
- **THEN** its length passes
- **AND** empty, 5-byte, or 51-byte values fail the applicable required/length rules

#### Scenario: Legacy characters remain valid locally

- **WHEN** a 6–50 byte current_password contains spaces, Unicode, or JSON-representable controls
- **THEN** the exact credential is forwarded for authoritative backend verification
- **AND** three é characters count as six UTF-8 bytes

#### Scenario: Wrong existing password

- **WHEN** a locally valid current_password is rejected by the backend
- **THEN** the backend's safe status/code/messages are retained without rewriting the credential or attempting another password

### Requirement: Validate a distinct new password using the register policy

The service SHALL require new_password to have 6–50 bytes, each within ASCII 0x21–0x7E, without composition requirements or normalization. Nonempty credentials that are byte-for-byte equal SHALL add a new_password field error; two absent/empty credentials SHALL not gain a redundant equality error. A distinct but invalid new_password SHALL still fail its applicable length or character rules.

#### Scenario: New password boundaries and simple credentials

- **WHEN** a distinct new_password consists of 6 or 50 allowed bytes, including lowercase-only, digits-only, or symbols-only values
- **THEN** it passes local new-password validation without composition requirements
- **AND** 5 or 51 bytes fail the relevant length rule

#### Scenario: Forbidden characters are not removed

- **WHEN** new_password contains space, tab, CR/LF, NUL, DEL, another forbidden control, or non-ASCII characters
- **THEN** new_password receives an invalid-character error and is not cleaned into a valid credential

#### Scenario: Equal credentials

- **WHEN** nonempty current_password and new_password are exactly equal
- **THEN** HTTP 400 includes a localized new_password difference error and ChangePassword is not called

#### Scenario: Exact comparison and unchanged forwarding

- **WHEN** valid credentials differ only by ASCII letter case or another allowed byte
- **THEN** they are not treated as equal through case folding or normalization
- **AND** both parsed strings reach the backend unchanged

### Requirement: Aggregate safe localized credential errors

Local credential validation SHALL collect all applicable errors for both fields using the existing HTTP 400 code/errors/data envelope, bad-request code, localized message lists, and empty data. Each rule SHALL contribute at most one message per field. Messages SHALL not echo either credential or repeat its field name as a prefix.

#### Scenario: Both credentials invalid

- **WHEN** both credentials violate applicable rules
- **THEN** both fields' errors are returned after one Me and without ChangePassword

#### Scenario: Multiple rules and empty input

- **WHEN** new_password is short and contains multiple forbidden bytes, or the two credentials are empty
- **THEN** applicable length/character errors aggregate with only one character message, while empty fields contain required and minimum messages without an equality message

#### Scenario: Supported locales

- **WHEN** invalid input uses English, Indonesian, or Chinese request locales
- **THEN** existing locale resolution supplies field messages without submitted credentials, including the new password-difference message

### Requirement: Return only authoritative password-change results

A successful non-nil backend response SHALL produce HTTP 200, code SS, message `Password changed successfully.`, Cache-Control no-store, and typed data containing exactly changed and login_required as booleans. Both flags SHALL retain backend values, including false. A nil successful dependency response SHALL fail safely as HTTP 500. The service SHALL not invent tokens, version, user identity, a local no-op, or another RPC to confirm success.

#### Scenario: Backend reports password change and login required

- **WHEN** ChangePassword returns changed=true and login_required=true
- **THEN** data is exactly {"changed":true,"login_required":true} in the standard success envelope
- **AND** no password, token, profile, or synthetic version is exposed

#### Scenario: Flags remain authoritative

- **WHEN** a non-nil successful dependency response contains false in either boolean
- **THEN** both booleans are serialized exactly rather than omitted or forced true
- **AND** browser cleanup follows login_required instead of inventing revocation state

#### Scenario: Missing successful result

- **WHEN** ChangePassword returns nil without an error
- **THEN** the response is a safe internal error without a fabricated success, later confirmation RPC, or browser logout side effect

### Requirement: Clear the refresh cookie only on confirmed login-required success

When a successful backend result has login_required=true, the response SHALL expire the documented host-only __Secure-refresh cookie at Path=/api/v1/auth using empty value, past expiry/removal semantics, HttpOnly, Secure, and SameSite=Lax. The response SHALL not require reading that cookie, send it as an RPC credential, issue replacement credentials, or claim to clear frontend memory itself. Frontend clients use login_required to clear their access token and log in again. Failures and results with login_required=false SHALL not clear the cookie.

#### Scenario: Successful change without an incoming refresh cookie

- **WHEN** a bearer-authenticated request succeeds with login_required=true, whether or not any cookie was received on /me/password
- **THEN** the response expires __Secure-refresh with the documented name/path/security attributes and no Domain
- **AND** the RPC used only the bearer for user authentication

#### Scenario: Failure leaves browser credential transport unchanged

- **WHEN** authentication, parsing, local validation, or the password dependency fails
- **THEN** the response does not clear or replace the refresh cookie and does not present the failure as a successful logout

#### Scenario: Backend does not request login again

- **WHEN** the successful backend result has login_required=false
- **THEN** its response retains that value and does not emit a refresh-cookie deletion

### Requirement: Preserve secure transport and downstream error authority

ChangePassword SHALL reuse the configured auth connection, verified TLS, original bearer, one server-owned service credential, locale, existing request metadata, earlier parent deadline, and cancellation. Backend method and namespace grants SHALL remain authoritative. Errors SHALL follow http-error-mapping without retries, fallback credentials, local session state, or credential leakage.

#### Scenario: Secure ordered transport

- **WHEN** a valid password change traverses the configured transport
- **THEN** one Me precedes one ChangePassword over one reused verified TLS connection with the expected full methods, original user metadata, and one configured service token

#### Scenario: Method or namespace denial

- **WHEN** the backend denies the caller method or iam namespace authority
- **THEN** the safe backend error is retained without new browser-derived namespace metadata, another identity, or an application retry

#### Scenario: Structured errors stay authoritative

- **WHEN** ChangePassword returns explicit HTTP status 409, 429, or 503 with public field errors/code and RetryInfo
- **THEN** the response retains the explicit status, safe code/messages, and retry guidance instead of forcing HTTP 400
- **AND** neither Me nor the mutation is repeated

#### Scenario: Deadline cancellation and concurrency

- **WHEN** a request has an earlier deadline, is canceled, or runs concurrently with distinct credentials and metadata
- **THEN** existing deadline/cancellation mapping and per-request isolation remain intact without later retries, cross-request values, or success-only cookie cleanup on failures

### Requirement: Preserve the existing auth and users endpoints

Adding the password endpoint SHALL retain existing Register/Login/Me/UpdateProfile/ListUsers URLs, envelopes, validation, RPC counts, and access policies. It SHALL not add an unversioned or legacy-users password alias, require a new confirmation field, or expose refresh/logout/CSRF functionality through this change.

#### Scenario: Existing endpoint regression

- **WHEN** existing public Register/Login, restricted/nonrestricted GET Me, PATCH profile, or admin/non-admin ListUsers requests execute after registration
- **THEN** their existing responses, role/restriction policies, and ordered RPC counts remain unchanged
- **AND** public Register/Login still perform no Me call and do not emit new cookies

#### Scenario: Password route ownership and isolation

- **WHEN** the new POST /api/v1/me/password route is registered
- **THEN** it belongs to authentication behavior while profile updates retain their own restricted-user policy
- **AND** no unversioned /me/password, /users/password, /users/me/password, or /auth/password alias is introduced
