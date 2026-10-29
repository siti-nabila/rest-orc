# Proposal

## Why

rest-orc sudah menyediakan Me dan perubahan profil, tetapi belum menyediakan jalur mengganti password bagi pengguna biasa maupun restricted user yang wajib mengganti credential. Tambahkan operasi self-service sesuai docs/grpc-auth-validation-rules.md bagian 4, 15, dan 17, dengan verifikasi credential dan pencabutan sesi tetap menjadi tanggung jawab grpc-auth.

## What Changes

- Tambahkan `POST /api/v1/me/password` pada feature auth, menggunakan satu autentikasi Me dan satu RPC `/auth.v1.AuthService/ChangePassword`; restricted user diperbolehkan tanpa gate admin/permission tambahan.
- DTO HTTP tetap di auth/domain. Validasi kedua password required dan 6–50 byte; current_password diteruskan persis tanpa pembatasan karakter baru, new_password mengikuti ASCII register dan harus berbeda secara persis. Agregasikan pesan locale lewat pkg/dictionary dan shared response mapper.
- Gunakan AuthService SDK dari published module yang sudah dipin pada connection auth existing; pertahankan bearer, satu service credential, locale, metadata, deadline, dan cancellation tanpa retry.
- Petakan `changed` dan `login_required` dari backend ke response typed, tanpa replacement token/version/identity buatan. Response sukses memakai Cache-Control no-store; ketika backend meminta login ulang, expire cookie refresh sesuai policy dokumen. FE menghapus access token memory dan meminta login ulang.
- Tambahkan unit, HTTP integration, dan TLS bufconn tests beserta dokumentasi, dengan verifikasi standalone GOWORK=off.

## Capabilities

### New Capabilities

- `my-password-change`: perubahan password pengguna terverifikasi termasuk restricted user, validasi credential lokal, hasil dan invalidasi browser berdasarkan backend, serta contract transport/error dan compatibility endpoint existing.

### Modified Capabilities

Tidak ada. auth-input-validation tetap mencakup Register/Login existing; me-profile, my-profile-update, auth-service-credentials, dan http-error-mapping tetap berlaku tanpa perubahan requirement.

## Impact

- Production: auth/domain, auth/usecase, auth/gateway/authservice, auth/handler langsung tanpa subfolder http; internal/client/authservice; wiring internal/app/auth_routes.go; definisi error di pkg/dictionary.
- Test: root test/features/auth, test/client/authservice, test/dictionary, test/app, test/integration/auth, test/integration/authservice, serta concrete shared app fixture dan mock yang terdampak constructor.
- Dokumentasi: README menjelaskan route, validasi, flags, cookie expiration, login ulang, serta grant method dan namespace iam yang diperiksa backend.
- Tidak memerlukan dependency upgrade atau environment baru. Backend dipelajari read-only: AuthService menerima dua password tanpa target/version/idempotency field, mengizinkan restricted, memverifikasi session/owner/current password, lalu mencabut sesi/refresh dalam transaksi. Operator tetap mengelola method/namespace grant.

## Non-Goals

Tidak mengubah response Register/Login existing atau membangun login/refresh/logout, penerbitan cookie/CSRF, penyimpanan JWT/session, password hashing/history, permission framework, atau mekanisme revocation di rest-orc. Tidak mengubah sibling repository, config backend, transport lifecycle, dependency pin, generated graph/protobuf, atau endpoint users. Tetap gunakan boundary existing dan hindari generic validator/mapper/factory atau abstraction tambahan.
