# Proposal

## Why

Register/login saat ini hanya memeriksa parsing HTTP; use case langsung memanggil gateway tanpa memvalidasi email/password. Validasi lokal sesuai `docs/grpc-auth-validation-rules.md` bagian 1–2 dan 13 diperlukan agar input invalid menghasilkan field errors HTTP 400 sebelum RPC bisnis, sambil tetap menerima credential login lama.

## What Changes

- Tambahkan aturan input murni milik users dan jalankan dari use case Register/Login sebelum gateway.
- Email: ASCII, required, 6–50 byte dan pola backend; gunakan salinan email yang di-trim/lowercase untuk pemeriksaan seperti backend, tetapi teruskan email asli tanpa perubahan.
- Password register: required, 6–50 byte, hanya `0x21–0x7E`, tanpa syarat komposisi. Password login: required dan 6–50 byte saja; jangan trim, mengubah kapitalisasi, atau membatasi karakter credential lama.
- Kumpulkan semua field errors memakai catalog existing dan shared mapper yang sudah mendukung validasi lokal. Tambahkan hanya pesan catalog milik users yang diperlukan untuk karakter password register invalid.
- Tambahkan domain/usecase/integration test untuk boundary, locale, no-RPC pada input invalid, nilai credential persis, dan regresi contract existing; dokumentasikan aturan endpoint pada README.

## Capabilities

### New Capabilities

- `auth-input-validation`: Aturan validasi lokal email/password register/login dan penghentian alur sebelum business RPC.

### Modified Capabilities

Tidak ada. Spec `http-error-mapping` existing sudah mengatur envelope, locale, status 400 lokal, dan prioritas HTTP status downstream; implementasi ini menggunakannya tanpa perubahan requirement.

## Impact

- Production: feature users pada `domain/` dan `usecase/{register,login}.go`; gateway/client, shared response mapper, route wiring, dan dependency pins tidak perlu berubah.
- Test: root `test/features/users/domain/`, `test/features/users/usecase/`, handler tests existing, serta `test/integration/users/`; README menjelaskan input yang diterima.
- Route publik existing `POST /api/v1/users/create` dan `/api/v1/users/login`, HTTP success 200, message, casing field `data.Token`, dan mapping response existing dipertahankan. Response register PENDING/login lengkap, cookie/CSRF, dan lifecycle lain pada plan besar berada di luar change ini.
- `confirm_password` tidak ditambahkan sebagai field wajib/didukung baru; request existing tetap email/password dan unknown fields mengikuti parsing existing.
- Pola email mengikuti source backend saat discovery: `+` ditolak, sedangkan `.`, `_`, `~`, `-` diterima hanya pada posisi yang lolos pola. Ini mempertahankan syarat plan bahwa simbol harus lolos validator backend; tidak memperluas pattern atau mengubah sibling service.
