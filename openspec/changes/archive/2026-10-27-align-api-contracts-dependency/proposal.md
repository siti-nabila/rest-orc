# Proposal

## Why

`rest-orc` lulus test menggunakan sibling modules dari parent `go.work`, tetapi build/test standalone gagal karena pin `api-contracts` belum menyediakan API yang sudah dipakai source dan test. Dependency perlu diselaraskan ke versi remote yang kompatibel agar checkout repository dapat diverifikasi dengan `GOWORK=off` tanpa sibling checkout.

## What Changes

- Pin `api-contracts` ke `v0.0.0-20261023084653-d32fd5c1bab4`, commit workspace yang sudah dapat diunduh melalui Go module proxy.
- Selaraskan direct requirement `error-package` ke `v0.0.10-0.20260927130242-e06b69ef855d` sesuai requirement api-contracts dan hasil module resolution standalone.
- Regenerasi `go.sum` dan hasil `go mod tidy` yang diperlukan tanpa mengubah versi dependency lain yang tidak terkait atau menambahkan local replacement/vendor.
- Dokumentasikan perintah standalone di README dan verifikasi build, targeted/full tests, vet, serta resolution module dengan `GOWORK=off` dan mode readonly.

## Capabilities

### New Capabilities

Tidak ada. Ini pemeliharaan dependency/build, tanpa behavior runtime baru; `.openspec.yaml` memakai `skip_specs: true`.

### Modified Capabilities

Tidak ada. Seluruh requirements dan skenario `http-error-mapping` tetap berlaku tanpa revisi.

## Impact

- File implementasi yang diperkirakan berubah saat apply: `go.mod`, `go.sum`, dan `README.md`.
- Source production, public REST contract, generated protobuf, dan aturan bisnis tetap sama. Gunakan regression suite existing, tanpa menambah test yang hanya mencocokkan isi go.mod.
- Tidak mengubah parent `go.work`, sibling repository, atau melakukan publish/tag/release. Kedua versi kandidat sudah tersedia remote sebagai pseudo-version; tidak perlu membuat release baru.
- Planning discovery memakai modfile sementara untuk mengevaluasi kandidat; dependency files repository belum diubah oleh change ini.
