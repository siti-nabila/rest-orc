# Design

## Context

See proposal.md for motivation. Discovery verified the following current source and contracts:

- `internal/app/users_routes.go` registers target sessions, detail and status with per-route authentication before the broad legacy role-admin users group. Self-profile remains on the authenticated `/me` group.
- `UpdateProfileRequest` already represents the four editable values with pointers. Its NormalizeAndValidate copies selected values, collects localized field errors and implements the chosen name/country/phone/address rules; HasFields selects non-null values. Self-profile owns a different RPC and response containing a profile ID.
- `docs/grpc-auth-validation-rules.md` sections 3, 5 and the REST table explicitly select `/users/{user_id}/profile`, required Version, optional key and local no-op after validation even for a target update. `docs/user-management-spec.md` still lists the older PATCH `/users/{user_id}` reference. Follow the selected profile route and reconcile that one reference during apply; do not add the older route as an alias.
- Pinned api-contracts d32fd5c1bab4 provides UpdateUserRequest with optional name/address/phone/country_code and expected_user_version, plus a FieldMask. UpdateUserResponse contains a safe User and Changed, with entity version inside User; it has no profile ID or top-level version.
- Read-only grpc-auth handler/user_management.go and features/user/update.go confirm optional key, required version/nonempty mask, one current authority check inside management processing, version conflict, authoritative changed/no-op, audit and backend-owned replay. A local empty-mask call would be rejected by backend, so REST's documented local no-op must terminate before mutation. A selected equal value must still dispatch.
- Existing gateway get_user.go maps the generated User into the 12-field UserDetail, with target range, version, status, timestamp and assignment checks. Root tests already cover self-profile validation and target detail/status HTTP/TLS behavior. The TLS harness registers one generated UserService server.
- Graphify commit f824b74379b1eb2a6eed01a2204c2c5d6ec0a054 was used to route profile/client investigation; current source was checked because the working tree contains newer completed changes. No graph regeneration is required.

## Goals / Non-Goals

**Goals:** Keep target identity, actor authority, field selection and target version separate; reuse proven normalization and safe User mapping; make local and backend no-op outcomes explicit.

**Non-Goals:** A generic management framework, options/setter constructor refactor, profile persistence, version lookup/cache, self-profile migration or local transaction/replay logic. Read and write permissions remain independent.

## Decisions

### 1. Dedicated target operation with existing composition pattern

Add UserProfileActor with UserID, MustChangePassword and Permissions, a small UpdateUserProfileGateway, a focused usecase and UserProfileHandler directly under users/handler. Register PATCH `/users/:user_id/profile` through internal/app/user_profile_routes.go using per-route authentication before the broad admin group; mirror shared test fixtures. Keep all current handler constructors and self-profile wiring unchanged.

The usecase guards nil context/zero actor as unauthorized, then restriction and exact `iam.user.update=true` as forbidden, before business validation, local no-op or mutation. Target=self still requires this grant. Roles/read/status/session grants and browser authority claims are not substitutes. HTTP syntax/path/header parsing may return 400 before usecase eligibility; valid unauthorized input cannot return a successful no-op. A shared prefix middleware group is avoided because it risks duplicate Me or role gating.

### 2. Reuse profile value validation while keeping request sources explicit

Use the existing users-domain UpdateProfileRequest as the JSON body contract; a separate UpdateUserProfileRequest carries UserID, Version, optional IdempotencyKey pointer and the selected profile values. Do not embed technical fields in a JSON-bound request. Omitted/null editable fields are absent; unknown fields, caller field_mask and immutable/authority fields are ignored. Accept JSON object or top-level null (empty selection); absent body, malformed JSON, arrays/scalars and wrong known-field types are 400. Ordinary decoder behavior suffices without a custom duplicate-member framework.

Delegate selected value normalization to the existing NormalizeAndValidate without changing self-profile error identities or semantics. Name accepts ASCII letters/spaces/apostrophes/hyphens, trims/collapses ordinary spaces and capitalizes word starts without lowercasing acronyms; normalized bounds 3–100 bytes. Country code adds plus to digits and validates plus/nonzero-leading 1–3 digits, with no trim. Only this request's explicit valid +62 country code enables removing one +62/62 prefix and one zero from phone; final phone is 12–20 ASCII digits. No prior country lookup/inference. Address is unchanged printable ASCII plus CR/LF, 3–200 bytes. Empty selected values fail, and independent field errors accumulate safely in the request locale.

Parse target using the existing detail ID rule, and Version with the existing positive-int64 decimal rule: ASCII digits only, leading zeros accepted, no trim/sign/float/overflow. Require exactly one case-insensitive parsed Version even for local no-op. Optional key needs presence distinct from absence: nil means not supplied; present key must have exactly one nonblank printable ASCII value of 1–200 bytes. Commas/internal spaces are opaque, no trim/generation/hash. Duplicate keys including identical/case-varied values fail; an empty present key is not absence. Reuse matching session validators without renaming them or introducing a status-specific dependency. Direct usecase/gateway requests validate these bounds too.

### 3. Authenticated local no-op and backend result are separate outcomes

After actor/target/header and body validation, HasFields=false returns HTTP 200/code SS/message `User profile updated successfully.` with data exactly `{"changed":false}`. No user, version, lookup, idempotency reservation/replay, audit or existence/version guarantee is fabricated. The same key on a local no-op cannot substitute for a later command. Version remains mandatory syntactically even though its current value cannot be compared locally.

With any valid selected field, dispatch one UpdateUser even if it equals Me or a previous detail value. REST does not compare actor or cached target state, compute version increments, require growth, or reject safe stored replay results. Backend owns current permission, target existence, version compare, persisted no-op/audit/outbox and optional-key replay. Caller retries perform a fresh Me and eligible command; no automatic retry/refetch/version/key replacement. No key means no REST replay guarantee or generated key.

### 4. Additive SDK and copied transport mapping

Add one-method UserProfileService using the generated UpdateUser signature, a client field/default SDK on the existing connection, and Client.UpdateUser in user_client.go. Add NewWithUserProfileService taking existing explicit dependencies plus the profile command SDK, building through NewWithUserStatusService and validating the new dependency after existing guards. Retain Service, ProfileService and all detail/status/session interfaces and constructors. The existing ProfileService continues to mean the self-profile generated service; do not widen or repurpose it. The wrapper owns nil context/request/response guards, bounded RequestContext and deferred cleanup.

Users gateway update_user_profile.go maps only selected normalized profile pointers into present optional protobuf fields and an explicit nonempty mask in deterministic name/address/phone/country_code order. ExpectedUserVersion always has presence. Copy each pointer and mask, so dependency mutation cannot affect the domain caller. Copy outgoing metadata, delete inherited version/idempotency-key, then set exactly one selected key only when present; absent key leaves zero key metadata. Version goes only in payload. Preserve bearer, safe locale/request metadata, earlier parent deadline/cancellation and the existing fail-closed environment-owned service credential policy.

The gateway rejects direct empty selection rather than issuing an invalid RPC. No GetUser, UpdateProfile, EvaluateAccess, ListUsers or session RPC is part of this mutation. Reusing the existing connection avoids per-request resources and preserves constructor/lifecycle cleanup behavior.

### 5. Safe User snapshot from the command, without inventing a profile response

For dispatched commands, data is exactly `{"changed":<boolean>,"user":<UserDetail>}`. User has exactly id, email, name, address, phone, country_code, status, version, created_at, must_change_password, roles and application_access as documented by user-detail. Version is a positive int64 decimal string at data.user.version; booleans including false remain explicit; collections are copied, preserve order and serialize as []. Stored response text is preserved without reapplying write validation or concatenating phone prefixes. No invented profile ID, duplicate top-level version, auth_version, password/dob/token or effective-permission fields.

Reuse the existing private safe mapper by extracting its User-input portion into a small package-private function shared by GetUser and UpdateUser; preserve the current GetUser wrapper's nil checks and behavior. Do not make an RPC simply to reuse its response type. Validate result/User nonnil, target-matching PostgreSQL ID, supported status, version 1–MaxInt64, valid required creation timestamp and nonnil/range-valid assignments. Invalid command success yields safe 500 with no partial data. Restricted or inactive target state is information, not an actor denial. No new read permission is required to map the backend-authorized mutation snapshot.

This command-shaped response is chosen because the SDK supplies a User, whereas the self-profile response supplies a separate profile identity. Reusing the old self-profile result would invent missing data or force a second RPC. Result/user/collection objects must be independent of caller/dependency data and concurrent requests.

### 6. Existing cache, browser and error boundaries

Extend LifecycleNoStore's target-path matcher for `/users/<segment>/profile` before CORS/auth, including malformed IDs and OPTIONS; standalone handler also sets no-store. Existing CORS supports PATCH, Version and Idempotency-Key. Do not change the self-profile no-store matcher or add BrowserPolicy, cookie fallback, Origin/CSRF proof requirements, Set-Cookie or token effects. This holds for self-target, local/backend no-op, replay and failure.

Use the existing shared error mapper unchanged, including explicit backend status/public code/fields, safe localized unknown-Reason description, Retry-After rounding, native 504/499 and no retry. Auth/common registries are the current configuration; do not globally register user errors. Verify SDK-encoded user VC/IC/RB plus other existing structured failures instead of assuming raw private messages become public.

## Risks / Trade-offs

- [Local no-op mistaken for target/version verification] → document mandatory syntax versus absent backend check; test no GetUser/UpdateUser and no invented snapshot/version, including nonexistent/stale target examples.
- [Optional key inherits an old command key] → copy/delete metadata first; absent/present/blank/duplicate paths tested independently, including empty selection.
- [Profile reuse changes self-profile behavior] → reuse values only, keep separate actor/technical request and SDK method; run full self-profile regression and preserve public constructors/error identities.
- [Actor data substituted for target or normalization inferred from stored country] → distinct actor/target/version fixtures, explicit presence masks and phone-only/country-only cases.
- [Safe User mapper extraction changes detail output] → exact existing detail envelope regressions plus malformed-success and shared-object isolation tests.
- [Backend replay result compared to current state or input+1] → preserve safe backend snapshots/no-op/version with fresh Me and no read/refetch/retry.

## Migration Plan

1. Implement additive domain/usecase/client/gateway/handler/routes and root scenario tests, landing README per layer. Reconcile only the old UpdateUser route reference; no dependency/env/backend/schema migration.
2. Verify fake HTTP/local no-op and separate detail→PATCH permission/version flows, existing UserService TLS bufconn dispatch and controlled mixed concurrency. Run focused race, build/full tests/vet and strict OpenSpec; audit the scoped diff against the pre-existing working tree.
3. Any rollout is separate authorization: provision backend `/user.v1.UserService/UpdateUser` IAM/method grants and exact permission, then perform real backend/frontend smoke checks. Rollback removes the additive route; it does not reverse committed profiles/audit/idempotency outcomes. Do not sync main specs or archive during apply.
