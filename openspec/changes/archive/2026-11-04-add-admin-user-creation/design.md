# Design

## Context

See proposal.md for motivation. Current source and dependency findings:

- Users composition registers dedicated target-user operations before the broad legacy role-admin `/users` group. Public Register/Login are registered earlier on `/users/create` and `/users/login`. GET collection remains a legacy role-admin list.
- `internal/features/auth/domain/auth_validation.go` owns register/login email and credential rules. ChangePassword also uses its private length/new-password helpers. These methods preserve error identities and raw forwarded credentials. Users owns profile normalization and the shared private safeUserDetail mapper already used by GetUser/UpdateUser.
- REST decisions in docs/grpc-auth-validation-rules.md select POST `/users`, required key, PENDING status and register/profile validation. Name, phone and country_code are required on create; address is optional and follows selected-address 3–200-byte rules. The backend permits wider email syntax and address/phone bounds. Follow the narrower chosen REST rules rather than silently widening existing endpoints.
- Main auth-input-validation explicitly rejects plus-addressing, validates a separate trimmed/lowercased ASCII email view and forwards raw email unchanged. The broader documentation's mention of `+` does not override its instruction to follow register rules. Existing unknown confirm_password input is ignored; create adds no confirmation feature or public-auth change.
- Pinned SDK d32fd5c1bab4 has CreateUserRequest{Name, Email, Password, optional Status, Address, Phone, optional CountryCode} and CreateUserResponse{User}. It has no expected version, field mask, Changed, registration/session/token result. The shared response Writer deliberately returns HTTP 200 successes.
- Read-only grpc-auth create.go, user_management.go and management_transaction.go confirm required key, canonical trim/lowercase email, explicit status, backend password fingerprint/hash, current permission before replay, default member assignment, safe command snapshot, atomic account/profile/audit/outbox and email conflict. Fresh backend account sets must_change_password=true; REST must not manufacture that value or manage the membership/session effects.
- Graphify revision f824b74379b1eb2a6eed01a2204c2c5d6ec0a054 routed auth/register/client investigation; current source was verified because the working tree includes newer completed changes. Generated graph stays unchanged. Root auth/users/client/app and HTTP/TLS scenario runners provide the test pattern; the TLS harness has one generated UserService registration.

## Goals / Non-Goals

**Goals:** A users-owned creation flow with explicit required input, one current actor, shared pure validation, copied outbound data, and a safe account result independent of actor identity.

**Non-Goals:** A management framework, registration delegation, creation of ACTIVE accounts, role/app grants, REST hashing/fingerprint/replay persistence, automatic activation/login, or migration of legacy ListUsers/public auth policies.

## Decisions

### 1. Dedicated permission-gated creation route

Add CreateUserActor{UserID, MustChangePassword, Permissions}, CreateUser body/request/result DTOs and a small CreateUserGateway in users/domain, a CreateUser usecase and UserCreationHandler directly under users/handler. Compose with user_creation_routes.go and mirror shared fixtures. Register exact POST `/users` with route-specific authentication before the broad users admin group, after public auth. No new prefix group, role gate or changes to existing handler constructors.

The usecase rejects nil context/zero actor as 401 and restricted/missing-or-false exact `iam.user.create` as 403 before business validation/dispatch. A non-admin permission-only actor is eligible; role admin, read/update/status/session grants or browser authority claims are insufficient. Syntax/header parsing may return 400 before usecase eligibility. Valid input invokes one Me then one CreateUser. There is no local success/no-op for an empty create body and no Register/GetUser/ListUsers/EvaluateAccess/grant/status/session RPC in this operation.

### 2. Explicit JSON and required-field validation

Require a JSON object; absent/empty/malformed body, null, array/scalar or incompatible known-field types fail 400. Body DTO uses pointer strings for email/password/name/phone/country_code/address/status so omission/null, empty and wrong types remain distinguishable. Only email/password/name/phone/country_code/address plus the constrained status field affect creation. Unknown fields, confirm_password, user/profile IDs, field masks, permissions, role/app assignments, must_change_password and technical key/version claims are ignored, never forwarded or trusted.

Missing/null email/password/name/phone/country_code are required failures. Normalize a copied request, supplying empty validation values for missing required profile fields so required/minimum/format rules retain existing catalogs. Optional omitted/null address stays unselected and maps to the protobuf's empty scalar; an explicit empty address fails the selected minimum. No create-field default comes from Me or a stored profile.

Extract only the pure credential rules needed by both auth and users into `pkg/validation/credentials.go`: email validation, 6–50-byte credential length and new-password character validation, accepting an existing FieldErrors accumulator and field/value arguments. Keep regex/constants private to that helper. AuthRequest's existing public ValidateRegister/ValidateLogin behavior and ChangePassword.Validate use these functions with their unchanged field names and policy split; login/current password must not acquire new-character checks. Preserve error objects/order/messages, raw forwarding and all signatures. This is actual two-feature reuse, not a generic validation engine or a cross-feature DTO import.

Email validation checks original input for non-ASCII before using a separate trim/lowercase view for required, 6–50 bytes and the exact existing register regex. Plus-addressing remains rejected. Forward original parsed email unchanged; backend canonicalizes it. Password is raw, required 6–50 bytes, every byte 0x21–0x7e, no trim/case/composition rule. Keep it only in the inbound body and secured downstream payload; do not log, echo, place in metadata or store it in REST.

Reuse users' UpdateProfileRequest.NormalizeAndValidate for required name/phone/country and optional address, merge its errors with credential/status errors, and normalize once in the usecase. Preserve name 3–100 bytes/capitalization/acronyms, country plus/nonzero-leading 1–3 digits, phone 12–20 digits with explicit +62 prefix/one-zero normalization, and exact address bytes 3–200 printable ASCII plus CR/LF. Gateway copies normalized values without repeating phone normalization. Return all independent field errors safely in the request locale.

### 3. Constrained PENDING status and one caller-owned key

Omitted/null status defaults to PENDING; an explicit status must be exactly PENDING without trim/case conversion. Reject ACTIVE/SUSPENDED/DEACTIVATED/unknown/empty strings, and wrong types are parsing failures. Add a focused users validation catalog entry for this initial-status policy; the existing four-status mutation message would misdescribe it. Gateway always constructs its own present PENDING pointer and rejects contradictory direct input before dispatch.

Require exactly one case-insensitive parsed Idempotency-Key, including identical-duplicate rejection. Reuse the current nonblank printable ASCII 1–200-byte rule, preserving opaque parsed content including commas/internal spaces; no trim/hash/generation/UUID restriction. Missing/invalid key cannot come from body/query/parent metadata. Create has no target path ID or Version requirement. Ignore unused Version values, including missing/malformed/duplicate headers and body/query versions; do not derive or send actor version.

Backend owns key scope, normalized-email account scope, password-sensitive fingerprint, current authority before replay, duplicate account and persistence. Do not hash/cache/store password or replay in REST. Every caller retry runs fresh Me and one eligible CreateUser. IC/RB/conflict/current denial returns through the shared mapper; no retry, key replacement or lookup after a conflict. A saved creation snapshot is not a fresh target read; later status/profile operations remain independent.

### 4. Additive shared SDK and feature gateway

Add UserCreationService with only the generated CreateUser signature, a client field/default SDK on the existing reusable connection, Client.CreateUser in user_client.go and additive NewWithUserCreationService through NewWithUserProfileService. Keep all old interfaces/constructors and their guard ordering unchanged, including the auth Service.Register method. The wrapper owns nil context/request/response guards, RequestContext and deferred cancellation/cleanup. No type assertions, setters, options refactor or per-request transport.

Gateway create_user.go validates key and required country/status preconditions for direct transport consumers, copies normalized scalar values and optional country, and constructs explicit PENDING. Copy outgoing metadata, remove inherited version/idempotency-key and set exactly one selected key. No expected-version/mask/target-ID payload exists. Preserve bearer, locale/request metadata, earlier deadlines/cancellation and the existing environment-owned service-token conflict policy. Do not log payloads or put credentials into metadata.

Use existing safeUserDetail directly on the command's User. No second RPC, new mapper layer or self-profile response DTO is needed. Validate result/User nonnil and all current safe-user invariants: positive PostgreSQL ID, positive int64 version, valid timestamp and nonnil/range-valid assignment entries. Additionally require returned email to equal the canonical trim/lowercase view of the submitted email and creation snapshot status PENDING; fail safe 500 on mismatch without rewriting the backend result or exposing partial data. Backend generates ID; never compare it to actor/body IDs, require version=1 or synthesize assignments. This preserves authoritative saved-result precision while enforcing the selected creation identity/status.

### 5. Command-shaped response without credentials or browser effects

Return HTTP 200/code SS/message `User created successfully.` with data exactly `{"user":<UserDetail>}`. HTTP 200 follows the current Writer convention and avoids a shared success-status refactor. User has the same 12 safe fields as detail: id/email/name/address/phone/country_code/status/version/created_at/must_change_password/roles/application_access. Preserve backend strings, false booleans, timestamps/assignment order; empty arrays become [] and version remains a decimal string. No profile ID, changed, registration_id/status wrapper, top-level version, password/dob/hash/auth_version/token/session/login_required/effective permissions.

The new account's backend restriction/membership is information, not permission for the caller. Copy user and collection results so dependencies/retries/concurrent calls cannot share mutable state. Never inject BrowserPolicy or issue/clear cookies/tokens, invoke Login/Refresh/ChangeUserStatus or infer activation/login on success/replay/error. No cookie fallback, CSRF proof or new Origin requirement for this bearer operation; existing CORS still checks supplied Origin.

### 6. Method-specific no-store and existing error contracts

Extend LifecycleNoStore before CORS/auth for the exact normalized API users collection path only when method POST, or OPTIONS requesting POST. Standalone creation handler also sets no-store. This avoids adding cache changes to GET ListUsers or its GET preflight. Existing CORS already permits POST, Idempotency-Key and JSON headers; no environment change.

Preserve the shared mapper and configured auth/common registries: explicit public status/code, localized safe fields, unknown user-Reason fallback, Retry-After rounding, 504/499 context mapping and no retry/raw internal-message leaks. Verify duplicate-email auth errors and SDK-encoded user IC/RB as real catalog fixtures without globally registering another namespace. Public Register/Login, current-password policy, self/target profile, detail/status and all sessions retain exact contracts/order/cookie effects.

## Risks / Trade-offs

- [Credential helper extraction changes public login/password behavior] → preserve pure rules/error identities and run existing auth/password scenarios plus exact raw forwarding/locale tests.
- [Backend email validator is wider than register] → document existing REST regex and retain plus rejection; canonical response comparison is separate from unchanged raw forwarding.
- [Optional address becomes required or empty selected value bypasses validation] → test omitted/null versus empty and 3/200-byte boundaries explicitly.
- [Unexpected ACTIVE account or wrong account is reported as success] → force PENDING payload and validate canonical response email/PENDING snapshot without fabricating state.
- [Collection POST inherits legacy admin middleware or changes GET caching] → exact method route before group, one-Me inventory and method-specific no-store regression tests.
- [Password/key leakage or local replay semantics drift] → no payload logging/cache/hash, safe DTO only, copied metadata and backend-owned replay/current authority verified with controlled fakes.

## Migration Plan

1. Implement shared pure validators with auth/password regressions, then additive users/client/gateway/routes and root tests/README per layer. No backend/schema/dependency/env migration.
2. Verify fake HTTP ordered creation/replay and separate follow-up grants; extend the existing UserService TLS mock, secure context/deadline tests and mixed concurrency with public auth/ListUsers/target operations. Run focused/race/full build/test/vet, strict OpenSpec and scoped diff/hash audit; keep main specs/archive/generated files intact during apply.
3. Rollout is separate authorization: provision exact create permission and `/user.v1.UserService/CreateUser` IAM/method grants, ensure backend default-member readiness, then real backend/frontend smoke tests. Rollback removes the new route but cannot undo committed accounts/profiles/audit/outbox/replay records. Do not activate or delete accounts as rollback automation.
