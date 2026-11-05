# Tasks

## 1. Snapshot Me pada boundary autentikasi

- [x] 1.1 Lengkapi Principal dan mapping Me dengan profil/state, optional created-at, typed scoped roles/application access, dan permissions sesuai design; validasi nil response/timestamp, skip nil nested entries, serta suppress seluruh privilege pada restricted snapshot. Tambahkan scenario/fixture root `test/auth` untuk snapshot penuh, nilai kosong/false, restricted dengan privilege nonempty, error identity, nil response, invalid timestamp, dan legacy HasRole; verifikasi dengan `rtk proxy env GOWORK=off go test -mod=readonly ./test/auth/... ./test/middleware/authorization/...`.
- [x] 1.2 Isolasi mutable snapshot saat mapping, WithPrincipal, dan PrincipalFromContext tanpa generic framework; pertahankan nil semantics internal. Tambahkan test mutasi slice/map/time dan concurrent readers, serta perjelas snapshot request pada dokumentasi autentikasi README; verifikasi data dependency/context tidak berubah dan `rtk proxy env GOWORK=off go test -mod=readonly -race ./test/auth/... ./test/middleware/auth/... ./test/middleware/authorization/...` lulus.

## 2. Handler dan contract DTO REST Me

- [x] 2.1 Tambahkan handler/response DTO di `internal/features/auth/handler` yang membaca verified principal dan memakai shared writer, tanpa client/usecase/gateway baru. Tambahkan runner/scenario/fixture di `test/features/auth/handler` untuk exact envelope/allowlist, JSON numeric IDs, decimal-string version di atas batas aman JavaScript, timestamp UTC/fraction/null, arrays/map kosong, false permissions, urutan assignments, phone passthrough, restricted minimum, missing principal, dan dependency constructor validation yang diperlukan; verifikasi `rtk proxy env GOWORK=off go test -mod=readonly ./test/features/auth/handler/...` lulus.
- [x] 2.2 Dokumentasikan response GET Me normal/restricted di README, termasuk field nested, version string, timestamp/null, phone yang sudah dikomposisi, collection shape, dan legacy metadata tanpa permission inference; verifikasi contoh JSON sesuai DTO/spec dan tidak memuat bearer/service credential atau mengubah contoh response endpoint existing.

## 3. Wiring authenticated route

- [x] 3.1 Tambahkan authenticated route group dan `internal/app/auth_routes.go`, lalu registrasikan hanya GET `/api/v1/me` dengan auth middleware/locale existing tanpa gate admin/permission tambahan. Tambahkan/extend test app dan HTTP integration terdekat untuk missing/blank bearer 401 tanpa Me, no alias, serta regression public Register/Login dan Me-before-ListUsers admin flow; verifikasi `rtk proxy env GOWORK=off go test -mod=readonly ./test/app/... ./test/integration/users/...` lulus dan route existing tidak berubah.
- [x] 3.2 Perbarui daftar route/auth policy README: Me memakai user bearer dan configured service credential, menerima non-admin/restricted user, dan tidak memakai user/application selector dari input. Jelaskan ownership Me pada feature auth dan pengelolaan profil/data user pada feature users, tanpa memindahkan implementasi Login/Register existing; verifikasi daftar route cocok dengan composition dan tidak mengubah public path `/api/v1/me`, dokumentasi gate admin ListUsers, atau environment keys.

## 4. Integration satu RPC dan isolation

- [x] 4.1 Reuse harness TLS bufconn di `test/integration/authservice` untuk real middleware/shared-client path GET Me: non-admin dan restricted 200, tepat satu empty Me RPC tanpa lookup tambahan, configured service token/user bearer/locale, query/body/header claims tidak memilih identitas/scope, dan consecutive requests memakai snapshot terbaru. Lengkapi root scenario/fixture serta penjelasan setup deterministic seperlunya; verifikasi `rtk proxy env GOWORK=off go test -mod=readonly ./test/integration/authservice/...` lulus tanpa real network, sibling import, atau perubahan fixture endpoint lain yang tidak diperlukan.
- [x] 4.2 Tambahkan integration scenario kegagalan backend 401/403, explicit HTTP status/retry guidance, dependency/internal error, deadline dan cancellation, serta concurrent request isolation. Pastikan error tidak menjadi 200/partial/stale/empty-permission fallback, tidak ada application retry/Me kedua, dan tidak ada token/internal detail pada response; verifikasi `rtk proxy env GOWORK=off go test -mod=readonly -race ./test/integration/authservice/...` lulus dengan metadata, call count, context propagation, dan response isolation assertions.

## 5. Verifikasi lintas perubahan

- [x] 5.1 Jalankan gofmt pada seluruh file Go berubah dan targeted tests terkait hingga lulus, lalu jalankan `rtk proxy env GOWORK=off go build -mod=readonly ./...`, `rtk proxy env GOWORK=off go test -mod=readonly ./...`, `rtk proxy env GOWORK=off go vet -mod=readonly ./...`, dan targeted race pada auth/middleware/Me handler/integration terkait; verifikasi semua checks lulus tanpa dependency update atau workspace override.
- [x] 5.2 Review final diff/status dan jalankan `rtk git diff --check` serta `rtk proxy openspec validate add-me-endpoint --strict`; verifikasi hanya scope Me/auth snapshot/wiring/test/docs yang berubah, seluruh scenario spec tercakup, endpoint existing kompatibel, seluruh test tetap di root `test/`, serta sibling repository/generated files/configuration/dependency tidak berubah.

## 6. Penyesuaian struktur handler sesuai permintaan user

- [x] 6.1 Tempatkan handler auth/users langsung di `internal/features/<feature>/handler/{auth_handler,users_handler}.go` sesuai feature, dengan DTO Me `auth/handler/me_response.go` dan package `handler`; pindahkan root test agar mencerminkan layout baru, perbarui imports serta AGENTS/README/artifact aktif, dan pertahankan seluruh behavior endpoint. Verifikasi seluruh reference aktif dan directory mengikuti layout handler baru, jalankan targeted test handler/app/integration, full build/test/vet dengan `GOWORK=off -mod=readonly`, serta diff/scope review tanpa mengubah generated graph atau sibling repository.
