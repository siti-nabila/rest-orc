# Spec Delta

## MODIFIED Requirements

### Requirement: Preserve request context and connection lifecycle

Service authentication SHALL reuse the startup-created auth connection without introducing per-request connections or shared mutable request metadata. It SHALL preserve parent cancellation/deadline, locale, request ID, and other safe existing outgoing metadata. Protected RPCs SHALL preserve their original user authorization; public Register/Login and cookie Refresh/Logout SHALL deliberately omit authorization, while bearer Logout SHALL contain exactly its one selected authorization value, as defined by auth-session-lifecycle. These changes SHALL use copied metadata without mutating the caller context. The configured request timeout SHALL remain bounded by any earlier parent deadline. Existing cleanup and protected RPC ordering SHALL remain unchanged, and no application retry SHALL be introduced.

#### Scenario: Metadata and earlier deadline

- **WHEN** an RPC is called with locale, request metadata, and a parent deadline earlier than the client timeout
- **THEN** safe existing metadata and the earlier deadline are preserved alongside the configured service credential
- **AND** authorization follows the operation's credential mode while the original context and metadata remain unchanged

#### Scenario: Cancellation before dispatch

- **WHEN** the parent context is already canceled or its deadline has expired
- **THEN** the call terminates without invoking a business handler
- **AND** no credential refresh, fallback, or application retry is attempted

#### Scenario: Concurrent requests remain isolated

- **WHEN** concurrent protected, public, bearer-logout, or cookie-lifecycle requests use distinct user credentials and metadata on one connection
- **THEN** each RPC receives only its selected credential mode, its own metadata, and the configured service credential without cross-request mutation or leakage

#### Scenario: Startup connection and cleanup

- **WHEN** multiple requests execute or application startup fails after the auth connection has been created
- **THEN** existing connection reuse and cleanup ownership are preserved
- **AND** protected users flows retain Me-before-business-RPC ordering while Refresh/Logout perform no Me

### Requirement: Preserve safe REST outcomes

Service authentication SHALL preserve public routes, credential input validation, and the http-error-mapping contract. Register/Login success SHALL follow the explicit auth-session-lifecycle response migration rather than legacy data.Token; other existing endpoint responses SHALL remain unchanged except the documented addition of CSRF-cookie cleanup on confirmed password login-required success. Credential values and sensitive metadata SHALL NOT appear in logs, startup errors, or added diagnostics. User access credentials SHALL appear only in designated safe Login/Refresh response fields, refresh credentials only in designated HttpOnly Set-Cookie headers, and CSRF proof only in its designated transport. The server service credential and CSRF signing secret SHALL never be returned. Backend authentication and authorization failures SHALL remain safe downstream errors rather than local field-validation errors.

#### Scenario: Existing response and local validation

- **WHEN** Register/Login succeeds or fails local credential validation after applicable browser checks pass
- **THEN** success retains HTTP 200 and existing success code/message with the revised safe lifecycle data/cookie contract
- **AND** invalid credential input retains HTTP 400 with localized field errors and no business RPC or cookie mutation

#### Scenario: Unknown or expired server credential

- **WHEN** a locally well-formed service token is unknown or expired according to grpc-auth
- **THEN** the resulting UNAUTHENTICATED error follows existing HTTP mapping, including HTTP 401 for normal mapping
- **AND** no secret, raw internal message, fallback identity, application retry, or success cookie mutation is exposed or used

#### Scenario: Downstream error details remain authoritative

- **WHEN** an authenticated RPC returns structured error details such as email conflict with http_status=409
- **THEN** its status, public code, safe localized body, and retry guidance retain existing mapping
- **AND** attaching service credentials does not turn it into a local HTTP 400 error
