# Proposal

## Why

Login, refresh, logout, dan change password sudah tersedia, tetapi pengguna belum dapat melihat sesi/perangkat sendiri atau mencabut sesi tertentu melalui REST. Lengkapi self-service sesi menggunakan RPC yang sudah ada, dengan kepemilikan dan revocation tetap authoritative di grpc-auth.

## What Changes

- Tambahkan auth-owned `GET /api/v1/me/sessions`, `DELETE /api/v1/me/sessions/{session_id}`, dan `POST /api/v1/me/sessions/revoke-all` pada shared authenticated `/me` group.
- Setiap request yang eligible menjalankan tepat satu Me lalu satu RPC sesi; non-admin nonrestricted boleh, restricted user ditolak tanpa RPC bisnis. Cookie tidak menjadi credential alternatif dan tidak ada Me/lookup tambahan.
- List menerima pagination `page`, `limit`, dan opaque `last_id`, memakai urutan server tetap `created_at DESC, id DESC`, dan memetakan safe session metadata serta `is_current`. Pertahankan seluruh paginator dan total int64 tanpa truncation atau perubahan helper halaman ListUsers.
- Revoke satu sesi memvalidasi UUID canonical dan mengembalikan hanya `changed`. Tidak mengarang `login_required`, mencari sesi saat ini, atau menghapus cookie berdasarkan tebakan; FE memakai `is_current` dari list yang sudah dibacanya.
- Revoke semua sesi mempertahankan `changed`/`login_required` backend dan membersihkan kedua cookie browser hanya pada success `login_required=true`, termasuk tanpa incoming cookie pada `/me`.
- Terapkan no-store pada response/error/preflight ketiga URL, reuse auth connection/TLS/service credential dan shared error mapper, serta tambahkan deterministic unit/HTTP/TLS/race scenarios.

## Capabilities

### New Capabilities

- `my-session-management`: list dan revoke sesi sendiri, pagination aman, bearer-owner authorization, safe metadata, authoritative mutation flags, dan conditional browser cleanup.

### Modified Capabilities

Tidak ada. Existing `auth-session-lifecycle` tetap mengatur transport cookie/CSRF; `me-profile` tetap menjadi sumber principal tanpa field session ID baru. Endpoint dan kontrak existing dipertahankan.

## Impact

Implementasi nanti menyentuh auth/domain, usecase, gateway/authservice, handler langsung di handler/, shared SDK wrapper internal/client/authservice, dan composition auth di internal/app. Reuse BrowserPolicy.Clear dan shared test fixtures; seluruh test tetap di root test/. SDK session/v1 sudah ada pada published dependency pin, sehingga tidak diperlukan upgrade module, protobuf, environment key, local storage/JWT parsing, maupun perubahan sibling/backend.

Scope tidak mencakup sesi milik user lain/admin, user/role/permission APIs, UI FE, perubahan Login/Refresh/Logout/password/profile/ListUsers, backend transaksi/audit/outbox, Git/PR/deployment, atau sync/arsip main specs pada tahap proposal.
