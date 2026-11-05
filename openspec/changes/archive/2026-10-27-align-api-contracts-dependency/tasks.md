# Tasks

## 1. Selaraskan dependency remote

- [x] 1.1 Perbarui pin api-contracts menjadi `v0.0.0-20261023084653-d32fd5c1bab4` dan error-package menjadi `v0.0.10-0.20260927130242-e06b69ef855d` di `go.mod`, lalu jalankan `rtk proxy env GOWORK=off go mod tidy` untuk memperbarui `go.sum` serta klasifikasi direct/indirect yang diperlukan. Verifikasi melalui `GOWORK=off go list -mod=readonly -m -json all` bahwa kedua module memakai versi/checksum remote, tidak ada dependency local replacement atau sibling main module, dan versi dependency lain tidak berubah tanpa kebutuhan.
- [x] 1.2 Jalankan targeted regression dengan `GOWORK=off` dan `-mod=readonly` pada `./test/auth/...`, `./test/client/authservice/...`, `./test/response/...`, `./test/features/users/gateway/authservice/...`, `./test/features/users/handler/http/...`, dan `./test/integration/users/...`; pastikan seluruhnya lulus tanpa `-modfile` dan tanpa perubahan production/test Go source. Bila diperlukan perubahan source/contract, laporkan scope tambahan sebelum melanjutkan.

## 2. Dokumentasikan verifikasi standalone

- [x] 2.1 Tambahkan bagian README tentang parent workspace yang dapat menutupi pin dependency lama dan cara menjalankan build/test/vet, module resolution/verification, serta tidy check dengan `GOWORK=off` sesuai design. Pertahankan instruksi existing dan aturan environment-only runtime; verifikasi perintah terdokumentasi dijalankan dari root repository tanpa modfile sementara, replacement, atau sibling checkout yang dibutuhkan.

## 3. Acceptance dan review akhir

- [x] 3.1 Dari dependency files repository sendiri, jalankan `GOWORK=off go build -mod=readonly ./...`, `GOWORK=off go test -mod=readonly ./...`, `GOWORK=off go vet -mod=readonly ./...`, `GOWORK=off go mod verify`, dan `GOWORK=off go mod tidy -diff` melalui RTK. Semua harus exit 0, tidy diff kosong, dan build/test/vet/verify tidak mengubah go.mod/go.sum; gunakan cache sementara hanya bila dibutuhkan environment sandbox dan catat hasilnya.
- [x] 3.2 Review `rtk proxy git diff -- go.mod go.sum README.md`, `rtk git diff --check`, dan `rtk git status --short`; verifikasi delta hanya dependency files, README, serta tracking change ini, tanpa local replacement/vendor, update versi yang tidak terkait, perubahan parent go.work/sibling, atau perubahan user yang tertimpa. Catat versi final dan hasil standalone; baru tandai seluruh tasks selesai setelah acceptance lulus.

## Hasil Verifikasi

- Pin final api-contracts: `v0.0.0-20261023084653-d32fd5c1bab4`; error-package: `v0.0.10-0.20260927130242-e06b69ef855d`. Versi dependency lain tetap sama; genproto/rpc menjadi direct requirement pada versi existing.
- Module graph dengan `GOWORK=off` menunjukkan dependency remote dengan checksum, tanpa replacement dan tanpa sibling main module.
- Targeted tests, seluruh test, build, dan vet dengan `GOWORK=off -mod=readonly` lulus memakai go.mod/go.sum repository, tanpa modfile sementara.
- `GOWORK=off go mod verify` lulus; `GOWORK=off go mod tidy -diff` exit 0 tanpa diff. Dependency files tidak berubah selama acceptance readonly.
- Verifikasi memakai `GOCACHE=/private/tmp/rest-orc-go-build`. Build exit 0 dengan warning nonfatal penulisan stat-cache module di luar sandbox; tidak memerlukan perubahan konfigurasi repository.
- Review diff/checksum membuktikan hanya go.mod, go.sum, README.md, dan tracking tasks change ini yang berubah sejak apply. Source/test Go, parent workspace, dan sibling tidak diubah. Gofmt/race detector tidak diperlukan karena tidak ada perubahan Go/concurrency.
