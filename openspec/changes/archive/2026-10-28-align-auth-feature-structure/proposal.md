# Proposal

## Why

Login/Register masih dimiliki feature users pada domain, validasi, usecase, gateway, handler, wiring, dan test, sedangkan Me sudah berada pada feature auth. Ownership dan beberapa penamaan perlu diselaraskan dengan struktur project serta skill terbaru, tanpa mengubah contract HTTP/downstream yang sudah berjalan.

## What Changes

- Pindahkan model request/response, rule validasi, usecase Register/Login, integrasi gateway authservice, serta handler Login/Register ke `internal/features/auth`. Definisi error dan catalog pesan validasi lokal berada di `pkg/dictionary`. Feature users menyisakan flow ListUsers existing dan tanggung jawab resource user.
- Tetap gunakan tipe domain feature sebagai DTO HTTP sesuai keputusan user. Auth request/response dipindahkan ke domain auth; tipe response Me yang saat ini dideklarasikan di handler ditempatkan di domain auth dengan bentuk JSON yang sama. Handler tetap melakukan parsing/context forwarding/mapping, dan domain tetap bebas dari Fiber/protobuf/concrete client.
- Gunakan direktori `internal/features/<feature>/handler/` langsung, termasuk `auth_handler.go` dan `users_handler.go`, sesuai keputusan struktur terbaru. Rampungkan pemisahan interface consumer agar usecase/gateway users tidak lagi mewajibkan kemampuan Register/Login.
- Batasi abstraksi pada kebutuhan nyata dan boundary existing. Mapping credential/token yang sederhana tetap di auth gateway; jangan menambah mapper interface/struct, wrapper penerus, generic framework, atau file terpisah hanya untuk beberapa assignment. Setiap abstraksi baru harus memiliki consumer dan tanggung jawab yang jelas serta menjaga flow mudah ditelusuri developer junior.
- Rangkai Login/Register/Me di `internal/app/auth_routes.go`, dan ListUsers di `users_routes.go`. Daftarkan route publik auth yang memakai prefix legacy users sebelum protected users group agar Fiber tidak menangkapnya lebih dahulu.
- Pertahankan `POST /api/v1/users/create`, `POST /api/v1/users/login`, `GET /api/v1/me`, dan `GET /api/v1/users`, termasuk payload/envelope, `data.Token`, validasi/locale, downstream HTTP status, service credential/TLS, context/deadline/cancellation, dan urutan/jumlah RPC.
- Rename implementasi shared service client `internal/client/authservice/client.go` menjadi `auth_client.go`, dan mapper `internal/auth/me_snapshot.go` menjadi `principal_mapping.go` sesuai tanggung jawab mapping UserData ke Principal. Pertahankan generic transport, boundary mapping, dan behavior snapshot request existing tanpa layer, transport, atau API client baru.
- Pindahkan test/fixture/mock sesuai ownership auth/users; tempatkan setup HTTP lintas feature yang benar-benar dipakai bersama pada root test shared fixture. Perbarui README dan aturan repository terkait ownership/DTO, lalu verifikasi dengan `GOWORK=off`.

## Capabilities

### New Capabilities

Tidak ada. Change ini adalah refactor internal; `.openspec.yaml` memakai `skip_specs: true`.

### Modified Capabilities

Tidak ada perubahan requirement pada `auth-input-validation`, `auth-service-credentials`, dan `http-error-mapping`. Behavior Me dari implementasi `add-me-endpoint` juga tetap dipertahankan. Tidak dibuat delta spec untuk perubahan lokasi/naming saja.

## Impact

- Production: `internal/features/{auth,users}`, `internal/app/{auth_routes,users_routes,routes}.go`, filename shared auth client dan principal mapper di `internal/auth`, serta catalog pesan validasi dan auth-named compatibility aliases pada dictionary untuk menjaga identity error existing.
- Test: unit domain/usecase/gateway/handler kedua feature, HTTP integration auth/users, transport integration authservice, app wiring, dan shared fixtures yang telah memiliki lebih dari satu consumer.
- Dokumentasi: README dan root AGENTS untuk ownership Login/Register/Me, handler tanpa subfolder http, dan DTO domain sesuai instruksi user. Aturan anti-heavy-abstraction pada skill global diperbarui terpisah saat penyusunan plan atas permintaan user; implementasi change ini tidak mengedit skill global atau artifact change lama.
- Tidak mengubah dependency, environment key, sibling repository, generated Graphify/protobuf, privilege policy, cookie/session lifecycle, atau public URL. Tidak menambah endpoint alias `/auth/login`/`/auth/register`, empty scaffold, database, retry, cache, atau business validation baru.
- Menggunakan hasil implementasi `add-me-endpoint` sebagai baseline. Change lama tetap terpisah; pembuatan plan ini tidak mengarsipkan atau mengubah task yang sudah selesai pada change tersebut.
