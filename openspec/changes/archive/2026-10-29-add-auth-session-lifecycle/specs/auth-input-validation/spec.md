# Spec Delta

## MODIFIED Requirements

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
