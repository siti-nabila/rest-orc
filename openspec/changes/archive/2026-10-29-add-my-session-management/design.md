# Design

## Context

Lihat `proposal.md` untuk motivasi dan scope. Source aktual menunjukkan:

- Auth sudah memiliki `/me`, `/me/password`, dan public `/auth/{refresh,logout,csrf}`. `registerFeatureRoutes` membuat satu authenticated `/me` group yang dibagi auth dan users profile.
- `Principal` memuat ID dan MustChangePassword tetapi tidak session ID. User principal berasal dari satu Me; tidak ada alasan menambah session ID ke Me atau decode JWT untuk feature ini.
- `internal/client/authservice/session_client.go` membungkus **AuthService** RefreshSession/Logout; interface SessionService di file tersebut bukan generated **session.v1.SessionService**. Kedua service berbeda meskipun namanya mirip.
- Published pin `api-contracts v0.0.0-20261023084653-d32fd5c1bab4` sudah memuat tiga RPC self-session. ListSessionsResponse memiliki Items, PageMeta dengan total int64, dan NextCursor. `pkg/pagination.Page.Total` justru int32.
- Backend read-only `internal/features/session/{list,list_cursor,self,global}.go`, handler list/self, dan domain session_lifecycle mengonfirmasi owner dari verified principal, nonrestricted requirement, canonical lowercase UUID, default limit 20, cursor maksimum 2048 byte, default created_at DESC, dan authoritative single/all revocation.
- Backend RevokeMySession mengembalikan hanya Changed. RevokeAllMySessions mengembalikan Changed/LoginRequired dan normal success saat ini true/true. Pemeriksaan sesi aktif terjadi lagi saat RPC; Me yang sukses sebelumnya tidak menjamin mutation masih eligible.
- BrowserPolicy.Clear sudah menghapus refresh/CSRF dengan atribut yang benar tanpa perlu incoming cookie. LifecycleNoStore saat ini belum mencakup URL self-session.
- Graphify revision d2257fc2baa509f00ae462606c0992c58045e4fe mencakup routing auth/pagination sebelum perubahan working tree terbaru; digunakan sebagai routing awal saja. Seluruh temuan diverifikasi dari source aktual.

## Goals / Non-Goals

**Goals:** alur auth handler → usecase → feature gateway → shared SDK/transport yang mudah ditelusuri, safe owned session output, satu Me lalu satu RPC, serta reuse browser cleanup tanpa menambah session state.

**Non-Goals:** admin/session user lain, flexible session sorting/search/filter API, local revocation transaction, JWT parsing/session registry, perubahan generic pagination atau Principal/Me contract, dependency/env baru, maupun pembangunan FE. Keputusan transaksi/audit/outbox tetap backend.

## Decisions

### 1. Auth memiliki self-session tanpa mengubah handler lifecycle existing

Tambahkan DTO/query, `SessionActor{UserID, MustChangePassword}`, dan tiga consumer-owned domain contracts di auth/domain. Usecase ListMySessions/RevokeMySession/RevokeAllMySessions menerima actor terverifikasi serta input yang diperlukan; guard nil context/zero ID lalu restricted sebelum rule domain dan gateway call. Return 401 untuk missing identity, 403 untuk restricted, tanpa admin/permission gate baru. Transport parsing dapat menghasilkan 400 sebelum usecase bila syntax query tidak valid; tidak ada RPC bisnis pada kasus itu.

Tambahkan `MySessionsHandler` di auth/handler langsung dengan constructor sendiri, tiga small usecase interfaces, writer dan existing BrowserPolicy. Daftarkan pada `meRoutes.Group("/sessions")` yang sudah authenticated, bukan group dengan middleware Me baru. Composition khusus boleh berada di `internal/app/my_session_routes.go`; tidak membuat generic registration framework atau memperpanjang dependency handler Register/Login.

Alternatif feature users ditolak karena operasi ini mengatur sesi credential diri sendiri. Reuse SessionHandler public lewat banyak optional dependencies ditolak karena mencampur credential-cookie flow dengan bearer-protected resource.

### 2. Bedakan AuthService lifecycle dari SessionService management

Tambahkan interface `SelfSessionService` tiga generated-signature RPC dan field `selfSessionService` pada shared auth client, dengan wrapper `self_session_client.go`. Generated sessionv1 SDK memakai transport auth existing. Jangan memperlebar interface SessionService Refresh/Logout, Service UserService, ProfileService, atau PasswordService.

Pertahankan constructor New/NewWithService/NewWithServices/NewWithPasswordService/NewWithSessionService serta guard ordering. Tambahkan satu explicit NewWithSelfSessionService injection point yang menerima dependency existing plus self-session SDK; constructor lama menyediakan default generated self-session SDK setelah nil guards. Semua wrapper menjaga RequestContext timeout/cancel/nil response semantics. Gateway self-session memiliki mapping terfokus; tidak ada SDK/protobuf pada domain/HTTP output.

Alternatif rename SessionService existing atau menggabungkan dua generated services dalam satu interface ditolak karena memperluas dampak dan mock operations yang tidak dibutuhkan.

### 3. Pagination kecil dengan default dan urutan server tetap

Kontrak awal menerima hanya kontrol pagination page, limit, last_id. Omitted page/limit menjadi 1/20. Nilai hadir harus nonempty digit ASCII yang bisa diparse tanpa overflow: page 1–2147483647, limit 1–100. Duplicate known pagination keys dan empty/noninteger/overflow values ditolak 400. Last_id omitted/empty berarti tanpa cursor; selain itu maksimal 2048 byte, diteruskan sesudah satu URL decoding tanpa trim, parse base64, atau perubahan isi. Duplicate last_id ditolak.

Kontrol list yang tidak ditawarkan (`sort`, `fields`, `search`, `keyword`, `statuses`, `application_code`, `created_from`, `created_to`) ditolak 400 jika hadir, agar tidak seolah didukung. Claim user_id/session_id/namespace/service identity dan unknown non-control input diabaikan, bukan forwarded. Body tidak dibaca untuk ketiga operasi; Version/Idempotency-Key/CSRF tidak diwajibkan atau dibentuk menjadi precondition.

Gateway membentuk PageQuery dengan page/limit/last_id, explicit sort created_at DESC lalu id DESC, Fields kosong dan Search nil. Backend mendukung sort lain, tetapi tidak diekspos pada tahap ini. Ini membatasi scope sesuai kebutuhan pagination awal; tetap stabil untuk cursor yang terikat owner/order/limit. Page merupakan metadata halaman, bukan janji offset scan; cursor tetap mengatur traversal backend. Client memakai next_cursor dan meresetnya jika limit berubah.

### 4. DTO halaman sesi mempertahankan metadata backend

Gunakan auth-owned `MySessionsPage` dengan Items nonnil, Total int64, Page/Limit/TotalPages int32, HasNext/HasPrev bool, dan NextCursor string; JSON keys mengikuti envelope pagination existing. Jangan memakai `pagination.Page` dengan narrowing cast, mengubah generic helper, menghitung ulang paginator, atau mengosongkan total hanya karena page tidak memiliki item.

Setiap item hanya berisi `id`, `user_id`, `status`, `created_at`, `last_seen_at`, `expires_at`, `revoked_at`, `device_label`, `user_agent`, `ip_address`, `is_current`. ID sesi string UUID, user_id JSON number; is_current termasuk false. Status ACTIVE/REVOKED dipertahankan, expiry merupakan timestamp terpisah dan bukan status EXPIRED buatan. Device/IP hanya metadata tampilan, bukan bukti identity/authority.

Gateway memeriksa nonnil response/meta/items, page positif, limit 1–100, total/total_pages nonnegative, canonical item UUID, positive item owner, known status dan protobuf timestamp validity. CreatedAt/ExpiresAt wajib; LastSeenAt/RevokedAt nil menjadi JSON null, valid time menjadi RFC3339 UTC dengan fractional precision. Historical expired/revoked sessions tetap boleh; jangan reuse future-expiry rule penerbitan cookie. Empty text tetap string kosong. Usecase memeriksa setiap owner sama dengan actor.UserID sebelum menampilkan hasil; mismatch gagal 500 tanpa partial data. Collection dibuat sendiri agar dependency object dan concurrent requests tidak berbagi mutable slice.

### 5. Revoke satu sesi tidak mengklaim sesi saat ini

Domain memvalidasi session_id persis dengan regex backend `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`. Tidak trim/case-fold, membatasi versi UUID, atau menambah UUID dependency; backend format saat ini juga tidak mengecualikan UUID all-zero. Nilai yang lolos dikirim persis dalam RevokeMySessionRequest; owner tidak dikirim sebagai target browser.

Response HTTP 200/SS, message `Session revoked successfully.`, data tepat changed boolean. Target absent, owner mismatch, status sesi berubah, atau revocation failure mengikuti error backend. Changed=false dipertahankan tanpa no-op lokal. Tidak ada list/get/Me kedua untuk memastikan target atau mengetahui current session; tidak ada login_required, cookie cleanup, atau token issuance pada single revoke, termasuk ketika target sebenarnya current session. FE menggunakan is_current dari list yang sudah dibaca, lalu menghapus access token memory bila relevan; subsequent protected request tetap diverifikasi backend.

### 6. Revoke semua sesi memakai hasil backend dan BrowserPolicy.Clear

RPC RevokeAllMySessions menerima emptypb.Empty dengan bearer; HTTP request tanpa body diterima. Response HTTP 200/SS, message `All sessions revoked successfully.`, data tepat changed/login_required termasuk false. Tidak memaksa kedua flag menjadi true atau menciptakan version/token.

Hanya success login_required=true yang memanggil BrowserPolicy.Clear; tidak membaca refresh cookie atau memerlukan CSRF untuk bearer resource ini. False flag/error/nil response tidak mengubah cookie. Cleanup tetap bekerja pada /me/sessions/revoke-all yang normalnya tidak menerima cookie Path=/api/v1/auth. Ini konsisten dengan password cleanup existing; backend saat ini normalnya true/true, tetapi REST tidak mengubah fake/future authoritative flags.

### 7. No-store dan keamanan transport tetap menyeluruh

Perluas matcher LifecycleNoStore untuk list, single revoke path, dan revoke-all sebelum CORS/auth; handlers juga memasang no-store untuk standalone tests. CORS origin/credentials/header policy tetap; protected methods hanya memakai bearer terverifikasi, tanpa cookie fallback atau signed-CSRF requirement baru. Missing bearer zero RPC; Me failure menghentikan pipeline; valid unrestricted satu Me → satu full method session.v1.SessionService. Grant service method/IAM namespace tetap operator-managed.

Reuse original bearer/locale/request metadata, one env-owned service token, verified TLS1.3, earlier deadline/cancellation, dan shared mapper. Error lokal 400 dengan catalog field messages aman; backend status/code/field errors/Retry-After tetap authoritative. Nil dependency result/invalid safe response 500; tidak ada retry, reconnect per request, identity fallback atau cleanup pada error.

List success memakai message `Sessions retrieved successfully.`. Test fixture HTTP shared harus merakit MySessionsHandler pada meRoutes yang sama. TLS bufconn menambahkan satu fake sessionv1.SessionService server, terpisah dari fake authv1.AuthService existing; tidak mengimplementasikan ulang transactions backend.

## Risks / Trade-offs

- [Single revoke current session tidak memberi flag baru] → FE memakai is_current yang sudah dibaca dan backend menolak request selanjutnya bila sesi dicabut; tidak ada lookup atau cookie cleanup rekaan.
- [Me dan mutation melihat state berbeda] → backend tetap memverifikasi owner/restriction/session saat RPC; teruskan failure tanpa retry atau success browser effects.
- [Total backend melebihi int32] → DTO feature memakai int64 dan test nilai >2147483647, tanpa memperluas kontrak ListUsers.
- [Cursor menampilkan page label yang berbeda dari jumlah offset] → teruskan metadata/cursor backend persis, tidak merekayasa offset/paginator.
- [Record historique expired disalahanggap invalid] → pisahkan mapping list timestamp dari issuance validation; uji masa lalu dan optional nil timestamps.
- [Nama SessionService tertukar] → gunakan SelfSessionService untuk SDK management, pertahankan lifecycle interface/constructor existing dan regression tests seluruh RPC.

## Migration Plan

1. Implementasikan perubahan additive, perbarui README dan tests setiap layer; tidak ada data/schema/module/env migration.
2. Sebelum rollout yang diotorisasi terpisah, operator memverifikasi tiga exact method grants serta namespace IAM; FE membaca paginator/is_current dan menangani login_required pada revoke-all.
3. Verifikasi fake SDK HTTP, TLS bufconn dan targeted race, lalu build/full tests/vet/diff/strict validation. Smoke test real backend/FE dilakukan terpisah; proposal/apply tidak melakukan deployment.
4. Rollback route/code tidak mengembalikan sesi yang sudah dicabut backend. Tidak mencoba restore token atau merekayasa login setelah rollback.
