# Design

## Context

Motivasi dan scope ada di `proposal.md`; acceptance contract ada di `specs/auth-service-credentials/spec.md`. Inspeksi read-only dilakukan pada 28 Oktober 2026.

### Kondisi rest-orc yang teramati

- Module adalah `github.com/siti-nabila/rest-orc`; pin api-contracts terpublikasi sudah dipakai. Baseline targeted tests config/grpcclient/authservice/app/integration users lulus dengan `GOWORK=off -mod=readonly`.
- Graph `graphify-out/graph.json` berasal dari commit `d2257fc2baa509f00ae462606c0992c58045e4fe`; dipakai sebagai routing awal lalu source diverifikasi karena working tree berubah. Hub yang relevan: konfigurasi, error infrastructure, shared transport, composition root, dan runner/scenarios existing.
- `config.Load()` mengisi typed config dari environment dan memanggil `Config.Validate()`. `app.New()` memvalidasi lagi sebelum membuat connection. `GRPCClientConfig` belum memiliki service credential; `.env.example` masih mencontohkan TLS disabled dan di-ignore oleh `.gitignore`; template nonsecret perlu dapat dilacak agar migration config ikut terdistribusi.
- `grpcclient.New(cfg)` membuat satu lazy connection; `RequestContext(parent)` memakai `context.WithTimeout`. `app.New()` menutup connection jika wiring berikutnya gagal; `App.Close()` menutupnya saat shutdown.
- `tlsconfig.New()` shared dengan HTTP dan menghasilkan minimum TLS 1.2, verifikasi default aktif, optional ServerName, dan system roots yang dapat ditambah CA PEM. Jangan menaikkan minimum seluruh client HTTP sebagai efek samping change auth.
- `authservice.Client` membungkus generated client untuk Me/ListUsers/Register/Login. Authenticator menambahkan bearer pada outgoing metadata; locale middleware/client flow mempertahankan metadata bahasa. Tidak ada service-token injection saat ini.
- Test config/app/grpcclient memakai fixture yang umumnya belum mengaktifkan TLS atau menyediakan token. Test fake `NewWithService` tidak melewati transport gRPC, sehingga tidak cukup untuk membuktikan call credentials benar-benar terkirim.

### Contract backend yang diverifikasi

- `grpc-auth/docs/service-authentication.md`, `internal/serviceauth/verifier.go`, dan `internal/interceptors/service_interceptor.go`: seluruh registered RPC wajib TLS dan tepat satu `x-service-token`; canonical raw base64url berukuran encoded 43–342, decoded 32–256 byte. Strict decode dilanjutkan re-encode equality; CR/LF yang diterima decoder tidak menjadi token canonical.
- `grpc-auth/internal/configs/tls.go` dan dokumen service authentication menetapkan TLS 1.3. CA/hostname harus diverifikasi; tidak ada plaintext fallback.
- Verifier meng-hash **encoded string** dengan SHA-256 dan membandingkannya dengan current/previous digest serta batas expiry server. Client tidak dapat memvalidasi digest match atau expiry dari opaque token.
- `internal/serviceauth/caller.go` dan `pkg/config/service_auth.go`: ServiceID, exact Methods, dan Namespaces dimiliki backend. `rest-orc` adalah service ID yang dikenali untuk trusted caller; tidak ada header namespace/service-name untuk meminta authority. ListUsers membutuhkan namespace `iam` dan permission pengguna; method/service authority tidak memberikan user privilege.
- Method yang sudah dipakai consumer: `/user.v1.UserService/Register`, `/user.v1.UserService/Login`, `/user.v1.UserService/Me`, `/user.v1.UserService/ListUsers`. Operator perlu mengizinkan exact methods yang diperlukan; proposal tidak mengubah caller grants atau menambahkan wildcard.
- Pinned grpc-go `v1.83.0` menyediakan `credentials.PerRPCCredentials`, `grpc.WithPerRPCCredentials`, dan `grpc.WithContextDialer`; transport meneruskan RPC context ke credential provider. Tidak perlu dependency tambahan atau import sibling internal package.

## Goals / Non-Goals

**Goals:** Satu credential server yang immutable pada connection auth; validasi startup jelas; kebijakan keamanan tetap di boundary konfigurasi/transport; flow caller tetap mudah dibaca dan diuji melalui wire gRPC in-memory.

**Non-Goals:** Tidak menambah service-auth framework/interface generik, credential cache/provider eksternal, refresh worker, environment polling, namespace resolver, atau policy gate baru di REST. Tidak mengubah middleware bearer, payload bisnis, lifecycle browser, generated protobuf, maupun backend. Tidak membuat secret runtime atau menjalankan deployment/readiness backend.

## Decisions

### 1. Tambahkan token ke typed transport config dan wajibkan pada auth application config

Tambahkan field `ServiceToken string` pada `GRPCClientConfig`, diisi `AUTH_GRPC_SERVICE_TOKEN` oleh loader existing. Application config untuk `Clients.AuthGRPC` wajib token valid dan TLS enabled, termasuk saat dibuat langsung melalui `app.New(cfg)`. `.env.example` berubah menjadi TLS enabled dengan token kosong serta komentar provisioning; contoh memang tidak bisa start sebelum secret disuplai. Hapus hanya entry ignore `.env.example` agar template nonsecret dapat dilacak; pertahankan ignore `.env` dan perubahan user lain.

Validasi format: batasi encoded length 43–342, `base64.RawURLEncoding.Strict().DecodeString`, decoded length 32–256, dan re-encode harus sama dengan input. Jangan trim, log, memasukkan raw token ke error parameter, atau memanggil `environmentError(key, value, cause)` untuk secret. Gunakan typed `dictionary.ValidationError` dengan field `clients.auth_grpc.service_token`, reuse `ErrValueRequired`, dan tambah catalog infrastructure problem hanya untuk invalid token/TLS required bila tidak ada semantik existing yang cocok. Error cause dari decoder tidak perlu diekspos.

Bagikan validasi format/TLS token antara application config dan transport melalui method konfigurasi kecil agar aturan tidak diduplikasi. Generic transport tanpa `ServiceToken` tetap dapat dipakai consumer lain sesuai behavior existing; ini bukan bypass aplikasi auth karena `Config.Validate()` selalu mewajibkan credential auth. Generic transport yang diberi token harus menolak token malformed atau TLS disabled, termasuk pemanggilan constructor langsung.

Alternatif: config auth yang membungkus/mengganti tipe GRPCClientConfig memperluas wiring dan fixture tanpa kebutuhan. Menyimpan secret di gateway/usecase atau membaca environment per request melanggar boundary dan ditolak. Optional auth-disable flag tidak kompatibel dengan backend.

### 2. Secure call credentials dipasang pada reusable connection

Tambahkan implementasi kecil `credentials.PerRPCCredentials` di `internal/client/grpcclient/service_credentials.go`, dibuat sekali dari token tervalidasi dan dipasang dengan `grpc.WithPerRPCCredentials` ketika `ServiceToken` nonempty. `RequireTransportSecurity()` selalu true. `GetRequestMetadata` menghormati context cancellation dan mengembalikan map baru berisi tepat satu key `x-service-token`; token tidak di-hash, ditrim, atau dipindahkan ke bearer/payload.

Provider menolak context yang sudah mempunyai outgoing `x-service-token`, termasuk jika nilainya sama dengan konfigurasi. Kembalikan catalog authentication error aman yang tersedia (`auth.ErrServiceAuthenticationFailed`); transport akan mempertahankan semantik RPC authentication failure. Jangan mengubah map/context asal, menambahkan entry kedua, atau mengganti credential yang disisipkan caller. Browser header tidak pernah dicopy sebagai service metadata; integration test mengunci batas ini. Tidak perlu interceptor sanitization tambahan karena fail-closed check dapat dilakukan di provider dengan RPC context.

Alternatif manual `metadata.AppendToOutgoingContext` pada setiap wrapper RPC menduplikasi logic dan tidak menjamin secure transport. Menambahkan credential ke `authorization` mencampurkan identitas dan ditolak. Tidak menambah option per request atau fallback credential.

### 3. Terapkan TLS 1.3 pada connection dengan credential, pertahankan shared HTTP behavior

Reuse `tlsconfig.New(cfg.TLS)`, lalu set minimum menjadi `tls.VersionTLS13` pada hasil khusus gRPC service-credential path sebelum membentuk `credentials.NewTLS`. Biarkan verifikasi chain/hostname dan SystemCertPool/CA append existing. Jangan set `InsecureSkipVerify`, menghapus CA yang invalid, atau membuat fallback plaintext. `TLS.ServerName` boleh kosong agar grpc-go memakai target authority; override eksplisit tetap diverifikasi terhadap SAN.

Validation yang bisa diketahui lokal gagal sebelum `grpc.NewClient`; handshake/certificate failures baru teramati saat lazy RPC pertama. Constructor success bukan bukti server reachable atau credential diterima. Config/README harus membedakan keduanya. CA-file error existing tetap mempertahankan cause dan context tanpa credential.

Alternatif menaikkan shared TLS builder global mengubah client HTTP yang tidak terkait. mTLS client certificate tidak diperlukan oleh contract backend yang diperiksa dan tidak ditambahkan. Secret manager/automatic rotation juga tidak dibutuhkan untuk acceptance saat ini.

### 4. Pertahankan context, composition, dan public outcomes

Connection options menambahkan call credentials tanpa mengganti `RequestContext(parent)` atau metadata forwarding. Bearer hanya dari flow existing, locale/request ID tetap di context, earlier deadline tidak diperpanjang, cancellation tetap mengalir. Simpan token di provider immutable, kembalikan map baru tiap call, dan jangan membaca env dari request path. Concurrent requests memakai connection yang sama tanpa mutable metadata bersama.

`app.New()` tetap menjadi pemilik transport, menggunakan config tervalidasi. Tidak perlu mengubah authservice/gateway/handler bila transport-level injection sudah cukup. Pertahankan cleanup startup/shutdown dan urutan Me → ListUsers. Tidak menambah probe Me/TesRPC atau health RPC saat startup, retry wrapper, credential fallback, maupun response/error remapping.

Existing main specs sudah dibaca lengkap: `auth-input-validation` tetap menentukan local 400 dan legacy success `data.Token`; `http-error-mapping` tetap menentukan downstream 401/403/409 serta safe descriptions/RetryInfo. Change menambah satu capability baru, bukan mengubah kedua spec itu. Public Register/Login tetap tanpa bearer pengguna walaupun transport service authentication wajib.

### 5. Verifikasi secure credential melalui production constructor dan bufconn

Gunakan test runner/scenarios root existing. Tambahkan TLS integration di `test/integration/authservice/` dengan generated UserService server fake dan `grpc/test/bufconn`, sehingga path `grpcclient.New` → `authservice.New` → generated RPC → server melihat metadata di wire. Test tidak mengimpor sibling module atau verifier internal backend; fake memeriksa contract minimum, bukan menyalin sistem permission backend.

Untuk memasang in-memory dialer tanpa network TCP, perluas constructor transport secara source-compatible menjadi `New(cfg, options ...grpc.DialOption)`. Existing callers tetap `New(cfg)`. Optional options hanya dipakai trusted composition/test untuk `grpc.WithContextDialer`; config-owned TLS dan credential options dipasang sesudah options tambahan. Application production tidak menerima dial options dari input HTTP/env. Tidak menambah test-only bypass TLS, global resolver/transport state, constructor family, atau mengekspos connection internals baru. Test memakai options dialer saja dan tetap menjalankan security options production.

Gunakan fixed test-only PEM CA/server/key dengan SAN dan validity window terkontrol, disimpan di fixtures root test; bukan credential/cert deployment. Untuk TLS validity/deadline/timeout, gunakan fake clock dari standard library `testing/synctest` yang sudah tersedia pada Go 1.26.2: bubble dimulai pada 2000-01-01 UTC. Buat server, client, bufconn, dan goroutine di bubble yang sama lalu tutup semuanya sebelum bubble berakhir; fixture valid/expired ditentukan relatif terhadap waktu virtual tetap itu. Verifikasi certificate unit dapat memakai fixed `CurrentTime`. Tidak perlu menambah clock hook test-only pada production API. Credential fixture dibentuk deterministik dari byte test, tidak mengklaim randomness provisioning. Timeout hanya safety bound dengan waktu virtual; koordinasikan concurrency lewat channels/barrier, bukan wall-clock sleep. Tidak membuka TCP listener, Docker, atau backend nyata.

Coverage:

- Config: missing/empty/whitespace, decoded 31/32/256/257, padding/invalid alphabet/noncanonical tail bits/CRLF, TLS false, safe typed errors, direct config validation, dan environment isolation.
- Credential/transport: keempat methods mendapat satu token; public tanpa bearer; protected bearer/locale/request-ID dipertahankan; conflicting outgoing key gagal sebelum handler; map/context nonmutation; generic no-token path tetap bekerja; constructor guard dan cancellation.
- TLS wire: trusted TLS1.3 success, mismatch SAN/untrusted chain/expired certificate, malformed/unreadable CA, TLS1.2-only/plaintext rejection, tanpa credential leakage ke handler. Baseline HTTP TLS tidak berubah.
- Context/concurrency: earlier parent deadline, already canceled/expired context, simultaneous bearer/locale berbeda tanpa leakage, satu connection, original token tetap dipakai setelah environment berubah dan hanya new client memakai token baru. Gunakan channels/barrier, bukan sleep.
- HTTP regression: browser `x-service-token`/identity claims tidak diteruskan; public success/local validation; protected ordering; backend auth/forbidden/409 errors tetap safe tanpa retry. Fake tests existing dipertahankan, tetapi tidak dianggap bukti wire credentials.

## Risks / Trade-offs

- [Startup config lama gagal] → Dokumentasikan kebutuhan `AUTH_GRPC_SERVICE_TOKEN` dan TLS true sebagai migration konfigurasi; tidak beri insecure compatibility mode.
- [Format valid dianggap token terdaftar] → Jelaskan backend tetap memeriksa digest, expiry, exact method/namespace; readiness aktual masih di luar scope.
- [Secret bocor lewat helper parse/error lama] → Gunakan static catalog problem, jangan raw-value parser untuk token; uji sentinel tidak muncul pada error/HTTP/diagnostic yang ditambahkan.
- [Token ganda karena metadata context] → Fail closed pada provider sebelum dispatch; uji original metadata tidak berubah dan handler zero calls.
- [TLS13 berdampak transport lain] → Scope ke connection dengan service credential; test generic transport dan HTTP suite existing.
- [Fake service melewati call credentials] → Tambah bufconn TLS yang memanggil production transport dan generated RPC; jangan hanya mengandalkan `NewWithService`.
- [Concurrent metadata berubah] → Immutable token + map per call; concurrency tests dan targeted race detector.
- [Dial options menambah surface internal] → Pertahankan satu constructor kompatibel; hanya trusted callers, production composition tanpa extra options; tidak menyuplai custom credential/insecure option dari browser atau environment.

## Migration Plan

1. Pada implementasi, perbarui `.env.example`/README tanpa secret nyata dan sesuaikan auth application fixtures dengan token test valid serta TLS enabled. Generic transport fixtures tanpa token tetap dapat menguji generic behavior.
2. Operator menyiapkan token acak 32–256 byte, canonical encoding, digest encoded string, expiry dan grants backend untuk caller `rest-orc`, serta trust CA/hostname TLS. Tidak membaca secret deployment atau mengubah konfigurasi sibling dalam change ini.
3. Setelah implementasi diverifikasi, deployment terpisah menginjeksi token dan TLS settings. Rotation: backend menyiapkan current/previous overlap secara eksplisit, operator mengganti environment dan me-restart/recreate client; tidak ada live reload/fallback token.
4. Jika rollout tidak kompatibel, hentikan rollout atau gunakan version set yang tetap mendukung service authentication; jangan menonaktifkan verifikasi TLS/service credential. Tidak ada migrasi data, dependency update, deploy, atau rotasi runtime yang dijalankan oleh change ini.
