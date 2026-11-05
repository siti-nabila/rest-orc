# Design

## Context

Lihat `proposal.md` untuk motivasi. Temuan source yang membentuk desain:

- `auth/domain.AuthResponse` hanya memiliki exported `Token` tanpa JSON tag; `auth/gateway/authservice/gateway.go` memetakan hanya field itu untuk Register dan Login. Golden bersama di `test/shared/fixtures/auth.go` mengunci `data.Token` serta token Register dummy.
- Published `api-contracts v0.0.0-20261023084653-d32fd5c1bab4` sudah memuat `UserTokenResponse` lengkap dan `AuthService.RefreshSession/Logout`. Logout memakai optional `*string refresh_token`: absent berarti bearer, present-empty invalid. Tidak diperlukan perubahan module pin atau generated code.
- Inspeksi read-only backend `grpc-auth/internal/sessions/logout.go` mengonfirmasi bahwa refresh mode menolak authorization metadata, bearer mode memerlukan satu header Bearer valid, dan tidak ada fallback. Handler Login backend sudah menghasilkan sesi, sedangkan Register PENDING tidak menghasilkan sesi.
- Auth client saat ini memakai satu transport untuk UserService, ProfileService, dan PasswordService. Locale middleware menyalin outgoing metadata; bearer baru dipasang authenticator pada route protected. Route session baru harus menjaga metadata dengan aturan credential sendiri tanpa menggunakan authenticator Me.
- ChangePassword saat ini hanya menghapus `__Secure-refresh` pada sukses `login_required=true`. Semua request protected existing dirakit lewat satu `/me` group atau admin `/users` group.
- Graphify `built_at_commit=d2257fc2baa509f00ae462606c0992c58045e4fe` sama dengan HEAD, tetapi working tree berisi implementasi yang belum committed. Graph dipakai sebagai petunjuk routing dan keputusan di atas diverifikasi terhadap source aktual.
- Keputusan browser di `docs/grpc-auth-validation-rules.md` bagian 4, 12, 15, dan 17 sudah menetapkan signed double-submit, exact Origin allowlist, cookie same-site, no automatic refresh retry, serta output lowercase. Main specs masih memuat kewajiban respons lama, termasuk klausul compatibility Me; delta pada change ini menggantikannya secara eksplisit tanpa mengubah behavior Me.

## Goals / Non-Goals

**Goals:** mempertahankan batas handler/usecase/gateway/client, memisahkan hasil internal yang membawa refresh dari DTO HTTP aman, dan menjaga operasi browser stateless antar-instance. Penolakan transport/CSRF harus terjadi sebelum RPC; keputusan status sesi tetap di backend.

**Non-Goals:** menyimpan/verifikasi JWT atau refresh token secara lokal, meniru transaksi rotasi/reuse/revocation backend, membuat server-side refresh single-flight, membangun FE, menambah session-management/admin APIs, atau mengganti role gate ListUsers. Tidak ada perubahan expiry baseline backend maupun deployment melalui workflow ini.

## Decisions

### 1. Gunakan hasil Register dan Login yang berbeda

Pisahkan DTO `RegisterResponse` dan `LoginResponse` di auth/domain dari hasil sesi internal. Register memetakan `token:""`, positive `registration_id`, dan status `PENDING`; nil/inconsistent backend success menjadi safe HTTP 500, bukan registrasi/sesi rekaan. Register tidak mengubah cookie yang sudah ada.

Login dan Refresh memakai hasil internal sesi yang mengandung access/refresh, session ID, waktu, serta must-change-password. Refresh selalu ditandai `json:"-"` jika berada pada type yang bisa diserialisasikan; handler tetap membangun DTO aman secara eksplisit. Response Login/Refresh berisi tepat `token`, `token_type:"Bearer"`, `session_id`, `access_expires_at`, `session_expires_at`, dan boolean `must_change_password` termasuk false. Timestamp valid dari protobuf dipetakan RFC3339 UTC dengan presisi yang tersedia. Nonempty credentials/session ID dan valid future expiry dengan access expiry tidak melewati session expiry wajib sebelum mengeluarkan success/cookie. Jangan decode opaque token atau access JWT untuk mengisi field hilang.

URL, HTTP 200, `SS`, `Register Successfully`, dan `Login Successfully` tetap; pesan baru Refresh `Session refreshed successfully.`, Logout `Logout successfully.`, dan CSRF `CSRF token issued successfully.` memakai writer bersama. Logout data tepat `{"changed":<backend bool>}`; bootstrap tepat `{"csrf_token":"<signed token>"}`. Seluruh response Register/Login/session endpoint, termasuk error, menggunakan no-store; password tetap no-store.

Alternatif mempertahankan alias `Token` sekaligus `token` ditolak karena membuat dua kontrak credential browser. Ini cutover yang perlu koordinasi FE, bukan perubahan casing diam-diam. Request Register/Login dan seluruh rule validasi tetap.

### 2. Session RPC dipisahkan dari Me dan password RPC

Tambahkan consumer-owned `RefreshGateway` dan `LogoutGateway`, usecase kecil yang memvalidasi input internal, serta gateway authservice dengan direct mapping. Shared client menambahkan interface SDK dua operasi dan wrapper pada `session_client.go`; signature interface Service dan PasswordService existing tidak diperlebar. Semua constructor existing tetap valid; tambahkan satu constructor explicit untuk menginjeksi SDK session bersama SDK existing untuk fake-service tests. Default constructor merakit generated AuthService pada connection yang sama; jangan buat connection baru.

`SessionHandler` di auth/handler menangani `/auth/refresh`, `/auth/logout`, `/auth/csrf`. Handler auth existing tetap memiliki Register/Login/Me/password dan menerima browser policy yang sama untuk penerbitan/cleanup. Browser policy berupa helper feature-owned di handler, dengan injected clock/entropy untuk test; tidak membuat framework/generic middleware security atau package global baru. Domain bebas Fiber/protobuf/concrete client; cookie dan CSRF tidak menjadi rule bisnis usecase.

Tambahkan public `/auth` group melalui composition auth, tanpa middleware Me. Public legacy `/users/create` dan `/users/login` tetap didaftarkan sebelum protected admin `/users`. Satu authenticated `/me` group serta ownership profile users tetap.

### 3. Pilih credential sebelum membentuk request RPC

| Operasi | Pilihan user credential | Jumlah RPC valid |
| --- | --- | --- |
| Login/Register | Email/password JSON; hapus authorization outgoing yang kebetulan ada | Satu Login/Register; zero Me |
| Refresh | Tepat satu nonempty refresh cookie; abaikan bearer yang masuk | Satu RefreshSession; zero Me |
| Logout dengan header Authorization hadir | Header harus satu `Bearer <nonempty-token>`; refresh field protobuf nil | Satu Logout; zero Me |
| Logout tanpa header Authorization | Tepat satu nonempty refresh cookie; optional protobuf field present | Satu Logout; zero Me |
| CSRF bootstrap | Cookie refresh hanya untuk binding CSRF lokal | Zero RPC |

Header Authorization hadir tetapi kosong/malformed/berulang tetap memilih jalur penolakan bearer HTTP 401, tidak pernah fallback cookie. Refresh mengabaikan Authorization untuk pemilihan credential dan menghapus seluruh outgoing authorization. Cookie-mode Logout juga menghapus authorization; bearer Logout mengganti authorization outgoing dengan tepat satu nilai terpilih. Salin metadata, jangan mutasi context caller; pertahankan locale/request metadata, parent cancellation/deadline, dan guard server-owned service token. Payload/body/query tidak bisa mengganti credential, session ID, atau authority; endpoints baru menerima empty body dan tidak memakai body/query sebagai sumber credential. Ambiguous duplicate refresh/CSRF cookies ditolak HTTP 401/403 sesuai credential yang ambigu, bukan memilih salah satunya.

Alternatif memanggil Me dahulu ditolak: access token expired harus tetap dapat refresh, dan Logout proof memiliki semantik terminal ownership sendiri. Restriction/reuse/expiry diperiksa backend; tidak menambah lookup, retry atau fallback.

### 4. Browser policy dengan signing secret khusus dan Origin exact

Tambahkan `AUTH_BROWSER_ALLOWED_ORIGINS` (comma-separated exact HTTPS origins) dan `AUTH_BROWSER_CSRF_SECRET` (canonical unpadded base64url, tepat 32 decoded bytes) sebagai konfigurasi wajib. Secret CSRF harus berbeda dari encoded auth service credential. Tidak ada default, bypass dev, reload otomatis, atau secret logging. Config.Validate juga berlaku untuk construction langsung, sebelum resource dibuka. Origin entries berupa scheme/host/optional port tanpa userinfo/path/query/fragment, wildcard, opaque `null`, atau HTTP. Daftar kosong/entry invalid gagal startup; value aktual dipilih operator di luar proposal. `.env.example` hanya placeholder, bukan secret siap pakai.

Login memerlukan satu Origin pada allowlist dan media type JSON (`application/json` dengan parameter valid diperbolehkan); tidak memerlukan CSRF sebelumnya. Origin hilang/invalid/ditolak menghasilkan safe 403; media type unsupported menghasilkan 415. Refresh, cookie Logout, dan CSRF bootstrap juga memerlukan exact Origin allowlist. Bearer Logout tidak memerlukan CSRF atau Origin untuk caller non-browser; bila browser mengirim Origin yang ditolak CORS, request ditolak. Register tidak mendapat kewajiban Origin baru, tetapi tetap mengikuti CORS jika Origin dikirim.

CORS pada `/api/v1` memakai daftar yang sama, credentials=true, Vary Origin serta header preflight yang relevan; tidak merefleksikan origin arbitrer/wildcard. Izinkan metode GET/POST/PATCH/PUT/DELETE/OPTIONS dan header Authorization, Content-Type, X-CSRF-Token, Version, Idempotency-Key, serta locale/request-ID yang memang dipakai. Preflight yang valid selesai tanpa RPC/Me/cookie dan tidak memerlukan credential atau CSRF. Cross-site deployment/SameSite=None serta proxy-derived dynamic origin tidak ditambahkan.

Signed double-submit memakai HMAC-SHA256 dari standard library dan nonce crypto/rand 32 byte. Format versi tetap `v1.<base64url nonce>.<base64url mac>`, panjang komponen tervalidasi. MAC mengikat version, nonce, dan SHA-256 digest refresh melalui encoding length-prefixed; raw refresh maupun digest tidak dimuat dalam token browser. Verifikasi menggunakan constant-time compare untuk cookie/header dan MAC. HMAC secret sama antar-instance, bukan service token. Test memiliki entropy/clock deterministik; kegagalan entropy gagal aman tanpa cookie parsial atau RPC lanjutan.

Alternatif plain double-submit ditolak karena tidak mengikat credential dan tidak melindungi injection. In-memory CSRF/session registry ditolak karena menambah state lokal dan masalah multi-instance. Bootstrap JSON memungkinkan FE beda origin tetapi satu site membaca token, meski cookie berada di host API.

### 5. Cookie lifecycle atomik pada response sukses

Nama refresh tetap `__Secure-refresh`; cookie CSRF baru `__Secure-csrf`. Keduanya host-only, Secure, SameSite=Lax, path `/api/v1/auth`; hanya refresh HttpOnly. Prefix `/api/v1` sekarang literal pada startup, jadi path diturunkan dari prefix composition yang sama tanpa menambah konfigurasi prefix baru. ChangePassword memakai policy path yang sama walaupun tidak menerima cookie pada `/me/password`.

Login/Refresh menyiapkan seluruh validasi result, DTO, signed CSRF, dan cookie sebelum memasang salah satunya. Untuk mutasi, siapkan nonce sebelum RPC agar entropy failure tidak terjadi setelah backend merotasi sesi; MAC dibuat memakai refresh hasil RPC. Expires memakai session expiry authoritative yang dibulatkan turun ke detik, Max-Age positif dibulatkan turun dari remaining lifetime; jangan memakai access TTL atau memperpanjang session expiry. Jika tidak ada satu detik lifetime yang bisa direpresentasikan, fail safe tanpa cookie. CSRF cookie mengikuti batas yang sama.

Bootstrap hanya membutuhkan refresh cookie nonempty dan Origin yang valid: tidak memverifikasi sesi, decode token, atau membuat klaim authenticated. Ia menghasilkan token CSRF baru, cookie CSRF session-only (tanpa Expires/Max-Age karena expiry backend tidak diketahui), dan JSON csrf_token; refresh cookie tidak berubah dan tidak diperpanjang. Token lama menjadi tidak cocok setelah refresh rotasi; backend tetap menentukan keabsahan refresh. Missing refresh menghasilkan 401; bad CSRF/Origin menghasilkan 403; ambiguous credentials gagal sebelum RPC.

Logout sukses termasuk changed=false menghapus kedua cookie dengan empty value, expiry lama/Max-Age removal dan atribut penerbitan yang sama. Password sukses login_required=true melakukan cleanup yang sama tanpa membaca refresh atau mewajibkan CSRF pada bearer password flow. LoginRequired=false dan semua error tidak menghapus/mengganti cookie; refresh error juga tidak dianggap logout. FE membuang token memory setelah konfirmasi/logout/login_required dan melakukan login ulang jika refresh gagal; REST tidak dapat menghapus memory FE.

### 6. Error dan verification boundary

Tambahkan hanya error config/origin/CSRF/result guards yang diperlukan ke catalog; mapping HTTP tetap di internal/response. Missing credential/malformed bearer 401, Origin/CSRF 403, unsupported Login content type 415, malformed JSON 400, invalid backend result/internal entropy failure 500. Error backend termasuk 409/429/503, field errors, Retry-After, timeout 504/cancellation 499 tetap mengikuti main spec http-error-mapping tanpa credential values. No-store dipasang sebelum CORS/Origin/handler checks pada path lifecycle agar error awal juga aman.

Unit/scenario tests mengikuti runner root test. HTTP integration memakai shared fixture composition dengan config/browser policy explicit; TLS bufconn memperluas fake AuthService existing, bukan mendaftarkan dua server pada nama service yang sama. Assertions menguji payload/optional presence, authorization absences, full methods, satu x-service-token, isolation concurrent, cookies dan no-store termasuk error. Backend reuse/revocation diuji sebagai controlled outcomes; tidak mengklaim telah memverifikasi transaksi real backend.

## Risks / Trade-offs

- [FE masih membaca Token atau otomatis login setelah Register] → tandai breaking change, perbarui golden/docs, koordinasikan cutover dan verifikasi browser sebelum rollout.
- [Strict refresh rotation sudah commit tetapi response hilang] → jangan retry; dokumentasikan single-flight FE termasuk antar-tab dan login ulang jika hasil ambigu. Tidak ada local replay cache.
- [Secret CSRF berbeda antar-instance] → satu shared operator-provided secret; test policy lintas instance dengan config sama. Pergantian secret menginvalidasi CSRF lama, bootstrap ulang tanpa mengganti refresh.
- [Bootstrap mint CSRF untuk token invalid] → bootstrap bukan bukti autentikasi; hanya backend Refresh/Logout memverifikasi sesi. Tidak memperpanjang refresh expiry.
- [HTTPS same-site cookie tidak terkirim di dev HTTP/cross-site] → dokumentasikan topology yang didukung; test HTTPS dan jangan melemahkan Secure/SameSite secara otomatis.
- [Login/Refresh backend response rusak atau terpotong] → fail safe tanpa cookie parsial, tanpa synthetic expiry/second call. Backend mutation mungkin sudah terjadi; caller dapat perlu login ulang.

## Migration Plan

1. Implementasikan serta verifikasi change pada pinned SDK; perbarui README dan golden khusus Register/Login serta password CSRF cleanup. Main specs disinkronkan melalui workflow terpisah setelah implementasi, bukan diedit langsung saat proposal.
2. Sebelum rollout yang kelak diotorisasi, operator menyiapkan origin HTTPS satu site, secret CSRF shared terpisah, dan grant `/auth.v1.AuthService/RefreshSession` serta `/auth.v1.AuthService/Logout` pada namespace IAM; FE menyiapkan response lowercase, cookies, bootstrap dan single-flight. Tidak ada data migration.
3. Rollout REST/FE secara terkoordinasi. Smoke test Login → bootstrap → Refresh → bearer/cookie Logout serta restricted Login → password cleanup. Workflow proposal/apply tidak menjalankan deployment.
4. Rollback membutuhkan pasangan REST/FE yang kompatibel. Mengembalikan REST lama tidak mencabut sesi backend atau menghapus cookie yang sudah terbit; jangan mengklaim rollback sebagai revocation. Jika perlu penghentian sesi, gunakan prosedur backend/operator yang diotorisasi terpisah.
