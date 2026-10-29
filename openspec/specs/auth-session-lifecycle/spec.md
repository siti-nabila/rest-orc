# auth-session-lifecycle Specification

## Purpose

Provide browser-safe session issuance, refresh, logout, and CSRF bootstrap while grpc-auth remains authoritative for credential validity, session expiry, rotation, restriction, and revocation.

## Requirements

### Requirement: Preserve public URLs while returning pending registration

The service SHALL retain `POST /api/v1/users/create`, HTTP 200, code SS, and message `Register Successfully`. A successful response SHALL contain exactly `token:""`, positive backend `registration_id`, and backend `registration_status:"PENDING"`. It SHALL NOT issue or remove session cookies, return refresh/session credentials, automatically log in the user, or synthesize a registration identity. Invalid or inconsistent successful backend results SHALL fail safely as HTTP 500. Existing email/password validation and exact credential forwarding SHALL remain unchanged.

#### Scenario: Pending registration

- **WHEN** locally valid registration succeeds with a backend PENDING registration
- **THEN** one Register and zero Me/Login RPCs occur and the response contains the pending-registration data without Set-Cookie

#### Scenario: Invalid registration success

- **WHEN** Register returns nil, zero registration identity, non-PENDING status, or session/access/refresh credentials inconsistent with registration
- **THEN** a safe HTTP 500 is returned without fabricated success, cookie changes, or credential leakage

### Requirement: Issue browser session transport on successful login

The service SHALL retain `POST /api/v1/users/login`, HTTP 200, code SS, and message `Login Successfully`, while replacing legacy `data.Token` with exactly `token`, `token_type:"Bearer"`, `session_id`, `access_expires_at`, `session_expires_at`, and `must_change_password`. The access token, session identity, expiry, and boolean SHALL come from the backend; timestamps SHALL use RFC3339 UTC and false SHALL remain explicit. The refresh credential SHALL appear only in the refresh Set-Cookie header, never in response JSON. Login SHALL require one allowed Origin and JSON content type, but no bearer, Me, or existing CSRF token.

#### Scenario: Complete login success

- **WHEN** an allowed-origin JSON login succeeds with a complete valid backend session
- **THEN** exactly one Login and zero Me RPCs occur, the safe lowercase data is returned, and refresh plus bound CSRF cookies are issued
- **AND** neither a legacy Token alias nor refresh_token appears in JSON

#### Scenario: Restricted login

- **WHEN** the backend successfully logs in a restricted user
- **THEN** must_change_password=true and authoritative session details are returned without adding role or permission claims
- **AND** existing restricted Me/profile/password access policies remain in force

#### Scenario: Origin content type and credential validation

- **WHEN** Login lacks an allowed Origin, uses unsupported content type, or fails existing email/password validation
- **THEN** it returns respectively safe HTTP 403, 415, or 400 with no Login/Me RPC and no cookie changes

### Requirement: Refresh with cookie proof independently of access JWT

The service SHALL provide `POST /api/v1/auth/refresh` using exactly one nonempty `__Secure-refresh` cookie, allowed Origin, and valid bound CSRF proof. It SHALL invoke exactly one `/auth.v1.AuthService/RefreshSession` with the original opaque cookie value, no authorization metadata, and no Me. Missing or expired access JWT SHALL NOT prevent refresh; any HTTP bearer SHALL be ignored for this operation. Success SHALL use HTTP 200, code SS, message `Session refreshed successfully.`, the same safe data fields as Login, and replacement refresh/CSRF cookies bound to the rotated backend credential. Bodies and query parameters SHALL NOT select credentials, identity, or service authority.

#### Scenario: Refresh after access token expiry

- **WHEN** valid cookie/Origin/CSRF proof accompanies no access token or an expired bearer
- **THEN** only one RefreshSession RPC is sent with no authorization metadata and successful rotated credentials replace both cookies

#### Scenario: Forged body or query credential

- **WHEN** an otherwise valid refresh includes body/query refresh_token, user_id, session_id, namespace, or service-token claims
- **THEN** those fields do not enter the RPC or alter the cookie-selected identity
- **AND** an empty request body remains accepted

#### Scenario: Missing or ambiguous refresh cookie

- **WHEN** refresh has no nonempty refresh cookie or multiple refresh cookie values
- **THEN** safe HTTP 401 is returned with zero RPCs and no cookie changes

#### Scenario: Restricted expired reused or revoked backend proof

- **WHEN** the backend rejects refresh for restriction, expiry, reuse, revocation, or another authoritative failure
- **THEN** its safe mapped error is returned without a second RPC, fallback login, fabricated session, or success cookie mutation

### Requirement: Logout selects exactly one credential mode without fallback

The service SHALL provide `POST /api/v1/auth/logout` without Me. Presence of HTTP Authorization SHALL select bearer mode and require exactly one well-formed `Bearer <nonempty-token>` value; the RPC refresh field SHALL be absent, regardless of incoming cookies. Without Authorization, Logout SHALL require exactly one nonempty refresh cookie plus allowed Origin and valid CSRF, send the optional refresh field present, and omit authorization metadata. Body/query fields SHALL NOT select credentials. Successful backend results SHALL produce HTTP 200, code SS, message `Logout successfully.`, exactly `data.changed` including false, and removal of both browser cookies. Restricted users SHALL remain eligible for backend logout verification.

#### Scenario: Bearer logout alongside automatic browser cookies

- **WHEN** Logout receives a well-formed bearer plus refresh/CSRF cookies
- **THEN** exactly one Logout sends that bearer with absent refresh field, no Me, and no requirement for cookie CSRF
- **AND** success expires both cookies

#### Scenario: Cookie logout without access token

- **WHEN** no Authorization header is present and cookie/Origin/CSRF proof is valid
- **THEN** exactly one Logout sends the original refresh value in a present optional field with no authorization metadata

#### Scenario: Malformed explicit bearer never falls back

- **WHEN** Authorization is present but empty, malformed, or repeated despite a usable refresh cookie
- **THEN** safe HTTP 401 is returned with zero RPCs and no fallback
- **AND** a well-formed bearer rejected by the backend also never falls back or retries

#### Scenario: Confirmed repeat logout

- **WHEN** the backend verifies proof of an already-ended session and returns changed=false
- **THEN** false remains explicit in the success data and both cookies are still expired
- **AND** REST does not independently declare proof of a revoked session valid

### Requirement: Bind double-submit CSRF to the current refresh credential

Cookie Refresh and cookie Logout SHALL require one allowed Origin, exactly one nonempty `__Secure-csrf` cookie, and exactly one matching `X-CSRF-Token` header carrying a server-signed token bound to the digest of the current refresh cookie. Verification SHALL authenticate the signature and binding with timing-safe comparisons. The CSRF token SHALL contain neither the raw refresh credential nor its digest and SHALL NOT act as a replacement RPC credential. Missing, malformed, ambiguous, mismatched, forged, or incorrectly bound CSRF SHALL return safe HTTP 403 before any RPC. Successful refresh SHALL issue CSRF proof for the new credential.

#### Scenario: Valid signed proof

- **WHEN** cookie and header contain the same correctly signed token for the selected refresh credential
- **THEN** the request passes CSRF validation without a local session lookup

#### Scenario: Tamper duplicate and binding failures

- **WHEN** CSRF is absent, malformed, duplicated, has an invalid signature, differs between cookie/header, or is signed for another refresh credential
- **THEN** HTTP 403 occurs without RPCs, cookie changes, or disclosure of the signing secret/refresh digest

#### Scenario: Rotation invalidates the old binding

- **WHEN** a request combines the newly rotated refresh cookie with the previous CSRF token
- **THEN** it is rejected before RPC even when the old token has a valid signature

### Requirement: Bootstrap CSRF without asserting session validity

The service SHALL provide `GET /api/v1/auth/csrf` requiring allowed Origin and exactly one nonempty refresh cookie, without bearer or an existing CSRF proof. It SHALL return HTTP 200, code SS, message `CSRF token issued successfully.`, exactly `data.csrf_token`, and a corresponding readable CSRF cookie. Bootstrap SHALL NOT invoke any RPC, decode/validate the opaque refresh credential, expose it or its digest, modify the refresh cookie, extend session expiry, or claim the user is authenticated. Its CSRF cookie SHALL be session-only because authoritative expiry is unavailable. Missing/ambiguous refresh cookie SHALL return HTTP 401; invalid Origin SHALL return HTTP 403.

#### Scenario: Recover proof on browser reload

- **WHEN** an allowed FE origin requests bootstrap with a refresh cookie and no bearer or CSRF cookie
- **THEN** a fresh bound token is returned in JSON and the CSRF cookie without any RPC or refresh-cookie mutation

#### Scenario: Unknown refresh credential is not authenticated

- **WHEN** bootstrap receives a nonempty opaque refresh cookie of unknown backend validity
- **THEN** bootstrap can issue proof of that cookie's binding but makes no authenticated-session claim
- **AND** subsequent Refresh/Logout must still receive authoritative backend verification

### Requirement: Enforce consistent secure cookie attributes and bounded lifetime

Refresh cookie `__Secure-refresh` SHALL be host-only, HttpOnly, Secure, SameSite=Lax, and Path=/api/v1/auth. CSRF cookie `__Secure-csrf` SHALL use the same host-only/Secure/SameSite/path attributes without HttpOnly. Login/Refresh persistence and Max-Age SHALL not exceed the backend absolute session expiry, and SHALL NOT use access-token TTL or invent a sliding session lifetime. Deletion SHALL use the same names/paths/security attributes and explicit removal semantics. No HTTP or cross-site fallback SHALL weaken the policy.

#### Scenario: Issue and rotate bounded cookies

- **WHEN** Login/Refresh returns a valid session expiring later than its access token
- **THEN** both cookies use the documented attributes and lifetime bounded by the session expiry, including fractional-second rounding
- **AND** refresh_token is not visible through a readable cookie

#### Scenario: Cookie removal without incoming cookies

- **WHEN** confirmed Logout or confirmed password change with login_required=true requires cleanup even without incoming cookies
- **THEN** both cookies receive empty values and removal instructions with their original names/path/attributes and no Domain

### Requirement: Validate complete results before committing browser effects

The service SHALL require nonempty backend access token, refresh token, and session ID, valid future access/session timestamps, and access expiry no later than session expiry before session success. Invalid/nil successful results, missing representable cookie lifetime, and local signing/entropy failures SHALL return safe HTTP 500 without partial cookies or invented fields. Local nonce generation failures SHALL occur before dispatching session-creating/rotating RPCs. Any error or password success with login_required=false SHALL leave cookies unchanged. A successful mutation followed by an invalid result SHALL not trigger a confirmation RPC or retry.

#### Scenario: Invalid successful dependency response

- **WHEN** Login/Refresh returns nil, a missing credential/session field, invalid timestamp, expired timestamp, or reversed expiry ordering
- **THEN** safe HTTP 500 is returned with no partial cookie issuance and no additional RPC

#### Scenario: Local generation failure

- **WHEN** entropy needed for a fresh CSRF nonce cannot be generated
- **THEN** safe HTTP 500 is returned before Login/Refresh dispatch, without cookie changes or credential output

#### Scenario: Dependency failure preserves browser state

- **WHEN** Login, Refresh, Logout, or password processing fails
- **THEN** no success-only cookie issuance, rotation, or deletion is emitted

### Requirement: Apply no-store and preserve safe authoritative errors

Every Register/Login/Refresh/Logout/CSRF response, including local rejection, preflight, and backend failure, SHALL use `Cache-Control: no-store`; password SHALL retain no-store. Existing shared error mapping SHALL preserve valid downstream HTTP status/code/localized field errors/Retry-After, timeout, and cancellation without raw secrets/internal details. Local missing credential/malformed bearer SHALL map to 401, Origin/CSRF denial to 403, unsupported Login media type to 415, malformed Login/Register JSON to 400, and unexpected internal/result failure to 500. Local rejection SHALL invoke zero business RPCs.

#### Scenario: Explicit downstream status remains authoritative

- **WHEN** a lifecycle RPC returns structured 409, 429, or 503 with safe field errors and retry guidance
- **THEN** the same mapped HTTP status and safe body/guidance are returned with no-store, zero retries, and unchanged cookies

#### Scenario: Early rejection and timeout are not cacheable

- **WHEN** Origin/CORS/CSRF validation fails or a dependency times out/is canceled
- **THEN** the response retains no-store and the established local or timeout/cancellation mapping without credential leakage

### Requirement: Require immutable environment-owned browser policy

Startup SHALL require a nonempty exact HTTPS origin allowlist from `AUTH_BROWSER_ALLOWED_ORIGINS` and a dedicated `AUTH_BROWSER_CSRF_SECRET` containing canonical unpadded base64url of exactly 32 bytes, distinct from the service credential. There SHALL be no secret default or development bypass. Wildcard/opaque origins, origins with userinfo/path/query/fragment, malformed entries, missing/invalid secret, or direct configuration bypass SHALL fail startup safely before resources are initialized. New runtime keys SHALL be documented in `.env.example`. Running instances SHALL use immutable validated startup configuration, with the same configured secret providing cross-instance compatibility.

#### Scenario: Safe startup failure

- **WHEN** required browser config is missing or invalid, or the secret equals the service credential
- **THEN** startup fails before serving/opening clients and reports only the field/reason without its secret value

#### Scenario: Multiple instances and explicit config

- **WHEN** two policy instances share valid startup configuration
- **THEN** one can verify tokens issued by the other and later environment changes do not mutate either instance
- **AND** constructing the application directly enforces the same requirements

### Requirement: Restrict browser origins and support credentialed preflight

Login, cookie Refresh/Logout, and CSRF bootstrap SHALL require an exact allowed Origin; bearer Logout without Origin SHALL remain available to non-browser clients. API CORS SHALL allow credentials only for configured origins with no wildcard reflection, vary responses by Origin, and permit required application methods and headers including Authorization, Content-Type, X-CSRF-Token, Version, Idempotency-Key, and existing locale/request-ID headers. Allowed preflight SHALL finish without credentials, CSRF proof, cookies, Me, or business RPCs. Requests carrying a denied browser Origin SHALL fail safely and not perform business operations.

#### Scenario: Allowed preflight

- **WHEN** an allowed FE origin sends a supported preflight for Login/Refresh or a protected endpoint
- **THEN** the response allows that exact origin, credentials, and the requested supported method/headers without invoking authentication or RPCs

#### Scenario: Denied or opaque browser origin

- **WHEN** a browser supplies null, an unlisted origin, an ambiguous Origin header, or a lookalike suffix origin
- **THEN** the request is rejected without business operations or permissive CORS headers

#### Scenario: Non-browser bearer logout

- **WHEN** a caller sends valid bearer Logout without Origin or CSRF
- **THEN** cookie checks do not block backend verification of the single bearer credential

### Requirement: Preserve downstream security context without local session authority

Lifecycle RPCs SHALL use the configured reusable verified TLS auth connection, exactly one server-owned service token, locale/request metadata, original cancellation, and the earlier of parent deadline/client timeout. Only the selected user credential SHALL enter each RPC; browser authority claims SHALL not alter method/namespace grants. No automatic application retry, fallback credential, local JWT/session storage, synthetic session extension, or extra lookup SHALL be introduced. Backend authorization/restriction/reuse/revocation decisions SHALL remain authoritative.

#### Scenario: Ordered isolated secure dispatch

- **WHEN** concurrent Refresh/Logout and existing protected operations use distinct metadata on the shared connection
- **THEN** each receives only its selected credentials, one environment-owned service token, its own metadata, and expected full method without cross-request leakage
- **AND** existing protected flows retain Me-before-business-RPC ordering

#### Scenario: Deadline and backend denial

- **WHEN** a lifecycle request has an earlier deadline/is canceled or the backend denies the method/IAM namespace
- **THEN** the established safe error is returned without another identity, connection, Me lookup, or application retry
