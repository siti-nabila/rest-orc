# auth-service-credentials Specification

## Purpose

Mengautentikasi rest-orc sebagai caller grpc-auth melalui credential milik server dan TLS terverifikasi, tanpa mencampurkan identitas service dengan bearer pengguna atau mengubah contract REST existing.

## Requirements

### Requirement: Environment-owned required service credential

The application SHALL require `AUTH_GRPC_SERVICE_TOKEN` from server environment configuration before startup. It SHALL accept only canonical unpadded base64url encoding of 32–256 decoded bytes, without trimming or rewriting the token. Missing or invalid credentials SHALL fail startup with a safe configuration error identifying the field without exposing its value. There SHALL be no default or development bypass.

#### Scenario: Valid credential boundaries

- **WHEN** configured service tokens canonically encode exactly 32 or 256 bytes
- **THEN** credential format validation succeeds and the original encoded value is retained
- **AND** no randomness, backend digest match, or expiry guarantee is inferred from format validity

#### Scenario: Missing or malformed credential

- **WHEN** the environment value is absent, empty, whitespace-only, padded, noncanonical, contains invalid characters, or decodes outside 32–256 bytes
- **THEN** startup fails before the application serves requests
- **AND** the error identifies the configuration field without including the supplied value

#### Scenario: Direct configuration cannot bypass required auth settings

- **WHEN** the application is constructed with an explicit configuration rather than loading environment variables
- **THEN** the same auth credential and TLS requirements are validated before resource initialization

### Requirement: Verified TLS for service credentials

The auth connection SHALL require TLS with minimum version 1.3 and normal CA-chain, certificate-validity, and hostname/SAN verification. It SHALL use the configured server-name override when present and otherwise the resolved target authority. A configured CA file SHALL augment trusted roots; an absent CA file SHALL retain system-root verification. Auth configuration SHALL reject TLS disabled. There SHALL be no plaintext or verification-bypass fallback, and service credentials SHALL never be sent on an insecure transport.

#### Scenario: Trusted TLS connection

- **WHEN** the server presents a valid trusted certificate matching the configured hostname or target authority and supports TLS 1.3
- **THEN** the connection can dispatch authenticated RPCs over TLS 1.3 or later

#### Scenario: Disabled TLS or unreadable CA

- **WHEN** auth TLS is disabled or an explicitly configured CA file is unreadable or contains no valid certificates
- **THEN** startup or client construction fails before any business RPC
- **AND** the application does not silently use plaintext or discard the configured CA

#### Scenario: Untrusted certificate or hostname mismatch

- **WHEN** the peer certificate is untrusted, outside its validity interval, or does not match the verified hostname
- **THEN** the TLS handshake fails without a business handler receiving the service credential
- **AND** the connection does not retry using insecure transport

#### Scenario: TLS downgrade rejected

- **WHEN** the peer supports only TLS 1.2 or plaintext
- **THEN** an auth connection does not dispatch an authenticated RPC to it
- **AND** unrelated generic HTTP client TLS behavior remains unchanged

### Requirement: One service credential on every auth RPC

Every RPC dispatched over the configured auth connection SHALL carry exactly one `x-service-token` entry containing the environment-owned encoded credential. This applies to Register, Login, Me, and ListUsers, including calls without a user bearer. Service credentials SHALL require secure transport and SHALL NOT be put in protobuf payloads or substituted for `authorization`. Credential generation SHALL not mutate the caller's context or metadata.

#### Scenario: Public register and login

- **WHEN** locally valid register or login input reaches its RPC without user authorization
- **THEN** the outgoing RPC carries exactly one configured service token and no invented user bearer
- **AND** the public HTTP route still does not invoke Me

#### Scenario: Protected RPCs

- **WHEN** Me or ListUsers is invoked with an existing user bearer
- **THEN** the RPC contains both the original `authorization` entry and one configured `x-service-token` entry
- **AND** the identities remain distinct

#### Scenario: Conflicting outgoing service metadata

- **WHEN** a caller context already contains one or more outgoing `x-service-token` values
- **THEN** the client fails closed before dispatch instead of appending, replacing, or forwarding those values
- **AND** the error is safe and the original context metadata remains unchanged

### Requirement: Server-owned caller authority

The client SHALL use only the configured service credential to establish caller identity. It SHALL NOT derive service identity, namespace, method authority, or service credentials from HTTP headers, query, or body fields. Service ID, credential digest/expiry, exact method allowlist, and exact namespaces SHALL remain authoritative backend configuration. The client SHALL NOT add identity or namespace metadata to claim privileges or infer user permissions from a successful service-authentication check.

#### Scenario: Browser supplied service identity

- **WHEN** an HTTP request supplies `x-service-token`, `x-service-name`, namespace, or user-ID claims outside the existing public contract
- **THEN** these claims do not select the service credential, add caller authority, or change the RPC identity
- **AND** the environment-owned token remains the only service credential sent

#### Scenario: Method or namespace denial

- **WHEN** grpc-auth rejects a verified caller's method or namespace with PERMISSION_DENIED
- **THEN** rest-orc retains the existing safe error mapping without expanding authority or retrying with another identity

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

### Requirement: Explicit credential reload and provisioning boundary

The running client SHALL use the credential captured from validated startup configuration. Changing environment variables SHALL NOT mutate an existing client's credential. A replacement credential SHALL take effect only after explicitly constructing a new configured application/client. Real token provisioning, server digest/expiry/rotation overlap, and method/namespace grants SHALL remain operator-managed and outside this change.

#### Scenario: Environment changes after startup

- **WHEN** the environment token changes while an existing client remains alive
- **THEN** that client continues to use its original immutable configured credential
- **AND** no background reload, automatic token generation, or alternate-token retry occurs

#### Scenario: Explicit replacement client

- **WHEN** a new application/client is constructed with a different valid credential during an operator-managed rotation
- **THEN** only the new client uses that replacement credential
- **AND** actual acceptance and overlap deadlines remain decisions of the backend verifier
