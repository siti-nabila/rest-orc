# Proposal

## Why

REST sudah menyediakan detail dan perubahan status user target, tetapi operator belum dapat mengubah profil target. Backend dan SDK `UpdateUser` sudah menyediakan partial update, entity-version conflict, serta idempotency opsional; endpoint baru melengkapi alur administrasi tanpa mengubah update profil sendiri.

## What Changes

- Tambahkan `PATCH /api/v1/users/{user_id}/profile`, milik feature users, untuk bearer unrestricted dengan exact `iam.user.update=true`. Satu Me diikuti satu UpdateUser jika ada field efektif, tanpa role-admin bypass atau lookup target tambahan.
- Target hanya dari path ID PostgreSQL positif. Header Version target wajib, presisi positif int64; Idempotency-Key opsional, tetapi jika hadir harus satu key valid. Jangan memakai Me.version atau input body/query sebagai fallback.
- Pilih hanya name/address/phone/country_code yang non-null dari JSON. Abaikan unknown/immutable fields dan field_mask caller; empty string terpilih tetap divalidasi. Gunakan aturan normalisasi profil existing, termasuk name 3–100 byte, address 3–200 byte, phone 12–20 digit, dan country code eksplisit.
- Tanpa field efektif, setelah eligibility/path/header validation, return sukses dengan data persis changed:false tanpa mutation, target lookup, versi buatan atau klaim pengecekan version terbaru. Field valid yang sama dengan nilai sebelumnya tetap dikirim ke backend.
- Untuk mutation, bentuk mask dari presence setelah validasi, kirim optional payload dengan target version, dan return changed beserta safe user snapshot dari UpdateUser dalam data.user. Gunakan shape user-detail yang sudah aman, version string desimal dan collection kosong konsisten; tidak menambahkan GetUser/read-permission requirement.
- Terapkan no-store sebelum CORS/auth serta tanpa cookie/token/CSRF effects. Backend tetap authoritative untuk current permission, version, replay, audit dan hasil changed; REST tidak cache/retry/refetch otomatis.
- Dokumentasikan alur detail → PATCH yang independen dan selaraskan referensi route UpdateUser lama ke route profile yang dipilih dalam keputusan REST terbaru.

## Capabilities

### New Capabilities

- `user-profile-update`: partial update profil user target dengan exact permission, aturan presence/normalisasi, Version wajib, key opsional, local no-op dan safe backend outcomes.

### Modified Capabilities

Tidak ada. `my-profile-update`, detail, status, public auth, ListUsers dan session flows mempertahankan kontrak serta eligibility existing.

## Impact

Implementasi nantinya menyentuh users/domain/usecase/gateway/handler, shared user SDK wrapper, composition routes, no-store matcher, root test/, README dan satu referensi route dalam docs/user-management-spec.md. SDK pinned saat ini mencukupi; tidak perlu dependency, env, protobuf, database, backend/sibling atau transport baru. Scope tidak mencakup create user, edit email/password/status/role/permission, perubahan self-profile, provisioning grant, deployment, main-spec sync maupun archive. Tahap ini hanya membuat artifact planning.
