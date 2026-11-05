# Tasks

## 1. Validator dan catalog domain users

- [x] 1.1 Tambahkan validator register/login murni pada `internal/features/users/domain/auth_validation.go` dengan shared email dan required/byte-length password rules. Email diperiksa ASCII sebelum membuat validation copy trim/lowercase; gunakan pola backend pada spec, tetapi jangan ubah AuthRequest. Tambahkan runner/scenarios di `test/features/users/domain/` untuk email 5/6/50/51 byte, empty, uppercase/whitespace, non-ASCII termasuk Unicode whitespace, accepted punctuation dan plus/malformed pattern, serta password required/length. Assert FieldErrors/catalog identity, agregasi, dan request nonmutation; verifikasi dengan `rtk proxy env GOWORK=off go test -mod=readonly ./test/features/users/domain/...`.
- [x] 1.2 Tambahkan pembatasan byte `0x21–0x7E` hanya pada validator register dan satu catalog field error milik users untuk karakter password invalid, memakai loader existing dengan pesan en/id/zh sesuai design. Tambahkan domain scenarios untuk batas 6/50, panjang 5/51, spasi/tab/CR/LF/NUL/DEL/non-ASCII, password sederhana tanpa komposisi, multi-error pada satu field, serta login credential lama termasuk tiga é (6 byte) dan whitespace/control yang panjangnya valid. Verifikasi domain tests lulus dan tidak ada credential yang diubah atau dimasukkan ke pesan error.

## 2. Hubungkan validator ke usecase

- [x] 2.1 Panggil validator yang sesuai pada `usecase/register.go` dan `usecase/login.go` setelah nil-context guard dan sebelum gateway. Perluas runner/scenarios dan mocks existing di `test/features/users/usecase/` untuk invalid→zero calls/zero response, valid→exact request/context serta satu call pada operasi yang benar, return value/error identity, nil context, dan nil constructor dependency. Verifikasi dengan `rtk proxy env GOWORK=off go test -mod=readonly ./test/features/users/usecase/...` tanpa mengubah constructor/interface atau menambahkan business decision di handler/gateway.

## 3. Contract HTTP dan dokumentasi

- [x] 3.1 Tambahkan integration scenarios real usecase/gateway/client dengan service fake di `test/integration/users/`: kedua endpoint invalid menghasilkan 400 dengan seluruh field messages dalam en/id/zh-CN dan tidak memanggil Register/Login/Me. Uji omitted/null/malformed/known-field type errors sesuai parser existing; pertahankan unknown fields termasuk confirm_password tanpa kebijakan baru. Jalankan `rtk proxy env GOWORK=off go test -mod=readonly ./test/features/users/handler/http/... ./test/integration/users/...` dan assert envelope serta ketiadaan nilai credential dalam response.
- [x] 3.2 Tambahkan integration success/forwarding regression untuk kedua public routes: email raw uppercase/whitespace tetap persis, password login legacy tetap persis, context/locale diteruskan, dan hanya business RPC yang sesuai dipanggil sekali tanpa Me. Kunci HTTP 200, message, dan data.Token existing; gunakan input lokal valid untuk downstream 409 field email serta 401 login dan assert mapping tidak berubah/tidak retry. Verifikasi handler/integration tests lulus tanpa response/gateway/client/composition changes.
- [x] 3.3 Tambahkan aturan request register/login di README: email mengikuti backend dengan raw forwarding, password register vs login, batas byte, field errors 400 sebelum RPC, dan response existing tetap. Jelaskan plus-addressing belum diterima pola backend serta confirm_password tidak diwajibkan; review contoh agar cocok dengan scenario spec dan tidak mendokumentasikan cookie/PENDING response baru.

## 4. Verifikasi keseluruhan dan review

- [x] 4.1 Jalankan gofmt pada seluruh Go files yang berubah, targeted tests domain/usecase/handler/integration, lalu `rtk proxy env GOWORK=off go test -mod=readonly ./...` dan `rtk proxy env GOWORK=off go vet -mod=readonly ./...`. Semua harus lulus; race detector hanya bila implementasi ternyata mengubah concurrency/shared state. Catat hasil dan bedakan masalah environment dari regression.
- [x] 4.2 Review `rtk proxy git diff`, `rtk git diff --check`, dan `rtk git status --short`, termasuk file baru. Verifikasi perubahan dibatasi pada validator/catalog users, pemanggilan usecase, test terkait, README, dan tracking artifacts; tidak ada dependency/sibling/response/route change atau credential/debug leak. Pastikan seluruh scenario spec terpenuhi, perubahan user terjaga, dan `rtk proxy openspec validate add-local-auth-validation --strict` lulus sebelum menyatakan apply selesai.

## Hasil verifikasi apply

- Gofmt dijalankan pada 11 Go files yang berubah.
- Targeted tests domain/usecase/handler/integration users lulus.
- `GOWORK=off go test -mod=readonly ./...` lulus.
- `GOWORK=off go vet -mod=readonly ./...` lulus.
- Command Go menggunakan `GOCACHE=/private/tmp/rest-orc-go-build`.
- `git diff --check` dan strict OpenSpec validation lulus.
- Review diff/status serta snapshot sebelum apply: delta hanya 14 file dalam scope; tidak ada file terhapus atau perubahan lain di luar scope apply.
- Tidak ada perubahan concurrency/shared state runtime; race detector tidak diperlukan untuk change ini.
