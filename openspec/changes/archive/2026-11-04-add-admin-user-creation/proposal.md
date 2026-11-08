# Proposal

## Why

REST sudah menyediakan detail, edit profil dan perubahan status user target, tetapi operator belum dapat membuat akun melalui CreateUser. Endpoint administrasi baru melengkapi alur create → detail/edit → aktivasi dengan permission dan idempotency backend yang sudah tersedia.

## What Changes

- Tambahkan users-owned `POST /api/v1/users` untuk bearer unrestricted dengan exact `iam.user.create=true`, satu Me lalu satu CreateUser. Role admin saja maupun permission read/update/status/session tidak menggantikan grant ini.
- Wajibkan satu Idempotency-Key nonblank printable ASCII 1–200 byte. Create tidak memakai target ID atau Version precondition; body/query/header authority dan target/version claims tidak dipakai.
- Terima JSON object dengan email, password, name, phone dan country_code wajib; address opsional. Email/password mengikuti register existing, profil memakai normalisasi existing satu kali, dan field errors digabung aman sesuai locale. Tidak ada local no-op untuk create.
- Tetapkan payload status PENDING; status omitted/null atau explicit PENDING diterima, sedangkan nilai bertentangan ditolak. Tidak ada grant role/aplikasi, aktivasi, login, token atau cookie effect dari REST.
- Return HTTP 200/code SS/message `User created successfully.` dengan data.user berupa safe user snapshot dari CreateUser. Pertahankan string version presisi, array kosong dan boolean; jangan menambah changed atau registration/session fields yang tidak ada pada RPC.
- Backend tetap owns current authority, canonical email, duplicate account, password hashing/fingerprint, default member assignment, transaksi/audit/outbox dan replay. REST tidak membaca user, mengaktivasi, cache replay atau retry otomatis.
- Terapkan no-store sebelum CORS/auth untuk POST collection dan preflight POST, tanpa mengubah kebijakan GET ListUsers. Dokumentasikan create lalu operasi detail/profile/status yang masing-masing memerlukan grant sendiri.

## Capabilities

### New Capabilities

- `admin-user-creation`: authenticated creation akun PENDING dengan exact create permission, credential/profile validation, required idempotency dan safe authoritative user response.

### Modified Capabilities

Tidak ada. Public Register/Login, password change, self-profile, ListUsers dan seluruh target-user/session contracts mempertahankan behavior existing; pemakaian ulang validator hanya merupakan refactor internal.

## Impact

Implementasi nantinya mencakup users domain/usecase/gateway/handler, shared user SDK client, route composition/no-store, root test/ dan README. Credential rules diekstrak secukupnya ke helper bersama agar auth/users tidak bergantung pada DTO lintas feature, dengan regression register/login/password. SDK pinned sudah menyediakan CreateUser; tidak perlu dependency/env/protobuf/backend/sibling/database baru. Scope tidak mencakup create ACTIVE, grant provisioning, reset password, email/update policy changes, frontend/deployment, main-spec sync atau archive. Tahap proposal hanya membuat artifact planning.
