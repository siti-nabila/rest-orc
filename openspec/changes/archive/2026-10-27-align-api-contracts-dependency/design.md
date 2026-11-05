# Design

## Context

Lihat `proposal.md` untuk motivasi. Discovery pada 27 Oktober 2026 menunjukkan:

| Komponen | Pin repository saat ini | Kandidat remote yang diverifikasi |
| --- | --- | --- |
| api-contracts | `v0.0.0-20260803064555-df4684bcd60d` | `v0.0.0-20261023084653-d32fd5c1bab4` |
| error-package | `v0.0.9` | `v0.0.10-0.20260927130242-e06b69ef855d` |

- Parent workspace adalah `/Users/bleki/Desktop/mini-projects/go.work`; dengan workspace aktif, dependency memakai sibling source. `go.mod` rest-orc tidak memiliki local replacement.
- Baseline `GOWORK=off go test -mod=readonly ./...` gagal: pin lama tidak memiliki `pkg/grpcerror/mapping`, `Decoded.RetryInfo`, common errors canceled/rate-limited/endpoint-not-found, dan `UserData.GetRoleCodes`. Kegagalan muncul di production dan test existing, bukan hanya change error mapper.
- Query Go proxy `api-contracts@latest` saat discovery mengembalikan pseudo-version Oktober 10; query commit workspace `d32fd5c1bab47d898ef63c01816b521ce7480b3c` berhasil me-resolve kandidat Oktober 23. Kandidat ini dipilih secara eksplisit; jangan memakai `@latest` sebagai pin atau menganggap default-branch latest sama dengan commit workspace.
- `go mod download -json` berhasil mengunduh kedua kandidat dari remote dengan checksum. api-contracts kandidat menyatakan requirement error-package persis seperti tabel. Keduanya mendukung Go directive repository `1.26.2`.
- Audit menggunakan salinan `go.mod`/`go.sum` sementara di `/private/tmp/`, lalu `go mod tidy` dengan `GOWORK=off`. `go list -m -json` menunjukkan kedua module berasal dari module cache dengan version/checksum, tanpa `Main: true` atau `Replace` sibling.
- Full test, build, dan vet dengan `GOWORK=off -mod=readonly -modfile=<audit>/go.mod` semuanya exit 0. Build sempat menampilkan warning nonfatal penulisan stat-cache di luar sandbox. Cache build sementara dipakai; tidak perlu perubahan konfigurasi repository untuk cache ini.
- Diff modfile audit hanya menaikkan kedua pin dan memindahkan `google.golang.org/genproto/googleapis/rpc` dari indirect menjadi direct dengan versi yang sama karena source/test mengimpor errdetails. Tidak diperlukan perubahan Go source.

## Goals / Non-Goals

**Goals:** Pin dua module immutable yang dapat diunduh dan pertahankan behavior existing. Hasil apply harus lulus standalone memakai dependency files repository sendiri, tanpa `-modfile`, workspace, atau local replacement.

**Non-Goals:** Tidak mengubah REST/gRPC schema, business rule, source/test implementation, toolchain version, parent workspace, sibling repository, atau menambah CI/script/vendor. Publish/tag dependency tidak diperlukan. Main spec `http-error-mapping` tidak berubah; `skip_specs: true` menghindari delta runtime yang tidak diperlukan.

## Decisions

### 1. Pin versi remote yang cocok dengan source workspace

Gunakan pasangan versi pada tabel, kemudian jalankan tidy dengan `GOWORK=off`. error-package tetap direct requirement karena rest-orc mengimpornya langsung. Pertahankan versi dependency lain; terima perubahan klasifikasi direct/indirect dan checksum yang dihasilkan tidy jika relevan dengan import aktual.

Alternatif mempertahankan pin lama dengan local `replace` atau parent workspace ditolak karena tidak menghasilkan standalone checkout. Menyesuaikan source ke contract lama juga ditolak karena dapat menghilangkan behavior yang sudah disepakati. Update seluruh dependency via `go get -u` tidak diperlukan.

### 2. Pisahkan bukti discovery dari acceptance apply

Audit modfile hanya membuktikan kandidat kompatibel tanpa mengubah repository pada fase propose. Acceptance apply wajib memakai `go.mod`/`go.sum` repository yang sudah diperbarui, dengan `GOWORK=off` dan tanpa `-modfile`:

```text
rtk proxy env GOWORK=off go list -mod=readonly -m -json all
rtk proxy env GOWORK=off go build -mod=readonly ./...
rtk proxy env GOWORK=off go test -mod=readonly ./...
rtk proxy env GOWORK=off go vet -mod=readonly ./...
rtk proxy env GOWORK=off go mod verify
rtk proxy env GOWORK=off go mod tidy -diff
```

Periksa module graph bahwa dependency tidak memakai local `Replace`/sibling sebagai main module. `-mod=readonly` memastikan build/test/vet tidak memperbaiki dependency files secara diam-diam. `tidy -diff` harus kosong dan exit 0 setelah pembaruan. Bila sandbox membatasi cache, inject `GOCACHE` sementara pada command saja; jangan mengubah environment runtime aplikasi.

Alternatif hanya mengulang suite dengan workspace aktif ditolak karena itulah yang menutupi masalah baseline.

### 3. Gunakan regression suite dan dokumentasikan mode standalone

Jalankan targeted tests untuk auth, client authservice, response, gateway/handler users, dan integration users setelah pin/tidy, lalu full suite. Suite ini memeriksa generated DTO yang digunakan, metadata, role authorization, response mapper termasuk RetryInfo, dan flow users. Tidak perlu menambah test yang mencocokkan string versi go.mod atau membuat koneksi downstream nyata.

Tambahkan petunjuk README tentang parent workspace yang dapat mengubah module resolution dan perintah standalone di atas. Instruksi lokal existing tetap berlaku; jangan menyatakan GOWORK sebagai konfigurasi runtime aplikasi.

## Risks / Trade-offs

- [Pseudo-version tersedia tetapi bukan semantic-version tag stabil] → Pin lengkap commit immutable yang sudah berhasil diunduh; jangan bergantung pada query `@latest` saat build.
- [Transitive dependency memerlukan versi lebih tinggi dari direct pin lama] → Selaraskan error-package dan review module graph/checksum dari tidy; jangan downgrade transitives atau menambahkan replacement untuk memaksa lulus.
- [Workspace memberi hasil hijau yang menyesatkan] → Acceptance selalu memakai `GOWORK=off`, `-mod=readonly`, dan dependency files repository sendiri.
- [Cache/network permission berbeda antar mesin] → Pisahkan kegagalan retrieval/cache dari compile/test regression; akses registry tetap diperlukan bila module belum tersedia di cache.
- [Working tree memiliki perubahan user sebelumnya] → Pertahankan isinya; diff change ini dibatasi pada dependency files, README, dan tracking tasks. Jika ternyata butuh perubahan source/contract, laporkan sebelum memperluas scope.

## Migration Plan

Apply memperbarui pin/checksum dan README, lalu menjalankan acceptance standalone. Tidak ada migrasi data, deployment, atau publishing dalam change ini. Jika nanti diperlukan rollback, kembalikan hanya delta dependency/documentation change ini; pin lama diketahui tidak mendukung standalone source saat ini, sehingga rollback bukan perbaikan kompatibilitas.
