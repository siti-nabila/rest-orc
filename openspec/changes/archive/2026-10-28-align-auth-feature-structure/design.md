# Design

## Context

Lihat `proposal.md` bagian Why. Ini refactor internal dengan `skip_specs: true`: requirement main spec `auth-input-validation`, `auth-service-credentials`, dan `http-error-mapping`, serta behavior Me yang sudah diimplementasikan, menjadi batas regresi.

Audit memakai Graphify sebagai routing awal lalu memverifikasi source dan test. Graph memakai commit dasar `d2257fc2baa509f00ae462606c0992c58045e4fe`, sementara working tree berisi perubahan belum di-commit. Collision tipe Handler pada extractor juga sudah diketahui, sehingga relasi graph tidak digunakan sebagai satu-satunya bukti dependency.

Temuan yang membatasi perubahan:

| Area existing | Temuan source | Keputusan |
| --- | --- | --- |
| `users/domain/gateway.go` | Menggabungkan ListQuery/filter, AuthRequest/AuthResponse, dan gateway tiga operasi | Pindahkan tipe/contract autentikasi ke auth; users hanya membutuhkan List |
| `users/domain/auth_validation.go` dan validation catalog | Hanya dipakai Register/Login; registry namespace existing `users` | Pindahkan rule ke domain auth dan catalog pesan ke pkg/dictionary; pertahankan identity legacy |
| `users/usecase/{register,login}.go` | Kedua usecase menerima gateway yang juga mewajibkan List | Pindahkan ke auth dengan interface consumer yang sempit |
| `users/gateway/authservice/gateway.go` | Mencampur List dengan Register/Login dan mapping token | Pisahkan adapter auth dari adapter users; shared client tetap satu |
| `users/handler/users_handler.go` | Handler List juga mewajibkan kedua auth usecase | Users handler hanya List; auth handler memiliki Register/Login/Me |
| Auth/users handler directory | Sudah langsung di `handler/`, tanpa subfolder http | Pertahankan hasil penyesuaian existing |
| DTO | AuthRequest/ListItem memiliki JSON tags di domain; Me response types berada di handler | Sesuai instruksi user, tipe DTO HTTP tetap domain; pindahkan tipe Me ke auth/domain dan pertahankan mapping di handler |
| `internal/app` | Users routes membuat seluruh usecase dan mendaftarkan public routes sebelum admin; auth routes hanya Me | Pindahkan construction/registration auth lalu pastikan urutan auth-public sebelum users-admin |
| `internal/client/authservice/client.go` | Shared generated-SDK wrapper dipakai authenticator dan gateway | Rename file ke auth_client.go; pertahankan package, tipe Client, API, dan transport |
| Root test | Auth domain/usecase/handler/integration scenarios masih bercampur users; users fixture NewApp dipakai TLS tests dan Me | Pindahkan test auth; jadikan combined app fixture shared karena benar-benar lintas consumer |
| Entry/config/transport/response | Entrypoint tipis, env terpusat, reusable transports, dan mapper HTTP terpusat sudah tersedia | Pertahankan; tidak membuka refactor baru untuk komponen yang telah sesuai |
| `internal/auth/me_snapshot.go` | Memvalidasi/memetakan UserData hasil Me ke Principal; tidak membangun mekanisme snapshot backend | Rename ke principal_mapping.go agar nama mengikuti tanggung jawab mapper, dengan behavior request snapshot tetap sama |

Baseline targeted tests pada feature, app, integration, client, dan response telah lulus dengan `GOWORK=off -mod=readonly`. Tidak ada implementasi di tahap propose.

## Goals / Non-Goals

**Goals:**

- Setiap feature memiliki model/DTO, validasi, consumer contract, usecase, gateway, handler, dan test sesuai tanggung jawabnya.
- Auth tidak mengimpor feature users untuk menjalankan Login/Register; users tidak mengimpor feature auth untuk ListUsers.
- Semua route existing tetap cocok dengan handler dan middleware policy yang sama, meskipun owner source berubah.
- Gunakan dependency kecil, mapping sederhana, penamaan yang jelas, dan flow yang dapat dibaca developer junior.

**Non-Goals:**

- Tidak merombak config, generic transport, lifecycle, error mapper, permission policy, atau shared principal yang telah memenuhi struktur.
- Tidak membuat DTO package/layer terpisah, HTTP DTO baru pada handler, service locator, abstraction/framework generik, scaffold, atau transport per request.
- Tidak mengubah URL, schema, validation rule, normalisasi credential, RPC ordering, retry, cookie/session behavior, dependency, environment, atau sibling repository.
- Tidak memperbaiki extractor Graphify atau mengganti nama tipe Handler hanya untuk menghindari collision tool. Generated graph/protobuf dan skill global tidak diedit.

## Decisions

### 1. Ownership auth berlaku pada seluruh vertical slice

Target production layout untuk bagian yang berubah:

```text
internal/
  auth/principal_mapping.go             UserData Me -> Principal, fungsi principalFromMe existing
  features/
    auth/
      domain/
        auth.go                         AuthRequest/AuthResponse dengan JSON contract existing
        gateway.go                      RegisterGateway dan LoginGateway
        auth_validation.go              Rule Register/Login existing
        me.go                           MeResponse dan nested response types
      usecase/
        register.go
        login.go
      gateway/authservice/
        gateway.go                      Adapter Register/Login dan mapping credential/token sederhana
      handler/
        auth_handler.go                 Register/Login/Me dan route constants
        me_response.go                  Principal -> domain MeResponse mapping existing
    users/
      domain/
        user.go                         ListItem dengan JSON tags existing
        gateway.go                      List query/filter dan List-only Gateway
      usecase/list.go
      gateway/authservice/
        gateway.go                      Adapter ListUsers saja
        list_request.go
        list_response.go
      handler/users_handler.go          Handler ListUsers saja
  client/authservice/auth_client.go     Shared wrapper, API dan package existing
  app/
    auth_routes.go
    users_routes.go
    routes.go
pkg/dictionary/
  validation_errors.go                  Definisi error validasi lokal
  validation_errors.yaml                Terjemahan pesan validasi lokal
```

Hapus deklarasi/implementasi auth dari feature users setelah consumer dipindahkan; jangan meninggalkan forwarding handler, deprecated feature shim, alias tipe auth di users, atau import antarkedua feature. Ini perubahan internal source, bukan compatibility shim untuk public HTTP API. Jangan membuat repository: adapter downstream tetap gateway.

**Alasan:** Hanya memindahkan handler masih membuat auth bergantung pada users untuk business model/validation/orchestration. Alternatif memindahkan semua ke `internal/auth` ditolak karena package tersebut memiliki tanggung jawab shared principal/request authentication, bukan vertical slice bisnis autentikasi.

### 2. DTO HTTP tetap tipe domain sesuai keputusan user

`auth/domain.AuthRequest` tetap memiliki field email/password dengan JSON tags existing. `auth/domain.AuthResponse` tetap hanya Token yang diserialisasi sebagai `Token` (tag eksplisit boleh `json:"Token"`). Handler melakukan Bind ke tipe domain tersebut dan memberikan result domain ke shared writer seperti existing. `users/domain.ListItem` dan `pagination.Page` tetap menjadi bentuk response ListUsers; tidak dibuat mapping DTO tambahan untuk ListUsers atau Login/Register.

Pindahkan deklarasi safe Me response types dari handler ke `auth/domain/me.go`, dengan exported names yang menjelaskan Me, misalnya MeResponse, MeRole, dan MeApplicationAccess. Pertahankan seluruh tag/tipe wire existing. Mapping principal ke domain MeResponse tetap berada di `auth/handler/me_response.go`; domain tidak perlu mengimpor shared principal, Fiber, protobuf, atau client untuk menjadi DTO. Me tetap membaca satu principal yang diperoleh middleware dari satu RPC.

Pilihan ini adalah instruksi eksplisit user untuk lokasi tipe DTO. Domain tetap berupa tipe/rule/contract data biasa; HTTP parsing, status/envelope writing, dan context forwarding tetap handler/response. Jangan menganggap instruksi DTO domain sebagai izin memasukkan Fiber atau protobuf ke domain.

**Alasan:** Memisahkan DTO handler pada refactor ini bertentangan dengan keputusan user dan menambah mapping yang tidak diminta. Memindahkan declaration Me sambil mempertahankan mapping mempertahankan safe field allowlist dan consistency behavior existing.

### 3. Interface mengikuti consumer, shared client tetap reusable

Usecase Register menerima `auth/domain.RegisterGateway` yang hanya memuat Register; Login menerima LoginGateway yang hanya memuat Login. Satu concrete auth gateway mengimplementasikan keduanya. Client contract pada auth gateway hanya membutuhkan Register/Login, sedangkan users gateway client hanya ListUsers. Users domain Gateway menyisakan List.

Handler auth memiliki interface RegisterUseCase dan LoginUseCase yang kecil serta shared writer. Constructor auth menjadi `New(register, login, writer)` dan memvalidasi dependency wajib. Users constructor menjadi `New(list, writer)`. Perubahan signature internal ini diselesaikan pada seluruh wiring/fixture/test dalam satu refactor; tidak dibuat overload atau wrapper untuk signature lama.

Pertahankan `internal/client/authservice.Client` dan constructor/shared Service API existing karena membungkus generated SDK dan digunakan beberapa consumer. Rename hanya file implementasinya dari client.go ke auth_client.go. Generic `grpcclient/client.go` dan `httpclient/client.go` tetap valid sebagai reusable transport; jangan memaksakan penamaan service pada transport generik, mengganti tipe menjadi nama redundan, atau mengganti gateway.go/list_request.go/list_response.go menjadi client.

Rename `internal/auth/me_snapshot.go` menjadi `principal_mapping.go`. Fungsi principalFromMe tetap memvalidasi response/timestamp dan memetakan UserData ke Principal dengan restricted suppression existing; tidak ada perubahan RPC atau cloning. Konsistensi snapshot profil/access berasal dari backend Me dan dipertahankan melalui principal request. Suffix snapshot tidak diperlukan untuk menamai fungsi mapping pada consumer. Pengecualian skill untuk genuine point-in-time snapshot bukan alasan untuk menyebut setiap mapper hasil RPC sebagai snapshot implementation.

**Alasan:** Feature-owned gateway interface mencegah consumer memerlukan dummy operasi tidak terkait. Alternatif satu domain Gateway berisi seluruh RPC auth/user mengulang coupling yang hendak dihapus; membuat connection/client baru per feature merusak lifecycle existing.

### 4. Pindahkan ownership route tanpa mengganti URL atau policy

`auth_routes.go` menerima shared auth client dan writer; membuat auth gateway, Register/Login usecases, lalu auth handler. Auth handler mendaftarkan public routes pada prefix legacy `/users`, dengan suffix `/create` dan `/login`, dan authenticated Me pada `/me`. Letakkan legacy prefix constant pada auth handler sendiri; auth tidak mengimpor users hanya untuk constant route. Users routes hanya membentuk pipeline ListUsers pada `/users` dengan gate admin existing.

`registerFeatureRoutes` mendaftarkan auth routes terlebih dahulu, kemudian users routes. Ini wajib karena Fiber protected group `/api/v1/users` yang terdaftar lebih awal dapat menangkap legacy public Login/Register. `server.go` tetap memiliki satu auth client/connection dan locale middleware; tidak perlu menambah root `/auth`, duplikasi route, per-request transport, atau global dependency.

| Public route | Owner setelah refactor | Middleware/RPC yang dipertahankan |
| --- | --- | --- |
| POST /api/v1/users/create | Auth Register | Public; zero Me; invalid local zero business RPC, valid tepat satu Register |
| POST /api/v1/users/login | Auth Login | Public; zero Me; invalid local zero business RPC, valid tepat satu Login |
| GET /api/v1/me | Auth Me | User authentication; tepat satu Me; tanpa gate admin/permission baru |
| GET /api/v1/users | Users List | Me sebelum gate admin dan ListUsers; tidak memanggil Register/Login |

TLS/service credential tetap dikirim connection existing. Me/ListUsers mempertahankan bearer pengguna; public Login/Register tidak memperoleh bearer buatan. Locale, metadata, earlier parent deadline, cancellation, retry guidance, dan no application retry tetap existing.

**Alasan:** Route URL adalah contract publik, sedangkan feature source adalah ownership internal. Mengganti prefix menjadi `/auth` atau mengandalkan forwarding melalui users menimbulkan breaking change/coupling yang tidak dibutuhkan.

### 5. Pindahkan validasi dan error definition tanpa mengubah identity

Rule auth validation berada di domain auth, sedangkan definisi error dan embedded catalog pesan validasi lokal ditempatkan bersama di `pkg/dictionary/validation_errors.go` dan `validation_errors.yaml`. Domain memakai error dari pkg/dictionary; tidak menyimpan salinan catalog atau forwarding alias. Jangan mengubah regex, byte limits, karakter register, dukungan credential login lama, urutan agregasi pesan, supported locale, atau raw input yang diteruskan. Pertahankan namespace catalog legacy `users` dan key `invalid_password_characters` sebagai identity kompatibilitas, dengan komentar singkat yang menjelaskan alasan; lokasi package baru tidak memerlukan perubahan identity error. Tidak ada copy catalog aktif di feature auth/users.

Gunakan auth-named dictionary aliases untuk guard yang dipindahkan dengan semantics sama: AuthGatewayClientRequired/ContextRequired ke guard UsersGateway existing, AuthUseCaseGatewayRequired/ContextRequired ke guard UsersUseCase existing, dan AuthHandlerUseCaseRequired ke UsersHandlerUseCaseRequired. Pertahankan identity/code/message underlying definitions; auth handler writer guard yang sudah ada tetap digunakan. Tidak menambah ad-hoc error strings atau public error code. Nama actual exported constants mengikuti prefix `Err` existing.

Error dependency tetap diteruskan dengan identity/cause existing. HTTP status/body tetap hanya dipetakan oleh `internal/response`. Pindahan source tidak menjadi kesempatan merapikan catalog legacy secara breaking.

**Alasan:** Mengubah namespace atau membuat error definition pengganti dapat mengubah identity/locale behavior saat refactor. Alternatif auth mengimpor users/domain untuk reuse error akan mempertahankan dependency antarkedua feature.

### 6. Test mengikuti ownership, fixture lintas feature benar-benar shared

- Pindahkan validation runner/scenarios dari users/domain ke auth/domain. Jika users/domain tidak lagi memiliki test nyata, jangan tinggalkan runner/folder placeholder.
- Pindahkan Register/Login usecase scenarios dan auth-only mocks dari users/usecase ke auth/usecase; users mock hanya List.
- Tambahkan auth gateway tests yang saat ini belum dimiliki gateway users untuk Register/Login: raw mapping, exact operation/call count, request context, nil context/client/response, dan identity dependency error. Pertahankan users gateway tests List/paginator.
- Pindahkan Login/Register handler scenarios dan auth fixtures ke auth/handler; sesuaikan constructor Me tests dengan dependency auth yang gagal jika dipanggil tak terduga. Users List handler tests tidak lagi membutuhkan dummy auth usecase.
- Pindahkan HTTP auth validation/forwarding/error scenarios dari integration users ke `test/integration/auth`. Pertahankan ListUsers scenarios di integration users dan TLS/context/security tests di integration authservice; split mixed error scenarios sesuai flow, tanpa menghapus coverage.
- Pindahkan combined HTTP setup dari integration/users/fixtures/app.go ke `test/shared/fixtures/app.go` karena auth, users, dan TLS integration menggunakannya. Gunakan concrete composition sederhana; tidak menambah prod test hooks atau fixture framework. Legacy auth golden bodies yang dipakai beberapa suite boleh ditempatkan di shared fixtures/auth.go; list-only data tetap users fixtures.
- Shared test app harus mendaftarkan public auth sebelum protected users seperti production, mempertahankan optional context-injection middleware dan `DisableDefaultDate` untuk synctest. Semua fixture/fake/scenario tetap di root test.

**Alasan:** Memindahkan production saja akan meninggalkan test auth bergantung pada users dan dummy List capability. Menggandakan setup semua feature untuk setiap suite berisiko menyamarkan perbedaan wiring/middleware order.

### 7. Acceptance dan verification refactor

Gunakan golden assertions existing sebagai acuan, bukan mengganti expected output agar cocok implementasi baru:

- Register/Login tetap HTTP 200, code SS, message `Register Successfully`/`Login Successfully`, dan hanya `data.Token`. Backend registration status, refresh token, atau session fields tidak otomatis ditambahkan.
- Invalid local input tetap 400/BR dengan daftar error terlokalisasi dan zero RPC; malformed/known-type error, null/unknown fields, confirm_password, raw credential forwarding, downstream 409/401/status override, serta retry guidance tetap behavior existing.
- Me mempertahankan field allowlist, decimal-string version, timestamp UTC/null, empty collection shapes, phone passthrough, restricted suppression, failure tanpa fallback, dan tepat satu RPC; perpindahan DTO tidak mengubah principal cloning.
- ListUsers mempertahankan query/date/role mapping, complete pagination/cursor, JSON empty items, serta admin flow. Tidak migrasi ke permission gate.
- Review import/type declarations atau `go list` memastikan auth/users tidak saling mengimpor, users tidak lagi memiliki auth capability, domain tidak mengimpor Fiber/protobuf/client, handler berada langsung di handler/, dan seluruh test berada pada root test. Gunakan focused review/commands; tidak menambah test yang sekadar mencerminkan filename atau scaffolding.
- Review setiap abstraksi baru terhadap batas pada decision 8: consumer dan tanggung jawabnya nyata, tidak ada layer/mapper/wrapper tambahan tanpa manfaat, dan flow bisnis tetap mudah ditelusuri developer junior.
- Jalankan targeted unit/integration/client tests, lalu full build/test/vet dan targeted race auth/context/integration dengan `GOWORK=off -mod=readonly`. Gunakan cache temporer yang writable bila environment memerlukan. Review scope/dependency/generated artifacts dan validasi OpenSpec strict.

### 8. Batasi abstraksi pada kebutuhan nyata

Pertahankan boundary handler/usecase/domain gateway/shared client yang memiliki tanggung jawab existing. Interface kecil untuk dependency consumer dan isolasi test tetap valid meskipun hanya memiliki satu implementasi. Jangan menambah layer, generic factory, strategy, builder, base type, atau wrapper hanya demi pola, kemungkinan reuse di masa depan, atau kemudahan mock tanpa kebutuhan behavior yang nyata.

Mapping Register/Login hanya menyusun credential request dan mengambil Token dari response. Gunakan assignment langsung atau fungsi private kecil di `auth/gateway/authservice/gateway.go`; tidak perlu auth_request.go/auth_response.go, mapper struct/interface, atau DTO perantara. File ListUsers `list_request.go`/`list_response.go` existing tetap dipertahankan karena memiliki mapping query/filter/pagination yang berbeda dan lebih kompleks. Pemisahan tanggung jawab tidak otomatis memerlukan pemisahan file/type.

Me tetap memproyeksikan principal existing melalui handler mapping; tidak menambah usecase, gateway, atau RPC untuk forwarding yang sama. Shared auth client tetap membungkus SDK dan menangani transport existing; tidak dibuat wrapper tambahan per operasi. Compatibility error aliases hanya dipakai untuk kebutuhan identity legacy pada decision 5, tanpa rantai alias atau forwarding lintas feature.

Gunakan DRY ketika tanggung jawab benar-benar sama. Jangan menyatukan flow Register/Login/ListUsers menjadi generic executor atau membuat fixture framework; sedikit pengulangan assignment literal lebih sederhana daripada coupling feature. Shared test fixture tetap concrete composition yang sudah dibutuhkan beberapa suite.

**Kriteria review:** Setiap interface/helper/type/layer baru harus dapat dijelaskan consumer dan manfaatnya saat ini. Hapus indirection yang hanya meneruskan panggilan tanpa tanggung jawab validation, mapping, policy, context, atau lifecycle. Developer junior harus dapat mengikuti handler ke usecase, gateway, dan shared client tanpa lompatan tambahan yang tidak diperlukan.

**Alasan:** Tujuan refactor adalah memperjelas ownership dengan contract tetap kompatibel. Menambah abstraksi tanpa kebutuhan akan memperpanjang alur dan biaya pemeliharaan tanpa membantu tujuan tersebut.

## Risks / Trade-offs

- [Public auth tertangkap admin middleware setelah perpindahan registrasi] → Daftarkan auth routes sebelum users routes dan uji success serta malformed/invalid public requests tanpa bearer/Me.
- [Payload berubah ketika tipe dipindahkan/exported] → Pertahankan JSON tags/types, khususnya capital Token dan seluruh Me fields, serta golden response existing.
- [Error identity/namespace berubah saat ownership dipindahkan] → Pertahankan catalog legacy sekali dan gunakan auth-named aliases ke guard existing; uji errors.Is/locale/HTTP output.
- [Test hijau karena fixture berbeda dari production] → Samakan urutan construction/registration pada shared test app dan app registration tests; uji actual RPC method/count/metadata memakai TLS bufconn existing.
- [Scope melebar menjadi cleanup seluruh project] → Gunakan daftar audit sebagai batas: hanya ownership auth/users, DTO domain, auth client filename, wiring/test/docs terkait; komponen compliant dipertahankan.
- [Refactor menambah heavy abstraction] → Terapkan decision 8, pertahankan mapping auth sederhana di gateway.go, dan tolak layer/wrapper/mapper tambahan tanpa consumer serta tanggung jawab nyata saat review.
- [Graphify incomplete/stale dan generic Handler collision] → Verifikasi actual Go imports/source. Perbaikan extractor atau extract-per-feature/merge adalah pekerjaan tooling terpisah, bukan perubahan generated graph dalam refactor ini.
- [Contoh skill global masih memakai handler/http, sementara user menetapkan handler/ dan DTO domain] → Keputusan user terbaru serta root AGENTS mengenai handler langsung menjadi acuan; dokumentasikan pilihan DTO domain secara eksplisit tanpa mengedit skill global.

## Migration Plan

1. Gunakan hasil implementasi add-me-endpoint yang sudah ada sebagai baseline; pertahankan change lama dan seluruh perubahan user yang tidak terkait. Tidak perlu archive otomatis untuk memulai refactor terpisah ini.
2. Kerjakan domain/DTO dan usecase/gateway auth beserta test, lanjutkan pemindahan handler/wiring secara utuh agar semua caller memakai signature baru. Pindahkan test/fixture terkait bersamaan dengan layernya.
3. Rampungkan naming shared auth client dan dokumentasi ownership, lalu jalankan acceptance/verification di atas. Semua runtime config dan published dependency tetap existing.
4. Deployment mengikuti proses biasa setelah review; tidak ada migrasi data atau secret provisioning baru. Deployment tidak dijalankan oleh change ini.
5. Rollback dengan mengembalikan refactor sebagai satu kesatuan source/wiring/test. Public URL/payload tidak memerlukan migrasi frontend atau perubahan backend.
