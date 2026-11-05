# Proposal

## Why

Pengguna sudah dapat membaca identitas melalui GET /api/v1/me, tetapi belum dapat memperbarui profil sendiri melalui REST. Tambahkan PATCH /api/v1/me/profile sesuai bagian 3, 12, 15, dan 17 dokumen acuan, dengan ownership terverifikasi dan partial update yang tidak menimpa field yang tidak dikirim.

## What Changes

- Tambahkan endpoint authenticated `PATCH /api/v1/me/profile` pada feature users. Non-admin nonrestricted boleh memperbarui dirinya; restricted user ditolak sebelum no-op atau RPC mutasi. Identitas berasal dari principal hasil Me, bukan body/query/header user ID.
- Gunakan DTO request domain dengan field opsional `name`, `phone`, `country_code`, dan `address`. Omitted/null diabaikan, string kosong terpilih divalidasi, unknown fields diabaikan. Terapkan normalisasi dan validasi lokal sebelum mutasi, dengan catalog pesan pada `pkg/dictionary`.
- Bentuk field mask eksplisit hanya dari field dikenal yang non-null dan valid. Tanpa field efektif, kembalikan HTTP 200 dengan `data={"changed":false}` setelah autentikasi/otorisasi, tanpa UpdateProfile atau version buatan. Field terpilih yang nilainya sama tetap diserahkan ke backend untuk keputusan changed.
- Panggil tepat satu `/profile.v1.ProfileService/UpdateProfile` setelah satu Me untuk request valid bermask nonempty. Pakai published `ProfileRequest`/`ProfileResponse`; RPC ini tidak memiliki target user ID atau expected version, sehingga tidak mewajibkan Version/Idempotency-Key atau menggantinya dengan UpdateUser admin.
- Petakan hasil backend ke DTO domain: `changed`, version string desimal, dan safe profile fields yang benar-benar dikembalikan. Pertahankan shared success/error writer dan pemilihan HTTP status downstream existing.
- Tambahkan ProfileService SDK pada reusable auth client dengan connection startup yang sama. Pertahankan bearer, service credential/TLS, locale, metadata, deadline/cancellation, dan constructor existing. Hindari per-request client, retry, generic patch/mapper framework, atau DTO berlapis.
- Gunakan satu authenticated group `/me` bersama untuk GET Me dan PATCH profile agar middleware prefix Fiber tidak memanggil Me dua kali. Registrasi/handler profile tetap users; auth tidak bergantung pada feature users, dan public auth tetap didaftarkan sebelum users-admin group.
- Tambahkan unit/integration/TLS bufconn tests dan dokumentasi endpoint, lalu verifikasi build/test/vet serta targeted race dengan `GOWORK=off -mod=readonly`.

## Capabilities

### New Capabilities

- `my-profile-update`: partial update profil pengguna terverifikasi, validasi/normalisasi, field presence/mask, authenticated no-op, response aman, serta propagasi downstream.

### Modified Capabilities

Tidak ada perubahan requirement existing. `me-profile`, `auth-input-validation`, `auth-service-credentials`, dan `http-error-mapping` menjadi batas regresi; kemampuan baru memakai contract tersebut tanpa mengubahnya.

## Impact

- Production: users domain/usecase/handler/gateway, shared auth client, app route wiring, dan catalog error yang diperlukan di pkg/dictionary. DTO HTTP tetap domain; handler langsung di handler/.
- Test: root test users/client/app/integration/shared fixtures, termasuk server ProfileService pada TLS bufconn existing.
- Dokumentasi: README dan command verifikasi. Published dependency sudah menyediakan RPC/field mask/country-code presence; tidak perlu dependency update atau environment key baru.
- Backend grpc-auth diperiksa read-only. Deployment tetap memerlukan method grant operator untuk UpdateProfile; plan tidak mengubah caller configuration backend, sibling repository, generated source/graph, ataupun public endpoint existing.
- Tidak mencakup admin UpdateUser, change password, cookie/refresh/logout, privilege fallback, database lokal, atau mekanisme optimistic concurrency baru.
