# Spec Delta

## MODIFIED Requirements

### Requirement: Reuse secure request transport and retain existing behavior

The three session RPCs SHALL use the existing reusable auth connection with verified TLS1.3, exactly one environment-owned service token, original bearer, locale/request metadata, and bounded parent deadline/cancellation. Method and IAM namespace grants SHALL remain backend/operator-owned. No browser authority claims, per-request connection, local JWT/session storage, dependency/env change, or backend/protobuf modification SHALL be introduced. Existing auth lifecycle, Me, password, profile and ListUsers URLs, responses, credential policies, and RPC ordering SHALL remain unchanged. Self-session routes SHALL NOT acquire unversioned or legacy-users aliases. Administrative user-target session routes SHALL be a separate capability with their own explicit permission/precondition contract and SHALL NOT replace, alias, or change self-session authorization and browser behavior.

#### Scenario: Secure ordered dispatch and denial

- **WHEN** eligible self-session requests run or the backend denies method/IAM namespace authority
- **THEN** one Me and at most one session RPC use the same secure connection and expected credentials/metadata, without expanded privileges or retry

#### Scenario: Concurrent existing and new requests

- **WHEN** list/revoke-one/revoke-all requests run concurrently with distinct owners, metadata and flag outcomes alongside existing endpoints
- **THEN** arguments, results, cookies, and metadata remain isolated and existing endpoint policies/order remain intact

#### Scenario: Route ownership and compatibility

- **WHEN** self-session routes are registered
- **THEN** they belong to auth on the existing shared authenticated /me group without duplicate Me middleware
- **AND** /me/sessions and /api/v1/users/me/sessions are not self-session aliases; /api/v1/users/{user_id}/sessions belongs only to separately specified target administration, with literal me invalid as a target ID

#### Scenario: Separate target administration

- **WHEN** independently permission-protected user-target session routes are added under /api/v1/users/{user_id}/sessions
- **THEN** self-session /api/v1/me/sessions URLs, one-Me ordering, unrestricted-owner eligibility, ignored Version/idempotency inputs and existing cleanup policies remain unchanged
