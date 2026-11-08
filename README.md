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

## Admin Session Management

Administrasi sesi user target dimiliki feature users. Satu Me memverifikasi actor;
user target berasal dari path dan tidak menggantikan identitas bearer. Semua route
memerlukan bearer aktif dan `must_change_password=false`.

| Method | Route | Exact permission |
| --- | --- | --- |
| GET | `/api/v1/users/{user_id}/sessions` | `iam.session.read=true` |
| DELETE | `/api/v1/users/{user_id}/sessions/{session_id}` | `iam.session.revoke=true` |
| POST | `/api/v1/users/{user_id}/sessions/revoke-all` | `iam.session.revoke=true` |

Non-admin dengan permission tepat boleh mengakses; role admin, wildcard, permission
false, atau klaim browser tidak menggantikan permission. ListUsers tetap memakai
gate role admin, sedangkan `/api/v1/me/sessions` tetap memakai policy sesi sendiri.

| Input | Validasi |
| --- | --- |
| `user_id` path | Digit ASCII, 1–2147483647; leading zeros diterima |
| `session_id` path | UUID kanonis lowercase, termasuk all-zero |
| `page` / `limit` | Default 1/20; 1–2147483647 / 1–100; duplicate ditolak |
| `last_id` | Opaque, sekali URL-decoded, maksimal 2048 byte; duplicate ditolak |
| `Idempotency-Key` | Kedua mutation: tepat satu header, 1–200 byte ASCII tercetak, tidak seluruh spasi; internal space/comma dipertahankan |
| `Version` | Revoke-all: tepat satu header digit ASCII, 1–9223372036854775807; leading zeros diterima |

List menolak sort/fields/search/keyword/statuses/application_code/created_from/created_to
termasuk nilai kosong; unknown query claims diabaikan. Body semua operasi diabaikan,
termasuk malformed JSON. Target, key, dan version tidak diambil dari body/query.

Contoh list, dengan urutan fixed `created_at DESC, id DESC` dan total `int64`:

```json
{"code":"SS","message":"User sessions retrieved successfully.","data":{"items":[{"id":"abcdefab-abcd-abcd-abcd-abcdefabcdef","user_id":23,"status":"ACTIVE","created_at":"2020-01-01T00:00:00.123456789Z","last_seen_at":null,"expires_at":"2020-01-08T00:00:00Z","revoked_at":null,"device_label":"","user_agent":"","ip_address":"","is_current":false}],"total":2147483648,"page":1,"limit":20,"total_pages":107374183,"has_next":true,"has_prev":true,"next_cursor":"opaque-next"}}
```

Status/timestamp historis, backend `is_current`, flags dan cursor dipertahankan;
owner row wajib sama dengan target. Ubah target/limit berarti caller perlu reset
cursor sesuai binding backend. Empty page memakai `items:[]` tanpa mereset total.

Contoh mutation success (HTTP 200), termasuk `changed=false`:

```json
{"code":"SS","message":"User session revoked successfully.","data":{"changed":false}}
{"code":"SS","message":"All user sessions revoked successfully.","data":{"changed":false,"version":"9007199254740993"}}
```

Revoke-all membutuhkan **entity Version user target** dari authoritative read yang
caller miliki. Versi Me adalah versi actor dan tidak dipakai sebagai target version.
`GET /api/v1/users/{user_id}` menyediakan sumber versi melalui capability User Detail
dengan exact `iam.user.read`; ListUsers tidak diperluas.
Version response berupa string desimal backend, termasuk nilai di atas 2^53, tanpa
menghitung input+1 atau menolak committed replay berdasarkan perbandingan versi.
Single revoke/list mengabaikan Version; list mengabaikan Idempotency-Key.

Setiap retry oleh caller menjalankan satu Me dan satu eligible business RPC. Backend
memiliki replay, fingerprint, current-authority checks, version/idempotency conflict,
audit dan transaksi. Tidak ada local replay cache atau automatic retry/version refresh.
Setelah response hilang, caller dapat mengulang request/key yang sama sesuai kontrak
backend; conflict 409 memerlukan caller menyelesaikan konflik, bukan retry otomatis
oleh REST. Status/code/field errors/Retry-After, 429/503, timeout 504 dan cancel 499
mengikuti mapper existing.

Semua response, error dan preflight memakai `Cache-Control: no-store`. Operasi admin
tidak memasang/menghapus cookie, menerbitkan credential, atau memerlukan CSRF, termasuk
target=actor, replay, dan changed=false. Tidak ada inferred login_required atau current
session outcome; request berikutnya tetap diverifikasi backend. `/users/me/sessions`
bukan alias self-session: literal `me` adalah target ID invalid. URL dan policy
`/api/v1/me/sessions`, public Register/Login, password dan profile tetap.

Shared generated SDK memakai satu verified TLS1.3 connection, service token dari env,
bearer asli, locale/request ID dan parent deadline/cancellation. Operator perlu grants
`/session.v1.SessionService/ListUserSessions`, `RevokeUserSession`, dan
`RevokeAllUserSessions`, namespace IAM, serta exact user permissions. Tidak ada grant
otomatis berdasarkan role. HTTP integration memakai fake SDK, wire tests memakai TLS
bufconn; smoke test backend/FE nyata dan deployment berada di luar verifikasi ini.

```sh
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache go test ./test/features/users/... ./test/client/authservice/... ./test/integration/users/... ./test/integration/authservice/... ./test/app/...
```

## User Detail

`GET /api/v1/users/{user_id}` adalah pembacaan detail target milik feature users.
Path `user_id` wajib digit ASCII dalam 1–2147483647; leading zeros diterima secara
numerik, tanpa trim/sign/hex/overflow. Body, query, Version dan Idempotency-Key
diabaikan, termasuk body malformed dan mutation headers berulang/tidak valid.

| Actor | Detail |
| --- | --- |
| Unrestricted dengan exact `iam.user.read=true`, termasuk non-admin | Satu Me lalu satu GetUser |
| Restricted, permission missing/false/wildcard, role-admin saja, session permission saja | 403 tanpa GetUser |
| Bearer absen / Me gagal | 401 / safe mapped error; tanpa GetUser |

Actor dan target independen, termasuk bila ID sama. Detail mengembalikan hanya
`id`, `email`, `name`, `address`, `phone`, `country_code`, `status`, `version`,
`created_at`, `must_change_password`, `roles`, dan `application_access`.
Version adalah entity version target sebagai string desimal positif int64;
creation time wajib valid dan mempertahankan fractional precision dalam UTC.
Assignments memiliki numeric role_id/application_id dan code/name/status serta
role_status/application_status; arrays kosong ditulis `[]`. Stored profile text,
assignment order/status dan restriction flag target dipertahankan. Tidak ada
password/dob/token/hash/auth_version, legacy role codes/names atau computed
permissions pada DTO; assignments target tidak memberikan authority ke actor.

Contoh response HTTP 200:

```json
{"code":"SS","message":"User retrieved successfully.","data":{"id":23,"email":"target@example.com","name":" Legacy NAME ","address":"Legacy\taddress","phone":"+62+628123","country_code":"+62","status":"ACTIVE","version":"9007199254740993","created_at":"2026-10-20T08:09:10.123456789Z","must_change_password":false,"roles":[{"role_id":11,"code":"iam.reader","name":"Reader","status":"REVOKED","role_status":"DISABLED"},{"role_id":9,"code":"clm.member","name":"Member","status":"ACTIVE","role_status":"ACTIVE"}],"application_access":[{"application_id":5,"code":"clm","name":"CLM","status":"REVOKED","application_status":"DISABLED"}]}}
```

Account status target boleh PENDING/ACTIVE/SUSPENDED/DEACTIVATED dan target
must_change_password=true tetap dapat dibaca oleh actor yang eligible. Ini bukan
policy restricted actor pada Me; informasi assignment backend tidak dihapus atau
diubah menjadi effective permissions. Nil/malformed success, target ID berbeda,
version di luar 1–MaxInt64, missing/invalid creation timestamp, atau nil/invalid
assignment ID gagal aman sebagai 500 tanpa partial detail.

Gateway memakai salinan metadata, menghapus inherited version/idempotency-key
pada read, dan mempertahankan bearer/locale/request ID/deadline/cancellation.
Shared UserDetailService membungkus generated GetUser pada connection existing,
tanpa memperlebar interface Service atau mengubah constructor lama.

Detail memakai authentication per route sebelum legacy role-admin users group;
ListUsers tetap memakai gate role yang sama. `/api/v1/me` tetap policy dan DTO
actor sendiri; `/api/v1/users/me` adalah ID target invalid, bukan self alias.
Public Register/Login serta profile/password/lifecycle/self/admin-session tetap.
Semua success/error/preflight detail memakai `Cache-Control: no-store`, termasuk
failed Me, invalid ID dan CORS rejection sebelum handler. Cookie tidak menjadi
fallback bearer dan detail tidak memerlukan CSRF atau memasang/menghapus cookie,
termasuk target=self. Backend error/status/code/locale/field/Retry-After memakai
mapper existing tanpa retry, cached detail atau fallback Me/ListUsers.

Caller dapat membaca detail target 23 lalu meneruskan `data.version` persis sebagai
header `Version` pada request revoke-all terpisah untuk target yang sama, bersama
`Idempotency-Key` milik mutation. Contoh urutan (access token tetap di memory caller):

```http
GET /api/v1/users/23
Authorization: Bearer <access-token>
```

```http
POST /api/v1/users/23/sessions/revoke-all
Authorization: Bearer <access-token>
Version: 9007199254740993
Idempotency-Key: user-23-revoke-request-1
```

Detail memerlukan `iam.user.read`, revoke-all memerlukan `iam.session.revoke`.
Read permission tidak memberikan session permission dan sebaliknya. Tiap request
menjalankan satu Me dan satu business RPC; mutation tidak melakukan GetUser internal.
Dua request ini tidak atomik: target dapat berubah setelah read dan menyebabkan
backend conflict 409. REST tidak mengganti versi dengan Me actor, menambah versi,
refetch detail atau retry mutation otomatis. Setiap GET memperoleh hasil baru,
tanpa cross-request cache; committed replay mutation mengikuti kontrak backend.

Operator menyiapkan exact method grant `/user.v1.UserService/GetUser`, namespace IAM,
dan permission user-read. Connection tetap TLS1.3 dengan satu env-owned service
credential dan deadline/cancellation existing. Pengujian fake SDK dan generated SDK
TLS bufconn tidak menggantikan smoke test backend/FE nyata atau deployment.

```sh
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache go test ./test/features/users/... ./test/client/authservice/... ./test/integration/users/... ./test/integration/authservice/... ./test/app/...
```

Verifikasi concurrency detail (2026-11-04): command berikut lulus 1.012 test tanpa
race, termasuk mixed detail/admin/self-session/public auth/ListUsers pada satu
TLS connection dengan actor, target, permission, version dan metadata berbeda.

```sh
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache go test -race ./test/features/users/... ./test/client/authservice/... ./test/integration/users/... ./test/integration/authservice/... ./test/integration/auth/... ./test/app/... ./test/features/auth/handler/...
```

### User status management

`PATCH /api/v1/users/{user_id}/status` requires bearer authentication, an unrestricted actor and exact `iam.user.status.update=true`. Roles and read/session permissions do not grant this operation. The actor from Me and the path target are independent.

| Input | Contract |
| --- | --- |
| user_id path | ASCII decimal 1–2147483647; leading zeros accepted |
| Version header | Exactly one ASCII decimal 1–9223372036854775807; target entity version, never Me.version |
| Idempotency-Key header | Exactly one nonblank printable ASCII value, 1–200 bytes; opaque spaces/commas preserved |
| status JSON string | Required, exactly PENDING, ACTIVE, SUSPENDED or DEACTIVATED |
| reason JSON string | Required; raw printable ASCII plus CR/LF only, then trim surrounding spaces/CR/LF; 3–500 bytes |

Reason validation rejects tabs, controls and Unicode before trimming, preserving internal spaces, punctuation and CR/LF. Missing/null fields fail validation; independent status/reason errors are localized together. Body/query/header claims cannot replace designated input sources. Backend owns transitions, current authority, self/last-admin protection, same-status no-op (including PENDING), replay, audit and session revocation. Every eligible mutation dispatches once, with no target read or REST retry.

Success uses HTTP 200/code `SS`, message `User status updated successfully.`, and exactly `{"user_id":23,"status":"ACTIVE","version":"9007199254740993","changed":false}` as data. Version remains a decimal string; false is explicit. No version increment or changed flag is inferred locally. Invalid dependency success fails safely without partial data.

Only the versioned PATCH route is registered; `me` is not a numeric target alias. All status outcomes and preflights use `Cache-Control: no-store`, including early CORS/auth failures. Status operations require bearer, have no cookie fallback/CSRF requirement, and never set/clear cookies or issue tokens, even for self, no-op, replay or activation. Login remains a separate action.

A caller with `iam.user.read` can GET `/api/v1/users/23` to obtain target `data.version`, then independently PATCH with `iam.user.status.update`:

```http
PATCH /api/v1/users/23/status
Authorization: Bearer <access-token>
Version: 9007199254740993
Idempotency-Key: user-23-activation
Content-Type: application/json

{"status":"ACTIVE","reason":"Registration approved"}
```

Read/status permissions do not imply one another. The two requests are non-atomic; stale Version returns conflict without automatic detail refetch or retry. Status-only callers can provide a target version from another authorized source. Backend accepts same-state requests and otherwise owns PENDING→ACTIVE/DEACTIVATED, ACTIVE→SUSPENDED/DEACTIVATED, SUSPENDED→ACTIVE/DEACTIVATED and DEACTIVATED→ACTIVE. Backend current authority, self-inactivation and last-admin checks remain authoritative, including before replay. REST preserves encoded public status/code/fields and the existing localized fallback for unregistered user Reasons; no global error registry change is needed.

Status verification uses root scenario runners: `GOWORK=off GOCACHE=/tmp/rest-orc-go-cache go test ./test/features/users/... ./test/client/authservice/... ./test/app/... ./test/integration/users/... ./test/integration/authservice/...`. Fake SDK HTTP tests cover authority, input, replay/error mapping and separate detail→status flow. TLS bufconn tests extend the existing UserService registration and verify the exact generated method/presence/metadata, TLS1.3, credential isolation, reusable connection and bounded cancellation. These controlled fakes do not prove real backend transaction/last-admin concurrency, grant provisioning, frontend integration or deployment; those require separate authorized rollout checks.

Verifikasi status (2026-11-04) lulus: full suite 1.551 tes/95 package; race suite terarah 898 tes/28 package; build, vet, strict OpenSpec, dan whitespace checks. Skenario concurrency menjalankan 36 request status/detail/admin-session/self-session/Register/Login/ListUsers dengan barrier terkontrol, actor/target/key/version/reason/permission/metadata/output/cookie terpisah, satu koneksi TLS dan tanpa duplikasi Me.

```sh
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go test ./...
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go test -race ./test/integration/authservice/... ./test/integration/users/... ./test/client/authservice/... ./test/features/users/... ./test/app/...
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go build ./...
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go vet ./...
rtk proxy openspec validate add-user-status-management --strict
```

### Update profil user target

`PATCH /api/v1/users/{user_id}/profile` memerlukan bearer unrestricted dan exact `iam.user.update=true`; role admin, permission read/status/session dan target=self tidak menggantikannya. Actor dari Me dan target path independen. Tanpa field efektif, eligibility dan input teknis tetap diperiksa sebelum sukses lokal.

| Input | Contract |
| --- | --- |
| user_id path | ASCII decimal 1–2147483647; leading zeros diterima |
| Version header | Tepat satu decimal positif int64 sampai 9223372036854775807, wajib juga pada no-op; versi target, bukan Me.version |
| Idempotency-Key header | Opsional; jika hadir tepat satu nonblank printable ASCII 1–200 byte, spaces/commas opaque; empty/duplicate tetap invalid |
| Body | JSON object atau null; hanya string non-null name/address/phone/country_code dipilih; unknown/immutable fields dan field_mask caller diabaikan |

Normalisasi memakai aturan profil existing: name ASCII letters/spaces/apostrophes/hyphens, trim/collapse ordinary spaces, capitalize word starts tanpa lowercase sisa huruf (3–100 byte); country code digit diberi plus lalu plus/nonzero-leading 1–3 digit tanpa trim; phone 12–20 digit setelah penghapusan satu prefix +62/62 lalu satu zero hanya jika country code +62 eksplisit dalam request ini. Address printable ASCII dan CR/LF 3–200 byte dipertahankan persis. Phone-only tidak menebak negara target; country-only tidak memilih phone. Empty string terpilih invalid, null/omitted diabaikan. Semua field error terpilih dilokalisasi bersama tanpa nilai pribadi; data caller tidak dimutasi.

Satu Me diikuti nol RPC mutasi untuk no-op lokal atau satu UpdateUser jika ada field efektif. Input sama dengan profil sebelumnya tetap dikirim. No-op lokal berdata tepat {"changed":false}, tanpa user/version, lookup, audit atau reservasi/replay key, dan tidak membuktikan keberadaan target atau version terbaru meski header Version wajib valid secara sintaks. Backend tetap authoritative untuk changed, target/version/current authority, transaksi/audit dan optional-key replay; REST tidak cache, refetch atau retry otomatis. Read dan update permission independen.

Mutation sukses memakai HTTP 200/code SS/message `User profile updated successfully.` dengan data tepat `{"changed":false,"user":<safe-user>}` (changed boleh true). `user` memiliki 12 field yang sama seperti detail: id/email/name/address/phone/country_code/status/version/created_at/must_change_password/roles/application_access. Versi presisi int64 berada di `data.user.version` sebagai string, timestamp UTC mempertahankan pecahan, array kosong [], false eksplisit, teks hasil backend tidak dinormalisasi ulang. Tidak ada profile ID/top-level version baru atau field credential/dob/auth_version/effective permission. Respons backend invalid/wrong-target menghasilkan safe 500 tanpa data parsial atau GetUser tambahan. Key tidak dikirim berarti tidak ada key pada metadata; inherited key/version dihapus dari salinan dan Version hanya pada payload.

Contoh update target:

```http
PATCH /api/v1/users/23/profile
Authorization: Bearer <access-token>
Version: 9007199254740993
Idempotency-Key: user-23-profile-edit
Content-Type: application/json

{"name":"  NASA  o'neil-jean  ","country_code":"62","phone":"0812345678901"}
```

Idempotency-Key boleh dihilangkan, tetapi jika dikirim wajib valid. JSON null/{} hanya menghasilkan local no-op sesudah semua guard; body kosong/malformed/array/scalar atau tipe field dikenal yang salah ditolak. Literal me bukan alias numeric; hanya route versioned PATCH /users/{user_id}/profile tersedia, tanpa PATCH /users/{user_id}, POST atau unversioned alias. Semua outcome dan preflight memakai no-store sebelum CORS/auth; tidak ada cookie fallback, CSRF/new Origin requirement, Set-Cookie, token atau login/logout inference, termasuk self/replay/no-op/error. Kebijakan self-profile existing tetap berlaku.

Caller dengan iam.user.read dapat GET /api/v1/users/23 untuk memperoleh data.version, kemudian melakukan PATCH profil dengan grant iam.user.update secara terpisah. Request kedua memvalidasi authority lagi; read/update grant tidak saling mewarisi. Conflict VC tidak memicu refetch/version replacement/retry. Selected same value tetap memanggil UpdateUser. Keyed caller retry menjalankan Me baru dan satu eligible command; backend memutuskan replay/current denial/IC/VC/RB. Unkeyed call tidak memperoleh key atau jaminan replay buatan. Key pada no-op lokal tidak direservasi untuk command berikutnya.

Verifikasi HTTP fake SDK dan TLS bufconn memakai root runners `./test/integration/users/...` dan `./test/integration/authservice/...`; UserService mock existing diperluas dengan UpdateUser, terpisah dari self-profile ProfileService. Pengujian memeriksa exact payload optional/mask/key/target-version, no-op lokal, safe snapshot, service credential tunggal, TLS1.3, koneksi reuse, locale/request ID/bearer, cancellation/deadline dan encoded public errors tanpa perlu global user registry baru. Fakes tidak membuktikan transaksi/replay persistence/locking backend nyata, grant provisioning, frontend atau deployment; real smoke/rollout terpisah.

Verifikasi profil target (2026-11-04): focused runners lulus 1.091 tes/28 package; full suite 1.744 tes/95 package; targeted race 1.351 tes/33 package tanpa race report. Skenario concurrency memakai barrier terkontrol untuk 36 request profil target/no-op lokal/self-profile/detail/status/session/Register/Login/ListUsers, termasuk permission denial, partial fields, key opsional, actor/target/version berbeda dan response/cookie terpisah pada satu koneksi TLS. Build, vet, strict OpenSpec dan whitespace checks lulus.

```sh
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go test ./...
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go test -race ./test/integration/... ./test/features/users/... ./test/features/auth/handler/... ./test/client/authservice/... ./test/app/...
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go build ./...
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go vet ./...
rtk proxy openspec validate add-user-profile-update --strict
```

### Admin user creation

`POST /api/v1/users` membuat akun melalui CreateUser, terpisah dari public Register `POST /api/v1/users/create`. Bearer unrestricted wajib mempunyai exact `iam.user.create=true`; non-admin dengan grant ini boleh membuat user. Role admin, wildcard, atau grant read/update/status/session saja tidak cukup. Satu request eligible menjalankan satu Me lalu satu CreateUser, tanpa Register, lookup target, grant role/aplikasi, aktivasi atau login tambahan. Nil/invalid actor ditolak 401; restriction/grant ditolak 403 sebelum validasi bisnis.

Body wajib JSON object dengan string non-null email, password, name, phone dan country_code. Address opsional: omitted/null tidak dipilih, tetapi string kosong gagal minimum 3 byte. Status omitted/null atau tepat PENDING diterima; nilai lain termasuk ACTIVE, string kosong dan pending ditolak dengan error initial-status tersendiri. Payload SDK selalu membawa pointer PENDING. Body kosong/malformed/null/array/scalar, tipe field dikenal yang salah, dan missing/null required gagal 400; {} bukan no-op. Field asing seperti confirm_password, IDs, roles, application_access, permissions, must_change_password dan field_mask diabaikan. Tidak ada default field dari Me.

Credential memakai helper murni bersama `pkg/validation/credentials.go`, dengan katalog/error order dan signature auth lama tetap. Email diperiksa ASCII pada input asli, lalu view trim/lowercase sepanjang 6–50 byte memakai regex register existing; plus-addressing tetap ditolak. Raw email diteruskan utuh, misalnya ` User.Name@Example.COM `; backend melakukan canonicalization. Password wajib 6–50 byte ASCII 0x21–0x7e, tanpa space/control/Unicode, trim atau aturan komposisi. Nilai sederhana seperti `abcdef`, `123456`, `!!!!!!` valid. Login/current_password tetap length-only untuk karakter password lama; register/new_password memakai aturan credential baru yang sama. Field errors independen credential/profile/status digabung dan dilokalisasi EN/ID/ZH tanpa echo input.

Profil dinormalisasi pada salinan satu kali di usecase. Name ASCII letters/spaces/apostrophe/hyphen menjadi 3–100 byte; `  NASA  o'neil-jean  ` menjadi `NASA O'Neil-Jean` tanpa mengubah akronim. Country digits ditambah plus, kemudian wajib 1–3 digit dengan digit awal nonzero; tidak di-trim atau diinferensikan. Phone wajib 12–20 digit. Hanya country eksplisit +62 yang menghapus satu prefix +62/62 dan paling banyak satu nol: `0812345678901` menjadi `812345678901`; `62628123456789` menjadi `628123456789`; `00628123456789` menjadi `0628123456789`. Gateway tidak mengulang normalisasi. Selected address 3–200 byte printable ASCII plus CR/LF disalin persis termasuk spasi; tab/Unicode ditolak.

Wajib tepat satu parsed `Idempotency-Key` case-insensitive, nonblank printable ASCII 1–200 byte. Comma dan space internal bersifat opaque. Duplicate termasuk identical/case-varied ditolak; key tidak di-trim, dibuat, dihash atau diambil dari body/query/parent. Tidak ada target ID atau Version precondition. Version malformed/repeated tidak dipakai. Gateway menyalin metadata, menghapus inherited version/key dan memasang satu key terpilih; bearer, locale, request ID, cancellation/deadline dan kebijakan fail-closed service token tetap berlaku.

```http
POST /api/v1/users
Authorization: Bearer <access-token>
Idempotency-Key: create-user-23
Content-Type: application/json

{"email":"user.name@example.com","password":"abcdef","name":"  NASA  o'neil-jean  ","phone":"0812345678901","country_code":"62","address":"Jl. Melati 7"}
```

Sukses memakai HTTP 200, code SS, message `User created successfully.`, dengan data tepat user. Contoh snapshot backend:

```json
{"code":"SS","message":"User created successfully.","data":{"user":{"id":23,"email":"user.name@example.com","name":"NASA O'Neil-Jean","address":"Jl. Melati 7","phone":"812345678901","country_code":"+62","status":"PENDING","version":"9007199254740993","created_at":"2026-11-04T08:09:10.123456789Z","must_change_password":true,"roles":[],"application_access":[]}}}
```

User mempunyai tepat 12 field detail. Roles berisi role_id/code/name/status/role_status; application_access berisi application_id/code/name/status/application_status. Numeric IDs harus positif dalam range PostgreSQL, version int64 positif berupa string presisi, timestamp UTC mempertahankan pecahan, arrays kosong [], false eksplisit, urutan dan teks backend dipertahankan. Email hasil harus persis canonical trim/lowercase input dan status snapshot PENDING. Nil/malformed/wrong-email/wrong-status menghasilkan safe 500 tanpa data parsial atau fallback GetUser. ID baru tidak dibandingkan dengan actor atau caller IDs; version tidak dipaksa 1. Tidak ada changed/top-level version/registration/login/session/token, profile ID, password/hash/dob/auth_version/effective permission pada respons. Salinan payload/status/country/metadata dan user/collections tidak berbagi mutable state antar request.

Client memakai interface satu-method UserCreationService dan constructor additive NewWithUserCreationService sesudah seluruh guard NewWithUserProfileService. Constructor lama tetap menyediakan generated CreateUser SDK pada koneksi reusable yang sama. Wrapper mengelola nil context/request/response, bounded RequestContext dan deferred cleanup; tidak menambah koneksi per request atau memperluas Service.Register.

Backend owns canonical email, duplicate account, ID/version, password hash/fingerprint, default member assignment, restriction flag, current authority sebelum replay, transaksi/locks/audit/outbox serta idempotency persistence. REST tidak menyimpan/menghash credential atau replay, retry otomatis, mengganti key maupun lookup setelah conflict. Retry caller menjalankan Me baru dan satu eligible CreateUser; snapshot replay boleh merupakan hasil tersimpan. Perubahan password/profil pada key sama dapat menghasilkan IC; duplicate-email EX dan IC/RB mengikuti mapper existing. Registry global tetap auth/common; encoded user public code/status dipertahankan dengan fallback aman terlokalisasi untuk Reason yang tidak diregistrasi, termasuk field errors, Retry-After dan 504/499.

Sesudah create, caller dapat GET `/api/v1/users/23` dengan iam.user.read, PATCH `/api/v1/users/23/profile` dengan iam.user.update, atau PATCH `/api/v1/users/23/status` dengan iam.user.status.update. Masing-masing mengautentikasi dan memeriksa grant sendiri; create tidak otomatis mengaktifkan akun atau menjamin state/version untuk mutasi berikutnya. Default member dan must_change_password dari backend bersifat informasi, bukan grant bagi caller.

POST collection serta OPTIONS dengan Access-Control-Request-Method POST memakai no-store sebelum CORS/auth, termasuk early error/replay. Allowed preflight tidak memanggil RPC; GET ListUsers dan preflight GET mempertahankan cache/role policy lama. Handler standalone juga no-store. Operasi bearer ini tidak memakai cookie fallback, BrowserPolicy, CSRF proof atau Origin wajib baru dan tidak pernah set/delete cookie atau menerbitkan token. CORS existing tetap memeriksa Origin bila dikirim; tidak ada env/CORS setting baru.

Verifikasi HTTP fake SDK ada di root `./test/integration/users/...`; wire menggunakan UserService mock existing pada TLS1.3 bufconn di `./test/integration/authservice/...`, tanpa registrasi UserService kedua. Skenario menguji payload/PENDING/country presence, raw credential, metadata key tanpa Version, satu credential layanan, bearer asli, locale/request ID, deadline/cancellation, dependency/default-member/method/IAM denial, malformed snapshot dan constructor defaults. Barrier concurrency mencampur 36 caller create/replay/conflict, invalid input, Register/Login/ListUsers, self/target profile, detail/status/session pada satu koneksi; actor/credential/key/profile/generated result, precision, collections, field errors dan cookie diperiksa terpisah. Fakes tidak membuktikan transaksi/locking/replay persistence backend nyata, grant provisioning, frontend atau deployment. Smoke backend serta provisioning exact permission/method grant dan default-member readiness tetap pekerjaan rollout terpisah.

```sh
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go test ./test/integration/authservice/...
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go test ./...
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go test -race ./test/integration/... ./test/features/users/... ./test/features/auth/... ./test/client/authservice/... ./test/app/...
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go build ./...
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go vet ./...
rtk proxy openspec validate add-admin-user-creation --strict
```

Hasil verifikasi create (2026-11-04): focused auth/password/users/client/app/integration **1.684 tes/41 package**, full suite **1.931 tes/96 package**, targeted race sesuai command di atas **1.684 tes/41 package**, seluruhnya lulus tanpa race report. Build, vet, strict OpenSpec, gofmt, serta tracked/untracked whitespace checks lulus. Audit hash membatasi edit ke 40 file dalam scope; 952 file baseline lainnya tetap, selain 13 artifact Graphify yang berubah melalui proses eksternal selama sesi dan dibiarkan apa adanya. Main specs/archive/dependency/env tidak diedit oleh implementasi ini.

### Role catalog reads

Role catalog memakai feature roles tersendiri dengan exact `iam.role.read=true` dari Me saat ini. Bearer unrestricted dengan grant ini boleh membaca tanpa role admin; role admin saja, wildcard/prefix, iam.user.read atau role-write grant tidak cukup. Nil context/zero actor ditolak 401 dan restriction/grant ditolak 403 sebelum validasi bisnis. Satu Me diikuti satu ListRoles (list/options) atau satu GetRole (detail), tanpa lookup/grant/mutasi tambahan. Backend tetap memeriksa authority saat read; katalog tidak memberi caller permission, assignment atau akses aplikasi.

List/options menerima satu nilai masing-masing page, limit, keyword, application_code, statuses dan last_id. Defaults page=1/limit=20; page 1–2147483647, limit 1–100, ASCII decimal dengan leading zeros diperbolehkan. Explicit blank/sign/whitespace/overflow dan duplicate query ditolak. Application optional; bila hadir wajib `[a-z][a-z0-9_]*`, 1–50 byte, tanpa trim/lowercase. Status CSV nonempty ACTIVE/DISABLED; spasi biasa di sekitar token dibuang, lalu dedup/sort. Omitted status tidak memfilter ACTIVE saja, termasuk options. Keyword maksimal 200 byte asli/normalisasi, printable ASCII atau ASCII whitespace tab/LF/VT/FF/CR; karakter lain termasuk NUL/DEL/Unicode ditolak sebelum collapse whitespace. Blank keyword berarti tanpa search. Error field independen digabung aman dalam EN/ID/ZH tanpa echo input.

Server menetapkan full safe field view, sort created_at DESC/updated_at DESC/id DESC dan search logical `keyword` dengan FULL_TEXT_TRIGRAM. Fields/sort/search/search_fields/search_mode/code serta created_from/created_to/updated_from/updated_to ditolak sebagai kontrol belum didukung, walau kosong. Query asing diabaikan tanpa authority. Last_id opaque maksimal 2048 byte sesudah URL decode satu kali; tidak di-trim/di-decode base64/dianggap ID. Reset cursor bila filter/keyword/sort berubah; backend memvalidasi query binding tanpa REST repair/refetch/retry. List/options memakai query sama sehingga cursor dapat dipakai silang dengan kontrol identik. Metadata tidak dihitung ulang dari jumlah item, page tidak di-clamp ke total_pages, dan total int64 dipertahankan pada DTO roles tanpa mengubah pagination users existing.

RoleService SDK memakai interface dua read method dan constructor additive NewWithRoleCatalogService sesudah seluruh dependency guard existing. Semua constructor lama tetap menyediakan generated role SDK pada koneksi auth yang sama. Wrapper mengelola nil context/request/response, bounded RequestContext/deferred cleanup, dan mempertahankan dependency error identity. Gateway menyalin payload/query/metadata, menghapus inherited version/idempotency-key untuk read, mempertahankan bearer/locale/request ID dan fail-closed service credential policy. Tidak ada transport per request, dependency/env baru, Version/key precondition, atau forwarding header Version/key yang malformed/repeated.

Sukses list memakai HTTP 200/code SS/message `Roles retrieved successfully.` dengan data tepat items/total/page/limit/total_pages/has_next/has_prev/next_cursor. Role memiliki tepat 13 field: id, code, name, description, application_id, application_code, status, is_system, version, role_code, created_at, updated_at, affected_user_count. ID/application ID numeric positif PostgreSQL-range; role_code int32 positif numeric berbeda dari ID/code; version int64 positif berupa string presisi. Description dan timestamps absent menjadi null eksplisit; timestamp valid menjadi UTC RFC3339Nano. Teks backend, urutan, false dan zero tetap, tidak dinormalisasi sebagai input baru. Affected_user_count berasal dari distinct users dengan assignment ACTIVE, termasuk bila account/role/app/app access sedang nonaktif; bukan jumlah online/effective access. Total int64 tetap JSON number tanpa konversi float/int32. Empty page tetap items=[] dengan total nonzero/cursor/backend flags bila demikian, bukan reset total atau metadata buatan.

Detail memakai HTTP 200/code SS/message `Role retrieved successfully.` dengan data tepat role dan permissions. Role memakai 13 field yang sama; permissions berisi id/application_id/application_code/code/description, atau [] bila kosong. ID target hanya dari path (leading zeros diterima), bukan actor ID, body/query, role_code atau code. System/IAM dan DISABLED roles tetap dapat dibaca. Backend response nil/malformed/wrong-target, invalid ID/version/status/timestamp/page metadata atau permission lintas application gagal safe 500 tanpa partial data atau lookup tambahan. Nullable values dan collections disalin agar tidak berbagi mutable response antar request. Tidak ada credential/auth_version/user records, assignment, inherited/effective permission atau grant side effect pada hasil baca ini.

Endpoint bearer role:

```http
GET /api/v1/roles?page=1&limit=20&application_code=clm&statuses=ACTIVE,DISABLED&keyword=legal
Authorization: Bearer <access-token>
```

```http
GET /api/v1/roles/00010
Authorization: Bearer <access-token>
```

```http
GET /api/v1/roles/options?application_code=clm&statuses=ACTIVE
Authorization: Bearer <access-token>
```

Options memakai HTTP 200/code SS/message `Role options retrieved successfully.`. Data tetap delapan field pagination yang sama; hanya items dipetakan ke empat field. Contoh:

```json
{"code":"SS","message":"Role options retrieved successfully.","data":{"items":[{"value":"10","label":"clm.legal_reviewer - Reviewer legal kontrak","code":"clm.legal_reviewer","role_code":"3"}],"total":1,"page":1,"limit":20,"total_pages":1,"has_next":false,"has_prev":false,"next_cursor":""}}
```

Value adalah ID role string; role_code merupakan numeric code legacy sebagai string, bukan value atau immutable code. Description null/empty memakai name sebagai fallback label; nonempty whitespace/CR/LF/Unicode historis pada response tetap dipertahankan. Label display-only tidak dikirim sebagai identitas/grant. Route options didaftarkan sebelum detail, tanpa GetRole per item, label RPC, fetch seluruh halaman atau default ACTIVE-only. Grant/revoke nanti tetap memeriksa eligibility sendiri. Filter numeric role ListUsers existing tidak berubah.

Role collection/options/detail dan malformed target memakai no-store sebelum CORS/auth serta pada handler standalone. Allowed GET preflight zero RPC; framework HEAD memakai protected read policy yang sama. Cookie-only tidak mengganti bearer, tidak ada mandatory Origin/CSRF baru, Set-Cookie/token atau inference login-required pada success/error. Existing CORS tetap memeriksa Origin bila dikirim. Semua public auth/user/session routes serta users GET/preflight/cache dan role gate legacy dipertahankan; tidak ada alias unversioned/mutasi role pada scope ini.

Verifikasi HTTP fake SDK berada di `./test/integration/roles/...`; TLS bufconn memperluas harness existing dengan satu RoleService mock registration, tetap bersama User/Profile/Auth/Session pada satu connection. Combined command `GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go test ./test/integration/authservice/... ./test/integration/roles/...` lulus 584 tes/6 package saat pengujian read dan context. Skenario mencakup exact payload sort/search/filter/cursor, query parity list/options, fresh Me dan current denial, field/locale errors, nil/malformed success, RNF/public encoded Reason, status/code/fallback/ceil Retry-After, 504/499, TLS1.3/service credential/bearer/locale/request ID dan deadline/cancellation/already-ended contexts tanpa retry. Registry global tetap auth/common; RNF mempertahankan 404/public code dengan fallback NotFound existing, sedangkan Reason lain mengikuti fallback mapper masing-masing. Fakes tidak membuktikan database full-text ranking, cursor binding/read snapshot backend nyata, provisioning exact IAM/method/user grant, frontend atau deployment; smoke/rollout tersebut tetap terpisah.

Verifikasi role catalog (2026-11-05): focused roles/client/app/auth-handler/users/integration lulus **1.925 tes/43 package**, full suite **2.318 tes/110 package**, dan targeted race **1.925 tes/43 package** tanpa race report. Barrier terkontrol menguji 39 request list/options/detail role, invalid field/current denial/RNF/malformed permission, user creation, public Register/Login, user ListUsers/self/target profile/detail/status/session; filter/search/cursor, ID/code/role_code, nullability, version/total presisi, label, locale, error/cookie terpisah pada satu koneksi TLS. Build, vet dan strict OpenSpec lulus. Perintah verifikasi:

```sh
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go test ./test/features/roles/... ./test/client/authservice/... ./test/app/... ./test/features/auth/handler/... ./test/features/users/... ./test/integration/...
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go test ./...
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go test -race ./test/integration/... ./test/features/roles/... ./test/features/users/... ./test/features/auth/handler/... ./test/client/authservice/... ./test/app/...
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go build ./...
GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go vet ./...
rtk proxy openspec validate add-role-catalog --strict
```

### User role management

Assignment list: `GET /api/v1/users/{user_id}/roles` (`iam.user.read`). Commands:
`POST /api/v1/users/{user_id}/roles/grant` (`iam.user.role.grant`) and
`POST /api/v1/users/{user_id}/roles/revoke` (`iam.user.role.revoke`). Each exact
permission is independent; neither read permission nor admin role substitutes for
an operation's grant. Role options are a separate `iam.role.read` request.

Commands require a JSON object with `role_ids` (1–20 decimal **strings**, values
1–2147483647) and string `reason`. Send role option `value`, not its label, code or
numeric role_code. Count is checked before normalization; duplicate numeric IDs
(including `"10"` and `"00010"`) fail 400. Valid IDs are sent in ascending order.
Reason allows printable ASCII plus CR/LF, is checked before trimming surrounding
space/CR/LF, and must then be 3–500 bytes. Internal whitespace remains unchanged.
Send exactly one `Version` header for the **target user**, positive decimal int64,
and one nonblank printable ASCII `Idempotency-Key` of 1–200 bytes. Keys stay opaque;
no generated key, body/query/actor-version fallback or automatic detail read occurs.

The backend owns whole-batch atomicity, current authority, version/idempotency
conflicts, audit and outbox. Grant requires an ACTIVE target, ACTIVE roles and
applications, and ACTIVE target application access for non-IAM applications.
REST does not automatically grant access or activate anything. Revoke follows
backend self platform-admin/last-active-admin guards and does not impose grant
prerequisites. Eligible repeated grants or absent/revoked assignment revokes can
return `changed=false`; committed replay preserves its saved result/version.
Every retry authenticates again and dispatches once; REST never retries, caches
replays, replaces keys/versions, pre-reads state or invokes session changes.

| Method | Route | Exact permission |
| --- | --- | --- |
| GET | `/api/v1/users/{user_id}/roles` | `iam.user.read` |
| POST | `/api/v1/users/{user_id}/roles/grant` | `iam.user.role.grant` |
| POST | `/api/v1/users/{user_id}/roles/revoke` | `iam.user.role.revoke` |

List supports one value per query key: page (default 1, max 2147483647), limit
(default 20, max 100), keyword (max 200 ASCII bytes, whitespace collapsed),
application_code (canonical lowercase ASCII, max 50 bytes), statuses (CSV
ACTIVE/REVOKED, default both) and last_id (opaque, max 2048 decoded bytes).
Ordering is fixed id ASC; keyword searches role name through the backend's logical
keyword document. Fields/sort/search controls, date ranges and exact code filters
are unsupported. Duplicate supported keys fail; other unknown claims are ignored.
URL-encode the cursor unchanged and reset it when target/filter/keyword changes.

HTTP 200/code SS messages are `User roles retrieved successfully.`,
`User roles granted successfully.` and `User roles revoked successfully.`.
List data has exactly `items`, `total` (int64), `page`, `limit`, `total_pages`,
`has_next`, `has_prev`, `next_cursor`, `user_id` (numeric) and `version` (exact
positive int64 decimal string). Empty items remain [] even on an out-of-range page.
Each item has exactly `id` (assignment ID), `role_id`, `code`, `role_code` (number),
`name`, `status`, `role_status`, `application_id`, `application_code`,
`application_status`, `is_system`, `application_access_status`, `granted_by`,
`granted_at`, `revoked_by`, `revoked_at` and `revoke_reason`. IDs remain positive
numeric PostgreSQL-range IDs. Access state, provenance actors/timestamps and revoke
reason are explicit null when absent; present timestamps are UTC RFC3339 with
fractional precision. Assignment status is ACTIVE/REVOKED; role/application
status is ACTIVE/DISABLED. Historical text is unchanged. History is informational
and does not promise effective permissions or grant eligibility; user detail's
existing short role view is unchanged.

Command data has exactly numeric `user_id`, numeric `changed_role_ids` ([] for
no-op), string `version`, and boolean `changed` (including false). Changed IDs
are the backend's ascending subset of the request. Result version is authoritative,
including replay; do not assume input+1. Invalid backend successes fail safely
without partial data. Read/list version can be used on a separately authorized
command, but it may already be stale. On conflict, the caller decides how to
refresh intent/version/key; REST does not refetch or resubmit automatically.

```sh
curl "$BASE_URL/api/v1/users/23/roles?application_code=clm&statuses=ACTIVE,REVOKED" \
  -H "Authorization: Bearer $ACCESS_TOKEN"

curl -X POST "$BASE_URL/api/v1/users/23/roles/grant" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Content-Type: application/json' \
  -H 'Version: 9007199254740993' -H 'Idempotency-Key: assign-reviewer-23' \
  -d '{"role_ids":["10","11"],"reason":"Penugasan reviewer"}'

curl -X POST "$BASE_URL/api/v1/users/23/roles/revoke" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Content-Type: application/json' \
  -H 'Version: 9007199254740993' -H 'Idempotency-Key: revoke-reviewer-23' \
  -d '{"role_ids":["10"],"reason":"Penugasan selesai"}'
```

These bearer routes emit no-store on success and early errors/preflight, never
issue/delete cookies or return tokens, and do not require new Origin/CSRF proof.
Existing exact-origin CORS still applies to supplied Origin. Backend method/IAM
and actor permission grants must already be configured; these endpoints do not
provision them. No new environment keys, SDK upgrade or local state are needed.

### Application catalog

`GET /api/v1/applications` and `GET /api/v1/applications/options` require an
unrestricted bearer actor with exact `iam.permission.read=true`. Admin role,
role/user read or application grant/revoke permissions do not substitute for it.
Each eligible request performs one Me followed by one ListApplications; application
definitions and option selections do not grant access or assign roles.

Both reads support one decoded value per query key: page (default 1, range
1–2147483647), limit (default 20, range 1–100), keyword (max 200 ASCII bytes before
and after whitespace normalization), statuses (CSV ACTIVE/DISABLED, ordinary spaces
trimmed, sorted/deduplicated) and last_id (opaque, max 2048 decoded bytes). Leading
zeros in page/limit are accepted. Omitted statuses includes both states, including
system/IAM applications; request ACTIVE explicitly when needed. Keyword searches
code/name through logical keyword/FULL_TEXT_TRIGRAM; ordering is fixed id DESC.
Duplicate supported keys and fields/sort/search/search_fields/search_mode,
application_code/code/date controls fail 400 even when empty; other unknown claims
are ignored. Invalid independent fields accumulate safe EN/ID/ZH errors. URL-encode
cursors unchanged and reset them when query filters/keyword change.

List success is HTTP 200/code SS/message `Applications retrieved successfully.`.
Data has exactly items/total/page/limit/total_pages/has_next/has_prev/next_cursor.
Each item has exactly numeric positive PostgreSQL-range id, backend code/name,
ACTIVE/DISABLED status and is_system (including false). Stored text/order remain
unchanged; no description, version, timestamp, count, assignment/access state or
permission mapping is invented. Total retains exact int64 precision; empty or
out-of-range pages return items=[] with unchanged backend metadata. Nil/malformed
successes fail safe 500 with empty error data and no partial rows or retries.

Options success is HTTP 200/code SS/message `Application options retrieved
successfully.` with the same eight page fields. Each option contains exactly
value (the unchanged application.code, **not numeric id**) and label (`code - name`).
Names/code, whitespace, punctuation and line breaks are preserved; an empty name
still produces the literal `code - ` label. There is no description fallback.
Options uses the same list query/page/cursor and never aggregates all pages or
looks up labels/access/roles separately.

| Method | Route | Exact permission |
| --- | --- | --- |
| GET | `/api/v1/applications` | `iam.permission.read` |
| GET | `/api/v1/applications/options` | `iam.permission.read` |

Both paths emit Cache-Control no-store on success, input errors, early CORS/auth/
dependency failures and allowed GET preflight (zero RPCs). Bearer reads accept
optional Origin with existing exact-origin CORS validation, never use cookies for
authentication, and do not require new CSRF proof, issue/delete cookies or return tokens.
Bodies (including malformed JSON) and unused/malformed/repeated Version or
Idempotency-Key headers are ignored. Inherited mutation metadata is removed from
a copy before business dispatch; existing public/user/role/assignment routes,
legacy ListUsers cache policy and shared safe errors remain unchanged.

```sh
curl "$BASE_URL/api/v1/applications?statuses=ACTIVE&keyword=contract&page=1&limit=20" \
  -H "Authorization: Bearer $ACCESS_TOKEN"

curl "$BASE_URL/api/v1/applications/options?statuses=ACTIVE" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

For example, list item `{"id":5,"code":"clm","name":"Contract Lifecycle Management","status":"ACTIVE","is_system":false}`
becomes option `{"value":"clm","label":"clm - Contract Lifecycle Management"}`.
Both retain the same backend page metadata; reset cursor after changing filters or
keyword. Shared safe status/code/locale/structured-field/Retry-After mapping and
504/499 context behavior remain unchanged, including generic localized fallback
for unregistered downstream Reasons. No automatic retry or alternate identity.

The configured service must already be allowed to call the exact ApplicationService
method in the IAM namespace, and the actor must retain its current catalog read
permission. These reads do not provision grants or application access, activate
applications/users or promise grant eligibility. They reuse the verified TLS
connection and one environment-owned credential, with bounded parent deadlines,
locale and request ID. No SDK upgrade, new environment keys or local state are
needed; integration tests use controlled fake/TLS services rather than Docker or
a live backend.

## User application access

Application access is a users-owned relationship, independent of the application
catalog and the short embedded access view in user detail. The history response
has exactly `items,total,page,limit,total_pages,has_next,has_prev,next_cursor,user_id,version`.
Each item has `application_id,application_code,name,application_status,is_system,status,granted_by,granted_at,revoked_by,revoked_at,revoke_reason`.
`granted_at` is required UTC RFC3339 with fractional precision; the other provenance
fields are nullable. Stored text/order are preserved; empty collections are `[]`
and false flags remain explicit. IDs are numeric positive PostgreSQL INTEGER;
total is nonnegative int64 and target entity version is a precise positive int64
**string**, including values beyond 2^53. No never-granted catalog rows are synthesized.

History accepts once-only `page,limit,last_id,keyword,application_code,statuses`.
Defaults are page 1/limit 20; bounds are page 1–2147483647 and limit 1–100 without
capping against totals. Statuses are ACTIVE/REVOKED CSV (spaces around tokens,
deduplication/canonical order); they describe stored access, independently of
ACTIVE/DISABLED application definitions. Application codes are exact lowercase
ASCII `[a-z][a-z0-9_]*`, 1–50 bytes. Keyword is at most 200 raw/normalized ASCII
bytes (printable plus whitespace), with whitespace collapsed. Access history uses
explicit application_code/name **literal case-insensitive CONTAINS**; the current
RPC does not support the logical FULL_TEXT_TRIGRAM used by application catalog
and role-assignment search. Percent, underscore and backslash remain literal input;
there is no rank or search-mode fallback. Ordering is fixed `application_id ASC`,
with full projection. Opaque `last_id` is passed once-decoded unchanged, maximum
2048 bytes; reset it after target/filter/keyword changes. Backend rejection is
returned without repair, retry, page loops or catalog supplementation. Duplicate
supported controls and fields/sort/search/search_fields/search_mode/date/code
controls fail 400 even if empty; unrelated unknown claims are ignored.

Grant takes required `role_ids` (1–20 string-decimal IDs, positive INTEGER range)
and `reason`. Count is checked before normalization; numeric duplicates such as
`"10"`/`"00010"` fail. IDs are copied and sorted. Revoke takes **reason only** and
revokes access plus **every active role in that application**, including more
than 20. Extra revoke `role_ids` of any JSON type are ignored and **cannot narrow
revocation**. Use selected-user-role revoke for an explicit subset instead.
Both access commands require exactly one `Version` (positive int64 ASCII decimal)
and `Idempotency-Key` (opaque, nonblank printable ASCII, 1–200 bytes), without
body/query/Me fallback or generated replacements. Reason allows printable ASCII
and CR/LF, trims surrounding space/CR/LF to 3–500 bytes, and preserves internal
bytes; tabs/non-ASCII/other controls fail. Unknown fields confer no authority.
Compound result data has exactly `user_id,application_id,access_status,changed_role_ids,version,changed`.

Access history uses the existing UserService SDK; compound access mutations use
ApplicationService. Their small injectable SDK seams follow the existing
constructor chain; all prior constructors retain generated SDK defaults on the
same connection. Wrappers validate nil context/request/response, preserve error
identity, and use the bounded request context with deferred cancellation. No new
connection, dependency, service credential or runtime environment variable is needed.

Gateway reads send full projection and `application_id ASC`, with nil search or
`Search{fields:[application_code,name], mode:CONTAINS}`. Filters and metadata are
copied; inherited version/key are removed, commands select exactly one key and
send the expected target version only in payload. History may contain disabled
system definitions or inactive-target records; required granted_at is checked,
optional provenance remains null, and malformed successes fail entirely as safe
500 without partial data or recovery RPCs. Caller-controlled list/detail target
versions can be submitted on a separately authorized command; they promise no
freshness and grant no command permission.

Compound mapping preserves backend version and changed flag. `changed=true`
with `changed_role_ids=[]` is valid for access-only changes; `changed=false`
requires no changed roles. Grant returns ACTIVE and a subset of selected roles.
Revoke returns REVOKED or explicit null (no stored access pair), and all actual
changed roles without a 20-role output cap. Roles can be cleaned up even without
an access row. IDs must be sorted/distinct/in range; malformed result metadata,
state, target or version fails safe 500. Application ID is range-checked but the
path code-to-ID resolution is backend-owned, without a catalog lookup. Nullable
fields and collections are independently owned. No input+1 version arithmetic,
partial repair, auto-retry, cookie or credential effect occurs.

| Access endpoint | Exact grants after one Me | Business RPC |
| --- | --- | --- |
| GET `/api/v1/users/{user_id}/applications` | `iam.user.read` | UserService/ListUserApplicationAccess |
| POST `/api/v1/users/{user_id}/applications/{application_code}/grant` | `iam.user.application.grant` AND `iam.user.role.grant` | ApplicationService/GrantApplicationAccess |
| POST `/api/v1/users/{user_id}/applications/{application_code}/revoke` | `iam.user.application.revoke` AND `iam.user.role.revoke` | ApplicationService/RevokeApplicationAccess |

Missing/invalid identity is 401; restricted or missing either exact grant is 403
before business validation (transport syntax can fail earlier). Wildcards,
prefixes, admin role, target=self or forged claims do not replace permission.
Commands require no additional user/role/permission catalog read grant. Their
permissions do not grant reads. Each eligible request performs one Me and the
selected business RPC only; current backend authority can still deny after Me.

Grant atomically creates/reactivates access and selected assignments, preserving
unselected roles and active provenance. Backend alone checks ACTIVE target,
ACTIVE nonsystem/non-IAM application and ACTIVE same-application roles. Revoke
cleans all active application roles and access, including for backend-authorized
inactive targets/disabled definitions, and does not rewrite old revoked history.
Backend owns system protection, audit/outbox, whole-set atomicity, optimistic
concurrency, no-op and retained replay. Every caller retry performs fresh Me and
one command; REST has no eligibility pre-read, replay cache or automatic retry.
Stale versions, changed retained payloads and busy-resource errors do not trigger
replacement versions/keys, activation or alternate commands. Credentials,
sessions and other applications are unaffected by REST orchestration.

Example access requests (bearer authentication on each request):

```http
GET /api/v1/users/23/applications?page=1&limit=20&statuses=ACTIVE,REVOKED&keyword=clm
Authorization: Bearer <access_token>
```

```http
POST /api/v1/users/23/applications/clm/grant
Authorization: Bearer <access_token>
Content-Type: application/json
Version: 9007199254740993
Idempotency-Key: <action_key>

{"role_ids":["10","11"],"reason":"Approved application access"}
```

```http
POST /api/v1/users/23/applications/clm/revoke
Authorization: Bearer <access_token>
Content-Type: application/json
Version: 9007199254740993
Idempotency-Key: <action_key>

{"reason":"Application access no longer required"}
```

List success is 200/SS/`User application access retrieved successfully.`:

```json
{"code":"SS","message":"User application access retrieved successfully.","data":{"items":[{"application_id":5,"application_code":"clm","name":"CLM","application_status":"ACTIVE","is_system":false,"status":"ACTIVE","granted_by":17,"granted_at":"2026-11-06T08:00:00.123456789Z","revoked_by":null,"revoked_at":null,"revoke_reason":null}],"total":1,"page":1,"limit":20,"total_pages":1,"has_next":false,"has_prev":false,"next_cursor":"","user_id":23,"version":"9007199254740993"}}
```

Compound success is 200/SS/`Application access granted successfully.` or
`Application access revoked successfully.`. An access-only grant can return
`{"user_id":23,"application_id":5,"access_status":"ACTIVE","changed_role_ids":[],"version":"9007199254740994","changed":true}`.
A revoke cleaning roles without a stored access pair can return
`{"user_id":23,"application_id":5,"access_status":null,"changed_role_ids":[10,11],"version":"9007199254740994","changed":true}`.
A full no-op returns changed=false and [] without local version arithmetic.

Only the three versioned access operations are added (standard HEAD follows
GET); no self alias or alternate write method is introduced. Numeric path user
IDs accept leading zeros; literal me is invalid. The independent access routes
run before the broad legacy users admin group. Read ignores body and unused
Version/key; commands select user/code from path and version/key only from their
headers. Use a separately authorized target detail or history read for Version,
not Me's actor version. Backend owns freshness and can reject a later command.

All access responses are no-store, including early CORS/auth/input/backend
failures and malformed targets/codes. Allowed exact-origin GET/POST preflight
finishes with zero RPCs. Supplied Origin retains existing validation; no new
mandatory Origin/CSRF proof is required. Cookies cannot authenticate these routes,
and no result, self operation, no-op, replay or error sets/deletes cookies or
returns credentials. Existing public/legacy users/catalog/assignment/profile/
status/session routing and cache policies remain unchanged.

### Application-access verification

Focused `GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk go test
./test/features/users/... ./test/client/authservice/... ./test/app/...
./test/features/auth/handler/... ./test/integration/...` passed 2,866 tests across
37 packages. This includes HTTP fake-SDK contracts, real SDK-encoded application
AASP/HTTP409 and user/role failures, structured EN/ID/ZH fields, unknown-Reason
localized fallback, ceil Retry-After and 499/504 handling without registry changes,
private input echo or retries.

`GOWORK=off GOCACHE=/tmp/rest-orc-go-cache rtk proxy go test
./test/integration/authservice/... -run
'TestAuthServiceTransport/application_access'` passed the access TLS scenarios.
The default generated SDK uses the exact UserService read/ApplicationService
commands on one existing verified TLS1.3 connection, configured service token,
original bearer, locale/request ID, copied metadata and bounded deadlines.
Already-ended contexts do not dispatch; cancellation completes without a retry.
The focused suite also passes a controlled 48-request mixed scenario covering
access reads/grants/revokes, catalog, selected role grants, detail, login, status,
profile, sessions, creation and legacy ListUsers, with isolated nullable data,
large versions/totals, keys/reasons/roles/cursors, errors and cookie effects.

These fakes and TLS bufconn verify REST orchestration and secure wire contracts;
they do not establish live database literal-search/cursor snapshot behavior,
compound transaction/audit/outbox persistence, retained replay or deployment.
Before a separately authorized rollout, verify the existing configured identity
has Me and the three exact access methods allowlisted, IAM scope, and the current
actor's corresponding read or AND command grants. Application catalog permission
remains independent. REST adds no persistence/config/dependency migration.
Rollback removes the additive REST routes; it never compensates committed backend
access/role changes automatically. Conflicts need caller-controlled resolution,
not hidden version refresh/key generation or automatic resubmission.

Final implementation checks (2026-11-06), all with `GOWORK=off
GOCACHE=/tmp/rest-orc-go-cache`:

- `rtk go test ./...`: 3,430 tests / 124 packages passed.
- `rtk go test -race ./test/integration/... ./test/features/users/... ./test/features/auth/handler/... ./test/client/authservice/... ./test/app/...`: 2,866 tests / 37 packages passed, no race report.
- `rtk go build ./...` and `rtk go vet ./...`: passed.
- `openspec validate add-user-application-access --strict`, gofmt and scoped tracked/untracked whitespace checks: passed.
- Scope review: 42 files (25 existing edits / 17 new); 1,260 other baseline files unchanged. Existing constructor signatures, three intended routes, one UserService/ApplicationService registration each, root test placement and unchanged dependencies/env/main specs/archive/Graphify verified. No live backend, Docker, sibling edits or deployment were used.
