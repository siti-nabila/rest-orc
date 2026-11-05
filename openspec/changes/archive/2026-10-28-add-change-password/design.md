# Design

## Context

Lihat proposal.md bagian Why. Acuan utama docs/grpc-auth-validation-rules.md bagian 4, 15, dan 17; kebijakan field error serta browser cleanup terkait berada pada bagian 13. Lima main specs dibaca termasuk seluruh skenarionya. Password change belum tercakup; tidak ada requirement existing yang perlu diganti.

Graphify dipakai sebagai routing awal, bukan sumber behavior mutakhir. built_at_commit masih d2257fc; source working tree sudah memuat UpdateProfile yang lebih baru. Degree hub tertinggi meliputi dictionary serta shared test utilities; path app → auth handler/usecase/gateway → shared client dan response diverifikasi pada source. Tidak meregenerasi graph.

| Source yang diperiksa | Fakta / konsekuensi |
| --- | --- |
| auth/domain/auth_validation.go | Required/min/max 6–50 serta validasi karakter register sudah ada; current password harus mengikuti kompatibilitas login |
| auth/handler/auth_handler.go | Constructor menerima Register/Login/writer; Me membaca principal dan RegisterAuthenticatedRoutes mendaftarkan GET kosong |
| app/routes.go, auth_routes.go, users_routes.go | Satu shared authenticated /me group dipakai auth GET dan users PATCH; tambah POST relatif tanpa middleware Me kedua |
| client/authservice/auth_client.go, profile_client.go | UserService dan ProfileService mempunyai interface kecil serta New/NewWithService/NewWithServices existing; belum ada AuthService SDK |
| go.mod dan published pb/auth/v1 | Pin api-contracts v0.0.0-20261023084653-d32fd5c1bab4 sudah mempunyai AuthService.ChangePassword, dua string request, dan bool changed/login_required; tidak perlu upgrade |
| grpc-auth/internal/handler/change_password.go | Memetakan request dua password dan kedua flag; nil result merupakan internal error |
| grpc-auth/internal/repositories/domain/password.go | Backend memvalidasi required/6–50 byte; aturan ASCII credential baru dan perbedaan password merupakan policy REST yang disepakati |
| grpc-auth/internal/features/auth_feature/change_password.go | Verifikasi current password, recheck session/account di transaksi, ubah credential/auth version, revoke sesi dan refresh; sukses saat ini true/true |
| grpc-auth/internal/sessions/principal.go, interceptors/session_interceptor.go | Restricted boleh ChangePassword; method juga memerlukan caller namespace iam dan service method grant; tidak bergantung pada role admin REST |
| server.go, config, auth handler, README | API prefix saat ini fixed /api/v1; belum ada penerbitan refresh/CSRF cookie atau configurable cookie policy |
| shared/fixtures/app.go dan integration/authservice transport.go | Ada concrete app wiring bersama dan TLS bufconn UserService/ProfileService untuk diperluas dengan AuthService |

Sibling grpc-auth hanya dibaca; sibling grpc-contracts tidak tersedia. Published module cache adalah acuan types/full method yang benar-benar dikonsumsi rest-orc. Source backend lokal menunjukkan behavior, bukan jaminan deployment sudah memiliki grant atau versi compatible.

## Goals / Non-Goals

**Goals:**

- Flow pendek: shared Me authentication → auth handler → password usecase → small domain gateway → shared AuthService SDK pada connection yang sama.
- DTO HTTP milik domain; aturan password pure dan pesan terpusat; legacy credential tetap persis.
- Hasil dan kebutuhan login ulang tetap berasal dari backend; local success-only browser cleanup tanpa session state baru.

**Non-Goals:**

- Tidak memperluas UserService/ProfileService interfaces, membuat auth gateway Register/Login menanggung method yang tidak dikonsumsi, atau membuat generic credential executor/factory/mapper.
- Tidak mengimplementasikan session revocation, password comparison/hash terhadap storage, CORS/Origin/CSRF framework, login response migration, refresh/logout, maupun frontend memory management.
- Tidak menambahkan Version/Idempotency-Key, retry, no-op shortcut, permission aggregation, environment/dependency baru, atau backend provisioning.

## Decisions

### 1. Request/response sederhana dan reuse rule password di auth/domain

Tambahkan domain/change_password.go dengan ChangePasswordRequest berisi dua string CurrentPassword/NewPassword dengan JSON tags current_password/new_password; response ChangePasswordResponse berisi Changed dan LoginRequired bool dengan JSON tags tanpa omitempty. Required fields tidak memerlukan pointer/presence wrapper: missing, JSON null, dan empty string semuanya invalid. JSON null top-level menjadi zero request lalu required/min errors; malformed JSON/known-field wrong types tetap 400 dari parsing. Unknown termasuk confirm_password diabaikan.

Tambahkan domain/change_password_validation.go: method Validate() error mengagregasi FieldErrors kedua field. Reuse validateAuthLength dan constants existing. Ekstrak loop karakter password register menjadi helper private berparameter field/value yang benar-benar digunakan Register dan ChangePassword; jangan memanggil ValidateRegister dengan email fiktif, menambahkan generic validation framework, atau mengubah Login.

- CurrentPassword: required, 6–50 byte saja; spasi, Unicode, kontrol representable tetap diteruskan tepat setelah JSON parsing. Tidak trim/case folding dan tidak melakukan credential proof lokal.
- NewPassword: required, 6–50 byte, setiap byte 0x21–0x7E; tidak ada komposisi huruf/angka/simbol, trim, atau transformasi.
- Bila kedua string nonempty dan identik byte-for-byte, tambahkan satu difference error pada new_password; tetap agregasikan rule lain yang gagal. Dua empty fields hanya required/min, tanpa difference error tambahan.
- Reuse auth dictionary required/min/max dan pkg/dictionary.ErrInvalidPasswordCharacters. Tambahkan hanya ErrPasswordMustDiffer pada pkg/dictionary/validation_errors.* dengan key password_must_differ, namespace users existing, serta en/id/zh; jangan membuat catalog duplikat di feature atau mengganti identity/password message existing.

**Alasan:** Dua scalar cukup untuk credential required. Helper hanya berbagi policy credential baru yang memang sama; rule current password tetap terpisah agar credential lama tidak terkunci oleh policy baru.

### 2. Usecase hanya bergantung pada satu kemampuan bisnis

Tambahkan domain.ChangePasswordGateway dengan satu ChangePassword(context.Context, ChangePasswordRequest) (ChangePasswordResponse, error). Tambahkan usecase/change_password.go dan NewChangePassword(gateway). Execute(ctx, userID uint64, request) memeriksa nil context (catalog auth existing), userID zero (common unauthorized), lalu request.Validate() dan satu gateway call dengan context/request yang sama.

Handler mengambil userID dari principal terverifikasi, bukan Bind atau browser claims. Scalar userID cukup sebagai guard; jangan membuat actor struct satu field, memasukkan internal/auth.Principal ke domain/usecase, atau meneruskan ID ke protobuf. MustChangePassword tidak menjadi restriction guard dan tidak perlu di-copy ke model baru: route ini memang boleh dipakai restricted user. Error identity/cause dependency diteruskan; tidak ada perbandingan dengan Me, hash/retry/second lookup, atau automatic Me setelah sukses.

**Alasan:** Ownership credential diverifikasi backend melalui bearer, termasuk recheck dalam transaksi. Local Me bukan pengganti otorisasi mutasi/backend revocation dan tidak boleh dijadikan grant lintas request.

### 3. AuthService SDK pada shared connection dengan injection kecil

Tambahkan PasswordService interface satu RPC generated signature (termasuk grpc.CallOption) dan field passwordService pada client/authservice.Client. AuthService berbeda dari UserService; SDK dibuat authv1.NewAuthServiceClient(transport.Connection()), sekali saat construction, tanpa connection/env/security/lifecycle baru. Outbound wrapper ChangePassword ditempatkan pada password_client.go, mengikuti RequestContext/defer cancel serta guard nil context/request/response dan error passthrough existing.

Pertahankan signature/behavior New, NewWithService, NewWithServices dan interfaces Service/ProfileService. Tambahkan NewWithPasswordService(transport, userService, profileService, passwordService) sebagai explicit injection point ketiga SDK. NewWithServices mempertahankan guard urutan transport/UserService/ProfileService sebelum memakai Connection(), lalu mendelegasikan ke constructor baru dengan generated AuthService SDK. Constructor baru memvalidasi seluruh dependency sebelum mengembalikan Client. Constructor existing tetap menghasilkan client yang lengkap; test lama tidak perlu menambahkan password method ke fake UserService.

Tambahkan hanya guard catalog yang diperlukan: dependency PasswordService dan ChangePassword request/response nil; reuse context/transport guards. Tidak membuat variadic optional service, SetService mutator, dependency bundle/framework, second Client, atau constructor factory. Nama PasswordService membatasi kemampuan consumer SDK pada operasi yang sedang dibutuhkan, tanpa menambah Refresh/Logout methods.

**Alasan:** Menambahkan ChangePassword ke UserService interface tidak cocok dengan generated SDK dan memaksa fake yang tidak terkait berubah. Satu injection constructor mengikuti pola existing tanpa abstraksi baru di luar boundary transport nyata.

### 4. Password gateway terpisah dari client Register/Login

Tambahkan auth/gateway/authservice/change_password.go berisi PasswordClient interface satu method shared wrapper, PasswordGateway, dan NewPassword(client). Domain contract diimplementasikan langsung: context guard → authv1.ChangePasswordRequest dua field → satu client call → nil guard → map Changed/LoginRequired ke DTO domain. Tidak perlu mapper type, request/response wrapper file, atau import generated types pada domain/usecase/handler.

Existing Gateway/Client/New Register/Login tetap tidak diperlebar. Gunakan nil-client/context guard auth existing serta response guard baru. Tidak memetakan userID/role/version/mask/idempotency dari browser, menambah namespace metadata iam, atau mengirim cookie sebagai user credential. Dokumen operator mencantumkan full method /auth.v1.AuthService/ChangePassword dan namespace iam yang harus sudah diizinkan credential server.

**Alasan:** Adapter memiliki mapping dan validasi result yang nyata, sedangkan generated transport tetap shared. Method/namespace authority tetap milik backend dan tidak dapat diperbaiki dengan mengarang header REST.

### 5. Handler auth dan satu shared /me group

Tambahkan consumer ChangePasswordUseCase pada auth handler; ubah New menjadi New(register, login, changePassword, writer), dengan guard usecase/writer existing, lalu perbarui semua caller internal dan test. RegisterAuthenticatedRoutes menambahkan POST PasswordRoute=/password pada router yang sama dengan GET path kosong. Implementasi handler boleh di handler/change_password.go agar auth_handler.go tetap mudah dibaca; package tetap handler langsung, tanpa http/.

Handler ChangePassword memeriksa principal exists dan ID nonzero, Bind request domain, meneruskan ctx.Context()/principal.ID/request ke usecase, lalu menulis result melalui shared writer. Rule bisnis dan normalisasi tidak ditaruh di handler. Header Cache-Control no-store dipasang pada handler baru sebelum parsing/dispatch, sehingga sukses dan error yang melewati handler tidak disimpan; kegagalan middleware Me tetap mengikuti response policy existing.

auth_routes.go membuat password gateway/usecase dan menginjeksi handler, tetap memakai meRoutes dari routes.go. Tidak membuat authenticated group /me atau /me/password kedua dan tidak memindahkan auth routes ke users. Mirror pada test/shared/fixtures/app.go; optional before middleware dan DisableDefaultDate tetap tersedia. Public /users/create,/users/login tetap didaftarkan sebelum admin /users, tanpa Me; users PATCH profile tetap menolak restricted di usecase-nya sendiri.

**Alasan:** Shared prefix middleware berjalan pada subpath; memasangnya dua kali akan memanggil Me dua kali. Business authorization khusus password tidak boleh menyalin restriction guard profile atau memakai gate admin.

### 6. Response authoritative dan cookie expiration yang terbatas

Success memakai HTTP 200, code SS, message Password changed successfully., data tepat dua booleans, tanpa omitempty:

```json
{"code":"SS","message":"Password changed successfully.","data":{"changed":true,"login_required":true}}
```

Map kedua flag persis untuk setiap nonnil response, termasuk kombinasi false; jangan memaksa true, membuat login_required dari changed, atau mengarang version/token. Backend lokal saat ini sukses true/true; preserving booleans mengikuti published response tanpa menganggap default flag sebagai fallback identity atau membuat local no-op. Nil response menjadi safe 500 dan tidak menjalankan cleanup.

Setelah dependency sukses dan sebelum success writer, hanya bila LoginRequired=true, expire __Secure-refresh dengan value kosong, Path=/api/v1/auth, HttpOnly, Secure, SameSite=Lax, tanpa Domain, memakai expiry di masa lalu/removal semantics yang didukung Fiber. Tetapkan expiry deterministik untuk test; jangan hanya ClearCookie tanpa atribut/path yang benar. Cookie tetap di-expire walau browser tidak mengirimnya pada /me/password karena path cookie /api/v1/auth berbeda. Tidak membaca refresh cookie atau menaruhnya pada request/metadata RPC.

Name/path/security attributes adalah contract tetap pada prefix /api/v1 yang saat ini fixed di app/server.go, bukan runtime config baru. Tidak membuat configurable cookie/session module hanya untuk satu expiration. Belum ada CSRF cookie di implementasi sehingga tidak mengarang nama cookie CSRF atau issuance flow; ketika lifecycle tersebut diperkenalkan, policy cleanup bersama harus diselaraskan dalam change tersendiri. Backend error/nil response/local validation/parse/authentication failure tidak menghapus atau menerbitkan cookie. LoginRequired=false tidak melakukan cleanup meski response mempunyai changed=true; jangan mengklaim pencabutan sesi tanpa flag backend.

FE bertanggung jawab menghapus access token memory dan meminta login ulang berdasarkan login_required; rest-orc tidak mengubah FE/storage atau JWT lokal. Revocation transaksi seluruh sesi/refresh tetap milik grpc-auth. Tidak menambahkan sesi pengganti, cookie refresh baru, Logout/Revoke RPC, atau automatic verification call setelah password berubah.

**Alasan:** Dokumen meminta browser cleanup saat password change memerlukan login ulang. Satu expiration di handler cukup pada fase ini dan tidak memaksa migrasi Login/Register ke lifecycle cookie yang belum diimplementasikan.

### 7. Error boundary dan verifikasi berbasis behavior

FieldErrors lokal tetap 400 melalui internal/response; missing principal 401. Backend password mismatch/auth/service/namespace denial serta status eksplisit, safe field messages, RetryInfo, timeout 504/cancellation 499 memakai mapper existing; tidak menulis mapping baru per handler atau mengubah status downstream menjadi 400. Jangan log/echo request password, token, raw internal error, atau sensitive metadata.

- Domain/catalog: required/byte boundary 0/5/6/50/51, Unicode byte count dan kontrol pada current, allowlist new, equality tepat/case-sensitive, multi-error en/id/zh, input nonmutation, dan Register/Login regressions untuk shared helper.
- Usecase/gateway/client: guards, context/request persis, zero mutation pada invalid, one call pada valid, error cause/identity, exact flags termasuk false, nil result, SDK constructors dan unchanged mock consumer interfaces.
- Handler/app/HTTP: parsing null/unknown/wrong types, verified actor bukan body, restricted allowed, exact success, conditional cookie attrs/path/expiry, no cleanup on failures, public/Me/profile/admin regression, dan precise RPC counts/order tanpa alias.
- TLS bufconn: register fake AuthService pada server existing bersama UserService/ProfileService; generated dispatch, full methods, caller grant denial outcomes, TLS1.3, one connection, exactly one service token, bearer/locale/request metadata, earlier deadline/cancellation dan concurrent request isolation. Gunakan controlled backend outcomes, bukan real backend verifier/database atau framework session tiruan.
- Targeted suites lalu GOWORK=off -mod=readonly build/test/vet dan targeted race. Semua test di root test, runner/scenario/mocks existing, deterministic synctest/controlled clock untuk deadline/cookie behavior.

## Risks / Trade-offs

- [Restricted user salah ditolak oleh pola profile/admin] → Local auth usecase hanya memeriksa identity, bukan restriction/role; uji Me restricted → ChangePassword dengan privileges kosong dan profile tetap forbidden.
- [Me dipanggil dua kali] → Gunakan meRoutes shared; assert satu Me dan satu password RPC, termasuk production registration regression dan concrete shared fixture.
- [Credential lama rusak oleh validasi/normalisasi baru] → Current hanya required/byte length; helper ASCII hanya Register dan new_password; test whitespace/Unicode/control serta raw payload/context.
- [Backend namespace/method denial setelah Me sukses] → Dokumentasikan iam dan exact grant; pertahankan safe403 tanpa provisioning, namespace claim, fallback, atau retry.
- [Password sudah diubah tetapi response hilang] → Jangan retry otomatis atau mengklaim gagal berarti backend pasti rollback; error timeout bukan alasan menghapus cookie sebagai sukses. FE dapat diminta login ulang melalui alur pemulihan produknya, di luar implementasi ini.
- [Cookie tidak terhapus karena path/attrs berbeda] → Gunakan documented host-only name/path/attrs dan past expiry, uji tanpa incoming cookie; tidak berasumsi REST dapat menghapus access token memory.
- [Bool result false disalahartikan sebagai true] → Map flags persis, cleanup hanya LoginRequired=true; tidak ada fabricated success pada nil/error.
- [Scope melebar menjadi auth lifecycle/heavy abstraction] → Tidak mengubah public response Login/Register atau config/env; setiap interface baru mempunyai consumer boundary saat ini dan mapping langsung.

## Migration Plan

1. Implementasikan DTO/rules/catalog serta unit tests; tambahkan shared AuthService SDK/wrapper dan small password gateway/usecase tanpa dependency update.
2. Rangkai handler dan shared /me route secara utuh, update constructor callers/shared test fixture, lalu conditional refresh-cookie expiration serta README.
3. Lengkapi HTTP/TLS/context/race regressions dan seluruh verification dengan GOWORK=off; operator memastikan existing credential memiliki method grant dan namespace iam pada deployment backend yang sesuai.
4. Tidak ada schema/data/env migration, backend edit, atau deployment dalam change ini. Rollback menghapus endpoint dan dependency/wiring-nya sebagai satu perubahan; tidak memulihkan password/sesi yang telah diubah backend. Login ulang/credential recovery tetap mengikuti backend dan FE.
