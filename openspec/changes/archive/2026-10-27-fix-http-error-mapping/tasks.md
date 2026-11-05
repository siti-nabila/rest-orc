# Tasks

## 1. Status dan envelope error downstream

- [x] 1.1 Pisahkan pemilihan status dari bentuk body di `internal/response/error_mapper.go`; prioritaskan metadata `http_status` integer 400–599 sebelum Reason/gRPC. Tambahkan scenario di `test/response/` untuk 400/409/429/503 dengan dan tanpa field errors, termasuk email conflict dan perbedaan status catalog/gRPC; verifikasi lewat `rtk go test ./test/response/...`.
- [x] 1.2 Pertahankan public metadata code dengan fallback Reason/catalog, field messages downstream, dan description catalog sesuai locale; mask raw unknown/internal status messages. Tambahkan scenario untuk metadata code berbeda/tidak tersedia, Reason unknown dengan status valid, multiple field messages, locale arbitrary, serta internal detail; verifikasi status/body dan success regression melalui `rtk go test ./test/response/...`.
- [x] 1.3 Lengkapi fallback gRPC sesuai spec saat status eksplisit hilang/malformed/out-of-range, dengan Reason terdaftar maupun tidak. Tangani wrapped gRPC errors dan native/wrapped deadline/cancellation tanpa menimpa status eksplisit; tambahkan matrix seluruh fallback, default 500, dan 504/499 di `test/response/`, lalu jalankan `rtk go test ./test/response/...`.

## 2. Mapping error lokal dan short-circuit existing

- [x] 2.1 Petakan local/wrapped `dictionary.FieldErrors` dan `errorpackage.Errors` ke HTTP 400 dengan envelope existing; tambahkan scenario input required/length/format/character errors, nested fields, multiple messages, dan locale di `test/response/`. Pertahankan mapping known catalog/Fiber dan generic 500 untuk error lokal/configuration yang tidak dikenali; verifikasi melalui `rtk go test ./test/response/...` tanpa menambah validator fitur baru.
- [x] 2.2 Pertahankan/perkuat test malformed JSON dan invalid query existing di `test/features/users/handler/http/`: assert HTTP 400 dan zero business-usecase calls. Verifikasi melalui `rtk go test ./test/features/users/handler/http/...`; jangan mengubah aturan input endpoint.

## 3. Header retry dari error downstream

- [x] 3.1 Bawa retry delay opsional dari decoder ke `MappedError` dan tulis `Retry-After` pada writer tanpa generic header abstraction atau perubahan constructor. Tambahkan scenario `test/response/` untuk round-up 1.2→2, zero, durasi besar tervalidasi, missing duration, protobuf invalid, negative, dan overflow-safe handling; assert header beserta status/body dan ketiadaan header pada success/error tanpa guidance. Verifikasi melalui `rtk go test ./test/response/...`.

## 4. Integration dan verifikasi menyeluruh

- [x] 4.1 Tambahkan scenario pada `test/integration/users/` dengan service fake existing: error berstatus eksplisit dan field details melewati client/gateway/usecase/handler sampai HTTP; sertakan retry guidance dan metadata/context locale yang relevan. Assert status/body/header, urutan/call count dan tidak ada retry; tambahkan protected invalid-query case dengan `Me` satu kali dan `ListUsers` nol kali. Verifikasi tanpa service/network nyata melalui `rtk go test ./test/integration/users/...`.
- [x] 4.2 Jalankan `rtk proxy gofmt -w` untuk Go files yang berubah, lalu targeted tests response/handler/integration, `rtk go test ./...`, dan `rtk go vet ./...`. Jalankan race detector hanya jika perubahan menyentuh concurrency/shared state; catat hasil serta mode dependency workspace aktual dan pisahkan failure pre-existing/environment dari regression.
- [x] 4.3 Review diff final dan status memakai `rtk proxy git diff`, `rtk git diff --check`, dan `rtk git status --short`, termasuk file baru di change ini. Verifikasi hanya shared response implementation, test terkait, dan tracking artifacts yang berubah; tidak ada dependency update, sibling edit, secret, generated artifact, atau perubahan user yang tertimpa. Laporkan pemenuhan scenario spec dan keterbatasan pin dependency yang sudah ada.

## Hasil Verifikasi

- Targeted test response, handler users, dan integration users lulus.
- `go test ./...` dan `go vet ./...` lulus pada parent workspace existing.
- `gofmt` dan review diff selesai; perubahan hanya shared response, scenario test terkait, dan tracking tasks. Perubahan user lain dipertahankan.
- Verifikasi memakai `GOCACHE=/private/tmp/rest-orc-go-build` karena cache default berada di luar writable sandbox. Tidak ada perubahan konfigurasi repository.
- Race detector tidak dijalankan karena tidak ada perubahan concurrency/shared state.
- api-contracts dan error-package masih di-resolve dari sibling melalui parent `go.work`; pin dependency standalone yang lebih lama tidak diubah.
