# Proposal

## Why

Registrasi sudah menghasilkan akun PENDING dan detail user sudah menyediakan entity Version target, tetapi operator belum dapat mengaktifkan atau mengubah status akun melalui REST. SDK/backend ChangeUserStatus sudah menyediakan mutation authoritative beserta idempotency, version conflict dan perlindungan akun.

## What Changes

- Tambahkan users-owned `PATCH /api/v1/users/{user_id}/status`: satu Me lalu satu ChangeUserStatus untuk bearer aktif unrestricted dengan exact `iam.user.status.update=true`, tanpa role-admin bypass.
- Target hanya dari path ID 1–2147483647. Body JSON wajib memuat status code dan reason; unknown fields diabaikan. Status menerima PENDING/ACTIVE/SUSPENDED/DEACTIVATED persis; reason divalidasi raw ASCII printable plus CR/LF sebelum trim, panjang hasil trim 3–500 byte.
- Wajibkan tepat satu Idempotency-Key dan Version entity user target dari header. Pertahankan presisi positif int64 dan salinan metadata; jangan menggunakan versi Me actor, lookup target otomatis, replay cache atau retry lokal.
- Return hanya user_id, status, version string desimal dan changed boolean dari backend, termasuk no-op/replay. Transition/current authority, larangan self inactivation, last-admin guard, transaction/audit/outbox dan pencabutan sesi tetap backend-owned.
- Terapkan no-store sebelum CORS/auth dan tanpa cookie/token/CSRF effects, termasuk target=self, no-op atau error. Pertahankan safe structured error/locale/retry guidance dan secure request context.
- Dokumentasikan alur caller baca detail → kirim target Version pada status mutation terpisah, eligibility read/status yang independen, serta backend conflict/replay tanpa automatic refetch/retry.

## Capabilities

### New Capabilities

- `user-status-management`: administrasi status akun target dengan exact permission, typed JSON validation, target version, idempotency metadata dan safe authoritative outcomes.

### Modified Capabilities

Tidak ada. User detail tetap read-only dengan iam.user.read; sesi, Me, public Register/Login, password/profile dan ListUsers tidak memperoleh policy/RPC tambahan melalui change ini.

## Impact

Implementasi nantinya menyentuh users/domain/usecase/gateway/authservice/handler langsung, shared generated-SDK client, composition/routes internal/app, matcher no-store, README dan root test/. SDK pin existing sudah mencukupi; tidak diperlukan perubahan dependency, env, protobuf/backend/sibling, database atau local JWT/session storage.

Scope satu PATCH endpoint beserta dokumentasi dan verifikasi. Tidak mencakup create/update profil user oleh admin, migrasi ListUsers permission gate, UI/status-option endpoint, grant provisioning, transaksi/revocation backend, automatic login/session recovery, Git/PR/deployment, main-spec sync atau archive. Tahap proposal hanya menghasilkan artefak planning.
