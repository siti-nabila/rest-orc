# Proposal

## Why

Pengguna sudah dapat mengelola sesi sendiri, tetapi operator dengan permission yang sesuai belum dapat melihat atau mencabut sesi user target melalui REST. SDK dan backend sudah menyediakan ketiga RPC administrasi, termasuk idempotency dan optimistic versioning.

## What Changes

- Tambahkan users-owned `GET /api/v1/users/{user_id}/sessions`, `DELETE /api/v1/users/{user_id}/sessions/{session_id}`, dan `POST /api/v1/users/{user_id}/sessions/revoke-all`.
- Gunakan satu Me pada request saat ini, bearer aktif nonrestricted, dan exact permission `iam.session.read` untuk list atau `iam.session.revoke` untuk mutation. Nama admin tidak berarti role-admin bypass; gate role-admin ListUsers existing tetap dipertahankan.
- List memvalidasi target ID/pagination, menggunakan fixed order dan opaque cursor seperti self-session, serta mempertahankan safe fields, total int64, dan owner target tanpa lookup tambahan.
- Kedua mutation mewajibkan satu Idempotency-Key yang valid, diteruskan sebagai satu metadata entry tanpa local replay/retry. Revoke-all juga mewajibkan Version positif dalam rentang int64 dari entity user target; jangan memakai versi Me actor atau mengambil versi otomatis.
- Single revoke mengembalikan hanya changed. Revoke-all mengembalikan changed dan version sebagai string desimal dari backend; tidak menambahkan login_required, menghapus cookie, atau menerbitkan credential, termasuk ketika target adalah actor sendiri.
- Terapkan no-store sebelum CORS/auth, pertahankan TLS/service identity/context/error authority, dan uji route ordering, payload/header presence, replay outcomes, concurrency serta regresi.
- Perjelas requirement kompatibilitas my-session-management: route admin merupakan capability terpisah, bukan alias dari self-session yang sebelumnya dikecualikan.

## Capabilities

### New Capabilities

- `admin-session-management`: administrasi sesi user target dengan exact permission, bounded pagination, idempotency metadata, target entity version, dan safe backend-owned results.

### Modified Capabilities

- `my-session-management`: ubah requirement `Reuse secure request transport and retain existing behavior` agar route admin target dapat ditambahkan sebagai capability terpisah tanpa mengubah policy/URL self-session atau menjadikannya alias.

## Impact

Implementasi nantinya menyentuh users/domain, usecase, gateway/authservice, handler langsung, shared auth SDK client, composition/routing internal/app, matcher no-store, README, dan root test/. SDK pin saat ini sudah mencukupi; tidak diperlukan perubahan dependency, protobuf, backend/sibling, environment, database, atau local JWT/session state.

Scope hanya tiga endpoint dan rekonsiliasi spec terkait. GetUser/detail/version-source API, migrasi ListUsers ke permission gate, UI, audit/idempotency transaction implementation, grants operator, Git/PR/deployment, sync main specs, dan archive bukan bagian tahap proposal ini. Caller harus memperoleh authoritative target version melalui sumber yang tersedia baginya; API ini tidak memperkenalkan sumber versi baru.
