# rest-orc

`rest-orc` adalah skeleton service orkestrasi yang menerima request dari
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

- Mempelajari pemisahan transport, feature, dan integration layer.
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
Feature / Orchestration
   ├── business contract ──► gRPC integration ──► L7/gRPC proxy
   └── business contract ──► HTTP integration ──► HTTP proxy
```

Feature layer nantinya bergantung pada contract berdasarkan kemampuan bisnis,
bukan pada protokol. Sebagai contoh, feature menggunakan contract
`ProductReader`; implementasinya dapat memakai gRPC atau HTTP tanpa membuat
feature mengetahui detail transport tersebut.

Load balancing tidak dilakukan oleh orchestrator. Setiap client mengarah ke satu
endpoint proxy per service. Untuk gRPC, proxy harus memahami HTTP/2 dan gRPC agar
request dapat dibagi pada level RPC.

## Fitur Saat Ini

- Fiber v3 sebagai HTTP server.
- Endpoint liveness `GET /livez`.
- Recovery middleware untuk menangkap panic pada request pipeline.
- Respons panic yang aman:

  ```json
  {
    "error": "internal server error"
  }
  ```

- Graceful shutdown saat menerima `SIGINT` atau `SIGTERM`.
- Strict YAML configuration menggunakan `goccy/go-yaml`.
- Environment variable dapat meng-override nilai YAML.
- Generic gRPC connection client dengan:
  - TLS atau plaintext;
  - batas ukuran message;
  - request timeout;
  - keepalive opsional;
  - `PermitWithoutStream` selalu `false`.
- Generic HTTP client dengan:
  - reusable `http.Transport`;
  - connection pooling;
  - request timeout;
  - context propagation;
  - TLS opsional;
  - keepalive yang dapat dinonaktifkan.
- Scenario-driven test dengan runner, fixture, dan mock terpisah.

Saat ini belum ada business route atau downstream service nyata. Konfigurasi
`auth_grpc` dan `backend_http` masih menjadi contoh wiring awal untuk eksperimen.

## Struktur Project

```text
.
├── cmd/api/                    Entrypoint aplikasi
├── internal/
│   ├── app/                    Composition root dan lifecycle Fiber
│   ├── client/
│   │   ├── grpcclient/         Koneksi gRPC reusable
│   │   ├── httpclient/         Client dan transport HTTP reusable
│   │   └── tlsconfig/          Pembuatan konfigurasi TLS
│   └── config/                 Loader, environment override, dan validation
├── tests/
│   ├── app/
│   ├── client/
│   │   ├── grpcclient/
│   │   └── httpclient/
│   ├── config/
│   └── shared/
├── env.yaml                    Konfigurasi lokal
└── AGENTS.md                   Aturan pengembangan repository
```

Jika feature mulai ditambahkan, struktur dapat berkembang menjadi:

```text
internal/
├── handler/                    Fiber request/response mapping
├── features/                   Business flow dan orchestration
├── contracts/                  Interface yang digunakan feature
└── integrations/               Implementasi HTTP atau gRPC per service
```

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

Periksa liveness:

```bash
curl http://localhost:8080/livez
```

Expected response:

```json
{"status":"OK"}
```

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

  backend_http:
    base_url: http://backend-proxy:8080
    request_timeout: 10s
    tls:
      enabled: false
      server_name: ""
      ca_file: ""
    keepalive:
      enabled: true
      max_idle_connections: 100
      max_idle_connections_per_host: 20
      max_connections_per_host: 100
      idle_connection_timeout: 90s
      response_header_timeout: 10s
```

Environment variable memiliki prioritas lebih tinggi daripada YAML. Contoh:

```bash
APP_PORT=9090 \
AUTH_GRPC_TARGET=localhost:50051 \
BACKEND_HTTP_BASE_URL=http://localhost:8081 \
go run ./cmd/api
```

Environment override lain mengikuti nama yang didefinisikan pada
`internal/config/environment.go`.

## Menggunakan Client Reusable

Koneksi gRPC untuk service yang berbeda dibuat dengan constructor yang sama:

```go
authConnection, err := grpcclient.New(cfg.Clients.AuthGRPC)
```

Generated protobuf client dibentuk dari koneksi tersebut:

```go
authClient := authv1.NewAuthServiceClient(authConnection.Connection())
```

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
- Belum ada business feature dan generated protobuf client.
- Proxy atau load balancer aktual belum menjadi bagian repository.
- Kontrak error bisnis dan response API belum difinalisasi.

Perubahan eksperimen sebaiknya tetap disertai test dan tidak mengorbankan
context propagation, deterministic behavior, atau resource cleanup.
