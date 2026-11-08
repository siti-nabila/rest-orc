# Proposal

## Why

REST sudah menyediakan administrasi akun, tetapi belum memiliki katalog role untuk melihat definisi role atau memilih role pada alur grant/revoke berikutnya. Tiga endpoint baca ini menyediakan metadata dan pilihan FE dari kontrak RoleService existing dengan permission serta pagination yang sama.

## What Changes

- Tambahkan roles-owned `GET /api/v1/roles`, `GET /api/v1/roles/{role_id}` dan `GET /api/v1/roles/options`; route options mendahului parameter detail.
- Wajibkan bearer unrestricted dengan exact `iam.role.read=true`; satu Me diikuti satu ListRoles untuk list/options atau satu GetRole untuk detail. Role admin saja atau grant user/role-write tidak menggantikan read permission.
- Terima page, limit, opaque last_id, keyword, application_code dan CSV statuses ACTIVE/DISABLED pada list/options. Kirim default sort created_at DESC, updated_at DESC, id DESC dan pencarian logical keyword/FULL_TEXT_TRIGRAM; field selection/search mode/sort tetap milik server. Tolak kontrol yang belum didukung, termasuk tanggal role.
- Petakan list metadata role dan detail role/permissions melalui DTO aman. Pertahankan version sebagai string presisi, nullable description/timestamps, affected_user_count authoritative, numeric IDs dan role_code; total pagination int64 tidak boleh menyempit ke int32.
- Options mempertahankan metadata pagination/cursor, memakai value ID desimal string, label code - description dengan fallback code - name, serta code dan role_code string. Tidak melakukan loop seluruh halaman atau RPC baru untuk label.
- Terapkan no-store sebelum CORS/auth serta error/locale/Retry-After/context/TLS existing, tanpa cookie/token effect, cache lokal, authority lookup atau retry otomatis. Seluruh route dan kontrak existing tetap.

## Capabilities

### New Capabilities

- `role-catalog`: protected paginated role list, safe role detail dengan permission mapping, dan adapter options FE menggunakan exact read grant.

### Modified Capabilities

Tidak ada. Capability transport/error existing dipakai tanpa perubahan requirement; legacy ListUsers role gate/filter numeric role code tetap.

## Impact

Implementasi berikutnya menyentuh internal/features/roles/{domain,usecase,handler,gateway/authservice}, shared generated-SDK wrapper role_client.go, komposisi app/cache matcher, root test/ dan README. SDK pinned sudah menyediakan RoleService; tidak perlu dependency/env/protobuf/database atau sibling edits. RoleService TLS mock baru akan didaftarkan sekali pada harness existing dengan koneksi auth yang sama. Scope tidak mencakup mutasi role/permission, assignment user, application catalog, status options, frontend, provisioning grant, deployment, main-spec sync atau archive. Tahap ini hanya membuat artifact planning.
