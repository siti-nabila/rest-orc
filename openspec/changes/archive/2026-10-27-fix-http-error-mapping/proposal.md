# Proposal

## Why

Mapper saat ini memaksa seluruh field errors downstream menjadi HTTP 400 dan belum membaca `ErrorInfo.metadata.http_status`. Perilaku ini dapat mengubah conflict dari service menjadi bad request dan belum memenuhi keputusan pada `docs/grpc-auth-validation-rules.md` bagian 13.

## What Changes

- Prioritaskan HTTP status downstream yang valid (400–599), termasuk saat field errors tersedia; gunakan catalog dari Reason, lalu mapping gRPC, lalu 500 sebagai fallback.
- Petakan field errors validasi lokal menjadi HTTP 400 dengan pesan sesuai locale. Pertahankan mapping error lokal yang dikenal dan default 500 untuk error yang tidak dikenali.
- Pertahankan envelope `code/errors/data`, kode dan pesan downstream yang aman, serta field errors tanpa memaksa status 400.
- Teruskan `RetryInfo` yang valid sebagai `Retry-After` tanpa retry otomatis.
- Tambahkan test mapper/writer dan integration dengan dependency terkontrol untuk prioritas status, fallback, locale, keamanan pesan, serta validasi lokal yang menghentikan RPC bisnis.

## Capabilities

### New Capabilities

- `http-error-mapping`: Aturan HTTP status, envelope error lokal/downstream, dan header retry pada shared response boundary. Ini mendokumentasikan capability existing yang belum memiliki spec OpenSpec.

### Modified Capabilities

Tidak ada; inventory `openspec list --specs` belum memiliki spec.

## Impact

- Production: `internal/response/error_mapper.go` dan `internal/response/writer.go`; error tetap diteruskan dari handler/gateway/client melalui boundary existing.
- Test: `test/response/`, test handler users existing, dan `test/integration/users/` sesuai kebutuhan alur lintas layer.
- Status error downstream dapat berubah dari 400/status catalog menjadi status eksplisit service; bentuk JSON, route, dan response sukses tetap kompatibel.
- Tidak menambahkan endpoint, aturan validasi fitur, konfigurasi, dependency, atau perubahan sibling repository. Cookie/CSRF dan fitur lain dalam bagian 13 di luar scope.
- Workspace saat ini me-resolve api-contracts ke sibling melalui `go.work`; publikasi/pembaruan pin dependency terpisah dari change ini.
