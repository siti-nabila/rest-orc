# Design

## Context

Lihat `proposal.md` bagian Why untuk motivasi dan `specs/me-profile/spec.md` untuk contract REST. Acuan utamanya adalah `docs/grpc-auth-validation-rules.md` bagian 4, 8, dan 17.

State yang telah diperiksa:

- `internal/auth.Authenticator` sudah memanggil shared client Me sekali untuk setiap request yang melewati middleware autentikasi. `Principal` saat ini hanya menyimpan ID, email, fullname, dan legacy role metadata.
- Dependency terpublikasi `api-contracts` yang sudah dipin menyediakan seluruh field profil/access melalui `user.v1.UserData`. Me menerima `google.protobuf.Empty`; identitas pengguna berasal dari bearer dan scope berasal dari service caller.
- Shared auth client/transport sudah mengirim service credential, menggunakan TLS, serta mempertahankan user authorization, locale, deadline, dan cancellation. Transport tidak perlu diubah untuk endpoint ini.
- `internal/app` memiliki public dan admin route group. ListUsers memakai autentikasi lalu gate admin; Register/Login terdaftar sebagai public routes.
- Pembacaan read-only backend mengonfirmasi satu snapshot Me yang mencakup profil/access. Restricted snapshot tidak membawa privilege; phone sudah berupa hasil komposisi backend. Backend gagal tanpa partial result ketika pembacaan/finalisasi snapshot gagal.
- Test buatan project berada di root `test/`. Harness TLS gRPC `bufconn` dan `testing/synctest` sudah tersedia di `test/integration/authservice`.

## Goals / Non-Goals

**Goals:**

- Jadikan hasil autentikasi sebagai satu snapshot request yang cukup untuk response Me, tanpa round trip tambahan.
- Pertahankan batas protobuf, request context, dan DTO REST dengan perubahan kecil yang mudah dibaca.
- Buat restricted state dan kegagalan dependency aman sejak snapshot dibentuk, serta cegah alias mutable data antarpembaca/request.

**Non-Goals:**

- Tidak membuat general-purpose snapshot framework, cache, orchestration usecase/gateway kosong, atau authorization engine baru.
- Tidak mengubah admin policy ListUsers, cara normalisasi bearer, runtime configuration, dependency version, generated files, atau sibling repository.
- Tidak menambah update profile, change password, cookie/session management, retry, atau endpoint alias.

## Decisions

### 1. Lengkapi Principal yang sudah dimiliki boundary autentikasi

Tambahkan field profil/state/access ke `internal/auth.Principal`: address, phone, country code, optional created-at, status, uint64 version, must-change-password, typed role/application assignments, dan map permissions. Pertahankan field serta behavior `HasRole` existing. Role/application assignment memakai value type milik `internal/auth`, bukan generated protobuf.

Pemetaan `UserData` dilakukan di boundary autentikasi, dengan helper mapping terarah bila diperlukan. Representasikan waktu internal sebagai optional `*time.Time`; validasi protobuf timestamp sebelum konversi. Nil Me response tanpa error memakai error catalog existing `ErrGRPCAuthMeResponseRequired`; timestamp invalid menjadi safe internal error tanpa membuat public error code baru. Timestamp absent tetap absent. Skip nil nested assignments dan pertahankan urutan/nilai backend, termasuk false permissions.

Untuk `must_change_password=true`, jangan masukkan legacy roles, scoped roles, application access, atau permissions ke snapshot, sekalipun dependency memberi nilai nonempty. Profil/state dan restriction marker tetap disimpan. Tidak ada inference privilege dari role codes/names atau pemilihan namespace lokal.

**Alasan:** Authenticator sudah memiliki response Me dan menjadi boundary identitas yang digunakan middleware. Mengembangkan snapshot existing menghindari DTO protobuf bocor ke handler, model paralel dengan sumber data berbeda, dan RPC kedua. Alternatif memanggil Me dari usecase/handler ditolak karena menggandakan RPC; menyimpan protobuf mentah di context ditolak karena coupling transport dan risiko eksposur field.

### 2. Context snapshot memiliki salinan datanya sendiri

Mapping dari dependency menyalin collections ke value type lokal. `WithPrincipal` dan `PrincipalFromContext` menyalin slice, map, dan optional time pointer, sehingga penulis/pembaca tidak mendapat reference mutable milik context atau dependency. Gunakan clone sederhana pada field yang benar-benar mutable; tidak perlu generic deep-copy library. Nil collections internal boleh tetap nil agar fixture/behavior principal existing tidak berubah; normalisasi JSON dilakukan di DTO.

**Alasan:** Principal kini memuat permission map dan collections yang lebih kaya. Shallow struct copy saja dapat mengubah hasil otorisasi/response request lain. Alternatif convention-only immutability ditolak karena API context existing mengembalikan value yang tetap memuat mutable references.

### 3. Handler Me hanya memproyeksikan snapshot ke DTO REST

Tambahkan `internal/features/auth/handler` dengan handler dan response DTO terpisah. Me dimiliki feature `auth` karena menyediakan snapshot identitas/akses autentikasi pengguna login. Pembacaan/pengelolaan profil serta perubahan data user sebagai resource tetap milik feature `users`. Login/Register termasuk tanggung jawab autentikasi, tetapi pemindahan ownership implementasi existing dari feature users ke auth tidak termasuk change ini.

Penyesuaian struktur yang diminta user menempatkan file langsung di `internal/features/<feature>/handler/`, memakai package `handler`. Me berada di `auth_handler.go` dengan DTO `me_response.go`; handler users existing dipindahkan ke `users/handler/users_handler.go` tanpa mengubah ownership atau behavior Login/Register/ListUsers. Root test mengikuti layout `test/features/<feature>/handler/`. Tidak ada subfolder `http/`.

`internal/auth` tetap menjadi shared boundary principal/request authentication; `internal/features/auth` memiliki handler/DTO feature. Handler mengambil principal terverifikasi dari request context lalu memakai shared success writer. Missing principal menghasilkan error unauthorized existing. Constructor mengikuti dependency validation/error catalog terdekat bila membutuhkan writer wajib.

Tidak ada usecase/gateway baru: sesudah middleware, endpoint hanya melakukan transport projection tanpa orchestration atau business decision tambahan. Tidak ada protobuf import, environment read, client call, role gate, atau permission fallback di handler.

Contract field lengkap ada di spec. Pilihan serialization yang harus dikunci oleh test:

- Success envelope existing dengan HTTP 200, code `SS`, dan message `Profile retrieved successfully.`
- Nama snake_case; ID tetap JSON number seperti API existing. `version` selalu string desimal melalui integer formatting, tanpa floating-point conversion.
- `created_at` valid diformat UTC RFC3339 dengan fractional precision yang tersedia; absent menjadi JSON null. Tidak menambahkan waktu saat ini.
- Arrays selalu `[]`, permissions selalu object `{}` ketika kosong, tanpa `omitempty` yang menghilangkan field contract.
- Phone/country code diteruskan persis; tidak concat, normalize, atau mengubah legacy repeated prefix.
- DTO hanya berisi allowlist field spec, termasuk value type REST untuk nested assignments. Tidak serialize Principal/protobuf secara langsung.

**Alasan:** Ownership pada feature `auth` mengikuti tanggung jawab Me, tanpa membuat feature `me` terpisah atau mencampurkannya dengan pengelolaan resource user. DTO eksplisit menjaga contract frontend dan batas keamanan. Alternatif raw protobuf JSON ditolak karena bergantung pada generated serialization/field tambahan; ID string ditolak untuk konsistensi ID existing, sedangkan version string diperlukan untuk menjaga presisi JavaScript.

### 4. Tambahkan authenticated route tanpa gate admin

Tambahkan authenticated group pada route composition yang hanya memasang middleware autentikasi existing. Registrasikan Me melalui `internal/app/auth_routes.go` pada `/api/v1/me`, lalu hubungkan dari feature registration. Nama file mengikuti ownership feature; public path tetap `/api/v1/me`, tanpa prefix `/auth` baru. Locale tetap berasal dari group `/api/v1` existing.

Jangan memasang Me di group admin dan jangan memperluas middleware ke prefix public users. Pertahankan urutan/policy Register/Login/ListUsers. Abaikan query/body yang tidak memiliki contract input; downstream tetap menerima empty Me request dan configured service identity.

**Alasan:** Backend Me sudah memverifikasi identitas/session dan membatasi scope caller, sehingga endpoint self-profile tidak membutuhkan local admin/permission grant. Alternatif reuse admin group menolak non-admin/restricted user; route public akan menghilangkan autentikasi yang diperlukan.

### 5. Gunakan mapper kegagalan existing tanpa fallback

Biarkan error Me dan transport mengalir melalui response error mapper existing, termasuk downstream HTTP status, locale, retry guidance, timeout, cancellation, serta default internal failure. Handler hanya berjalan setelah snapshot sukses. Jangan menambah retry, stale cache, default profile, atau empty permissions sebagai pengganti error.

**Alasan:** Satu boundary mapping sudah memiliki contract yang disepakati. Mapper endpoint khusus akan menggandakan precedence HTTP error dan berisiko mengubah response existing.

### 6. Verifikasi melalui test per boundary dan wire integration

- `test/auth`: full snapshot mapping, nil/invalid response, restricted suppression, context cloning, dan legacy role behavior.
- `test/features/auth/handler`: typed JSON contract, exact large version, timestamp/null, collection shape, phone passthrough, safe field allowlist, dan missing principal. Path ini mencerminkan handler feature auth, terpisah dari `test/auth` untuk shared authentication boundary.
- `test/app` serta integration: registration, non-admin/restricted success, public/admin regression, no alias, unused target claims, dan exactly one Me tanpa aggregation RPC.
- Reuse TLS `bufconn` harness di `test/integration/authservice` untuk real middleware/client path dengan configured service token, user bearer, locale, parent deadline/cancellation, errors, fresh consecutive snapshots, dan request isolation. Callback fake backend menyediakan snapshots terkontrol; tidak memakai real network/backend/database.
- Test `synctest` memakai HTTP app config `DisableDefaultDate` bila diperlukan untuk menghindari global fasthttp date ticker keluar dari lifecycle bubble. Pertahankan shared fixture existing untuk endpoint lain.
- Jalankan targeted tests, broad build/test/vet, dan race detector pada area snapshot/context dengan `GOWORK=off` serta `-mod=readonly`. Tidak membutuhkan dependency update.

**Alasan:** Unit test mengunci detail DTO/mapping, sedangkan integration membuktikan one-RPC dan metadata pada wiring nyata. Alternatif hanya mock handler tidak mendeteksi double call atau kesalahan route policy.

## Risks / Trade-offs

- [Principal bertambah field dan biaya cloning per request] → Salin hanya data snapshot yang dibutuhkan; tidak menambah framework/cache. Uji aliasing dan targeted race; hindari refactor di luar boundary terkait.
- [Shared authenticator dipakai ListUsers juga] → Pertahankan HasRole untuk snapshot normal dan jalankan regression admin flow. Restricted suppression mengikuti contract backend yang memang tidak memberi privilege.
- [ID uint64 dalam JSON bisa melewati presisi JavaScript] → Pertahankan representasi number existing sesuai spec baru; version memakai string. Perubahan representasi ID lintas API membutuhkan keputusan terpisah.
- [Response Me sukses tetapi nil/timestamp invalid] → Tolak sebagai safe internal error sebelum menyimpan principal, tanpa membuat identitas default.
- [Backend mengembalikan privilege pada restricted snapshot] → Suppress sejak mapping dan uji fixture nonempty yang disengaja.
- [Status server/configuration production berbeda dari fake] → Integration menguji contract terpublikasi dengan TLS/service credential; provisioning service identity dan deployment tetap menjadi tanggung jawab environment existing.

## Migration Plan

1. Implementasikan snapshot/mapping dan isolation, lalu handler DTO dan route; lengkapi test serta README sesuai task.
2. Jalankan verifikasi `GOWORK=off`, scope/diff review, dan validasi OpenSpec sebelum change dinyatakan siap arsip.
3. Tidak ada migrasi database, konfigurasi baru, atau dependency rollout. Deployment mengikuti proses repository existing dan tidak dijalankan dalam change ini.
4. Rollback dengan mengembalikan implementasi change beserta route Me; endpoint existing tidak membutuhkan migrasi data atau penyesuaian contract.
