# Proposal

## Why

FE belum mempunyai `GET /api/v1/me` untuk membaca profil dan akses pengguna login, walaupun contract terpublikasi dan backend sudah menyediakan snapshot Me lengkap. Middleware autentikasi existing sudah memanggil Me sekali, tetapi membuang field di luar principal minimum; snapshot tersebut perlu dipertahankan agar endpoint baru tidak melakukan RPC kedua.

## What Changes

- Tambahkan `GET /api/v1/me` sebagai bagian feature `auth`, dengan bearer pengguna dan service credential existing, tanpa gate admin/permission tambahan. Identitas dan cakupan akses tetap berasal dari hasil satu RPC Me backend.
- Lengkapi snapshot request milik `internal/auth` dengan profil, status, user version, restriction marker, scoped roles/application access, dan permissions dari response Me. Generated protobuf tetap berhenti di boundary autentikasi; jangan menyimpan bearer/token di snapshot response.
- Sajikan DTO REST eksplisit memakai envelope sukses existing, HTTP 200, nama field snake_case, version string desimal, timestamp UTC, serta collection/map kosong `[]`/`{}`. Phone diteruskan persis tanpa concat ulang country code.
- Restricted user tetap memperoleh profil minimum dan `must_change_password=true` tanpa privilege legacy maupun baru. Dependency/authentication failure tetap error melalui mapper existing, tanpa profil parsial, stale snapshot, atau empty-permission fallback.
- Pertahankan seluruh route/response existing, termasuk public Register/Login dan gate admin ListUsers; tambahkan test mapping, satu RPC, context/metadata, restricted/error, snapshot isolation, serta verifikasi `GOWORK=off`.

## Capabilities

### New Capabilities

- `me-profile`: Pembacaan profil dan akses pengguna login melalui satu snapshot backend pada `GET /api/v1/me`, dengan DTO REST aman dan restricted behavior.

### Modified Capabilities

Tidak ada. Requirement `auth-input-validation`, `auth-service-credentials`, dan `http-error-mapping` tetap berlaku tanpa perubahan.

## Impact

- Target implementasi: snapshot/mapping `internal/auth`, handler/DTO baru `internal/features/auth/handler`, registrasi `internal/app/auth_routes.go` dan authenticated route group, README, serta root `test/auth`, `test/features/auth/handler`, `test/app`, dan `test/integration`.
- Sesuai penyesuaian struktur handler yang diminta, file ditempatkan langsung di `handler/`: auth memakai `auth_handler.go` dan `me_response.go`, users memakai `users_handler.go`. Import, root test, dan dokumentasi diselaraskan; behavior Login/Register/ListUsers tetap sama.
- Me adalah snapshot identitas/akses pengguna login milik feature `auth`; pembacaan/pengelolaan profil dan perubahan data user sebagai resource tetap milik feature `users`. Pemindahan ownership implementasi Login/Register existing dari feature users ke auth tidak termasuk change ini.
- Reuse middleware auth, shared auth client/transport, TLS/service credential, locale, dan response writer. Tidak menambahkan usecase/gateway kosong untuk projection snapshot yang tidak memiliki orchestration baru.
- Tidak mengubah public API existing, dependency pin, generated files, konfigurasi runtime, sibling repository, deployment, permission gate endpoint lain, cookie/session lifecycle, update profile, atau change password.
- Pemilihan nama field DTO dan representasi nilai baru diperjelas di design/spec agar implementasi dan test memiliki contract yang konkret.
