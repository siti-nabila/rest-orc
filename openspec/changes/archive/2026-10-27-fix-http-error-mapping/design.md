# Design

## Context

Motivasi ada pada `proposal.md`; contract perilaku ada pada `specs/http-error-mapping/spec.md` dan acuan keputusan pada `docs/grpc-auth-validation-rules.md` bagian 13.

- `internal/response/error_mapper.go` memeriksa `grpcerror.Decode` lebih dahulu. `mapGRPCError` langsung menghasilkan 400/BR saat ada field errors; tanpa field errors, status/code/message berasal dari registry atau switch gRPC yang belum lengkap.
- Local `*dictionary.Error` dan beberapa `*fiber.Error` sudah dipetakan; local `dictionary.FieldErrors` dan `errorpackage.Errors` belum memiliki jalur response per field. `pkg/dictionary/ValidationError` adalah error konfigurasi, bukan validasi request, sehingga tidak boleh diklasifikasikan otomatis sebagai bad request.
- `MappedError` hanya membawa status/body; `Writer.Write` belum menulis `Retry-After`. `internal/app/server.go` sudah memasang writer ini sebagai Fiber ErrorHandler; wiring baru tidak diperlukan.
- Client auth meneruskan error service tanpa mengubahnya. Handler users sudah menghentikan malformed JSON/invalid query sebelum use case, tetapi belum menerapkan seluruh aturan validasi field pada dokumen user-management.
- `test/response` memakai runner/scenario dan Fiber in-memory. Baseline `rtk go test ./test/response/...` lulus pada workspace saat discovery.
- `go list -m -json github.com/siti-nabila/api-contracts` menunjukkan module aktif dari sibling `/Users/bleki/Desktop/mini-projects/api-contracts` melalui parent `go.work`. Decoder aktif sudah menyediakan `Metadata`, `FieldErrors`, dan `RetryInfo`; encoder aktif mendukung kedua tipe field errors. Versi cache yang dipin `go.mod` lebih lama dan belum memiliki seluruh API yang saat ini sudah digunakan rest-orc. Perbedaan ini pre-existing.

## Goals / Non-Goals

**Goals:**

- Semua keputusan status/body/header tetap berada pada shared response boundary dengan fungsi kecil untuk pemilihan status, pemilihan pesan, dan validasi retry delay.
- Pisahkan pemilihan status dari bentuk body sehingga field errors tidak mengubah status downstream yang sudah dipilih.
- Uji output HTTP aktual dan no-call behavior menggunakan dependency terkontrol.

**Non-Goals:**

- Menambah validator required/length/character baru pada endpoint atau mengimplementasikan endpoint yang belum tersedia. Untuk change ini, jenis validasi tersebut diuji sebagai input error lokal ke mapper; short-circuit endpoint diuji lewat parsing/query validation existing.
- Mengubah domain contract, generated protobuf, client/gateway, composition, konfigurasi, dependency pin, parent `go.work`, atau sibling repository.
- Mengubah HTTP success response, route, cookie/CSRF, urutan authentication, retry, atau idempotency.

## Decisions

### 1. Pilih status downstream sebelum membentuk body

Gunakan `grpcerror.Decode` existing. Parse hanya metadata `http_status` sebagai integer dan terima rentang 400–599. Urutannya: status eksplisit valid → registry dari Reason → fallback gRPC → 500. Jangan menebak status dari field name atau menambahkan nama wire `http_code` baru. Decode tetap bekerja untuk wrapped gRPC errors.

Alternatif mempertahankan `HasFieldErrors` sebagai cabang awal ditolak karena mengubah conflict menjadi 400. Membuat setiap handler memilih status juga ditolak karena menduplikasi kebijakan lintas feature.

Lengkapi fallback sesuai spec. Gunakan catalog common existing bila semantiknya cocok; untuk status yang tidak memiliki definisi common khusus, pertahankan generic safe catalog body dengan status fallback yang benar tanpa membuat business catalog baru. Jangan memakai catalog auth tertentu hanya untuk mendapatkan status conflict generik.

### 2. Tentukan code/message secara terpisah

Prioritas code: public metadata `code` nonempty → definisi Reason → code dari fallback. Registered description tetap di-resolve menurut request locale. Structured field messages adalah public localized messages dari service; pertahankan isi/list-nya. Tanpa Reason terdaftar, gunakan description generik dari fallback/common catalog, bukan `decoded.Message` mentah. Error internal/unknown tidak boleh membocorkan pesan transport/stack/credential. Status eksplisit tetap dipertahankan walaupun description harus diganti generik.

Field errors memakai `contractresponse.FieldError`, description memakai `contractresponse.DescriptionError`, dan `data` tetap empty. Tidak menambahkan metadata ke JSON.

Alternatif menyalin raw status message untuk Reason yang belum dikenal ditolak karena dapat membocorkan detail dependency. Mengubah seluruh pesan downstream menjadi teks 400 juga ditolak karena kehilangan semantik dan locale.

### 3. Tangani local field errors tanpa mengubah validator fitur

Tambahkan pengenalan kedua tipe field errors existing (`dictionary.FieldErrors` dan `errorpackage.Errors`), termasuk ketika di-wrap, di shared mapper. Format public/localized field messages secara konsisten dengan contract existing, termasuk nested fields dan banyak pesan per field. Gunakan catalog bad request untuk status/code; error lokal lain tetap melalui catalog/Fiber mapping existing lalu generic 500. Jangan memakai `err.Error()` sebagai description publik untuk error lokal yang tidak dikenali.

Tangani `context.DeadlineExceeded` dan `context.Canceled`, termasuk wrapping, setelah memprioritaskan structured gRPC/catalog errors agar status eksplisit tidak tertimpa. Gunakan common timeout/canceled catalog dengan 504/499. Cancellation hanya menghasilkan response jika koneksi masih dapat ditulis; tidak membuat mekanisme delivery/retry baru.

Alternatif mengubah seluruh error lokal menjadi 400 ditolak: error konfigurasi dan error internal tidak termasuk kesalahan input caller.

### 4. Bawa hanya informasi retry yang dibutuhkan writer

Tambahkan informasi retry opsional pada `MappedError` (misalnya string `RetryAfter`), lalu writer menulis header itu sebelum JSON jika tersedia. Pertahankan signature constructor dan pola wiring existing. Tidak perlu generic header framework.

Gunakan `RetryInfo` dari decoder aktif. Validasi adanya duration dan validitas protobuf-nya; delay harus nonnegative. Hitung delay-seconds secara aman dari seconds/nanos tervalidasi, round up jika nanos positif, dan izinkan zero yang eksplisit. Hindari konversi duration yang saturate/overflow; delay yang tidak dapat direpresentasikan diabaikan. Missing/invalid/negative delay tidak menghasilkan header. Tidak menggunakan waktu sekarang dan tidak menjalankan retry otomatis.

Alternatif HTTP-date ditolak karena membutuhkan clock tanpa manfaat untuk relative retry delay. Mengubah `api-contracts` ditolak karena decoder workspace sudah menyediakan data yang dibutuhkan.

### 5. Verifikasi melalui scenario existing dan integration terkontrol

- `test/response`: matrix status eksplisit valid vs catalog/gRPC dengan/tanpa field errors; 400/409/429/503; invalid/missing status; unknown Reason; seluruh fallback; wrapping; local field errors; locale; masking; retry delay boundaries; success regression.
- `test/features/users/handler/http`: pertahankan/perkuat no-usecase assertions untuk malformed JSON dan invalid query. Ini membuktikan short-circuit existing tanpa menambah validasi fitur.
- `test/integration/users`: gunakan service fake existing dan Fiber in-memory untuk error service yang melewati client/gateway/usecase/handler sampai writer, serta protected invalid query yang boleh memanggil `Me` tetapi tidak `ListUsers`. Assert call count/order, metadata locale/context yang relevan, response status/body/header, dan tidak ada retry.
- Seluruh test baru tetap pada root `test/`, runner menjadi satu-satunya `TestXxx`; tidak memakai service/network nyata atau wall clock.

## Risks / Trade-offs

- [Caller terbiasa menerima 400 untuk semua field errors] → Uji contoh email conflict 409 dan pertahankan bentuk envelope; ini koreksi status yang memang diminta.
- [Status/code catalog berbeda dari metadata downstream] → Uji prioritas metadata secara eksplisit; pisahkan status dari keamanan body.
- [Test lulus hanya karena parent workspace] → Laporkan mode dependency aktual dan baseline. Publikasi/pin api-contracts adalah pekerjaan terpisah; jika implementasi memerlukan API baru yang belum tersedia, berhenti dan laporkan tanpa memperluas scope.
- [Test field errors dianggap implementasi seluruh validator user-management] → Gunakan fixtures untuk local mapping; jangan menandai validator endpoint baru sebagai selesai.
- [Retry duration invalid atau overflow] → Validasi protobuf dan aritmetika delay sebelum membentuk header; jangan memakai clock atau retry otomatis.

## Migration Plan

Tidak ada migrasi data atau konfigurasi. Setelah apply, jalankan targeted tests lalu suite/vet dan review diff; deployment bukan bagian change ini. Rollback melalui pengembalian perubahan mapper/writer beserta test pada change ini jika nantinya dibutuhkan.
