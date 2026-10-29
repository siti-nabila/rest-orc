# Proposal

## Why

Login saat ini membuang refresh credential dan metadata sesi yang sudah tersedia pada SDK, sehingga browser belum dapat memperbarui access token atau mengakhiri sesi melalui REST. Lengkapi lifecycle browser setelah Me, update profil, dan change password dengan tetap menyerahkan validasi serta revocation sesi kepada grpc-auth.

## What Changes

- Tambahkan `POST /api/v1/auth/refresh`, `POST /api/v1/auth/logout`, dan `GET /api/v1/auth/csrf`, tanpa middleware Me. Refresh memakai cookie; logout memilih tepat satu mode bearer atau cookie tanpa fallback.
- **BREAKING:** ubah response Login dari `data.Token` menjadi `data.token`, `token_type`, `session_id`, `access_expires_at`, `session_expires_at`, dan `must_change_password`. Pertahankan URL, HTTP 200, success code/message, dan validasi credential existing. Refresh token hanya dikirim melalui cookie host-only `__Secure-refresh`, bukan JSON.
- **BREAKING:** selaraskan response Register dengan hasil backend PENDING: `token:""`, `registration_id`, dan `registration_status:"PENDING"`, tanpa sesi atau cookie baru. Ini menggantikan golden fixture legacy yang masih mengasumsikan token Register.
- Pasang cookie refresh `HttpOnly; Secure; SameSite=Lax; Path=/api/v1/auth`, signed double-submit CSRF yang terikat pada refresh credential, Origin allowlist, dan credentialed CORS untuk origin FE eksplisit. Tambahkan konfigurasi environment wajib untuk origin serta secret CSRF terpisah, tanpa default secret.
- Rotasikan refresh/CSRF hanya pada hasil refresh sukses; hapus keduanya pada logout sukses dan change-password sukses dengan `login_required=true`. Terapkan `Cache-Control: no-store` pada seluruh response lifecycle, termasuk error.
- Reuse auth connection, SDK published yang sudah dipin, service credential, locale, deadline/cancellation, dan shared error mapper; jangan melakukan retry rotasi atau menyimpan session/JWT secara lokal.

## Capabilities

### New Capabilities

- `auth-session-lifecycle`: penerbitan browser session transport pada Login, hasil Register PENDING, refresh tanpa access JWT, logout satu mode credential, bootstrap CSRF, cookie/Origin/CORS policy, konfigurasi, dan isolasi RPC.

### Modified Capabilities

- `auth-input-validation`: pertahankan seluruh aturan input serta public routes, tetapi gantikan kewajiban `data.Token`/tanpa cookie dengan response Register/Login lifecycle yang eksplisit.
- `auth-service-credentials`: pertahankan keamanan service caller dan context; jelaskan pemilihan metadata authorization pada RPC lifecycle serta pengecualian response Register/Login yang kini berubah.
- `my-password-change`: perluas cleanup sukses bersyarat ke cookie CSRF bersama refresh, dan selaraskan regression requirement dengan kontrak lifecycle yang baru.
- `me-profile`: pertahankan seluruh behavior Me dan isolasi snapshot, sambil memperbarui klausul compatibility Register/Login agar tidak bertentangan dengan migrasi lifecycle.

## Impact

Perubahan implementasi nanti mencakup `internal/features/auth/{domain,usecase,gateway/authservice,handler}`, shared SDK wrapper `internal/client/authservice`, konfigurasi environment, `internal/app`, dan catalog error lokal yang diperlukan. Seluruh test tetap di root `test/`, termasuk golden Register/Login, fake SDK HTTP integration, dan TLS bufconn integration.

FE harus mengadopsi field lowercase, menunggu aktivasi setelah Register, menggunakan `credentials: include`, mengambil CSRF saat dibutuhkan, dan mengoordinasikan satu refresh termasuk antar-tab. Operator menyediakan secret CSRF yang sama antar-instance, HTTPS dengan FE/API satu site, serta backend method/namespace grants. Tidak ada upgrade dependency, edit sibling/backend/protobuf/generated graph, database, implementasi FE, admin/session-management endpoints, perubahan permission gate users, atau deployment dalam change ini.
