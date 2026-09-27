# rest-orc

`rest-orc` adalah service orkestrasi yang menerima request dari
frontend, menjalankan alur pemrosesan atau penggabungan data, lalu berkomunikasi
dengan downstream service melalui gRPC maupun HTTP.

Project ini tidak memiliki database dan tidak menyimpan JWT secara lokal.
Tanggung jawab utamanya adalah menjadi boundary HTTP, meneruskan request
context, memanggil downstream yang diperlukan, dan mengembalikan hasil
orkestrasi kepada caller.

> **Status project:** project ini dibuat untuk belajar dan bereksperimen dengan
> Go, Fiber, gRPC, HTTP client, konfigurasi, load balancer, recovery, graceful
> shutdown, dan struktur test. Project ini belum dimaksudkan sebagai template
> production yang siap digunakan tanpa penyesuaian dan security review.

## Tujuan

- Mempelajari pemisahan transport, use case, domain, dan adapter dalam setiap
  feature.
- Mencoba orkestrasi beberapa downstream service.
- Mendukung downstream berbasis gRPC dan RESTful HTTP.
- Mempelajari connection reuse, keepalive, timeout, dan graceful shutdown.
- Mempelajari penggunaan load balancer atau proxy di depan downstream.
- Menyediakan test yang scenario-driven dan mudah dibaca.

## Gambaran Arsitektur

```text
Frontend
   │
   ▼
Fiber HTTP Server
   │
   ▼
Middleware
   │
   ▼
internal/features/<feature>/handler/http
   │
   ▼
internal/features/<feature>/usecase
   │
   ▼
Contract milik feature
   ├──► repo/grpcauth ──► auth gRPC client ──► reusable connection ──► auth service
   ├──► downstream adapter ──► reusable HTTP client ──► downstream service
   └──► repo/orm (opsional) ──► database
```

Project menggunakan struktur **feature-first**. Seluruh kode yang hanya dimiliki
oleh satu business feature ditempatkan di bawah
`internal/features/<nama_feature>`. Sebagai contoh, transport HTTP untuk feature
`users` berada di `internal/features/users/handler/http`, bukan di
`internal/handler` global.

Use case bergantung pada contract berdasarkan kemampuan bisnis, bukan pada
Fiber, generated protobuf, `net/http`, ORM, atau concrete client. Implementasi
contract diinjeksikan dari `internal/app` saat aplikasi dibuat.

Untuk alur orkestrasi tanpa database, cabang `repo/orm` tidak digunakan. Feature
memanggil adapter downstream yang menggunakan reusable gRPC atau HTTP client.
Folder `repo/orm` hanya relevan jika suatu feature benar-benar memiliki
persistence berbasis ORM.

Load balancing tidak dilakukan oleh orchestrator. Setiap client mengarah ke satu
endpoint proxy per service. Untuk gRPC, proxy harus memahami HTTP/2 dan gRPC agar
request dapat dibagi pada level RPC.

## Status Implementasi

Komponen yang sudah memiliki implementasi:

- Entrypoint dan graceful shutdown di `cmd/api`.
- Composition root dan Fiber server di `internal/app`.
- Loader dan validation konfigurasi di `internal/config`.
- Reusable gRPC, HTTP, dan TLS client di `internal/client`.
- Service-specific auth gRPC client dengan operasi `Me`, `ListUsers`, `Register`,
  dan `Login` di `internal/client/grpcauth`.
- Request principal dan authenticator berbasis operasi `Me` di `internal/auth`.
- Middleware authentication serta role-based authorization di
  `internal/middleware/{auth,authorization}`.
- Contract dan read model feature users di `internal/features/users/domain`.
- Adapter `ListUsers` berbasis auth gRPC di
  `internal/features/users/repo/grpcauth`.
- Use case dan Fiber handler `ListUsers`, `Register`, dan `Login` di
  `internal/features/users/{usecase,handler/http}`.
- Generic query dan response pagination di `pkg/pagination`.
- Standard success response dan error mapping di `internal/response`.
- Internal infrastructure error di `pkg/dictionary`.
- Endpoint liveness `GET /livez`, endpoint publik `POST /api/v1/users/create`
  dan `POST /api/v1/users/login`, endpoint admin `GET /api/v1/users`, serta
  panic recovery.
- Scenario-driven test untuk komponen yang sudah aktif.

Komponen berikut sudah disiapkan sebagai scaffold, tetapi masih kosong:

- `internal/features/users/repo/orm`.
- `internal/client/httpbackend`.
- `internal/middleware/{logging,recovery,requestid,timeout}`.
- `pkg/httpclient` dan `pkg/validator`.

ORM repository belum digunakan karena data user berasal dari auth service.
Pipeline users sudah dirakit melalui `internal/app`, sedangkan scaffold lain di
atas belum aktif pada request pipeline.

## Struktur Project

```text
.
├── cmd/
│   └── api/                              Entrypoint aplikasi
├── internal/
│   ├── app/
│   │   ├── app.go                        Lifecycle aplikasi dan reusable client
│   │   ├── http_server.go                Setup Fiber dan shared middleware
│   │   ├── routes.go                     Route group berdasarkan access policy
│   │   └── users_routes.go               Composition dan registrasi users
│   ├── auth/                             Principal dan autentikasi request-level
│   ├── client/
│   │   ├── grpcauth/                     Auth client untuk Me dan ListUsers
│   │   ├── grpcclient/                   Reusable low-level gRPC connection
│   │   ├── httpclient/                   Reusable low-level HTTP client
│   │   ├── tlsconfig/                    TLS configuration builder
│   │   └── httpbackend/                  Scaffold adapter backend HTTP
│   ├── config/                           YAML/env loader dan validation
│   ├── features/
│   │   └── users/
│   │       ├── domain/                   Model, rule, dan contract feature
│   │       ├── handler/
│   │       │   └── http/                 Fiber request/response mapping
│   │       ├── usecase/                  Business flow dan orchestration
│   │       └── repo/
│   │           ├── grpcauth/             ListUsers adapter berbasis auth gRPC
│   │           └── orm/                  Adapter persistence opsional
│   ├── middleware/
│   │   ├── auth/                         Authentication melalui auth service
│   │   ├── authorization/                Role-based authorization
│   │   ├── logging/                      Request logging scaffold
│   │   ├── recovery/                     Recovery middleware scaffold
│   │   ├── requestid/                    Request ID middleware scaffold
│   │   └── timeout/                      Request timeout scaffold
│   └── response/                         HTTP response writer dan error mapper
├── pkg/
│   ├── dictionary/                       Internal infrastructure errors
│   ├── httpclient/                       Scaffold reusable public helper
│   ├── pagination/                       Generic query dan response pagination
│   └── validator/                        Validation scaffold
├── tests/                                Seluruh runner, scenario, mock, fixture
├── env.yaml                              Konfigurasi lokal
└── AGENTS.md                             Aturan pengembangan repository
```

Direktori kosong tidak disimpan oleh Git. Saat mulai mengimplementasikan sebuah
scaffold, tambahkan source file yang benar-benar dibutuhkan; jangan menambahkan
placeholder atau abstraction hanya agar folder tetap ada.

## Penempatan Logic dalam Feature

Contoh berikut menggunakan feature `users`.

### `domain`

Tempatkan model bisnis, value object, rule murni, dan contract yang diperlukan
use case di `internal/features/users/domain`.

Domain tidak boleh mengimpor Fiber, HTTP DTO, generated protobuf, concrete
client, atau ORM. Bila use case membutuhkan pembacaan user, contract dapat
berbentuk kecil seperti `UserReader` atau `UserRepository` sesuai sumber datanya.

### `usecase`

Tempatkan business decision dan orchestration di
`internal/features/users/usecase`. Use case menerima dependency melalui
constructor dan hanya bergantung pada contract feature.

Contoh tanggung jawab use case:

- Memvalidasi aturan bisnis.
- Menentukan dependency yang harus dipanggil.
- Mengatur urutan side effect.
- Menggabungkan hasil beberapa downstream.
- Mengembalikan domain output atau business error.

Use case tidak membaca Fiber context, menulis JSON, membuat protobuf request,
atau menjalankan query ORM secara langsung.

### `handler/http`

Tempatkan semua detail transport Fiber di
`internal/features/users/handler/http`:

- Registrasi route milik feature.
- Parsing path, query, header, dan body.
- Transport-level validation.
- Mapping HTTP request ke input use case.
- Penerusan request context.
- Mapping output use case ke response DTO.
- Pemanggilan shared response writer.

Handler tidak boleh mengandung business decision atau memanggil ORM/downstream
client secara langsung.

### `repo/orm`

Tempatkan implementasi concrete repository berbasis ORM di
`internal/features/users/repo/orm`. Package ini mengetahui query, ORM model,
transaction, dan mapping persistence, lalu mengimplementasikan contract yang
digunakan use case.

Jangan gunakan package ini untuk generic HTTP/gRPC downstream call. Jika feature
`users` memperoleh data dari service lain, gunakan adapter/client downstream
yang namanya menjelaskan service atau protokolnya.

### `repo/grpcauth`

`internal/features/users/repo/grpcauth` mengimplementasikan contract repository
users dengan memanggil `internal/client/grpcauth.ListUsers`. Package ini memetakan
query domain ke protobuf request dan protobuf response ke
`pagination.Page[domain.ListItem]` tanpa menghilangkan metadata paginator,
termasuk `next_cursor`.

Use case hanya bergantung pada `domain.Repository`; generated protobuf tetap
berhenti di client dan concrete repository.

### Shared infrastructure

- Connection pooling, keepalive, timeout transport, dan TLS tetap berada di
  `internal/client`.
- Mapping response/error HTTP bersama tetap berada di `internal/response`.
- Wiring concrete dependency ke use case dan handler berada di `internal/app`.
- Middleware lintas feature berada di `internal/middleware`.
- Jangan memindahkan logic feature ke `pkg`; `pkg` hanya untuk komponen yang
  benar-benar reusable dan aman digunakan di luar satu feature.

## Registrasi Route

`internal/app/http_server.go` hanya membuat Fiber server, shared middleware,
auth client, dan route groups. Detail composition setiap feature dipisahkan ke
file seperti `internal/app/users_routes.go`.

Seluruh business route ditempatkan di bawah version group `/api/v1`:

```go
v1 := httpServer.Group("/api/v1")
routes := newRouteGroups(v1, authentication.Handle)
```

Endpoint infrastructure seperti `GET /livez` tetap berada di root dan tidak
menggunakan API version.

Admin middleware tidak ditulis ulang pada setiap endpoint. Helper
`routeGroups.admin(prefix)` membuat Fiber group dengan authentication dan
authorization role `admin`:

```go
func (routes routeGroups) admin(prefix string) fiber.Router {
    return routes.router.Group(
        prefix,
        routes.authentication,
        authorization.RequireRole(requestauth.RoleAdmin),
    )
}
```

Feature mendaftarkan endpoint relatif terhadap version group tersebut. Untuk
users, prefix feature adalah `/users`, sehingga prefix akhirnya menjadi
`/api/v1/users`. Handler membagi registrasi menjadi dua:

- `RegisterPublicRoutes` untuk `POST /create` dan `POST /login`, tanpa
  authentication maupun authorization middleware.
- `RegisterAdminRoutes` untuk `GET` pada path kosong, dengan authentication dan
  authorization role `admin`.

Public routes didaftarkan sebelum admin group pada prefix yang sama. Urutan ini
penting di Fiber agar middleware admin `/api/v1/users` tidak menangkap login dan
register lebih dahulu.

Jika feature admin `products` ditambahkan, buat
`internal/app/products_routes.go`, lalu panggil registrasinya dari
`registerFeatureRoutes`:

```go
if err := registerProductsRoutes(
    routes.admin(productsRoutePrefix),
    dependencies,
); err != nil {
    return err
}
```

Dengan pola ini, middleware admin tetap didefinisikan sekali. Wiring constructor
repository, use case, dan handler tetap eksplisit di composition root karena
setiap feature dapat memiliki dependency yang berbeda; bagian itu sengaja tidak
disembunyikan dalam generic factory atau reflection.

## Alur Startup dan Request

Alur startup:

```text
cmd/api/main.go
   └── config.Load
         └── app.New
               ├── membuat reusable clients
               ├── membuat adapter/repository
               ├── menginjeksi dependency ke use case
               ├── menginjeksi use case ke handler
               └── mendaftarkan handler ke Fiber
                     └── Listen sampai shutdown signal
```

Alur request bisnis:

```text
HTTP request
   └── Fiber middleware
         └── features/<feature>/handler/http
               ├── parse dan transport validation
               ├── mapping ke use case input
               └── usecase.Execute(ctx, input)
                     ├── business validation/orchestration
                     └── domain contract
                           └── concrete adapter/repository
                                 └── downstream atau database

hasil/error kembali melalui jalur yang sama
   └── handler memetakan output
         └── internal/response.Writer
               └── HTTP response
```

Request context harus diteruskan dari handler ke use case dan seluruh dependency.
Jangan menggantinya dengan `context.Background()`. Reusable client dibuat sekali
saat startup dan tidak dibuat ulang untuk setiap request.

Header HTTP `Accept-Language` dinormalisasi pada request boundary dan diteruskan
ke downstream gRPC melalui metadata `x-language`. Locale menggunakan tag BCP-47;
misalnya `zh-CN` dapat memakai message catalog `zh`.

Alur konkret `ListUsers` yang sudah aktif:

```text
GET /api/v1/users + Authorization: Bearer <token>
   └── auth middleware
         └── grpcauth.Me(ctx + authorization metadata)
               └── principal disimpan pada request context
                     └── authorization.RequireRole("admin")
                           └── users handler
                                 ├── parse pagination, date/role filters, keyword
                                 └── users usecase
                                       └── users domain.Repository
                                             └── repo/grpcauth
                                                   └── grpcauth.ListUsers(ctx)

ListUsers response
   └── repo memetakan protobuf ke domain pagination
         └── usecase mengembalikan hasil
               └── handler menaruh hasil pada field data
                     └── response.Writer menghasilkan JSON HTTP
```

`Me` dan `ListUsers` menerima request context yang sama. Middleware menambahkan
header `Authorization` sebagai gRPC metadata `authorization`, sehingga repo
tidak perlu membaca Fiber context atau menerima token sebagai parameter khusus.

## Menjalankan Project

Persyaratan:

- Go `1.26.2` atau versi yang kompatibel dengan `go.mod`.
- File `env.yaml` tersedia pada working directory aplikasi.

Jalankan langsung:

```bash
go run ./cmd/api
```

Atau gunakan Makefile:

```bash
make run
```

Setelah socket berhasil listen, aplikasi menampilkan URL dan port di terminal:

```text
rest-orc is running at http://0.0.0.0:8080 (port 8080)
client auth_grpc target: localhost:9090 (port 9090)
```

Pesan tersebut berasal dari Fiber `OnListen`, sehingga tidak dicetak jika
server gagal membuka port. Informasi client menunjukkan target konfigurasi,
bukan jaminan downstream sudah menerima koneksi karena koneksi gRPC dapat
bersifat lazy.

Periksa liveness:

```bash
curl http://localhost:8080/livez
```

Expected response:

```json
{"status":"OK"}
```

## Register dan Login

Kedua endpoint berikut bersifat publik. Caller tidak perlu mengirim bearer token
dan request tidak memanggil service `Me`:

```bash
curl \
  -X POST \
  -H 'Content-Type: application/json' \
  -d '{"email":"new-user@example.com","password":"secret-password"}' \
  http://localhost:8080/api/v1/users/create
```

```bash
curl \
  -X POST \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"secret-password"}' \
  http://localhost:8080/api/v1/users/login
```

Malformed JSON menghasilkan HTTP `400`. Hanya `GET /api/v1/users` yang melewati
authentication `Me` dan pemeriksaan role `admin`.

URL yang tidak cocok dengan route terdaftar menghasilkan HTTP `404` yang
dibedakan dari data bisnis yang tidak ditemukan:

```json
{"code":"NF","errors":{"description":"Endpoint not found."},"data":[]}
```

## List Users

Endpoint ini hanya dapat diakses principal dengan role name `admin`:

```bash
curl \
  -H 'Authorization: Bearer <token>' \
  'http://localhost:8080/api/v1/users?page=1&limit=20&last_id=&created_from=2026-08-01&created_to=2026-08-31&role=1,2&keyword=blek'
```

Query yang didukung saat ini:

- `page`: nomor halaman positif; opsional.
- `limit`: jumlah item positif; opsional.
- `last_id`: cursor dari response sebelumnya; opsional.
- `created_from`: tanggal awal inklusif berformat `YYYY-MM-DD`; opsional.
- `created_to`: tanggal akhir inklusif berformat `YYYY-MM-DD`; opsional.
- `role`: daftar role code positif yang dipisahkan koma, misalnya `1,2`;
  opsional.
- `keyword`: teks pencarian; opsional. Pemilihan field dan mode pencarian
  ditentukan oleh auth service, bukan caller HTTP.

Jika `page` atau `limit` tidak diberikan, nilai `0` diteruskan agar default tetap
ditentukan oleh auth service. Tanggal ditafsirkan dalam zona waktu
`Asia/Jakarta`; `created_to` mencakup seluruh tanggal yang diberikan. Nilai
pagination/role yang tidak valid, tanggal malformed, atau rentang tanggal
terbalik menghasilkan HTTP `400`.

Contoh response sukses:

```json
{
  "code": "SS",
  "message": "Users retrieved successfully.",
  "data": {
    "items": [
      {
        "id": 17,
        "email": "admin@example.com",
        "name": "Admin User",
        "address": "Jakarta",
        "phone": "+621111111"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 20,
    "total_pages": 1,
    "has_next": false,
    "has_prev": false,
    "next_cursor": ""
  }
}
```

Struktur paginator dari auth service dipertahankan seluruhnya pada field
`data`. Hanya setiap `items` yang dipetakan menjadi read model milik feature,
supaya protobuf tidak bocor sampai handler dan contract HTTP tidak bergantung
langsung pada generated code.

## Konfigurasi

Konfigurasi default dibaca dari `env.yaml`. Field yang tidak dikenal ditolak
agar typo tidak terabaikan.

Contoh konfigurasi ringkas:

```yaml
app:
  name: rest-orc
  env: development
  port: 8080

server:
  read_timeout: 10s
  write_timeout: 10s
  idle_timeout: 60s
  shutdown_timeout: 10s

clients:
  auth_grpc:
    target: auth-proxy:50051
    request_timeout: 5s
    max_receive_message_bytes: 4194304
    max_send_message_bytes: 4194304
    tls:
      enabled: false
      server_name: ""
      ca_file: ""
    keepalive:
      enabled: false
      time: 5m
      timeout: 20s

```

Environment variable memiliki prioritas lebih tinggi daripada YAML. Contoh:

```bash
APP_PORT=9090 \
AUTH_GRPC_TARGET=localhost:50051 \
go run ./cmd/api
```

Environment override lain mengikuti nama yang didefinisikan pada
`internal/config/environment.go`.

## Menggunakan Client Reusable

Koneksi gRPC untuk service yang berbeda dibuat dengan constructor yang sama:

```go
authConnection, err := grpcclient.New(cfg.Clients.AuthGRPC)
```

Service-specific auth client dibentuk dari koneksi tersebut:

```go
authClient, err := grpcauth.New(authConnection)
```

Satu `authClient` dapat diinjeksikan ke middleware untuk `Me` dan ke
`features/users/repo/grpcauth` untuk `ListUsers`. Masing-masing consumer tetap
bergantung pada interface kecil sesuai kemampuan yang dipakainya.

Setelah field konfigurasi service ditambahkan, HTTP service menggunakan generic
HTTP client:

```go
productClient, err := httpclient.New(cfg.Clients.ProductHTTP)
```

Satu instance client dibuat per konfigurasi service, digunakan kembali untuk
semua request, lalu ditutup saat shutdown. Client tidak dibuat per request.

## Recovery

Panic yang terjadi secara synchronous pada Fiber handler, feature, atau
dependency yang dipanggil dalam request pipeline akan ditangkap oleh recovery
middleware dan dikonversi menjadi HTTP `500`. Detail panic tidak dikirim kepada
client, dan aplikasi tetap dapat melayani request berikutnya.

Recovery Fiber tidak dapat menangkap panic dari goroutine terpisah. Setiap
background goroutine harus memiliki recovery boundary sendiri. Normal failure
tetap harus dikembalikan sebagai `error`; panic recovery hanya menjadi pengaman
terakhir untuk unexpected failure.

## Test

Jalankan seluruh test:

```bash
go test ./...
```

Jalankan hanya test di root `tests/`:

```bash
go test ./tests/...
```

Jalankan race detector:

```bash
go test -race ./tests/...
```

Test mengikuti pola:

```text
runner_test.go
    └── test_scenarios.All()
            ├── success scenario
            ├── validation scenario
            └── dependency error scenario
```

Runner hanya menjadi entrypoint toolchain. Scenario, mock, dan fixture
ditempatkan pada folder terpisah agar test tetap mudah dibaca dan dirawat.

## Catatan Eksperimen

Beberapa bagian masih sengaja sederhana agar mudah dipelajari:

- Belum ada retry karena keamanannya bergantung pada idempotensi operasi.
- Belum ada circuit breaker, tracing, metrics, atau structured application log.
- Endpoint users yang aktif adalah public register/login dan admin ListUsers;
  filter tanggal/role dan pencarian keyword sudah diekspos, sedangkan field
  search, mode search, sort, dan field selection tetap ditentukan downstream.
- Proxy atau load balancer aktual belum menjadi bagian repository.
- Kontrak error bisnis dan response API belum difinalisasi.

Perubahan eksperimen sebaiknya tetap disertai test dan tidak mengorbankan
context propagation, deterministic behavior, atau resource cleanup.
