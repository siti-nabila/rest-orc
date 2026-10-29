# Design

## Context

Lihat proposal.md bagian Why. Scope mengikuti docs/grpc-auth-validation-rules.md bagian 3, 12, 15, dan 17; spec existing sudah diperiksa lengkap dan tidak perlu diubah.

Graphify dipakai sebagai routing awal: hub utama mencakup dictionary, shared HTTP test utilities, users handler, dan app wiring. Metadata graph masih menunjuk commit dasar d2257fc; seluruh keputusan berikut diverifikasi pada source working tree. Graph tidak diregenerasi.

| Bukti source | Implikasi |
| --- | --- |
| users handler/usecase/gateway saat ini hanya ListUsers | Tambahkan flow profile di users tanpa memindahkan Me atau auth operations |
| internal/app/auth_routes.go membuat authenticated group /me; users_routes.go membuat admin /users | Menambahkan authenticated group kedua /me/profile dapat menjalankan middleware prefix dua kali |
| internal/auth.Principal berisi ID dan MustChangePassword, dipasang middleware Me | Gunakan fakta identity/restriction tersebut; jangan membaca klaim target dari browser |
| internal/client/authservice.Client membungkus UserService pada connection startup | ProfileService adalah SDK berbeda, tetapi dapat memakai connection/TLS/credential yang sama |
| go.mod mem-pin api-contracts v0.0.0-20261023084653-d32fd5c1bab4 | Published package pb/profile/v1 sudah memiliki ProfileService, ProfileRequest, ProfileResponse, FieldMask, dan CountryCode pointer; tidak memerlukan upgrade |
| grpc-auth/internal/handler/profile_session.go | Mengadaptasi ProfileRequest dan mengembalikan profile/version/changed, tanpa expected version atau target ID |
| grpc-auth/internal/features/profile/{update_fields,update_session}.go | Mask nil berarti legacy replacement; mask eksplisit membatasi field. Backend memverifikasi ulang session/owner di transaksi dan menentukan changed/version |
| grpc-auth/internal/sessions/principal.go dan interceptors/session_interceptor.go | Restricted ditolak dengan common.ErrForbidden untuk UpdateProfile. Method ini membutuhkan bearer dan service method grant; tidak menambahkan namespace claim browser |
| pkg/dictionary/validation_errors.* dan response.ErrorMapper | Catalog lokal sudah terpusat; field error lokal menjadi 400 dan status downstream tetap authoritative |
| test/shared/fixtures/app.go dan TLS bufconn transport | Perlu meniru satu shared /me group serta mendaftarkan ProfileService pada server test yang sama |

Sibling grpc-auth hanya dibaca. Path sibling grpc-contracts tidak tersedia; published module cache menjadi acuan protobuf yang benar-benar dipakai rest-orc. Pin backend lokal lebih lama daripada rest-orc, sehingga source sibling tidak dianggap bukti bahwa workspace atau deployment backend otomatis kompatibel. Field yang dipakai sudah dicocokkan dengan published package.

## Goals / Non-Goals

**Goals:**

- Satu pipeline jelas: authentication Me → users handler → profile usecase → domain updater → gateway → shared auth client → ProfileService.
- Presence, normalization, validation, mask, dan no-op dapat diuji tanpa Fiber/protobuf dalam domain.
- Tidak ada salinan DTO HTTP, per-request connection, duplicate authentication, atau fallback profile/version.

**Non-Goals:**

- Tidak menambahkan Version/Idempotency-Key precondition pada RPC yang tidak menyediakannya, ataupun mengganti operasi dengan admin UpdateUser.
- Tidak membuat generic patch type, reflection-driven validator/mapper, generic executor, route registry framework, repository, profile cache, atau global request state.
- Tidak mengubah auth lifecycle, cookies, published dependency, environment keys, sibling code/config, generated files, atau contract endpoint existing.

## Decisions

### 1. DTO domain dan presence memakai pointer string biasa

Tambahkan users/domain/profile.go berisi UpdateProfileRequest dengan empat `*string` dan JSON tags name/phone/country_code/address. Nil dan null memiliki arti sama, sehingga tidak dibutuhkan optional wrapper dengan status absent/null/value. String kosong tetap pointer nonnil dan tidak berubah menjadi absence. Pakai Fiber Bind existing; malformed JSON atau tipe known field yang salah menjadi fiber.ErrBadRequest. JSON null mengikuti parser struct existing sebagai request tanpa selected fields; body kosong/malformed tetap parsing failure. Unknown fields tidak dipindahkan ke protobuf.

DTO response domain berisi Changed bool, Version *string dengan omitempty, dan Profile *Profile dengan omitempty. Profile hanya memiliki ID/UserID uint64 serta Name/Address/Phone/CountryCode string. Ini DTO REST sekaligus output domain sesuai keputusan user; tidak membuat copy output/HTTP DTO lagi. Nil Profile/Version hanya dipakai pada local no-op. Normalisasi mengembalikan salinan request dengan nilai baru; tidak menulis ke string pointer milik caller.

**Alasan:** Empat pointer cukup untuk semantik PATCH yang disepakati. RawMessage map atau generic Patch[T] menambah kompleksitas untuk membedakan dua keadaan yang justru mempunyai behavior sama.

### 2. Rule pure pada domain, catalog pada pkg/dictionary

Tempatkan normalisasi dan validasi di users/domain/profile_validation.go, dengan fungsi kecil per field. Normalisasi country_code lebih dahulu menentukan apakah normalisasi +62 boleh diterapkan pada phone. Presence tidak berubah karena hasil normalisasi string kosong: selected empty tetap invalid. Kumpulkan FieldErrors seluruh field sebelum mutation/no-op decision.

| Field | Keputusan konkret |
| --- | --- |
| name | Periksa raw ASCII letters/space/apostrophe/hyphen sebelum trim/collapse; ordinary spaces saja yang dinormalisasi. Panjang hasil 3–100 byte. Kapitalisasi huruf awal setelah awal string, space, apostrophe, atau hyphen; huruf lain dipertahankan, termasuk akronim NASA |
| country_code | Digit ASCII 1–3 karakter ditambah +; validasi hasil terhadap +[1-9][0-9]{0,2}. Tidak trim, menebak negara, atau menerima empty selected |
| phone | Hanya bila request secara eksplisit mempunyai country_code valid +62: hapus satu prefix +62 atau 62, kemudian maksimal satu leading 0. Selain itu jangan ubah prefix/0. Hasil wajib 12–20 digit ASCII; tidak membuang whitespace, separator, Unicode, atau plus yang tersisa |
| address | Pertahankan nilai persis. Panjang 3–200 byte; printable ASCII, CR, LF diperbolehkan, tab/kontrol lain/Unicode ditolak |

Country-code-only update tidak menambahkan phone ke mask. Phone-only tidak memakai country code dari Me dan tidak membutuhkan RPC tambahan. Boundary length diperiksa setelah normalisasi yang diizinkan; contoh +62/081234567890 menjadi 11 digit dan ditolak.

Reuse required/min/max dari published auth dictionary bila tepat. Tambahkan hanya pesan format/character yang belum tersedia pada pkg/dictionary/validation_errors.{go,yaml}, dengan en/id/zh dan namespace existing tanpa mengganti error password. Jangan membuat ad-hoc public errors, memasukkan rule feature ke pkg, mengubah validasi Login/Register, atau menormalkan output backend dengan rule input baru.

**Alasan:** Domain memiliki business rule dan pkg/dictionary memiliki definisi/pesan. Validasi raw character set mencegah Unicode atau kontrol menjadi valid karena terhapus saat normalisasi. Helper hanya digunakan flow ini sehingga tetap di feature, bukan pkg helper baru.

### 3. Usecase menjaga restriction sebelum no-op dan memverifikasi owner hasil

Tambahkan users/usecase/update_profile.go. Dependency hanya domain.ProfileUpdater dengan satu UpdateProfile operation; domain.Gateway List existing tetap List-only. Untuk input autentikasi, domain.ProfileActor cukup berisi UserID dan MustChangePassword yang diisi handler dari principal terverifikasi; ini dua fakta business guard, bukan copy seluruh principal atau DTO yang di-Bind dari browser.

Urutan Execute(ctx, actor, request):

1. Nil context mengikuti guard catalog existing; actor ID zero menghasilkan unauthorized.
2. Restricted menghasilkan common.ErrForbidden, termasuk ketika request tidak memilih field.
3. Normalize/validate seluruh selected input; jika error, return FieldErrors.
4. Tidak ada selected field → return response Changed=false tanpa Profile/Version dan tanpa gateway.
5. Selected fields ada → panggil updater satu kali dengan context yang sama. Jangan membandingkan dengan data Me untuk menghindari mutation RPC.
6. Pertahankan error identity/cause dependency. Pada success pastikan returned profile owner sama dengan actor.UserID; mismatch menjadi safe internal error, tanpa profile fallback atau call kedua.

Tidak ada lookup target, expected-version substitution, retry, session invalidation lokal, atau penulisan audit REST. Backend tetap memverifikasi user/session/state mutakhir saat mutation. Local no-op hanya membuktikan eligibility dari Me pada request itu; tidak mengklaim lock transaksi/audit/version check backend.

**Alasan:** Me memperbolehkan restricted membaca profil, sedangkan UpdateProfile tidak. Guard hanya di gateway/backend tidak cukup untuk local no-op yang tidak memanggil mutation. Small actor value menjaga usecase bebas dari Fiber dan concrete authentication package.

### 4. ProfileService menggunakan shared client dan transport existing

Tambahkan ProfileService interface satu method pada internal/client/authservice dan field profileService pada Client; generated profilev1.ProfileServiceClient dibuat sekali dari transport.Connection() yang sama dengan UserService. Implementasi UpdateProfile boleh berada di profile_client.go agar nama file menyatakan outbound SDK responsibility. Method mengikuti pattern RequestContext/defer cancel, nil context/request/response guards, dan error passthrough existing.

Pertahankan Service interface UserService serta signature New(transport) dan NewWithService(transport, userService). Tambahkan NewWithServices(transport, userService, profileService) sebagai titik construction/injection eksplisit untuk dua SDK. New dan NewWithService mendelegasikan construction dengan generated stubs yang diperlukan; NewWithServices memvalidasi transport dan kedua dependency. Constructor tambahan ini menghindari perubahan seluruh mock UserService hanya untuk memenuhi ProfileService dan mengikuti pola injection existing; tidak menambah generic factory, option builder, atau transport baru. Nil dependency errors tetap catalog, tanpa secret.

Tambahkan guard error request/response/ProfileService yang benar-benar diperlukan di pkg/dictionary, tanpa mengganti definition/code existing. Tidak mengubah grpcclient, service_credentials, TLS loader, lifecycle resource, atau pembacaan env. ProfileService method grant tetap operator-managed di backend, bukan env/metadata claim baru REST. Tidak menambahkan requirement namespace iam yang tidak ada pada self-profile path backend saat ini.

**Alasan:** Memperluas UserService interface dengan RPC dari service lain akan menuntut generated client/mocks yang tidak bisa mengimplementasikannya. Shared connection dengan dua generated SDK adalah kebutuhan nyata, bukan dua jaringan atau per-request wrapper.

### 5. Adapter profile kecil tanpa memperlebar dependency ListUsers

Tambahkan users/gateway/authservice/update_profile.go dengan concrete ProfileGateway, consumer ProfileClient interface satu UpdateProfile RPC, dan NewProfile constructor. Existing Gateway/Client/New untuk ListUsers tidak diubah menjadi gabungan List/Profile. ProfileGateway mengimplementasikan domain.ProfileUpdater memakai shared authservice.Client; tidak membuat reusable transport atau SDK sendiri.

Bangun ProfileRequest dari normalized selected pointers. FieldMask selalu nonnil, nonempty, dan berurutan deterministik name/address/phone/country_code untuk field yang benar-benar selected. CountryCode pointer nonnil hanya jika selected. Scalar protobuf field yang unselected dapat tetap zero; explicit mask mencegah zero tersebut menimpa stored values. Request tanpa selected field tidak boleh terkirim sebagai nil mask/legacy replacement, termasuk saat adapter dipanggil langsung; reject dengan known bad-request error sebagai boundary guard.

Map response secara langsung ke domain DTO, tanpa mapper interface/struct atau request/response files terpisah untuk beberapa assignment. Tolak nil response/profile, zero profile.ID/profile.UserID, atau version=0 sebagai safe internal error. Convert uint64 version dengan strconv.FormatUint tanpa float. Gunakan bool Changed backend persis, termasuk false; profile dan version tetap disertakan untuk semua RPC success. Usecase memeriksa owner terhadap actor. Return profile strings apa adanya, tanpa concat country code atau normalisasi ulang data legacy.

**Alasan:** Dua concrete adapter pada package yang sama memisahkan contract consumer yang nyata, tanpa memaksa List-only mocks memiliki dummy UpdateProfile. Membuat ProfileMapper/Repository atau generic execution layer tidak memberi manfaat.

### 6. Satu authenticated /me group pada composition root

Di internal/app/routes.go, bentuk satu `meRoutes := routes.authenticated(authhandler.MeRoute)` dan teruskan router tersebut ke registerAuthRoutes serta registerUsersRoutes. Auth mendaftarkan GET path kosong pada meRoutes; users mendaftarkan PATCH /profile pada router yang sama. Jangan membentuk Group("/me/profile", authentication) baru di bawah group middleware /me existing.

auth_routes.go tetap membuat Register/Login/Me handler dan public /users/create,/users/login. users_routes.go membuat List usecase/handler dan profile gateway/usecase; users handler menerima ListUseCase, UpdateProfileUseCase, dan shared writer. Handler UpdateProfile membaca principal context, menolak missing principal, Bind tipe domain request, lalu meneruskan ProfileActor dan context ke usecase. Business restriction/normalization/no-op tetap usecase/domain.

Registrasi auth public tetap sebelum protected /users admin group. GET /api/v1/me tetap dimiliki auth dan tidak diberi guard restricted global; ListUsers tetap memakai role admin existing. Me group dibentuk di composition root, tanpa import users pada auth feature atau auth feature pada users. Simpan route suffix profile pada users handler; tidak menambah alias /me/profile tanpa API version, /users/me/profile, atau /auth/me/profile.

Update seluruh caller internal/constructor test yang terdampak. Mirror concrete wiring tersebut pada test/shared/fixtures/app.go; pertahankan optional context middleware dan DisableDefaultDate untuk synctest. Tidak perlu global router cache atau framework registrasi feature.

**Alasan:** Dua authenticated prefix yang bertumpuk dapat menggandakan Me. Mengurutkan semua users routes sebelum auth akan berisiko menaruh admin /users di depan public Login/Register. Satu shared router menyelesaikan overlap tanpa mengubah public policy atau ownership feature.

### 7. Response dan error memakai satu boundary existing

Handler memakai response.Writer existing, message `Profile updated successfully.`, code SS, status 200. Shape RPC success:

```json
{"code":"SS","message":"Profile updated successfully.","data":{"changed":true,"version":"9007199254740993","profile":{"id":9,"user_id":17,"name":"NASA O'Neil","address":"Jakarta","phone":"812345678901","country_code":"+62"}}}
```

Shape local no-op persis:

```json
{"code":"SS","message":"Profile updated successfully.","data":{"changed":false}}
```

Tidak menambahkan user/session/privilege fields dari Me, version buatan, timestamp, atau fallback profile ketika backend gagal. Owner/profile IDs tetap number sesuai representasi ID existing; version string penuh. Backend Changed=false dengan nonempty mask tetap memakai shape pertama dengan false dan authoritative profile/version.

Local field errors melewati FieldErrors → response 400; missing principal 401, restricted 403. Downstream http_status/code/public field messages/Retry-After tetap melalui shared mapper, termasuk status 409/429/503 dengan field errors dan deadline/cancellation 504/499 bila response masih writable. Unknown/invalid successful response menjadi safe internal error. Tidak ada rewrite mapper atau catalog error lintas fitur.

### 8. Verification mengikuti behavior, bukan struktur file

- Domain unit: all pointer presence combinations yang relevan, normalisasi per field, byte/ASCII boundaries, independent multi-field errors, locale, dan input nonmutation. Test catalog menjaga pesan existing.
- Usecase unit: nil context/identity, restricted sebelum validation/no-op, zero updater call pada invalid/no-op, exact normalized arguments/context, one call pada selected unchanged values, owner mismatch, dan dependency identity.
- Gateway/client unit: explicit exact mask/country pointer, nil/error/zero-version guards, token-only existing methods regression, generated ProfileService dispatch, precise version conversion, dan no mutation of caller metadata.
- Handler/app/integration: malformed known types; missing principal/bearer; non-admin eligible; restricted termasuk `{}`; forged target/mask ignored; exact response shapes; single Me then UpdateProfile; single Me and zero mutation pada no-op/invalid; GET Me/legacy public auth/admin ListUsers tetap sama.
- TLS bufconn: register ProfileService pada server yang sama; assert one connection, full method, TLS 1.3, configured x-service-token, original bearer/locale/request metadata, earlier deadline/cancellation, backend error outcomes, dan concurrent request isolation. Gunakan mocks/fakes server, bukan backend/database real. Pertahankan date ticker control dan cleanup existing.
- Targeted checks, lalu full build/test/vet dan targeted race memakai GOWORK=off -mod=readonly; review diff, imports, public golden responses, scope, generated/dependency files, dan OpenSpec strict.

## Risks / Trade-offs

- [Double Me akibat middleware prefix overlap] → Satu shared /me group dan exact RPC-count tests pada production wiring dan shared fixture.
- [Omitted field menjadi destructive full replacement] → Pointer DTO dan explicit nonnil/nonempty field mask; adapter guard menolak empty selection.
- [Restricted user lolos melalui local no-op] → Guard usecase sebelum normalize/no-op dan integration test restricted `{}`.
- [Normalisasi menghapus karakter invalid atau menebak negara] → Raw allowlist pada name, normalization country-dependent hanya dari request, test Unicode/control/prefix dan normalized length.
- [Backend response salah owner atau version kehilangan presisi] → Fail-closed structure/owner guards, decimal string, dan golden large-version tests.
- [Operator belum memberi method grant] → Dokumentasikan full method ProfileService/UpdateProfile; downstream denial tetap aman. Jangan mengedit backend/env credentials untuk menyamarkan denial.
- [No-op lokal berbeda dari transaksi backend unchanged] → Dokumentasikan shape dan RPC-count berbeda; tidak mengklaim audit/latest version ketika mutation dilewati.
- [Scope melebar menjadi lifecycle atau heavy abstraction] → Tidak mengubah schema/login/cookies; setiap type/interface/file baru harus punya consumer/responsibility saat ini, tanpa generic Patch/Mapper/factory layer.

## Migration Plan

1. Implementasikan domain/catalog/usecase beserta unit tests, lalu shared ProfileService client dan gateway dengan published pin existing.
2. Rangkai satu shared /me group dan users handler secara utuh bersama update constructor/shared test fixture; jaga endpoint existing lewat regressions.
3. Lengkapi HTTP/TLS tests, README dan seluruh verifikasi. Tidak ada data migration, dependency update, env key baru, atau provisioning otomatis.
4. Saat deployment terpisah, operator memastikan credential caller existing mempunyai grant full method UpdateProfile yang diperlukan. Deployment dan backend configuration tidak dijalankan oleh change ini.
5. Rollback mengembalikan penambahan endpoint dan wiring sebagai satu unit; tidak memerlukan perubahan frontend untuk endpoint existing atau rollback schema data backend.
