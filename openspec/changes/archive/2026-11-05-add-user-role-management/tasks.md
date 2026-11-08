# Tasks

## 1. Users-owned contracts and validation

- [x] 1.1 Add users-owned actor, raw/typed list query, 17-field assignment view, int64-safe list data and command input/result DTOs without Fiber/protobuf/concrete client imports; add root `test/features/users/domain` scenarios verifying exact JSON fields, []/false/null behavior, independent collections and versions above 2^53.
- [x] 1.2 Implement numeric path, six-key list parsing, pagination/defaults, application/status/keyword normalization, unsupported/duplicate controls and 2048-byte opaque cursor rules; add deterministic domain scenarios for all boundaries and accumulated EN/ID/ZH field errors, then run the users domain runner.
- [x] 1.3 Implement required string-array role selection, original 1–20 count, numeric-identity duplicate rejection, owned ascending IDs, raw reason validation and exact one-key/Version parsing using applicable existing primitives; verify domain/header scenarios for wrong types/presence, 21 entries, `10`/`00010`, forbidden bytes, key opacity, precise versions and no source fallback. Document these public input rules in README and add only necessary dictionary-backed local validation entries with matching locale checks.

## 2. Compatible shared UserService wrappers

- [x] 2.1 Add the three-method UserRoleManagementService seam, client field/default SDK on the current connection and additive constructor after NewWithRoleCatalogService, preserving prior signatures/guard precedence; add `test/client/authservice` mocks/scenarios verifying old constructors/defaults and each nil dependency guard, with concise constructor ownership comments.
- [x] 2.2 Add ListUserRoles, GrantUserRoles and RevokeUserRoles wrappers in `internal/client/authservice/user_client.go` with original error identity, RequestContext/deferred cancellation and nil context/request/response guards; extend client scenarios to verify one exact SDK call, safe internal dictionary errors and earlier deadlines/cancellation, then run the authservice client runner.

## 3. Assignment gateway and defensive mappings

- [x] 3.1 Add explicitly named UserRoleManagementGateway and full list request/response mapping in users `gateway/authservice`; add gateway scenarios verifying target, explicit id ASC, logical keyword/FULL_TEXT_TRIGRAM, filters/cursor, no projection leakage, all 17 fields, nullable timestamps/provenance, total above int32, exact target version and empty/out-of-range pages without extra RPCs.
- [x] 3.2 Add grant/revoke mapping with copied metadata, removal of inherited mutation preconditions, exactly one selected key and owned expected_user_version/role IDs; add gateway scenarios for parent/request immutability, changed subset/no-op/replay, invalid sortedness/subset/flag/target/version, malformed list metadata/items and empty safe failures, then run the users gateway runner.
- [x] 3.3 Exercise dependency error identity, current authority/eligibility/atomicity/conflict outcomes and context termination across all three gateway methods; verify no lookup, partial result, automatic retry, version arithmetic or alternate command occurs. Document in README the backend-owned grant prerequisites, revoke safety, no-op/replay and application-access boundary; preserve the shared mapper registry and confirm fallback/structured-field/Retry-After scenarios.

## 4. Independent permission usecases

- [x] 4.1 Add focused list/grant/revoke usecases with small users-owned contracts and invalid-principal/restriction/exact-permission guards before business normalization; add root usecase scenarios verifying each permission-only non-admin actor, false/missing/wildcard/prefix/unrelated grants, restriction, nil context, invalid input precedence and zero or one gateway call.
- [x] 4.2 Preserve whole-batch inputs/results and dependency failures without local policy, read/role permission coupling, cache or retries; verify usecase scenarios for eligible no-ops/replays, target/result ownership and backend denials. Document the three distinct permissions and separately authorized list/detail/catalog version/option reads in README, then run the users usecase runner.

## 5. REST handlers composition and browser policy

- [x] 5.1 Add the explicit UserRoleManagementHandler directly under users `handler/` with operation-specific interfaces, designated path/query/body/header parsing, shared writer and standalone no-store; add handler scenarios verifying exact success messages/fields, malformed/non-object/wrong-typed inputs, read body/header tolerance, forged sources, safe errors and zero cookie/token effects, then run the users handler runner.
- [x] 5.2 Add `internal/app/user_role_routes.go`, invoke it before the broad legacy users admin group and extend LifecycleNoStore path matching before CORS/auth; add app/browser scenarios verifying exactly three routes, one authentication, malformed targets, HEAD/read policy, allowed GET/POST OPTIONS with zero RPCs, denied Origin/missing bearer no-store, literal me/unsupported aliases and unchanged public/legacy users/catalog routing/cache behavior.
- [x] 5.3 Add controlled fake-backed HTTP flows under `test/integration/users` covering one Me then the selected business RPC, permission separation, full history/nulls/precision, command batches/no-op/replay/errors and no hidden catalog/detail/access/session calls; update README route tables, all response fields, curl examples using string role_ids and target Version/key, cursor reset and conflict guidance, then verify documented examples against the HTTP scenarios.

## 6. Shared TLS transport and mixed-request isolation

- [x] 6.1 Extend the existing integration UserService mock Server with the three methods, keeping one UserService registration and existing RoleService registration; add `test/integration/authservice` TLS scenarios for the reused verified connection, one environment credential, original bearer, exact methods/payload presence, safe copied metadata, locale/request ID, nil/error handling and deadline/cancellation. Run that integration runner and confirm no environment/dependency changes are required.
- [x] 6.2 Add controlled concurrent list/grant/revoke plus existing user/catalog/public flows using distinct actors/targets/query/IDs/reasons/keys/versions/results, and assert isolated nullable output, metadata, errors, call ordering and no cookie effects; run targeted race tests across affected client/users/auth/browser/app/integration packages. Document existing backend method/IAM/permission prerequisites and absence of automatic access provisioning in README; verify no test needs Docker or a live backend.

## 7. Cross-cutting verification and scope review

- [x] 7.1 Format changed Go files and run affected runners followed by `GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go test ./...`, full `go build ./...` and `go vet ./...` using RTK where supported; verify all pass and record actual results, retaining targeted race evidence from group 6 and resolving failures attributable to this change.
- [x] 7.2 Run `openspec validate add-user-role-management --strict`, review full diff/status and `git diff --check`, and audit every requirement against delivered scenarios/README; verify root-only authored tests, no generic Graphify type collisions introduced, no generated artifacts/dependencies/configuration/sibling edits, no extra public contracts or RPCs, and preservation of pre-existing working-tree changes. Report checks and material limitations without archiving, committing or deploying this change.
