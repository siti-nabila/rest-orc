# Design

## Context

Lihat proposal.md untuk motivasi. Graphify pada revision f824b74379b1eb2a6eed01a2204c2c5d6ec0a054 sama dengan HEAD saat inspeksi dan dipakai untuk menelusuri composition/users/auth; semua keputusan diverifikasi di source aktual.

- `internal/app/routes.go` menyediakan router versioned dan authentication handler. `users_routes.go` memasang broad `/users` group dengan authentication lalu RequireRole untuk ListUsers; public auth didaftarkan lebih dahulu. Menambah authenticated prefix group `/users` dapat memberi middleware ganda pada route berikutnya.
- Principal sudah membawa UserID, MustChangePassword, dan cloned Permissions map dari Me. Tidak ada permission middleware atau target detail/version endpoint baru. ListUsers DTO existing tidak membawa version; versi Me adalah versi actor, bukan target.
- Shared client memiliki AuthService SessionService untuk refresh/logout dan SelfSessionService untuk sesi sendiri. Published api-contracts pin sudah memuat ListUserSessions, RevokeUserSession, dan RevokeAllUserSessions pada session.v1.SessionService.
- Backend read-only `handler/admin_sessions.go`, `features/session/{admin,admin_revocation,list,list_cursor}.go`, serta domain/idempotency.go menetapkan exact permission, IAM namespace, target 1–2147483647, canonical UUID, required idempotency metadata, dan expected entity version 1–MaxInt64. Backend melakukan current-authority check sebelum replay dan replay sebelum comparison terhadap versi target terkini.
- Single revoke SDK response hanya Changed. Global admin response Changed/Version uint64, dengan backend Version positif int64; tidak ada LoginRequired. Transaction/audit/outbox/counters tetap backend.
- Self-session mapping sudah memakai Total int64, safe historical timestamps dan opaque cursor; generic pagination.Total tetap int32. BrowserPolicy tidak diperlukan oleh operasi admin.
- Main spec my-session-management melarang user-target route pada requirement kompatibilitas. Delta change ini mempersempit larangan itu menjadi larangan alias self-session, sambil mendefinisikan administrasi target sebagai capability terpisah.

## Goals / Non-Goals

**Goals:** alur users handler → usecase → users gateway → shared SDK/transport yang mudah diikuti, exact permission dari satu Me, pemisahan actor/target/version, validasi input aman, dan tidak ada kebocoran metadata antar-request.

**Non-Goals:** generic permission framework, role-admin migration, target detail lookup, automatic version refresh/retry, local idempotency/revocation registry, perubahan self-session DTO/behavior, atau perubahan backend/contracts/env/dependencies. Nama admin menjelaskan operasi administrasi; eligibility ditentukan permission, bukan role.

## Decisions

### 1. User resource ownership dan route-specific authentication

Tambahkan users/domain DTOs, AdminSessionActor{UserID, MustChangePassword, Permissions}, tiga small consumer-owned gateway contracts, dan tiga usecase. Usecase memeriksa nil context/zero actor ID → unauthorized, restricted → forbidden, lalu exact map key bernilai true: iam.session.read untuk list atau iam.session.revoke untuk kedua mutation. Role, wildcard, prefix permission, namespace yang diklaim browser, serta permission palsu di body bukan authority. Transport syntax errors dapat menghasilkan 400 lebih dahulu; request valid dengan permission tidak cukup selalu 403 tanpa business RPC. Backend tetap memeriksa current authority saat RPC dan sebelum replay.

Tambahkan handler terpisah `internal/features/users/handler/admin_sessions_handler.go`, bukan memperluas constructor handler ListUsers/profile. Handler membentuk actor hanya dari PrincipalFromContext dan mengabaikan body ketiga operasi. Authentication dijalankan satu kali; tidak ada EvaluateAccess/GetUser/ListMySessions/Me kedua.

Composition khusus `internal/app/admin_session_routes.go` mendaftarkan tiga full relative paths `/users/:user_id/sessions...` dengan authentication sebagai handler pada masing-masing route, sebelum broad admin `/users` group existing. Jangan membuat prefix authentication group baru untuk `/users`, memasukkan route ini ke routes.admin, atau mengubah RequireRole. Gunakan handler route-registration dengan injected authentication handler atau explicit route registrations di composition, tanpa generic router factory. Shared test fixture mencerminkan urutan ini.

Alternatif role-admin group ditolak karena role tidak menggantikan exact permission baru. Prefix group tambahan ditolak karena bisa menangkap ListUsers dan memberi Me kedua. Non-admin yang memiliki permission tepat harus lolos; role-admin tanpa permission harus ditolak, sementara ListUsers tetap mempertahankan gate lamanya.

### 2. Input target dan pagination dimiliki users

users/domain memiliki AdminSessionsQuery dan DTO halaman/items sendiri. Gunakan direct focused mapping sesuai wire shape yang sudah dipakai auth; jangan mengimpor auth/domain ke users atau memindahkan self-session implementation/generic pagination hanya untuk mengurangi assignment sederhana.

Path user_id harus nonempty ASCII decimal digits, diparse tanpa overflow dalam rentang 1–2147483647, tanpa trim/sign/hex. Leading zeros diterima sebagai representasi angka dan dipetakan ke numeric target yang sama; tidak mengubah route atau actor. Session ID persis regex backend `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, tanpa trim/case-fold/version restriction/all-zero exclusion.

List memakai page default 1, limit default 20; present page 1–2147483647 dan limit 1–100 wajib nonempty ASCII digits tanpa overflow/whitespace. Duplicate page/limit/last_id ditolak. Cursor omitted/empty berarti tanpa cursor; maksimum 2048 byte setelah satu URL decoding, tidak diparse/di-trim/diubah. Gateway mengirim fixed created_at DESC lalu id DESC, Fields nil/Search nil. Tolak sort/fields/search/keyword/statuses/application_code/created_from/created_to bila hadir termasuk empty. Unknown non-control query/authority claims diabaikan. Pada mutations, query/body bukan sumber target, key atau version; empty body diterima dan malformed ignored body tidak diparse.

Alternatif flexible list controls ditunda sesuai scope self-session awal; generic ListUsers query parser ditolak karena contract/default/filter berbeda.

### 3. Satu Idempotency-Key untuk mutation, tanpa replay lokal

HTTP header name case-insensitive. Kedua mutation mewajibkan tepat satu parsed Idempotency-Key header value, nonblank, panjang 1–200 byte, seluruh byte printable ASCII 0x20–0x7e. Space di dalam nilai diperbolehkan; nilai seluruh spasi tidak. Nilai hasil parsing HTTP diteruskan tanpa aplikasi melakukan trim/lowercase/generate/hash. Tidak wajib UUID. Repeated header ditolak walaupun nilainya sama; comma dalam satu header adalah bagian opaque key, bukan daftar yang dipecah. Batas ASCII adalah constraint transport gRPC nonbinary metadata; backend sendiri menerima bounded UTF-8 text, sehingga Unicode/control yang tidak dapat ditransport harus gagal lokal 400.

Request domain mutation memuat key eksplisit. Gateway membuat salinan outgoing metadata, menghapus inherited idempotency-key/version, lalu memasang tepat satu idempotency-key tervalidasi hanya untuk mutation. List menghapus kedua metadata precondition tersebut tanpa meneruskan HTTP Version/key. Metadata lain yang aman, bearer, locale, request ID, deadline dan cancellation dipertahankan. Service token tetap kebijakan existing: satu env-owned credential, conflicting outgoing service token fail closed, bukan ditimpa gateway.

Local validation menggunakan field errors/catalog aman, tanpa menggemakan key. Backend tetap pemilik scope actor+method+target+key, fingerprint, expiry, replay, authority revalidation, conflict, transaction, audit/outbox. Request ulang caller dengan key sama tetap satu Me dan satu RPC per HTTP request; tidak ada HTTP local replay cache, hidden retries atau fallback identity. Fake SDK tests menyimulasikan replay success/conflict/revoked-authority, bukan mengimplementasikan storage backend.

### 4. Version milik entity target dan hanya untuk revoke-all

Revoke-all mewajibkan tepat satu header Version berupa nonempty ASCII decimal digits dalam 1–9223372036854775807, parse tanpa float/sign/trim/overflow. Leading zeros diterima secara numerik. Set nonnil ExpectedUserVersion pointer uint64 di protobuf. Single revoke/list tidak memerlukan atau meneruskan Version; nilai pada body/query dan Me actor.Version tidak digunakan.

Caller harus memasok target version dari authoritative read yang dimilikinya. Scope ini tidak menambah GetUser, memperluas ListUsers, atau menjanjikan FE sudah punya version source di rest-orc. Jangan menyamakan page total, session ID, auth_version, atau actor version dengan entity version. Sesudah conflict jangan fetch/replace version atau otomatis mengulang mutation. Sesudah lost response caller boleh mengulang request yang sama/key yang sama mengikuti backend contract; REST tidak membandingkan input version dengan current target atau response version karena replay dapat mengembalikan committed result lama setelah authority check.

### 5. SDK terpisah dan mapping safe responses

Tambahkan interface AdminSessionService berisi tiga generated signatures, field adminSessionService, wrappers `admin_session_client.go`, dan satu explicit NewWithAdminSessionService injection point yang menerima dependency existing plus SDK admin. Pertahankan semua signatures/guard ordering constructor lama, termasuk NewWithSelfSessionService; constructor existing menyiapkan default generated admin SDK pada connection yang sama setelah nil guards. Jangan memperlebar Service/ProfileService/PasswordService/AuthService SessionService/SelfSessionService. Wrapper menjaga nil-context/request/response errors dan RequestContext cleanup seperti existing.

Gateway users `NewAdminSessions` mengisolasi SDK/protobuf dari domain/HTTP. List page berisi Items nonnil, Total int64 dan authoritative Page/Limit/TotalPages/HasNext/HasPrev/NextCursor tanpa recompute, narrowing, sort ulang atau total reset pada empty page. Row safe fields persis id,user_id,status,created_at,last_seen_at,expires_at,revoked_at,device_label,user_agent,ip_address,is_current. Validasi canonical UUID, positive PostgreSQL-range owner, ACTIVE/REVOKED, required valid CreatedAt/ExpiresAt, optional valid timestamp/null, paginator positif/nonnegative sesuai bounds. Historical expiry diperbolehkan; UTC RFC3339Nano mempertahankan fractional precision. Usecase menolak row.UserID yang berbeda dari target, bukan actor, tanpa partial data. Pertahankan backend is_current; jangan memaksakan false untuk target lain. Buat koleksi/optional values baru agar dependency object dan concurrent requests terisolasi.

Success HTTP 200/code SS:
- List: message `User sessions retrieved successfully.`, data typed page.
- Single: message `User session revoked successfully.`, data tepat changed boolean termasuk false.
- All: message `All user sessions revoked successfully.`, data tepat changed boolean dan version string desimal; version backend wajib 1–MaxInt64, termasuk nilai >2^53, tidak dihitung input+1 atau diverifikasi harus lebih besar dari input.

Nil/invalid successful response → safe 500. Tidak menambah user_id/login_required/auth_version/token fields pada mutation output. BrowserPolicy tidak diinject/dipanggil: success/error/replay dan target=self tetap tanpa Set-Cookie atau token issuance. Jika actor mencabut sesi sendiri melalui admin API, backend memutuskan validitas request berikutnya; response tidak dapat mengklaim current-session outcome.

### 6. No-store, error authority dan compatibility

Perluas LifecycleNoStore agar path `/users/<segment>/sessions` dan `/users/<segment>/sessions/<segment>` sesuai API prefix ditandai sebelum CORS/auth, termasuk invalid IDs dan OPTIONS. Handler juga memasang no-store untuk standalone tests. Matcher tidak memerlukan ID valid agar 400/401/403/500 tidak cacheable; existing public/login/self patterns dipertahankan. Existing CORS sudah mengizinkan DELETE/POST, Version dan Idempotency-Key; tidak menambah origin/credential/header policy.

Reuse http-error-mapping untuk explicit status/code/localized fields/Retry-After, 401/403/404, 409 version/idempotency conflict, 429/503, timeout 504 dan cancel 499. Tidak mengubah shared mapper untuk endpoint ini. TLS1.3, satu server credential, original bearer, namespace/method authority backend dan earlier deadline/cancellation tetap.

Delta my-session-management memuat full existing requirement `Reuse secure request transport and retain existing behavior` beserta semua tiga scenarios, hanya memperjelas route separation dan menambah coexistence scenario. `/api/v1/users/me/sessions` bukan self alias: literal me gagal target ID validation. `/me/sessions` tanpa version prefix tetap bukan route baru. Existing `/api/v1/me/sessions` tidak menuntut admin permission/Version/key dan retain cleanup policy sendiri. ListUsers masih role-admin; public Register/Login tetap didaftarkan sebelum protected target routes dan broad legacy admin group.

## Risks / Trade-offs

- [Prefix middleware menghalangi permission-only user atau memberi Me ganda] → per-route authentication, route ordering tests nyata, non-admin permission success serta ListUsers/public-auth regressions.
- [Me authority berubah sebelum replay/mutation] → backend tetap memeriksa current authority sebelum replay; REST meneruskan denial tanpa local cache/retry.
- [Version target belum tersedia lewat REST detail endpoint] → dokumentasikan prerequisite caller; jangan menambah GetUser atau memakai versi actor dalam scope ini.
- [Key Unicode ditolak transport atau metadata ganda bocor] → explicit printable ASCII bounds, header multiplicity tests, copied metadata dengan satu selected key, concurrency tests.
- [Global admin outcome dianggap logout actor] → tidak ada BrowserPolicy/Set-Cookie/login_required bahkan target=self; FE/backend handling didokumentasikan tanpa inference.
- [Total/version kehilangan presisi] → Total int64, version string desimal, boundary tests >int32 dan >2^53/MaxInt64.
- [Duplikasi mapping kecil dengan self-session] → users mempertahankan DTO/validation sendiri dan parity tests; hindari shared framework/refactor yang mengubah contract existing.

## Migration Plan

1. Implementasikan secara additive dengan root tests dan README tiap layer; tidak ada schema/env/dependency migration atau perubahan sibling.
2. Sebelum rollout yang diotorisasi terpisah, operator menyiapkan tiga exact method grants/IAM namespace dan user exact permissions; caller menyiapkan authoritative target version serta stable mutation keys. Tidak mengaktifkan permission berdasarkan role secara otomatis.
3. Verifikasi fake SDK HTTP, generated SDK TLS bufconn (perluas fake SessionService existing, jangan register service yang sama dua kali), bounded deadlines/cancellation, mixed concurrency/race, build/full tests/vet/diff dan strict OpenSpec.
4. Real backend/FE smoke test serta deployment dilakukan terpisah. Rollback menghapus routes additive; tidak dapat mengembalikan sesi yang telah dicabut atau replay record backend, dan tidak mengembalikan token lokal.
