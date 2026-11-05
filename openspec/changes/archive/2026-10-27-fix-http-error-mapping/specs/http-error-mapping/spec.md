# Spec Delta

## Purpose

Menentukan HTTP status dan response error yang konsisten untuk validasi lokal rest-orc dan kegagalan downstream, sambil menjaga contract error, locale, dan informasi retry yang aman bagi caller.

## ADDED Requirements

### Requirement: Local validation and local error status

The service SHALL return HTTP 400 for request validation errors produced locally before the business RPC, including required, length, format, and disallowed-character errors. Local field errors SHALL use the existing `code/errors/data` envelope, with `errors` mapping each field to its localized list of messages. Known local errors SHALL retain their catalog/shared HTTP mapping; unrecognized errors SHALL return HTTP 500 with a generic localized internal-error response.

#### Scenario: Local field validation error
- **WHEN** local request validation returns one or more field errors for required, length, format, or disallowed-character checks
- **THEN** the response has HTTP status 400, the bad-request code, all field messages in the request locale, and empty `data`
- **AND** the rejected request does not invoke the business RPC

#### Scenario: Existing parsing rejection
- **WHEN** an existing endpoint rejects malformed JSON or an invalid query before its use case executes
- **THEN** it returns HTTP 400 and does not execute the business use case or business RPC
- **AND** authentication RPCs already required by a protected route remain permitted

#### Scenario: Known local error
- **WHEN** a known local catalog error or an already supported HTTP transport error reaches the response boundary
- **THEN** its existing mapped HTTP status and safe localized response are retained
- **AND** an unmatched endpoint retains its distinct endpoint-not-found response

#### Scenario: Unknown local error
- **WHEN** an unrecognized local error containing internal details reaches the response boundary
- **THEN** the response has HTTP status 500 and a generic localized internal-error body without those details

### Requirement: Explicit downstream HTTP status takes precedence

The service SHALL use `ErrorInfo.metadata.http_status` as the actual HTTP response status when it represents an integer from 400 through 599. This status SHALL take precedence over catalog and gRPC fallback status, regardless of whether field errors are present. Field errors SHALL determine the error body shape without forcing HTTP 400.

#### Scenario: Email conflict with field error
- **WHEN** the downstream returns `http_status=409` and a field error indicating that email is already used
- **THEN** the HTTP response status is 409 and the email error is retained in the field-error envelope

#### Scenario: Downstream validation with field error
- **WHEN** the downstream returns `http_status=400` and an email field error
- **THEN** the HTTP response status is 400

#### Scenario: Explicit status differs from catalog or gRPC
- **WHEN** a downstream error supplies a valid status of 409, 429, or 503 that differs from its catalog status or gRPC fallback
- **THEN** the actual HTTP response uses the explicit downstream status, both with and without field errors

#### Scenario: Unknown Reason with explicit status
- **WHEN** the downstream supplies an unregistered Reason and a valid explicit HTTP status
- **THEN** the service retains that HTTP status rather than replacing it with 500
- **AND** public code/message handling still follows the safe-envelope requirement

### Requirement: Deterministic fallback and cancellation mapping

When downstream HTTP status is absent, malformed, or outside 400–599, the service SHALL first use the catalog definition identified by Reason. Without a known definition, the service SHALL apply the following gRPC fallback: InvalidArgument→400, Unauthenticated→401, PermissionDenied→403, NotFound→404, AlreadyExists/Aborted→409, ResourceExhausted→429, FailedPrecondition→400, Unavailable→503, DeadlineExceeded→504, Canceled→499 if the connection remains writable, and all other codes→500. A field-error body SHALL NOT override this selection. Native deadline/cancellation errors, including wrapped errors, SHALL map to 504/499 when no structured downstream status takes precedence. The service SHALL NOT retry automatically.

#### Scenario: Invalid status with known Reason
- **WHEN** metadata status is absent, malformed, below 400, or above 599 and Reason is registered
- **THEN** the catalog status is used even when field errors are present

#### Scenario: Missing catalog definition
- **WHEN** explicit status is unusable and Reason is absent or unknown
- **THEN** each gRPC code uses the specified fallback table, including for errors with field details

#### Scenario: Transport failure and unknown code
- **WHEN** a transport failure has no structured details
- **THEN** its recognized gRPC code uses the fallback table and an unrecognized error uses HTTP 500
- **AND** internal transport messages are not exposed

#### Scenario: Deadline or cancellation
- **WHEN** a deadline or cancellation reaches the response boundary as a gRPC status or a native context error, directly or wrapped
- **THEN** it maps to 504 or 499 respectively if no explicit downstream status or applicable catalog definition takes precedence
- **AND** a disconnected client is not assumed to receive a response

### Requirement: Safe error envelope and locale compatibility

The service SHALL retain the existing `code/errors/data` envelope and empty error `data`, without adding internal metadata to the JSON response. A nonempty public downstream code from structured error metadata SHALL be preserved; when absent, code SHALL come from the known Reason or the selected fallback. Registered descriptions SHALL use the request locale; structured public field messages SHALL be preserved as supplied by the service. Raw unregistered/internal gRPC status messages SHALL NOT be exposed and SHALL use generic localized catalog descriptions instead. HTTP status selection SHALL remain independent of this message fallback. Response success behavior SHALL remain unchanged.

#### Scenario: Structured downstream code and localized fields
- **WHEN** a service returns a public metadata code and multiple localized field messages
- **THEN** the response retains that code and all field messages, including multiple messages for a single field
- **AND** it contains no raw metadata, credentials, or internal dependency details

#### Scenario: Registered description
- **WHEN** a downstream error identifies a registered Reason without field errors
- **THEN** its description is resolved in the HTTP request locale, including existing arbitrary-language handling

#### Scenario: Unsafe raw message with explicit status
- **WHEN** an unregistered downstream error supplies a valid HTTP status and a raw status message containing internal details
- **THEN** the explicit HTTP status is retained while the description is replaced by a generic localized catalog message

#### Scenario: Success regression
- **WHEN** a request succeeds
- **THEN** its existing HTTP status, envelope, and payload remain unchanged

### Requirement: Safe Retry-After forwarding

When a downstream error includes a valid nonnegative retry delay in `RetryInfo`, the service SHALL emit `Retry-After` as integer delay-seconds rounded up to avoid recommending an earlier retry. Missing, invalid, negative, or unrepresentable durations SHALL omit this header without changing status/body selection. This guidance SHALL NOT trigger an automatic retry or change dependency call order/count.

#### Scenario: Valid retry delay
- **WHEN** a service returns a retry delay of 1.2 seconds with an error
- **THEN** the response includes `Retry-After: 2` with the independently selected status and error body
- **AND** the failed business call is not repeated

#### Scenario: Zero retry delay
- **WHEN** a service returns an explicitly supplied zero retry delay
- **THEN** the response includes `Retry-After: 0`

#### Scenario: Unusable retry guidance
- **WHEN** retry guidance is absent or its duration is missing, malformed, negative, or cannot be safely represented
- **THEN** the response omits `Retry-After` and retains the selected status/body
