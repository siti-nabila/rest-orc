# Spec Delta

## MODIFIED Requirements

### Requirement: Preserve existing endpoints and isolate request data

The Me endpoint SHALL preserve its own response and authorization policy, existing Register/Login/ListUsers paths, and ListUsers success/error shapes and admin authorization. Public Register/Login SHALL adopt only the explicit response/browser-policy changes in auth-session-lifecycle while retaining existing credential validation and zero-Me behavior. Authentication middleware SHALL NOT be broadened over public or lifecycle routes, and ListUsers SHALL NOT migrate its admin gate to permissions through this change. Mutable profile/access data SHALL remain isolated from dependency objects, other requests, and callers reading the request snapshot; it SHALL NOT be stored in global mutable state.

#### Scenario: Existing public and admin flows

- **WHEN** Register/Login and ListUsers requests execute after Me and lifecycle route registration
- **THEN** public routes remain free of bearer/Me requirements with their specified lifecycle responses, and ListUsers retains its Me-before-ListUsers admin flow
- **AND** neither /users/me nor an unversioned /me alias is introduced

#### Scenario: Snapshot mutation and concurrent requests

- **WHEN** a dependency reuses mutable response collections or concurrent requests carry different snapshots
- **THEN** modifying a returned snapshot does not mutate dependency/context data or another response
- **AND** bearer, locale, role/application collections, and permissions do not leak between requests
