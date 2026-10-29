# Proposal

## Why

grpc-auth mewajibkan TLS 1.3 dan opaque service token pada seluruh RPC, tetapi transport rest-orc masih mengizinkan plaintext dan belum memasang service credential. Dependency terpublikasi sudah kompatibel; autentikasi caller perlu dilengkapi sebelum menambah endpoint protected.

## What Changes

- Tambahkan `AUTH_GRPC_SERVICE_TOKEN` sebagai konfigurasi secret wajib tanpa default, mengikuti canonical unpadded base64url yang mewakili 32–256 byte sesuai verifier backend.
- Pasang satu `x-service-token` dari konfigurasi server melalui secure gRPC per-RPC credentials pada reusable auth connection, termasuk untuk Register/Login, Me, dan ListUsers. Nilai dari browser atau outgoing metadata tidak boleh menggantikan atau menggandakan credential tersebut.
- **BREAKING (konfigurasi startup saja):** auth connection wajib TLS dengan minimum versi 1.3; konfigurasi lama tanpa token atau dengan `AUTH_GRPC_TLS_ENABLED=false` harus ditolak sebelum aplikasi berjalan. Trust CA dan hostname/SAN tetap diverifikasi tanpa plaintext/skip-verification fallback.
- Pertahankan bearer pengguna, locale, metadata lain, parent context, deadline/cancellation, connection reuse, urutan RPC, dan seluruh public REST contract. Namespace, service ID, expiry, dan method allowlist tetap merupakan konfigurasi authoritative backend; tidak menambah header untuk mengklaim privilege.
- Tambahkan test konfigurasi, call credentials/TLS melalui transport in-memory, dan regresi HTTP; dokumentasikan provisioning/rotation yang dilakukan di luar repository serta verifikasi standalone `GOWORK=off`.

## Capabilities

### New Capabilities

- `auth-service-credentials`: Autentikasi rest-orc sebagai caller grpc-auth melalui environment-owned service credential dan TLS, dengan isolasi identitas pengguna, context preservation, serta safe failure.

### Modified Capabilities

Tidak ada. `auth-input-validation` dan `http-error-mapping` tetap berlaku tanpa perubahan requirement; public Register/Login tetap tidak memerlukan bearer pengguna.

## Impact

- Target implementasi: `internal/config`, `internal/client/grpcclient`, wiring `internal/app` bila diperlukan, catalog infrastructure error yang relevan, `.env.example` (termasuk menghapus ignore khusus template tersebut di `.gitignore`), README, serta root `test/config`, `test/client`, `test/app`, dan `test/integration`.
- Reuse shared TLS builder; kebijakan TLS 1.3 berlaku pada transport dengan service credential tanpa mengubah baseline TLS client HTTP generik. Constructor transport existing tetap source-compatible.
- Tidak mengubah Go dependency pins, generated files, route, DTO, response/error mapper, sibling repository, backend allowlist, atau deployment.
- Provisioning token acak dan backend digest/expiry/allowlist harus dilakukan operator secara terpisah. Change ini tidak membuktikan deployment readiness, tidak membuat token nyata, dan tidak melakukan rotasi otomatis, cookie/CSRF, IP forwarding, atau fitur Me/session baru.
