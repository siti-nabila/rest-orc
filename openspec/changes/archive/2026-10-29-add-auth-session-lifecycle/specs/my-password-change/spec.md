# Spec Delta

## MODIFIED Requirements

### Requirement: Clear the refresh cookie only on confirmed login-required success

When a successful backend result has login_required=true, the response SHALL expire both documented host-only cookies, __Secure-refresh and __Secure-csrf, at Path=/api/v1/auth using empty values and past expiry/removal semantics. Both SHALL use Secure and SameSite=Lax with no Domain; refresh SHALL remain HttpOnly and CSRF SHALL remain readable. The response SHALL not require reading either cookie, send refresh as an RPC credential, require CSRF for this bearer-authenticated password operation, issue replacement credentials, or claim to clear frontend memory itself. Frontend clients use login_required to clear their access token and log in again. Failures and results with login_required=false SHALL not clear either cookie.

#### Scenario: Successful change without an incoming refresh cookie

- **WHEN** a bearer-authenticated request succeeds with login_required=true, whether or not cookies were received on /me/password
- **THEN** the response expires both cookies with their documented names/path/security attributes and no Domain
- **AND** the RPC used only bearer for user authentication without CSRF or refresh-cookie verification

#### Scenario: Failure leaves browser credential transport unchanged

- **WHEN** authentication, parsing, local validation, or the password dependency fails
- **THEN** the response does not clear or replace either cookie and does not present failure as a successful logout

#### Scenario: Backend does not request login again

- **WHEN** the successful backend result has login_required=false
- **THEN** its response retains that value and does not emit refresh-cookie or CSRF-cookie deletion

### Requirement: Preserve the existing auth and users endpoints

The password operation SHALL retain its existing URL, envelope, credential validation, RPC counts, and restricted-user eligibility. Me/UpdateProfile/ListUsers SHALL retain their existing URLs, responses, validation, and access policies. Public Register/Login SHALL retain their legacy URLs, unchanged credential validation, and zero-Me single-business-RPC behavior while adopting the explicit auth-session-lifecycle response/browser-policy migration. Refresh/Logout/CSRF SHALL exist only at their documented versioned auth paths and SHALL NOT become password aliases or change its authentication pipeline. No confirmation field SHALL become required.

#### Scenario: Existing endpoint regression

- **WHEN** public Register/Login, restricted/nonrestricted GET Me, PATCH profile, admin/non-admin ListUsers, or ChangePassword requests execute
- **THEN** each retains its specified role/restriction policy and ordered RPC counts
- **AND** public Register/Login have lifecycle responses, Register emits no cookie, Login emits its session cookies, and password cleanup emits two deletions only on confirmed login-required success

#### Scenario: Password route ownership and isolation

- **WHEN** POST /api/v1/me/password and the lifecycle routes are registered
- **THEN** password remains auth-owned on the shared authenticated /me group while profile updates retain their own restricted-user policy
- **AND** no unversioned /me/password, /users/password, /users/me/password, or /auth/password alias is introduced
