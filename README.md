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
internal/features/<feature>/handler
   │
   ▼
internal/features/<feature>/usecase
   │
   ▼
Contract milik feature
   ├──► gateway/authservice ──► auth gRPC client ──► reusable connection ──► auth service
   ├──► downstream adapter ──► reusable HTTP client ──► downstream service
   └──► repository/orm (opsional) ──► database
```

Project menggunakan struktur **feature-first**. Seluruh kode yang hanya dimiliki
oleh satu business feature ditempatkan di bawah
`internal/features/<nama_feature>`. Sebagai contoh, transport HTTP untuk feature
`users` berada di `internal/features/users/handler`, bukan di
`internal/handler` global.

Use case bergantung pada contract berdasarkan kemampuan bisnis, bukan pada
Fiber, generated protobuf, `net/http`, ORM, atau concrete client. Implementasi
contract diinjeksikan dari `internal/app` saat aplikasi dibuat.

Untuk alur orkestrasi tanpa database, cabang `repository/orm` tidak digunakan. Feature
memanggil adapter downstream yang menggunakan reusable gRPC atau HTTP client.
Folder `repository/orm` hanya relevan jika suatu feature benar-benar memiliki
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
  `Login`, `UpdateProfile`, `ChangePassword`, `RefreshSession`, dan `Logout` di `internal/client/authservice`.
- Request principal dan authenticator berbasis operasi `Me` di `internal/auth`.
- Handler Register/Login/Me/ChangePassword di `internal/features/auth/handler`; DTO HTTP,
  validasi credential, dan contract gateway di `internal/features/auth/domain`.
- Middleware authentication serta role-based authorization di
  `internal/middleware/{auth,authorization}`.
- Contract dan read model feature users di `internal/features/users/domain`.
- Adapter `ListUsers` dan `UpdateProfile` berbasis auth gRPC di
  `internal/features/users/gateway/authservice`.
- Use case Register/Login dan gateway auth di
  `internal/features/auth/{usecase,gateway/authservice}`.
- Use case dan Fiber handler `ListUsers`/`UpdateProfile` di
  `internal/features/users/{usecase,handler}`.
- Generic query dan response pagination di `pkg/pagination`.
- Standard success response dan error mapping di `internal/response`.
- Internal infrastructure error dan catalog pesan validasi lokal di `pkg/dictionary`.
- Endpoint liveness `GET /livez`, endpoint publik `POST /api/v1/users/create`
  dan `POST /api/v1/users/login`, endpoint authenticated `GET /api/v1/me`,
  endpoint admin `GET /api/v1/users`, authenticated `PATCH /api/v1/me/profile`, serta panic recovery.
- Scenario-driven test untuk komponen yang sudah aktif.

ORM repository belum digunakan karena data user berasal dari auth service.
Pipeline users dirakit secara eksplisit melalui `internal/app`. Direktori baru
hanya ditambahkan ketika memiliki implementasi nyata; project tidak menyimpan
scaffold kosong.

## Struktur Project

```text
.
├── cmd/
│   └── api/                              Entrypoint aplikasi
├── internal/
│   ├── app/
│   │   ├── app.go                        Lifecycle aplikasi dan reusable client
│   │   ├── server.go                     Setup Fiber dan shared middleware
│   │   ├── routes.go                     Route group berdasarkan access policy
│   │   ├── auth_routes.go                Composition Register/Login/Me
│   │   └── users_routes.go               Composition ListUsers/UpdateProfile
│   ├── auth/                             Principal dan autentikasi request-level
│   │   └── principal_mapping.go          Mapping Me ke Principal
│   ├── client/
│   │   ├── authservice/                  Shared auth client untuk auth dan users
│   │   │   └── auth_client.go            Wrapper generated auth SDK
│   │   ├── grpcclient/                   Reusable low-level gRPC connection
│   │   ├── httpclient/                   Reusable low-level HTTP client
│   │   └── tlsconfig/                    TLS configuration builder
│   ├── config/                           Environment loader dan validation
│   ├── features/
│   │   ├── auth/
│   │   │   ├── domain/                   DTO HTTP, validasi, dan contract auth
│   │   │   ├── gateway/authservice/      Mapping credential/token sederhana
│   │   │   ├── handler/
│   │   │   │   ├── auth_handler.go       Handler HTTP Register/Login/Me
│   │   │   │   └── me_response.go        Mapping Principal ke domain MeResponse
│   │   │   └── usecase/                  Validasi dan orchestration Register/Login
│   │   └── users/
│   │       ├── domain/                   Model, rule, dan contract feature
│   │       ├── gateway/
│   │       │   └── authservice/          Mapping auth transport dan domain
│   │       ├── handler/                  Fiber request/response mapping
│   │       │   └── users_handler.go      Handler HTTP users
│   │       └── usecase/                  Business flow dan orchestration
│   ├── middleware/
│   │   ├── auth/                         Authentication melalui auth service
│   │   ├── authorization/                Role-based authorization
│   │   └── locale/                       Normalisasi locale request
│   └── response/                         HTTP response writer dan error mapper
├── pkg/
│   ├── dictionary/                       Infrastructure errors dan pesan validasi
│   └── pagination/                       Generic query dan response pagination
├── test/                                 Scenario, mock, fixture, dan integration test
├── .env.example                          Daftar environment variable runtime
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

DTO HTTP memakai tipe domain feature: AuthRequest, RegisterResponse, LoginResponse dan MeResponse
beserta nested types berada di auth/domain; ListItem, UpdateProfileRequest,
UpdateProfileResponse, serta Profile berada di users/domain. Tidak
ada package DTO atau salinan transport tambahan. Domain tidak mengimpor Fiber,
generated protobuf, concrete client, atau ORM. Parsing/envelope HTTP tetap pada
handler/response, dan mapping Principal ke MeResponse tetap pada handler.
Contract dibentuk berdasarkan kemampuan yang dibutuhkan consumer; RegisterGateway
dan LoginGateway masing-masing hanya memuat satu operasi, sedangkan Gateway users
hanya memuat List.

Rule validasi tetap di domain auth/users sesuai feature. Definisi error dan terjemahan lokal berada
di `pkg/dictionary/validation_errors.go` dan `validation_errors.yaml`; namespace
legacy `users` dipertahankan untuk menjaga identity error.

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

Register/Login dimiliki auth/usecase dan menjalankan validasi domain sebelum
memanggil auth gateway. Users/usecase menangani ListUsers dan UpdateProfile.

Use case tidak membaca Fiber context, menulis JSON, membuat protobuf request,
atau menjalankan query ORM secara langsung.

### `handler`

Tempatkan semua detail transport Fiber di
`internal/features/users/handler`:

- Registrasi route milik feature.
- Parsing path, query, header, dan body.
- Transport-level validation.
- Mapping HTTP request ke input use case.
- Penerusan request context.
- Mengembalikan tipe domain sebagai DTO HTTP; mapping Principal ke domain MeResponse
  dilakukan pada auth/handler/me_response.go.
- Pemanggilan shared response writer.

Handler tidak boleh mengandung business decision atau memanggil ORM/downstream
client secara langsung.

### `repository/orm`

Tempatkan implementasi concrete repository berbasis ORM di
`internal/features/users/repository/orm`. Package ini mengetahui query, ORM model,
transaction, dan mapping persistence, lalu mengimplementasikan contract yang
digunakan use case.

Jangan gunakan package ini untuk generic HTTP/gRPC downstream call. Jika feature
`users` memperoleh data dari service lain, gunakan adapter/client downstream
yang namanya menjelaskan service atau protokolnya.

### `gateway/authservice`

`internal/features/users/gateway/authservice` mengimplementasikan contract gateway
users dengan memanggil `internal/client/authservice.ListUsers`. Package ini memetakan
query domain ke protobuf request dan protobuf response ke
`pagination.Page[domain.ListItem]` tanpa menghilangkan metadata paginator,
termasuk `next_cursor`.

Auth memiliki adapter `internal/features/auth/gateway/authservice` untuk
Register/Login. Mapping credential request dan hasil PENDING/session berada di gateway.go/session.go;
tidak perlu mapper type/interface atau file tambahan untuk beberapa assignment.
ListUsers tetap memakai list_request.go/list_response.go untuk mapping query,
filter, dan pagination yang lebih kompleks. Generated protobuf tetap berhenti
di shared client dan concrete gateway.

Pertahankan boundary yang punya tanggung jawab nyata. Hindari wrapper penerus,
generic executor/factory, dan abstraksi untuk kemungkinan reuse di masa depan.
Jangan memaksakan DRY pada flow berbeda; alur harus mudah ditelusuri developer junior.

### Shared infrastructure

- Connection pooling, keepalive, timeout transport, dan TLS tetap berada di
  `internal/client`.
- Mapping response/error HTTP bersama tetap berada di `internal/response`.
- Wiring concrete dependency ke use case dan handler berada di `internal/app`.
- Middleware lintas feature berada di `internal/middleware`.
- Jangan memindahkan logic feature ke `pkg`; `pkg` hanya untuk komponen yang
  benar-benar reusable dan aman digunakan di luar satu feature.

## Registrasi Route

`internal/app/server.go` hanya membuat Fiber server, shared middleware,
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
auth dan users, prefix legacy `/users` tetap dipertahankan untuk compatibility:

- Auth handler `RegisterPublicRoutes` mendaftarkan `POST /create` dan `POST /login`
  di `/api/v1/users`, tanpa authentication maupun authorization middleware.
- Auth handler `RegisterAuthenticatedRoutes` mendaftarkan `GET /api/v1/me`
  dengan authentication tanpa gate admin.
- Users handler `RegisterAdminRoutes` mendaftarkan `GET /api/v1/users`,
  dengan authentication dan authorization role `admin`.

`registerFeatureRoutes` membuat satu authenticated group `/me` yang digunakan
bersama auth GET path kosong dan users PATCH `/profile`. Authentication tidak
dipasang ulang pada subpath sehingga Me hanya dipanggil sekali. Tidak ada
admin gate pada PATCH tersebut; ListUsers tetap memakai admin group.

`registerFeatureRoutes` mendaftarkan auth_routes.go sebelum users_routes.go.
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
gateway, use case, dan handler tetap eksplisit di composition root karena
setiap feature dapat memiliki dependency yang berbeda; bagian itu sengaja tidak
disembunyikan dalam generic factory atau reflection.

## Alur Startup dan Request

Alur startup:

```text
cmd/api/main.go
   └── config.Load
         └── app.New
               ├── membuat reusable clients
               ├── membuat gateway/repository adapter
               ├── menginjeksi dependency ke use case
               ├── menginjeksi use case ke handler
               └── mendaftarkan handler ke Fiber
                     └── Listen sampai shutdown signal
```

Alur request bisnis:

```text
HTTP request
   └── Fiber middleware
         └── features/<feature>/handler
               ├── parse dan transport validation
               ├── mapping ke use case input
               └── usecase.Execute(ctx, input)
                     ├── business validation/orchestration
                     └── domain contract
                           └── concrete gateway/repository adapter
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
         └── authservice.Me(ctx + authorization metadata)
               └── principal disimpan pada request context
                     └── authorization.RequireRole("admin")
                           └── users handler
                                 ├── parse pagination, date/role filters, keyword
                                 └── users usecase
                                       └── users domain.Gateway
                                             └── gateway/authservice
                                                   └── authservice.ListUsers(ctx)

ListUsers response
   └── gateway memetakan protobuf ke domain pagination
         └── usecase mengembalikan hasil
               └── handler menaruh hasil pada field data
                     └── response.Writer menghasilkan JSON HTTP
```

`Me` dan `ListUsers` menerima request context yang sama. Middleware menambahkan
header `Authorization` sebagai gRPC metadata `authorization`, sehingga gateway
tidak perlu membaca Fiber context atau menerima token sebagai parameter khusus.

Principal menyimpan snapshot Me untuk request tersebut: profil, status/version,
restriction marker, legacy roles, scoped roles/application access, dan permissions.
Generated protobuf berhenti di boundary `internal/auth`. Snapshot menyalin slice,
map, dan optional timestamp dari dependency serta saat disimpan/dibaca melalui
context, sehingga mutasi pembaca tidak mengubah principal request lain.
`must_change_password=true` mempertahankan profil/state minimum dan mengosongkan
seluruh privilege, termasuk legacy role metadata. Snapshot tidak di-cache antar
request dan kegagalan Me tidak menghasilkan fallback profil/permissions.

## Menjalankan Project

Persyaratan:

- Go `1.26.2` atau versi yang kompatibel dengan `go.mod`.
- Seluruh environment variable wajib tersedia; lihat `.env.example`.

Untuk setup lokal, salin contoh konfigurasi, isi service token dan TLS settings
sesuai provisioning grpc-auth, lalu export nilainya ke shell. Template sengaja
tidak menyediakan token default dan tidak dapat dijalankan tanpa secret tersebut:

```bash
cp .env.example .env
set -a
. ./.env
set +a
go run ./cmd/api
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

## Identitas dan Akses Pengguna Login

`GET /api/v1/me` berada pada feature `auth`, sebagai snapshot identitas/akses
autentikasi. Pengelolaan profil/data user sebagai resource tetap milik feature
`users`. Login/Register dimiliki feature `auth`; URL legacy `/api/v1/users/create`
dan `/api/v1/users/login` tetap; response Register/Login kini mengikuti kontrak lifecycle di bawah.

Me membutuhkan `Authorization: Bearer <access_token>` pengguna dan memakai
service credential server existing. Non-admin dan restricted user dapat membaca
snapshot mereka tanpa gate admin atau permission tambahan. Identitas dan scope
ditentukan backend; body/query/header target user, aplikasi, atau permission
kiriman browser tidak dipakai untuk memilih snapshot.

Middleware autentikasi memanggil tepat satu RPC Me, lalu handler memetakan
principal request tersebut ke DTO REST tanpa RPC tambahan, aggregation, atau
cache antar-request. Locale, metadata, deadline, dan cancellation mengikuti
transport existing. Error Me tetap melalui mapper HTTP existing, tanpa partial
profile, stale snapshot, atau empty permissions sebagai fallback sukses.

Contoh response normal:

```json
{
  "code": "SS",
  "message": "Profile retrieved successfully.",
  "data": {
    "id": 17,
    "email": "member@example.com",
    "fullname": "Member User",
    "address": "Jakarta",
    "phone": "+628123",
    "country_code": "+62",
    "created_at": "2026-10-20T08:09:10.123456789Z",
    "status": "ACTIVE",
    "version": "9007199254740993",
    "must_change_password": false,
    "role_codes": [3],
    "role_names": ["member"],
    "roles": [
      {"role_id": 9, "code": "clm.member", "name": "Member", "status": "ACTIVE", "role_status": "ACTIVE"}
    ],
    "application_access": [
      {"application_id": 5, "code": "clm", "name": "CLM", "status": "ACTIVE", "application_status": "ACTIVE"}
    ],
    "permissions": {"clm.document.read": true}
  }
}
```

ID berupa JSON number; `version` adalah string desimal agar presisi uint64 tetap
utuh. `created_at` memakai UTC RFC3339 dengan fractional precision yang tersedia,
atau null ketika tidak diberikan backend. Phone diteruskan persis dari Me karena
sudah dikomposisi backend; REST tidak menambahkan country code lagi, termasuk
pada legacy repeated prefix. `role_codes`/`role_names` hanya metadata compatibility,
bukan sumber inference permission. Permission false tetap false; key absent tidak
memberikan grant.

Restricted user tetap menerima HTTP 200 dan envelope yang sama. Contoh data
minimum ketika backend tidak menyediakan timestamp:

```json
{
  "id": 17,
  "email": "member@example.com",
  "fullname": "Member User",
  "address": "Jakarta",
  "phone": "+628123",
  "country_code": "+62",
  "created_at": null,
  "status": "ACTIVE",
  "version": "7",
  "must_change_password": true,
  "role_codes": [],
  "role_names": [],
  "roles": [],
  "application_access": [],
  "permissions": {}
}
```

Semua collection privilege restricted dikosongkan, termasuk jika backend memberi
nilai nonempty. Pada snapshot normal, collection yang kosong juga berbentuk `[]`
dan permissions `{}`. Response nil atau timestamp invalid ditolak sebagai internal
error. Tidak ada alias `/me`, `/api/v1/users/me`, atau `/api/v1/auth/me`.

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

Validasi lokal dijalankan sebelum RPC Register/Login:

| Field | Register | Login |
| --- | --- | --- |
| `email` | Wajib, ASCII, 6–50 byte, sesuai pola email backend | Sama dengan register |
| `password` | Wajib, 6–50 byte, setiap byte `0x21–0x7E` (ASCII tercetak tanpa spasi) | Wajib, 6–50 byte; tidak membatasi karakter credential lama |

Email diperiksa melalui salinan yang di-trim dan diubah menjadi lowercase.
Email asli tetap diteruskan persis ke service, termasuk kapitalisasi dan spasi
ASCII di tepi. Non-ASCII tetap ditolak, termasuk Unicode whitespace di tepi.
Pola backend menerima `.`, `_`, `~`, dan `-` pada posisi internal yang
diizinkan, tetapi belum menerima plus-addressing seperti `user+tag@example.com`.

Password dihitung berdasarkan **byte**, tidak pernah di-trim atau diubah.
Register tidak mewajibkan kombinasi huruf besar, angka, atau simbol;
`aaaaaa`, `123456`, dan `!!!!!!` lolos validasi lokal. Login tetap menerima
spasi, Unicode, dan karakter kontrol yang dapat direpresentasikan setelah
parsing jika panjangnya valid. Contohnya, tiga `é` berukuran 6 byte: lolos
validasi lokal login, tetapi ditolak register.

Seluruh kegagalan validasi dikumpulkan menjadi HTTP `400`, code `BR`, dengan
daftar pesan per field pada `errors` dan `data: []`, tanpa memanggil service.
Pesan mengikuti locale request yang tersedia (`en`, `id`, `zh-CN`). Nilai
kosong mendapat pesan wajib dan minimum; kegagalan panjang dan karakter
dikembalikan bersama jika keduanya berlaku. Input yang lolos lokal tetap
mengikuti hasil service, termasuk HTTP `409` untuk email yang sudah digunakan
atau `401` untuk login gagal.

Request cukup berisi email/password. `confirm_password` tidak diwajibkan;
field tambahan tetap mengikuti parser existing dan tidak divalidasi atau
diteruskan sebagai field baru. Response sukses tetap HTTP `200`, code `SS`,
message `Register Successfully` / `Login Successfully`; data lowercase mengikuti bagian Browser Session Lifecycle. Login memerlukan allowed Origin dan JSON content type.

Malformed JSON atau tipe field yang tidak sesuai menghasilkan HTTP `400`.
`GET /api/v1/users` melewati authentication `Me` dan pemeriksaan role `admin`.
`GET /api/v1/me` memakai authentication yang sama tanpa gate admin.

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

Konfigurasi runtime hanya dibaca dari environment variable. Seluruh key yang
didukung didaftarkan di `.env.example` dan dipetakan oleh
`internal/config/environment.go`. Nilai wajib yang hilang atau malformed
membuat aplikasi gagal saat startup.

Contoh override saat menjalankan aplikasi:

```bash
APP_PORT=9090 \
AUTH_GRPC_TARGET=localhost:50051 \
go run ./cmd/api
```

File `.env` tidak dibaca otomatis oleh aplikasi; export nilainya melalui shell,
container runtime, CI/CD, atau secret manager.

### Autentikasi Caller grpc-auth

Auth client wajib memakai `AUTH_GRPC_TLS_ENABLED=true` dan
`AUTH_GRPC_SERVICE_TOKEN` dari environment server. Konfigurasi lama dengan TLS
disabled atau tanpa token ditolak sebelum aplikasi berjalan, termasuk saat
config dibuat langsung melalui constructor aplikasi. Tidak ada bypass development.

Token adalah opaque credential: canonical base64url **tanpa padding** dari
32–256 byte acak yang diprovisikan di luar repository. Jangan trim token,
memasukkannya ke source/log, atau mengambilnya dari header/body browser.
Operator mendaftarkan SHA-256 dari **encoded string** pada konfigurasi caller
backend beserta ServiceID `rest-orc`, expiry, exact method allowlist, dan
namespace yang diperlukan. Format valid secara lokal belum membuktikan token
terdaftar atau belum expired.

RPC yang dipakai saat ini adalah `/user.v1.UserService/Register`,
`/user.v1.UserService/Login`, `/user.v1.UserService/Me`, dan
`/user.v1.UserService/ListUsers`. Backend menetapkan grants dan namespace;
ListUsers membutuhkan `iam` serta permission pengguna yang sesuai. Jangan
memakai wildcard atau metadata service-name/namespace sebagai pengganti grants.

Reusable auth connection mengirim tepat satu `x-service-token` pada setiap RPC
melalui secure call credentials. `authorization: Bearer <user token>` tetap
terpisah; Register/Login publik tetap tidak memerlukan bearer pengguna.
Outgoing context yang sudah memuat `x-service-token` ditolak sebelum dispatch
agar tidak mengirim credential ganda. Locale, request metadata, deadline parent,
dan cancellation dipertahankan.

TLS auth memakai minimum versi 1.3 dan verifikasi CA, masa berlaku certificate,
serta hostname/SAN. `AUTH_GRPC_TLS_SERVER_NAME` opsional untuk override hostname;
jika kosong, grpc-go memverifikasi target authority. `AUTH_GRPC_TLS_CA_FILE`
opsional menambah CA ke system roots; path/PEM invalid menyebabkan error.
Tidak ada fallback plaintext atau `InsecureSkipVerify`. TLS client HTTP generik
tetap memakai kebijakan existing.

Connection dibuat sekali dan bersifat lazy. Startup sukses belum membuktikan
handshake, penerimaan credential, atau authority backend; kegagalan tersebut
terlihat pada RPC pertama dan mengikuti mapper error existing tanpa retry baru.
Untuk rotasi, operator menyiapkan current/previous overlap dan expiry di backend,
mengganti environment lalu me-restart aplikasi. Client yang sudah berjalan
mempertahankan credential startup; tidak ada hot reload atau fallback token.

ProfileService memakai connection, TLS, credential, dan timeout auth yang sama.
Operator perlu memberi credential caller existing grant untuk full method
`/profile.v1.ProfileService/UpdateProfile`. Startup tidak mem-provision grant;
penolakan backend tetap melewati mapper error existing. Tidak ada env key baru.
`NewWithServices` menyediakan injection UserService/ProfileService untuk test;
constructor dan interface UserService existing tetap tersedia.

## Menggunakan Client Reusable

Koneksi gRPC untuk service yang berbeda dibuat dengan constructor yang sama:

```go
authConnection, err := grpcclient.New(cfg.Clients.AuthGRPC)
```

Service-specific auth client dibentuk dari koneksi tersebut:

```go
authClient, err := authservice.New(authConnection)
```

Satu `authClient` di auth_client.go diinjeksikan ke middleware untuk `Me`, ke
`features/auth/gateway/authservice` untuk Register/Login, dan ke
`features/users/gateway/authservice` untuk `ListUsers` dan `UpdateProfile`. Masing-masing consumer tetap
bergantung pada interface kecil sesuai kemampuan yang dipakainya.

Setelah field konfigurasi service ditambahkan, HTTP service menggunakan generic
HTTP client:

```go
productClient, err := httpclient.New(cfg.Clients.ProductHTTP)
```

Satu instance client dibuat per konfigurasi service, digunakan kembali untuk
semua request, lalu ditutup saat shutdown. Client tidak dibuat per request.

## Mengganti Password Sendiri

`POST /api/v1/me/password` menerima JSON `current_password` dan `new_password`
pada feature auth; DTO HTTP tetap berada di auth/domain.

- Keduanya wajib dan panjangnya 6–50 **byte**. Omitted/null/empty menghasilkan
  required dan minimum-length errors; malformed JSON atau tipe dikenal yang
  salah menghasilkan 400. Unknown fields diabaikan; confirm_password tidak wajib.
- Current password diteruskan persis, termasuk spasi, Unicode, atau kontrol
  credential lama; tidak di-trim, diubah kapitalisasinya, atau di-hash lokal.
- New password hanya menerima byte `0x21–0x7E`, mengikuti register tanpa syarat
  komposisi. Nilainya harus berbeda secara persis dari current password.
- Validasi mengumpulkan error kedua field dengan pesan en/id/zh dari
  pkg/dictionary; credential tidak dimasukkan ke pesan error.

Password change memakai identitas principal hasil Me. Restricted user boleh
mengganti password tanpa gate admin/permission tambahan. Usecase memvalidasi
input sebelum mutation; backend memverifikasi current password dan ownership
session secara authoritative, lalu mencabut seluruh sesi dan refresh credential.
Tidak ada hashing, penyimpanan JWT/session, retry, atau revocation lokal.

RPC memakai `/auth.v1.AuthService/ChangePassword` dari AuthService SDK pada
connection auth existing. Operator perlu memastikan credential caller mempunyai
grant method tersebut dan namespace `iam` di backend. REST tidak menerima
namespace/service identity dari browser dan tidak menambah environment key.
`NewWithPasswordService` menyediakan injection SDK tambahan; constructor
New/NewWithService/NewWithServices dan interface UserService/ProfileService
existing tetap tersedia.

Gateway password memetakan hanya dua string request serta dua flag backend,
`changed` dan `login_required`, ke DTO domain tanpa generated message di HTTP.
Kedua boolean tetap ditulis, termasuk false; tidak ada token/version buatan.
Nil response dependency menjadi safe 500, dan error tetap melalui shared mapper.

Response sukses memakai `Cache-Control: no-store`:

```json
{"code":"SS","message":"Password changed successfully.","data":{"changed":true,"login_required":true}}
```

Jika backend mengembalikan `login_required=true`, handler menghapus cookie
host-only `__Secure-refresh` dan `__Secure-csrf` dengan value kosong dan instruksi penghapusan (`Max-Age=0` atau expiry di masa lalu),
`Secure; SameSite=Lax; Path=/api/v1/auth`, tanpa Domain; hanya refresh HttpOnly. Ini tetap
dilakukan tanpa incoming cookie pada `/me/password`; cookie tidak dibaca atau
dikirim sebagai credential RPC. Error atau `login_required=false` tidak
mengubah cookie. FE menghapus access token memory dan meminta login ulang.
Login/Refresh menerbitkan cookie melalui lifecycle terpisah; password hanya menghapus refresh dan CSRF saat login_required=true. Tidak ada credential/session baru yang disimpan lokal.

Route password dipasang oleh auth pada satu shared authenticated `/me` group,
bersama auth GET Me dan users PATCH profile. Tidak ada middleware Me kedua
atau gate admin. Public Register/Login tetap terdaftar sebelum admin `/users`;
GET Me tetap menerima restricted dan PATCH profile tetap menolaknya.

Contoh request:

```http
POST /api/v1/me/password
Authorization: Bearer <access_token>
Content-Type: application/json

{"current_password":"existing-password","new_password":"replacement-password"}
```

Verifikasi unit dan HTTP integration password:

```bash
rtk proxy env GOWORK=off go test -mod=readonly ./test/features/auth/... ./test/client/authservice/... ./test/dictionary/... ./test/app/... ./test/integration/auth/... ./test/integration/users/...
rtk proxy env GOWORK=off go test -mod=readonly -race ./test/integration/authservice/... ./test/integration/auth/... ./test/integration/users/... ./test/auth/... ./test/middleware/auth/...
```

## Memperbarui Profil Sendiri

`PATCH /api/v1/me/profile` dimiliki feature users, dengan DTO HTTP di domain.
Field yang dapat dipilih adalah `name`, `address`, `phone`, dan `country_code`.
Field omitted/null dan unknown diabaikan; string kosong tetap dipilih dan invalid.
JSON malformed atau tipe field dikenal yang salah menghasilkan 400.

- Name: ASCII letters, spasi biasa, apostrophe, dan hyphen; trim/collapse spasi,
  kapitalisasi awal kata setelah spasi/apostrophe/hyphen tanpa menurunkan huruf
  lainnya. Panjang setelah normalisasi 3–100 byte. `  NASA  o'neil-jean  `
  menjadi `NASA O'Neil-Jean`. Karakter invalid ditolak sebelum normalisasi.
- Country code: 1–3 digit ASCII ditambah `+`, hasil harus cocok
  `^\+[1-9][0-9]{0,2}$`; tanpa trim atau tebakan negara.
- Phone: 12–20 digit ASCII setelah normalisasi. Hanya jika request menyertakan
  country code valid `+62`, hapus satu prefix `+62` atau `62`, lalu maksimal
  satu leading `0`. Tanpa country code tersebut, nilai tidak diubah atau
  ditebak dari Me. Country-code-only tidak memilih phone.
- Address: 3–200 byte, printable ASCII serta CR/LF; isi termasuk spasi/newline
  diteruskan persis. Tab, kontrol lainnya, Unicode, dan digit Unicode ditolak
  sesuai allowlist masing-masing field.

Error field lokal teragregasi, memakai catalog en/id/zh, dan menghasilkan 400.
Usecase memeriksa identity/restriction sebelum validasi atau local no-op.
Restricted user mendapat 403 termasuk untuk `{}`; non-admin yang eligible boleh
memperbarui profil sendiri. Hasil mutation harus milik principal terverifikasi.

Input tanpa field efektif (`{}`, JSON null, unknown-only, atau null-only)
menghasilkan 200 dengan data tepat `{"changed":false}`, tanpa mutation RPC,
profile/version buatan, audit transaksi, atau klaim versi backend terbaru.
Selected field tetap memanggil backend walaupun sama dengan nilai dari Me.
Backend `changed=false` tetap mengembalikan profile dan version authoritative.

Mutation sukses memakai shared envelope berikut (version string tetap presisi):

```json
{"code":"SS","message":"Profile updated successfully.","data":{"changed":true,"version":"9007199254740993","profile":{"id":9,"user_id":17,"name":"NASA O'Neil","address":"Jakarta","phone":"812345678901","country_code":"+62"}}}
```

Local no-op memakai envelope yang sama dengan data tepat `{"changed":false}`.
Identitas diperoleh dari principal hasil satu Me; field mask eksplisit dibangun
gateway dengan urutan name/address/phone/country_code. Target ID, mask, privilege,
namespace, Version, dan Idempotency-Key dari browser tidak mengubah authority
atau menjadi precondition. Backend memverifikasi session/owner saat mutation.

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

Jalankan integration test auth, users, dan transport:

```bash
rtk proxy env GOWORK=off go test -mod=readonly ./test/integration/auth/... ./test/integration/users/... ./test/integration/authservice/...
```

Runner/scenario/mock mengikuti ownership feature pada `test/features/auth` dan
`test/features/users`. Setup HTTP yang dipakai auth, users, serta TLS wire tests
berada di `test/shared/fixtures/app.go`; golden auth bersama tetap di shared fixtures.

Verifikasi profile mencakup domain/catalog, usecase, client, gateway, handler,
HTTP integration, serta TLS bufconn ProfileService pada connection auth yang sama:

```bash
rtk proxy env GOWORK=off go test -mod=readonly ./test/features/users/... ./test/client/authservice/... ./test/dictionary/... ./test/app/... ./test/integration/users/... ./test/integration/auth/...
rtk proxy env GOWORK=off go test -mod=readonly -race ./test/integration/authservice/... ./test/integration/users/... ./test/auth/... ./test/middleware/auth/...
```

Jalankan race detector:

```bash
go test -race ./...
```

Seluruh runner, scenario, mock, dan fixture test berada di root `test/`. Tidak
ada file `*_test.go` di dalam `cmd/`, `internal/`, atau `pkg/`:

```text
test/<layer>/<module>/runner_test.go
    └── test_scenarios.All()
            ├── success scenario
            ├── validation scenario
            └── dependency error scenario
```

Runner hanya menjadi entrypoint toolchain. Scenario, mock, dan fixture
ditempatkan pada folder terpisah agar test tetap mudah dibaca dan dirawat.

### Verifikasi Standalone

Jika parent directory memiliki `go.work`, command Go dapat memakai source dari
sibling repository sehingga pin dependency yang tidak kompatibel di `go.mod`
tidak terdeteksi. Gunakan `GOWORK=off` dari root repository untuk memverifikasi
versi dependency terpublikasi di `go.mod` dan `go.sum` tanpa sibling checkout:

```bash
rtk proxy env GOWORK=off go list -mod=readonly -m -json all
rtk proxy env GOWORK=off go build -mod=readonly ./...
rtk proxy env GOWORK=off go test -mod=readonly ./...
rtk proxy env GOWORK=off go vet -mod=readonly ./...
rtk proxy env GOWORK=off go mod verify
rtk proxy env GOWORK=off go mod tidy -diff
```

Pada output module list, dependency harus memiliki versi remote dan tidak
memakai local `Replace` atau sibling sebagai main module. `-mod=readonly`
mencegah build, test, dan vet mengubah dependency files secara diam-diam.
`go mod verify` memeriksa integritas module cache; `go mod tidy -diff` harus
menghasilkan diff kosong dan exit code 0. Akses module registry diperlukan
jika dependency belum tersedia di cache.

`GOWORK=off` hanya mengatur resolusi module untuk command Go tersebut, bukan
konfigurasi runtime aplikasi. Jika perlu mengubah dependency, jalankan tidy
dengan `GOWORK=off`, lalu review perubahan `go.mod` dan `go.sum` sebelum
mengulangi verifikasi di atas.

## Catatan Eksperimen

Beberapa bagian masih sengaja sederhana agar mudah dipelajari:

- Belum ada retry karena keamanannya bergantung pada idempotensi operasi.
- Belum ada circuit breaker, tracing, metrics, atau structured application log.
- Feature auth memiliki public Register/Login pada URL legacy users dan authenticated Me;
  feature users memiliki admin ListUsers dan self-profile PATCH. Filter tanggal/role dan pencarian keyword
  sudah diekspos, sedangkan field
  search, mode search, sort, dan field selection tetap ditentukan downstream.
- Proxy atau load balancer aktual belum menjadi bagian repository.
- Kontrak error bisnis dan response API belum difinalisasi.

Perubahan eksperimen sebaiknya tetap disertai test dan tidak mengorbankan
context propagation, deterministic behavior, atau resource cleanup.

## Browser Session Lifecycle

Startup memerlukan `AUTH_BROWSER_ALLOWED_ORIGINS` (origin HTTPS persis, dipisah koma, tanpa path/wildcard) dan `AUTH_BROWSER_CSRF_SECRET` (base64url kanonis tanpa padding dari tepat 32 byte). Secret harus terpisah dari service token dan sama antar-instance, tanpa default/dev bypass/reload otomatis. Semua key ada di `.env.example`; provision secret di luar repository. Direct configuration divalidasi sebelum resource dibuka. Error konfigurasi menyebut field tanpa nilainya. Deployment FE/API memakai HTTPS dalam satu site.

Status lokal lifecycle: credential hilang/bearer malformed 401, Origin/CSRF ditolak 403, content type Login unsupported 415, JSON malformed 400, hasil dependency/internal generation invalid 500. Shared response mapper mempertahankan locale, explicit downstream status/field errors dan Retry-After; tidak ada retry otomatis atau credential dalam diagnostics.

Lifecycle memisahkan DTO HTTP dari hasil sesi internal. Register mengembalikan `token:""`, `registration_id`, `registration_status:"PENDING"`; akun menunggu aktivasi, tanpa auto-login/cookie. Login/Refresh mengembalikan `token`, `token_type:"Bearer"`, `session_id`, `access_expires_at`, `session_expires_at` (RFC3339 UTC), dan `must_change_password` termasuk false. **Breaking:** `data.Token` diganti `data.token`, tanpa alias. Raw refresh hanya lewat HttpOnly cookie. Usecase Refresh/Logout memanggil satu gateway dengan satu mode proof; validitas sesi, restriction, reuse dan revocation tetap authoritative di grpc-auth, tanpa Me/retry/fallback/local token decoding.

Session SDK wrappers memakai auth connection existing untuk `/auth.v1.AuthService/RefreshSession` dan `/auth.v1.AuthService/Logout`; operator perlu method grants dan namespace IAM. Cookie mode menghapus outgoing authorization lewat salinan metadata; bearer Logout memakai tepat satu authorization dan optional refresh field absent. Locale/request metadata, deadline/cancellation dan satu x-service-token tetap, tanpa mengubah context caller atau retry.

Cookie refresh `__Secure-refresh` adalah host-only HttpOnly/Secure/SameSite=Lax, Path=/api/v1/auth; `__Secure-csrf` memakai atribut sama tanpa HttpOnly. Login/Refresh membatasi expiry/Max-Age ke absolute session expiry, bukan access TTL; hasil invalid tidak memasang cookie parsial. CSRF memakai HMAC-SHA256 atas length-prefixed version/nonce/digest refresh dengan nonce crypto/rand 32 byte; token `v1.<nonce>.<mac>` tidak mengungkap refresh/digest. Cookie dan X-CSRF-Token harus sama dan signature/binding valid. Nonce disiapkan sebelum RPC mutation. Bootstrap hanya menerbitkan CSRF session-cookie dan JSON token, tidak memverifikasi sesi atau memperpanjang refresh. Login memerlukan exact allowed Origin + application/json (parameter charset diperbolehkan); cookie Refresh/Logout dan bootstrap juga memerlukan Origin. Bearer Logout tanpa Origin tidak memerlukan cookie/CSRF.

Routes lifecycle (tanpa middleware Me):

| Method | URL | Proof |
| --- | --- | --- |
| GET | /api/v1/auth/csrf | allowed Origin + refresh cookie; zero RPC |
| POST | /api/v1/auth/refresh | allowed Origin + refresh/CSRF cookies + X-CSRF-Token |
| POST | /api/v1/auth/logout | Authorization hadir memilih bearer; tanpa header memilih cookie + Origin/CSRF |

Header Authorization kosong/malformed/berulang pada Logout ditolak 401 dan tidak fallback. Bearer valid mengabaikan cookie untuk RPC; mode cookie menghapus outgoing authorization. Refresh mengabaikan bearer, termasuk expired. Body/query tidak menjadi sumber credential; request tanpa body diterima. Bootstrap data hanya `{"csrf_token":"<signed proof>"}`. Refresh memakai message `Session refreshed successfully.`, Logout `Logout successfully.` dengan data `{"changed":false}` atau true dari backend. Semua lifecycle response/error/preflight memakai Cache-Control no-store. Logout sukses termasuk changed=false menghapus kedua cookie; change password hanya pada success login_required=true. Error/nil result atau login_required=false tidak mengubah cookie.

CORS /api/v1 memakai exact allowlist, credentials=true, Vary Origin, application methods dan header Authorization/Content-Type/X-CSRF-Token/Version/Idempotency-Key/Accept-Language/X-Language/X-Request-ID. Preflight selesai sebelum authentication tanpa RPC/cookie. Origin null/unlisted/duplicate ditolak. FE memakai credentials: include, menyimpan access token hanya di memory, mengambil CSRF lewat bootstrap setelah reload (juga untuk beda origin dalam satu site), serta single-flight refresh termasuk antar-tab. Rotasi tidak di-retry otomatis; bila response hilang setelah commit, login ulang. Mutasi bisnis juga tidak di-retry otomatis sesudah refresh. Secret CSRF berbeda antar-instance memerlukan bootstrap ulang; gunakan secret yang sama.

Verifikasi lifecycle dengan fake SDK dan TLS bufconn (tanpa real backend/database):

```bash
rtk proxy env GOWORK=off go test -mod=readonly ./test/features/auth/... ./test/client/authservice/... ./test/config/... ./test/app/... ./test/integration/auth/... ./test/integration/users/... ./test/integration/authservice/...
rtk proxy env GOWORK=off go test -mod=readonly -race ./test/features/auth/... ./test/client/authservice/... ./test/integration/authservice/... ./test/integration/auth/... ./test/integration/users/... ./test/auth/... ./test/middleware/...
```

TLS wire tests menguji full method, optional presence Logout, satu service credential, deadline/cancellation, metadata/cookie isolation dan mixed concurrent Refresh/Logout/Me pada connection yang sama. Backend outcomes reuse/revocation/restriction disimulasikan untuk memastikan mapping; ini tidak menggantikan test transaksi backend atau smoke test browser FE. Tidak ada deployment atau perubahan operator grants otomatis.

## My Session Management

Self-service sesi dimiliki feature auth pada shared authenticated `/me` group:

| Method | URL | Business RPC setelah satu Me |
| --- | --- | --- |
| GET | `/api/v1/me/sessions` | `/session.v1.SessionService/ListMySessions` |
| DELETE | `/api/v1/me/sessions/{session_id}` | `/session.v1.SessionService/RevokeMySession` |
| POST | `/api/v1/me/sessions/revoke-all` | `/session.v1.SessionService/RevokeAllMySessions` |

Semua operasi memerlukan bearer aktif dan principal unrestricted. User non-admin
boleh mengelola sesi sendiri; `must_change_password=true` ditolak 403 tanpa RPC
sesi. Missing bearer ditolak 401 tanpa RPC; kegagalan Me menghentikan alur. Tidak
ada Me/permission lookup kedua, cookie credential fallback, atau session state
lokal. Grant tiga exact methods di atas dan namespace IAM tetap dikelola operator;
service token tetap satu credential dari environment dengan verified TLS1.3 dan
shared connection existing. Tidak ada runtime key atau dependency baru.

List hanya menerima `page` (default 1, maksimum 2147483647), `limit` (default 20,
1–100), dan `last_id` (default kosong, maksimum 2048 byte sesudah satu URL decoding).
Page/limit yang hadir wajib nonempty ASCII digits; zero, overflow, whitespace,
atau duplicate page/limit/last_id ditolak 400. Cursor diteruskan opaque tanpa trim
atau decoding isi. Urutan server tetap `created_at DESC, id DESC`; gunakan
`next_cursor` untuk traversal dan reset cursor saat mengubah limit. `page` adalah
metadata, bukan janji offset scan. `sort`, `fields`, `search`, `keyword`, `statuses`,
`application_code`, `created_from`, dan `created_to` ditolak bila hadir, termasuk
nilai kosong. Body ketiga operasi tidak dibaca; claim user/namespace/service,
Version, dan Idempotency-Key tidak menjadi target, credential, atau precondition.

Contoh list success (total int64 dan metadata backend dipertahankan, termasuk
nonzero total pada empty page; items selalu array):

```json
{"code":"SS","message":"Sessions retrieved successfully.","data":{"items":[{"id":"abcdefab-abcd-abcd-abcd-abcdefabcdef","user_id":17,"status":"ACTIVE","created_at":"2020-01-01T00:00:00.123456789Z","last_seen_at":null,"expires_at":"2020-01-08T00:00:00Z","revoked_at":null,"device_label":"","user_agent":"","ip_address":"","is_current":false}],"total":2147483648,"page":1,"limit":20,"total_pages":107374183,"has_next":true,"has_prev":true,"next_cursor":"opaque-next"}}
```

Item hanya berisi safe fields pada contoh. Optional timestamps absent menjadi
null; false/empty text tetap eksplisit. Status backend ACTIVE/REVOKED berbeda
dari expiry; historical expired/revoked sessions tetap valid untuk ditampilkan.
Device/IP hanya metadata tampilan. Nil/invalid response atau owner mismatch
menghasilkan safe 500 tanpa partial data; credentials, hashes, dan auth_version
bukan output.

Single revoke menerima UUID lowercase canonical dengan hyphens, tanpa trim atau
case conversion, dan membiarkan backend memeriksa existence/ownership. Success:

```json
{"code":"SS","message":"Session revoked successfully.","data":{"changed":false}}
```

`changed` bisa true/false sesuai backend. RPC ini tidak mengembalikan flag sesi
saat ini: REST tidak menambahkan `login_required`, lookup tambahan, atau cookie
cleanup, termasuk saat sesi yang dicabut sebenarnya current. FE menggunakan
`is_current` dari list sebelumnya untuk membersihkan access token memory;
protected request selanjutnya tetap diverifikasi backend.

Revoke-all menerima empty body dan mengirim empty business request. Success:

```json
{"code":"SS","message":"All sessions revoked successfully.","data":{"changed":true,"login_required":true}}
```

Kedua boolean diteruskan termasuk false. Hanya success `login_required=true`
yang menghapus `__Secure-refresh` dan `__Secure-csrf`, host-only Secure/SameSite=Lax,
Path=/api/v1/auth, refresh saja HttpOnly. Cleanup bekerja tanpa incoming cookie
pada `/me`, tidak membaca cookie atau memerlukan CSRF untuk bearer operation ini.
FE membersihkan access token memory dan kembali login bila flag tersebut true.
Error/nil response/login_required=false tidak mengubah cookie. Tidak ada token
pengganti atau automatic login.

Seluruh response/error/preflight ketiga URL memakai Cache-Control no-store,
termasuk early CORS/auth denial. Exact-origin CORS policy existing tetap berlaku.
Backend errors/status/field messages/Retry-After dan deadline/cancellation memakai
shared mapper, tanpa retry atau success-only browser effects saat gagal. Tests
menggunakan fake SDK, TLS bufconn, dan concurrent/race scenarios; real backend/FE
smoke test dan deployment dilakukan terpisah setelah diotorisasi.
