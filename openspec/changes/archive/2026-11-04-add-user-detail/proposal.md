# Proposal

## Why

Administrasi sesi sudah tersedia, tetapi caller belum dapat membaca detail dan entity Version user target melalui REST. GetUser backend dan SDK sudah menyediakan sumber authoritative ini, sehingga revoke-all admin tidak perlu memakai versi Me actor atau lookup tersembunyi.

## What Changes

- Tambahkan users-owned `GET /api/v1/users/{user_id}`: satu Me lalu satu GetUser untuk bearer aktif unrestricted dengan exact permission `iam.user.read=true`.
- Gunakan hanya ID target pada path, berupa digit ASCII dalam 1–2147483647; leading zeros diterima. Body/query/Version/Idempotency-Key bukan sumber target, authority, atau precondition.
- Kembalikan DTO detail eksplisit: profil, status, timestamp, restriction marker, assignment roles/application access dan entity version sebagai string desimal positif int64. Jangan expose password, credential, legacy protobuf fields atau privilege hasil agregasi tambahan.
- Pisahkan actor dan target, tolak nil/malformed/mismatched successful result dengan safe 500, pertahankan safe backend errors dan per-request isolation.
- Pasang authentication per route sebelum broad legacy admin group, serta no-store sebelum CORS/auth. Public auth, ListUsers role gate, Me, profile/password dan kedua keluarga route sesi tetap seperti sekarang.
- Dokumentasikan alur caller GET detail → gunakan data.version pada revoke-all user yang sama, dengan permission read/revoke terpisah, tanpa auto retry atau automatic detail lookup pada mutation.

## Capabilities

### New Capabilities

- `user-detail`: detail aman user target dan sumber entity version melalui exact user-read permission.

### Modified Capabilities

Tidak ada. `admin-session-management` tetap tidak menambah lookup GetUser pada request sesi; sumber versi baru merupakan capability independen. `me-profile` tetap merupakan profil actor dengan policy dan response sendiri.

## Impact

Implementasi nantinya menyentuh users/domain, usecase, gateway/authservice, handler langsung, shared generated-SDK client, composition/routes internal/app, matcher no-store, README dan root test/. SDK pin existing sudah memadai; tidak diperlukan perubahan dependency, protobuf/backend/sibling, env, database atau local JWT/session storage.

Scope satu endpoint detail. Tidak mencakup migrasi ListUsers ke permission gate, create/update/status user, role/application mutation, effective-permission/session/audit aggregation, UI, provisioning grants, Git/PR/deployment, sync main specs atau archive. Proposal ini hanya menghasilkan artefak planning.
