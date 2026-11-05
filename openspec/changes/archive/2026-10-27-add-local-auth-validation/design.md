# Design

## Context

Motivasi dan scope ada pada `proposal.md`; behavior terukur ada pada `specs/auth-input-validation/spec.md`. Acuan keputusan adalah bagian 1–2 dan 13 pada `docs/grpc-auth-validation-rules.md`.

- `domain.AuthRequest` saat ini berisi dua string `Email`/`Password` di `domain/gateway.go`. Register/Login use case hanya memeriksa context nil, lalu meneruskan request ke gateway.
- HTTP handler melakukan `Bind().Body`, mengembalikan Fiber bad request untuk parsing gagal, dan menulis response sukses melalui shared writer. Validasi feature belum ada.
- Gateway memetakan email/password langsung ke protobuf. Public routes didaftarkan sebelum admin group dan tidak memanggil `Me`; tidak diperlukan perubahan composition.
- Shared mapper sudah menerima `dictionary.FieldErrors`/`errorpackage.Errors`, memformat pesan locale dan status 400, serta mempertahankan status eksplisit service. Tidak diperlukan mapper baru.
- Catalog auth terpublikasi sudah memiliki required, minimum, maximum, dan invalid-email errors. `ErrAlphaOnly` tidak sesuai untuk password yang membolehkan angka/simbol; belum ada pesan invalid password character yang cocok.
- Backend yang dibaca secara read-only: `grpc-auth/internal/repositories/domain/auth.go` mendefinisikan pola email dan byte-length 6–50. Handler login dan `prepareRegistration` menggunakan email trim/lowercase sebelum validasi, tetapi tidak mengubah password. Regex tersebut menolak `+` dan menerima `~` pada posisi internal yang diizinkan.
- Baseline targeted usecase/handler/integration tests lulus dengan `GOWORK=off -mod=readonly` pada dependency pins yang sudah diselaraskan.

## Goals / Non-Goals

**Goals:** Validasi murni yang dapat diuji tanpa network; penolakan sebelum gateway; alur usecase tetap mudah dibaca; shared rules tidak diduplikasi dan kebijakan register/login tetap terlihat jelas.

**Non-Goals:** Tidak menambah validator framework/interface, helper generik di pkg, dependency, environment key, response DTO, middleware, atau endpoint baru. Tidak mengubah sibling, generated contract, query ListUsers, error mapper, atau response register/login. Pengayaan PENDING/session/token/cookie, Origin/CSRF, dan confirm_password berada di luar change ini.

## Decisions

### 1. Validator milik domain users, dipanggil dari usecase

Tambahkan `internal/features/users/domain/auth_validation.go` dengan entrypoint berbeda yang jelas untuk register dan login (misalnya value-receiver `AuthRequest.ValidateRegister()` dan `ValidateLogin()`). Keduanya memakai helper kecil untuk email dan required/length password; hanya register menambah pemeriksaan karakter. Hindari flag boolean yang membuat caller harus menebak mode validasi, mutasi request, atau generic validation engine.

Usecase tetap memeriksa context nil terlebih dahulu agar error identity existing tidak berubah, kemudian validasi, lalu gateway yang sesuai. Input invalid mengembalikan zero AuthResponse dan FieldErrors; input valid meneruskan context dan AuthRequest yang sama. Constructor dan consumer interfaces tetap sama.

Alternatif validasi hanya di HTTP handler ditolak karena pemanggilan usecase langsung akan melewati aturan. Memindahkan business validation ke gateway juga ditolak karena mencampur aturan feature dengan mapping protobuf.

### 2. Validasi email melalui salinan, bukan normalisasi request

Periksa ASCII pada nilai email asli terlebih dahulu. Buat salinan `strings.ToLower(strings.TrimSpace(request.Email))` hanya untuk required, panjang byte, dan regex backend. Compile regex yang sudah diperiksa sekali pada package domain. Teruskan original email tanpa perubahan saat valid.

Raw non-ASCII tetap invalid walaupun trim/lowercase dapat menghapus/mengubah karakter itu; jangan memberi error invalid-email dua kali untuk kegagalan ASCII dan pola. Salinan yang kosong mendapat required dan minimum; format diperiksa hanya jika salinan nonempty. Salinan 6/50 byte diterima bila pola valid; 5/51 ditolak. Length setelah trim mengikuti normalisasi authoritative backend dan aturan plan tanpa mengubah argument RPC.

Pattern literal pada spec diambil dari validator backend. Pernyataan `.`, `_`, `-`, `+` pada plan bersyarat lolos validator backend; karena `+` tidak lolos pola saat discovery, jangan memperluasnya secara lokal. Dukungan plus-addressing memerlukan keputusan contract terpisah. Hindari `net/mail` atau regex email alternatif karena accepted set-nya dapat berbeda dari backend.

### 3. Bedakan password baru dari credential login

Password selalu diperiksa berdasarkan byte pada string hasil parsing, tidak pernah di-trim atau diubah. Common rules adalah required, min 6, max 50. Password kosong mendapat required dan minimum; tidak ada character error tambahan pada kosong. Untuk register, tambahkan satu character error jika ada byte di luar `0x21–0x7E`; error panjang dan karakter yang sama-sama gagal tetap terkumpul. Password `aaaaaa`, `123456`, atau `!!!!!!` valid secara lokal.

Login tidak memanggil pemeriksaan karakter register. Whitespace, Unicode, dan kontrol yang dapat direpresentasikan setelah parsing tetap diteruskan jika panjangnya valid; backend tetap menentukan kecocokan credential. Contoh tiga `é` bernilai 6 byte, valid untuk panjang login dan invalid untuk karakter register.

Alternatif satu validator password yang sama ditolak karena akan memblokir credential lama. Tidak menambah syarat uppercase/digit/symbol atau memeriksa keunikan email melalui RPC pendahuluan.

### 4. Agregasikan catalog errors yang sudah kompatibel dengan mapper

Kumpulkan error seluruh field menggunakan `dictionary.FieldErrors`. Urutan pesan per field: required → minimum/maximum → format/character. Jangan mengembalikan error pada kegagalan field pertama; return nil secara eksplisit jika map kosong agar tidak menghasilkan typed-nil error.

Reuse `authdictionary.ErrRequired`, `ErrMinLength`, `ErrMaxLength`, dan `ErrInvalidEmail`. Tambahkan satu entry statis catalog milik domain users untuk karakter password register invalid, dengan pola embedded catalog yang sudah digunakan project dan loader library existing. Field message: English `contains invalid characters`, Indonesia `mengandung karakter tidak valid`, Chinese `包含无效字符`. Ini bukan business rule infrastructure di `pkg/dictionary` dan tidak memerlukan perubahan registry response, dependency, atau sibling api-contracts.

Error baru hanya berada di FieldErrors sehingga root HTTP code tetap BR/400 dari shared mapper. Jangan membuat string error ad-hoc atau memasukkan nilai credential dalam pesan/log/assertion diagnostic baru. Localization tetap dilakukan di response boundary menurut header existing, termasuk fallback locale.

### 5. Pertahankan transport dan response legacy secara eksplisit

Biarkan handler melakukan parsing existing dan meneruskan typed request; tidak menambah unknown-field rejection atau mandatory confirm_password. Pilihan plan `[tidak perlu]` dipakai dengan mempertahankan dua field request existing; properti confirm_password tetap mengikuti unknown-field parsing, tanpa new validation/forwarding.

Jangan menambahkan field registration_id/status, mengganti `data.Token` menjadi `data.token`, mengubah success message/200, atau memasang cookie. Itu bagian plan lebih besar yang secara eksplisit dikecualikan oleh permintaan mempertahankan response existing. Input yang lolos lokal masih dapat ditolak backend; error 409/401 dan retry guidance existing tidak ditangkap/dikonversi menjadi local 400 oleh usecase.

### 6. Test pada root test dengan boundary yang nyata

- Domain: `test/features/users/domain/` baru berisi runner/scenarios yang menguji email/pola/ASCII dan password register/login, byte-boundary, agregasi, identity catalog, dan request nonmutation.
- Usecase: perluas `test/features/users/usecase/` melalui runner existing dan mocks.Gateway yang sudah mempunyai RegisterCalls/LoginCalls. Uji invalid→zero calls/zero response; valid→exact arguments/context, satu call pada operasi yang tepat, success/error identity; constructor/context-nil regression.
- HTTP/integration: gunakan real usecase/gateway/client dengan fake service di `test/integration/users/`, sehingga validasi tidak dibypass oleh mock usecase. Uji multi-field 400 dalam en/id/zh-CN, semua register character boundaries yang relevan, credential login lama, email uppercase/whitespace tetap persis, serta tidak ada Me atau business RPC pada public invalid request. Pertahankan handler malformed/type-error coverage dan success fixtures existing.
- Gunakan payload valid untuk skenario dependency error (409 field email/401 login), assert satu call tanpa retry dan envelope existing. Tidak mengakses server/database/clock/random nyata.
- README menambahkan aturan input dan penjelasan credential lama tanpa mendokumentasikan response baru. Targeted suite mendampingi setiap kelompok perubahan, lalu full suite dan vet memakai `GOWORK=off` agar pin dependency tetap terbukti.

## Risks / Trade-offs

- [Validasi lokal menduplikasi contract email backend] → Catat sumber regex, uji accepted/rejected cases termasuk plus, dan jangan memperluas accepted set secara diam-diam. Backend tetap authoritative.
- [Copy email untuk validasi dianggap normalisasi argument] → Assert nilai raw yang sampai fake RPC identik, termasuk kapitalisasi/spasi.
- [Login tidak sengaja memakai rule register] → Uji credential Unicode/whitespace/control yang panjang byte-nya valid melalui usecase dan HTTP.
- [Scope melebar ke response lifecycle dari plan besar] → Kunci success fixture existing dan tidak ubah DTO/gateway/middleware.
- [Catalog baru membuat dependency ke transport] → Catalog domain hanya berisi error/message; FieldErrors dikenali mapper tanpa Fiber/protobuf di domain.

## Migration Plan

Tidak ada migrasi data, dependency, atau konfigurasi. Setelah apply, dokumentasikan input yang kini ditolak lebih awal, jalankan seluruh acceptance dan review perubahan sebelum archive. Deployment/commit/push bukan bagian change ini. Jika rollback diperlukan, kembalikan hanya delta validator, pemanggilan usecase, test, dan dokumentasi change ini.
