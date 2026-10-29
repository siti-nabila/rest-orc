# Aturan Validasi dan Mapping grpc-auth di rest-orc

Status: keputusan plan diperbarui pada 23 Oktober 2026; menjadi acuan implementasi berikutnya. Route dan mapping baru belum diimplementasikan.

## Cara Membaca

- Keputusan menggabungkan pilihan user dan contract backend yang diperiksa pada 23 Oktober 2026.
- Kolom aturan mencatat required/optional, panjang, allowlist, normalisasi, dan pesan error; isian [sesuai saran REST] berarti bentuk yang dijelaskan pada baris itu disetujui.
- Bedakan aturan register, login, dan penggantian password jika kebijakannya berbeda.
- Batas backend adalah acuan contract saat inspeksi, bukan keputusan baru. Aturan REST harus kompatibel dengan batas tersebut.
- Contoh nilai hanya menjelaskan bentuk data, bukan nilai default.
- **Saran REST** menjelaskan bentuk transport yang disepakati. Ringkasan endpoint ada pada bagian 15, options pada bagian 16, dan permission/response pada bagian 17; route baru belum diimplementasikan.

## 1. Alur dan Tanggung Jawab

```text
Input FE
  --> parsing dan validasi transport
  --> normalisasi dan validasi aturan feature
  --> mapping ke protobuf
  --> pemanggilan grpc-auth
  --> mapping response atau error REST
```

Request tidak valid berhenti sebelum RPC bisnis dipanggil. Middleware authentication dapat lebih dahulu memanggil `Me` pada route protected. grpc-auth tetap memverifikasi credential, permission, dan aturan bisnis authoritative.

| Layer | Tanggung jawab |
| --- | --- |
| `handler/http` | Parsing JSON/path/query/header, validasi transport, presence field, request/response DTO |
| `domain` | Aturan input dan normalisasi murni milik feature |
| `usecase` | Menjalankan validasi dan normalisasi sebelum orchestration |
| `gateway/authservice` | Mapping model domain ke/dari protobuf |
| `internal/client/authservice` | Reusable client dan pemanggilan RPC |
| `internal/response` | HTTP status, error per field, dan response envelope |

## 2. Register dan Login

Saran REST: register memakai `POST {base_url}/users/create`, login memakai `POST {base_url}/users/login`. Keduanya mempertahankan route existing, dengan body JSON `{"email":"nama@example.com","password":"Contoh123!"}`.

| Fitur | Field | Untuk apa | Aturan dari user | Batas backend saat ini | Keputusan REST |
| --- | --- | --- | --- | --- | --- |
| Register/login | `email` | Identitas login | ASCII; pola email mengikuti backend | Required; 6–50 byte | REST meneruskan nilai tanpa perubahan; backend login/admin create tetap trim/lowercase. Simbol `.`, `_`, `-`, `+` diterima jika lolos validator backend. |
| Register | `password` | Credential akun baru | Tanpa kompleksitas komposisi | Required; 6–50 byte | ASCII printable tanpa spasi (`0x21–0x7E`), tanpa kewajiban campuran huruf besar/kecil/angka/simbol. Tidak di-trim atau diubah kapitalisasinya. |
| Login | `password` | Membuktikan credential existing | Tanpa kompleksitas atau pembatasan karakter baru | Required; 6–50 byte | Diteruskan persis, termasuk credential lama yang tidak memenuhi aturan karakter password baru. |
| Register | `confirm_password` | Konfirmasi kecocokan password bila dibutuhkan FE/REST | Belum diminta sebagai field wajib | Tidak ada pada protobuf | Perlu/tidak: `[tidak perlu]`; jika ada, hanya diperiksa di REST dan tidak diteruskan |

Backend saat ini tidak mensyaratkan kompleksitas password. Jika backend kelak menambah kebijakan komposisi, terapkan pada pembuatan/penggantian credential; login/current_password tetap dapat membuktikan credential existing.

Password tidak dinormalisasi otomatis. Jangan trim, mengubah kapitalisasi, atau membuang karakter dari nilai credential.

## 3. Update Profile Sendiri

Saran REST: `PATCH {base_url}/me/profile`, dengan body JSON berisi field yang dipilih dari `name`, `phone`, `country_code`, dan `address`. Identitas pemilik profil diambil dari bearer token. Aturan omitted/null/string kosong tetap mengikuti keputusan pada tabel di bawah.

| Field | Untuk apa | Aturan dari user | Normalisasi | Batas backend saat ini | Keputusan REST |
| --- | --- | --- | --- | --- | --- |
| `name` | Nama lengkap pengguna | Huruf ASCII, spasi, apostrof, dan tanda hubung `-` | Trim tepi; gabungkan spasi menjadi satu; Proper Case | Field terpilih 3–100 karakter Unicode di backend | REST ASCII-only; panjang setelah normalisasi. Proper Case mengkapitalkan awal kata tanpa memaksa sisa huruf lowercase agar singkatan tetap utuh. Required pada create; PATCH hanya jika dipilih. |
| `phone` | Nomor telepon pengguna | Digit ASCII setelah normalisasi | Untuk country_code +62: hapus prefix +62/62, lalu satu awalan 0; nilai 8… tetap | Maksimal 20 karakter Unicode di backend | REST 12–20 digit **sesudah normalisasi**. Contoh 081234567890 menjadi 11 digit dan ditolak oleh minimum yang dipilih. Prefix negara lain tidak dihapus. Required pada create; PATCH hanya jika dipilih. |
| `country_code` | Kode negara nomor telepon | Tanda + lalu 1–3 digit ASCII, digit awal bukan 0 | Tambahkan + pada input digit; jangan menebak negara | Validator backend `^\+[1-9][0-9]{0,2}$`; 2–4 byte termasuk + | Mengikuti validator backend, bukan batas 5 pada draft lama; contoh 62 menjadi +62. Required pada create; PATCH hanya jika dipilih. Jika country_code omitted pada PATCH, jangan menebak negara existing; nomor tanpa prefix tetap diterima. |
| `address` | Alamat pengguna | ASCII printable 0x20–0x7E dan CR/LF; kontrol lain ditolak | Tidak diubah | Maksimal 250 karakter Unicode di backend | Opsional pada create; jika dipilih 3–200 byte. PATCH omitted/null diabaikan; string kosong ditolak sesuai minimum. |

### Semantik Update Parsial

| Bentuk input | Usulan perilaku | Keputusan |
| --- | --- | --- |
| Field tidak dikirim | Diabaikan | Nilai existing tidak berubah; tidak masuk field_mask. |
| `address` atau `phone` berupa string kosong | Mengosongkan nilai jika diizinkan aturan REST | `[return error karena aturan validasi diatas]` |
| `name` berupa string kosong | Ditolak karena batas minimum backend | Mengikuti contract backend |
| Field bernilai `null` | Diabaikan | Sama dengan omitted; required pada create tidak berarti wajib hadir dalam PATCH. |
| Body tanpa field yang dapat diubah | Diabaikan sebagai no-op | HTTP 200 dengan envelope sukses dan data {"changed":false}; tidak memanggil RPC mutasi dan tidak mengarang version. Authentication/authorization tetap dijalankan. |
| Field di luar allowlist | Tolak atau abaikan sesuai keputusan contract REST | `[abaikan]` |

rest-orc membentuk field_mask dari field dikenal, non-null, dan lolos validasi. No-op tanpa field efektif juga berlaku pada PATCH profil target dan metadata role. Path ID, header Version/Idempotency-Key yang diwajibkan, authentication, dan permission tetap divalidasi; no-op lokal tidak menjanjikan pengecekan keberadaan/version terbaru target karena RPC mutasi tidak dipanggil. Owner profil berasal dari principal terverifikasi, bukan `user_id` dari FE.

## 4. Password dan Lifecycle Authentication

| Fitur/RPC | Field | Untuk apa | Acuan contract | Keputusan REST |
| --- | --- | --- | --- | --- |
| `ChangePassword` | `current_password` | Pembuktian credential existing | Required; 6–50 byte | Tanpa kompleksitas/pembatasan karakter baru; diteruskan persis. |
| `ChangePassword` | `new_password` | Password pengganti | Required; 6–50 byte | Aturan credential baru mengikuti register; tanpa kompleksitas komposisi; harus berbeda dari current_password sesuai keputusan REST. |
| `RefreshSession` | `refresh_token` | Credential opaque | Required; rotasi tanpa retry otomatis | POST /auth/refresh tanpa body credential. REST membaca cookie HttpOnly dan mengisi RefreshSessionRequest.refresh_token; tidak meneruskan bearer dan tidak mewajibkan access token valid. |
| `Logout` | `refresh_token` | Mode refresh untuk logout | Wajib nonempty jika dipilih; tanpa bearer metadata | REST membaca cookie ketika header Authorization tidak dikirim; bukan dari JSON browser. |
| `Logout` | Header `Authorization` | Mode bearer untuk logout | Tepat satu mode credential pada RPC | Jika header dikirim, pilih mode bearer dan abaikan cookie untuk RPC ini; tanpa header pilih cookie. Bearer invalid tidak otomatis fallback. Hapus cookie setelah logout berhasil. |
| `Me` | Tidak ada body | Membaca profil, status akun, role, akses aplikasi, dan permission pengguna login | Identitas berasal dari token; target pengayaan backend pada bagian 8 | Tanpa input `user_id` atau `application_code`; cakupan mengikuti namespace service caller **Saran REST:** `GET {base_url}/me`, bearer token, tanpa body/query target user atau aplikasi. |

Cookie adalah transport REST: grpc-auth menerima refresh_token pada field protobuf dan bearer pada metadata. Refresh/logout tidak memakai middleware Me yang mengharuskan access token; credential mode diverifikasi backend.

Backend saat ini membuat registrasi `PENDING` tanpa sesi. Login menghasilkan access token, refresh token, session ID, expiry, dan `must_change_password`. Change password mencabut seluruh sesi dan meminta login ulang.

## 5. Administrasi User

| Fitur/RPC | Field | Untuk apa | Acuan contract | Keputusan REST |
| --- | --- | --- | --- | --- |
| `CreateUser` | `email` | Identitas login user baru | Required; 6–50 karakter; backend trim/lowercase | Pakai aturan register atau berbeda: `[mengikuti aturan register]` |
| `CreateUser` | `password` | Password awal user | Required; 6–50 karakter; tidak dikirim kembali | Pakai aturan register atau berbeda: `[mengikuti aturan register]` |
| `CreateUser` | `name` | Nama lengkap user | Required; 3–100 karakter | Pakai aturan profil atau berbeda: `[mengikuti aturan profil]` |
| `CreateUser` | `address` | Alamat user | Maksimal 250 karakter | Pakai aturan profil atau berbeda: `[mengikuti aturan profil]` |
| `CreateUser` | `phone` | Nomor telepon user | Maksimal 20 karakter | Mengikuti normalisasi dan validasi phone profil. |
| `CreateUser` | `country_code` | Kode panggilan telepon | Optional di protobuf; validator domain + lalu 1–3 digit | Required di REST sesuai aturan profil; dipakai untuk normalisasi phone. |
| `CreateUser` | `status` | Status awal akun | Backend mewajibkan explicit PENDING/ACTIVE | REST menetapkan PENDING; aktivasi oleh admin melalui ChangeUserStatus menjadi ACTIVE. Tidak ada aktivasi mandiri; input status yang bertentangan dengan PENDING ditolak. |
| `GetUser` | `user_id` | User target; dipetakan ke protobuf `id` | ID positif dalam rentang PostgreSQL INTEGER | Bentuk path REST: `[sesuai saran REST]` **Saran REST:** `GET {base_url}/users/{user_id}`; ID target berada pada path. |
| `UpdateUser` | `user_id` | User yang profilnya diubah | ID positif dalam rentang PostgreSQL INTEGER | Bentuk path REST: `[sesuai saran REST]` **Saran REST:** `PATCH {base_url}/users/{user_id}/profile`; field profil pada body JSON. |
| `UpdateUser` | `name`, `address`, `phone`, `country_code` | Field profil yang diubah admin | Presence cocok dengan `field_mask`; batas panjang sama seperti create | Pakai aturan profil atau berbeda: `[pakai aturan profil]` |
| `UpdateUser` | `expected_user_version` | Versi user saat terakhir dibaca | Required; positif dalam rentang int64 | Sumber field/header REST: `[sesuai saran REST]` **Saran REST:** Header `Version: <version>` dari response user target terakhir, dipetakan ke `expected_user_version`. |
| `ChangeUserStatus` | `user_id` | User target | ID positif dalam rentang PostgreSQL INTEGER | Bentuk path REST: `[sesuai saran REST]` **Saran REST:** `PATCH {base_url}/users/{user_id}/status`, body JSON `status` dan `reason`. |
| `ChangeUserStatus` | `status` | Status tujuan akun | Code string dan transisi backend existing | Input string dari pilihan status; tidak mengganti tipe protobuf menjadi enum. REST memvalidasi allowlist; grpc-auth tetap memvalidasi transisi. |
| `ChangeUserStatus` | `reason` | Alasan tindakan untuk audit | Required; trim; 3–500 karakter | Karakter, simbol, dan baris baru: `[setuju dengan usulan]` |
| `ChangeUserStatus` | `expected_user_version` | Versi user terakhir | Required; positif dalam rentang int64 | Sumber field/header REST: `[sesuai saran REST]` **Saran REST:** Header `Version: <version>` dari response detail user target, bukan Me actor/admin. |

Transisi backend: `PENDING → ACTIVE/DEACTIVATED`, `ACTIVE → SUSPENDED/DEACTIVATED`, `SUSPENDED → ACTIVE/DEACTIVATED`, `DEACTIVATED → ACTIVE`. Tidak dapat kembali ke `PENDING`.

Email, password, status, role, dan permission tidak diubah melalui update profil. Backend menangani status restricted, larangan self suspend/deactivate, dan perlindungan admin terakhir.

## 6. Role

| Fitur/RPC | Field | Untuk apa | Acuan contract | Keputusan REST |
| --- | --- | --- | --- | --- |
| `GetRole` | `role_id` | Role target | ID positif dalam rentang PostgreSQL INTEGER | Bentuk path REST: `[sesuai saran REST]` **Saran REST:** `GET {base_url}/roles/{role_id}`. |
| `CreateRole` | `application_code` | Aplikasi pemilik role | Required; aplikasi bisnis aktif; kode lowercase maksimal 50 | Pilihan dari catalog dan label FE: `[get list application, label dari application.name bentuk tampilan di FE nya '<application.code> - <application.name>']` |
| `CreateRole` | `code` | Kode immutable role baru | Required; pola application.role; lowercase/angka/underscore; maksimal 100 | REST menerima string kode lengkap yang diisi FE, bukan pilihan role existing. GetRoles memasok pilihan grant/revoke/filter. Backend memeriksa prefix dan keunikan. |
| `CreateRole` / `UpdateRole` | `name` | Nama tampilan role | Trim; 1–150 karakter | Huruf ASCII, spasi, apostrof, dan -; normalisasi/Proper Case mengikuti nama profil. PATCH omitted/null diabaikan; string kosong invalid. |
| `CreateRole` / `UpdateRole` | `description` | Penjelasan fungsi role | Opsional; trim; maksimal 100; selected-empty mengosongkan | Simbol dan baris baru: `[diperbolehkan]` |
| `CreateRole` | `permission_codes` | Permission awal role | Permission terdaftar dari aplikasi yang sama; daftar kosong diperbolehkan | Cara pemilihan FE: `[diambil dari list permission dari BE, bentuk tampilan di FE nya '<permission.code> - <permission.description>']` |
| `UpdateRole` | `role_id`, `expected_role_version` | Target dan versi role terakhir | ID positif; versi required positif int64 | Bentuk path dan sumber versi: `[sesuai saran REST]` **Saran REST:** `PATCH {base_url}/roles/{role_id}`; header `Version: <version>` dari detail role; body `name`/`description` yang dipilih. |
| `UpdateRole` | Presence name / description | Memilih metadata yang diubah | field_mask hanya memuat field efektif | Omitted/null/unknown diabaikan. Name kosong ditolak; description kosong menghapus nilai. Tanpa field efektif, no-op REST tanpa RPC mutasi. |
| `ChangeRoleStatus` | `role_id`, `expected_role_version` | Target dan versi role terakhir | Required | Bentuk path dan sumber versi: `[sesuai saran REST]` **Saran REST:** `PATCH {base_url}/roles/{role_id}/status`; header `Version: <version>` dari detail role; body `status` dan `reason`. |
| `ChangeRoleStatus` | `status` | Status tujuan role | Required ACTIVE atau DISABLED | FE membaca opsi GetStatuses rest-orc dari code string backend existing; tanpa RPC/enum baru pada grpc-auth. |
| `ChangeRoleStatus` | `reason` | Alasan perubahan | Required; trim; 3–500 karakter | Karakter/simbol/baris baru: `[diperbolehkan, dan setuju dengan usulan]` |
| `ReplaceRolePermissions` | `role_id`, `expected_role_version` | Target dan versi role terakhir | Required | Bentuk path dan sumber versi: `[sesuai saran REST]` **Saran REST:** `PUT {base_url}/roles/{role_id}/permissions`; header `Version: <version>` dari detail role. |
| `ReplaceRolePermissions` | `permissions.codes` | Daftar final permission | Wrapper wajib hadir; daftar kosong menghapus seluruh mapping; hanya permission terdaftar dari aplikasi yang sama | Bentuk body REST dan pemilihan FE: `[FE get list permission nya, bentukan list di FE nya adalah '<permission.code> - <permission.description>']` **Saran REST:** Body `{"permissions":{"codes":["clm.document.read"]},"reason":"Penyesuaian akses"}`. `codes` adalah daftar akhir; `[]` menghapus seluruh mapping. |
| `ReplaceRolePermissions` | `reason` | Alasan penggantian permission | Required; trim; 3–500 karakter | Karakter/simbol/baris baru: `[setuju, sesuai dengan usulan]` |

Kode permission maksimal 150 karakter dan dapat memiliki lebih dari tiga segmen, misalnya `iam.user.status.update`. Backend melindungi role sistem/IAM dari mutasi yang dilarang.

## 7. Assignment Role dan Akses Aplikasi

| Fitur/RPC | Field | Untuk apa | Acuan contract | Keputusan REST |
| --- | --- | --- | --- | --- |
| `ListUserRoles` | `user_id` | User pemilik assignment | ID positif | Bentuk path REST: `[sesuai saran REST]` **Saran REST:** `GET {base_url}/users/{user_id}/roles`; GET membaca daftar role milik user pada path. |
| `GrantUserRoles` / `RevokeUserRoles` | `user_id` | User target | ID positif | Bentuk path REST: `[sesuai saran REST]` **Saran REST:** Grant: `POST {base_url}/users/{user_id}/roles/grant`; revoke: `POST {base_url}/users/{user_id}/roles/revoke`. Body `role_ids` dan `reason`. |
| `GrantUserRoles` / `RevokeUserRoles` | `role_ids` | Role yang dipilih | Required; 1–20 entri; setiap ID positif | Cara pemilihan dan pesan error duplikat: `[tidak boleh duplikat role, return error jika duplikat, pada sisi FE harus get list role, dengan format tampilannya '<role.code> - <role.description>']` |
| `GrantUserRoles` / `RevokeUserRoles` | `reason` | Alasan tindakan | Required; trim; 3–500 karakter | Karakter/simbol/baris baru: `[sesuai usulan]` |
| `GrantUserRoles` / `RevokeUserRoles` | `expected_user_version` | Versi user terakhir | Required; positif int64 | Sumber versi: `[sesuai saran REST]` **Saran REST:** Header `Version: <version>` dari detail user target. |
| `ListUserApplicationAccess` | `user_id` | User pemilik akses aplikasi | ID positif | Bentuk path REST: `[sesuai saran REST]` **Saran REST:** `GET {base_url}/users/{user_id}/applications`. |
| `GrantApplicationAccess` / `RevokeApplicationAccess` | `user_id` | User target | ID positif | Bentuk path REST: `[sesuai saran REST]` **Saran REST:** Grant: `POST {base_url}/users/{user_id}/applications/{application_code}/grant`; revoke: `POST {base_url}/users/{user_id}/applications/{application_code}/revoke`. |
| `GrantApplicationAccess` / `RevokeApplicationAccess` | `application_code` | Aplikasi target | Required; kode lowercase maksimal 50 | FE memilih value string dari GetApplications; kode berada pada path. Grant mengirim role_ids dan reason, revoke hanya reason. Grant ulang mengikuti idempotency/no-op backend, bukan otomatis error duplikat. |
| `GrantApplicationAccess` | `role_ids` | Role awal yang dipilih untuk aplikasi | Required; 1–20 entri dari aplikasi yang sama | Cara pemilihan FE: `[tidak boleh duplikat role, return error jika duplikat, pada sisi FE harus get list role, dengan format tampilannya '<role.code> - <role.description>']` |
| `GrantApplicationAccess` / `RevokeApplicationAccess` | `reason` | Alasan tindakan | Required; trim; 3–500 karakter | Karakter/simbol/baris baru: `[diperbolehkan]` |
| `GrantApplicationAccess` / `RevokeApplicationAccess` | `expected_user_version` | Versi user terakhir | Required; positif int64 | Sumber versi: `[sesuai saran REST]` **Saran REST:** Header `Version: <version>` dari detail user target. |

FE mengambil pilihan siap tampil dari GetRoles/GetPermissions/GetApplications rest-orc (bagian 16), lalu mengirim value string, bukan label gabungan. Body role_ids pada endpoint baru berupa string desimal 1–2147483647, diparse ke uint64 protobuf. Filter role List User existing tetap CSV role code numerik. Duplikat ID dalam satu payload ditolak; grant ulang ACTIVE yang eligible tetap mengikuti no-op backend. Input application_code hanya satu pada path; tidak menambahkan error duplikat untuk grant existing yang idempotent. Usulan: jangan mencari ID berdasarkan nama bebas yang mungkin duplikat. Jumlah entri role diperiksa sebelum deduplikasi agar batas backend tetap dipertahankan.

## 8. Permission Efektif dan Evaluasi Akses

### Akses Pengguna Login melalui Me

Keputusan: profil dan akses pengguna login dibaca melalui satu RPC Me. Implementasi backend lokal selesai sesuai [Me snapshot](../../grpc-auth/docs/me-access-snapshot.md). Publikasi/pin api-contracts, consumer, dan deployment tetap tahap terpisah. Me.phone sudah berupa country_code + phone; REST tidak melakukan concat ulang. Detail/list/profile tetap mengikuti mapping existing.

| Data response Me | Untuk apa | Mapping REST yang disepakati |
| --- | --- | --- |
| Identitas dan profil | Data pengguna yang sedang login | Diterima dari grpc-auth dan dipetakan ke response REST |
| `status`, `version`, `must_change_password` | Status akun, versi entity user, dan pembatasan penggantian password | Versi user tidak menjadi versi kebijakan permission |
| `roles`, `application_access` | Role dan akses aplikasi aktif | Cakupan mengikuti namespace service caller; legacy `role_codes` dan `role_names` hanya metadata compatibility |
| `permissions` | Permission efektif pengguna login | `map[string]bool`; permission yang dimiliki bernilai `true`; key yang tidak ada dianggap tidak memiliki izin |

- Me tidak menerima target `user_id` atau filter `application_code` dari FE. Identitas diambil dari token; cakupan akses ditetapkan oleh namespace service caller.
- Tidak diperlukan endpoint FE terpisah untuk `GetMyEffectivePermissions` atau agregasi Me dengan RPC permission di rest-orc.
- FE memakai permission map untuk menu/tombol. rest-orc dan service lain memvalidasi request protected menggunakan hasil Me yang diperoleh langsung dari grpc-auth untuk request tersebut, bukan map kiriman FE atau hasil dari request sebelumnya.
- Permission route ditentukan server. grpc-auth tetap memverifikasi otorisasi RPC protected; service domain seperti CLM tetap memeriksa aturan resource.
- Restricted user memperoleh profil minimum tanpa collection privilege. Kegagalan pengambilan akses menghasilkan error, bukan permission map kosong sebagai fallback.

### Pembacaan Permission User Target oleh Admin

| Fitur/RPC | Field | Untuk apa | Acuan contract | Keputusan REST |
| --- | --- | --- | --- | --- |
| `GetUserEffectivePermissions` | `user_id` | User target yang dipilih admin | ID positif; berbeda dari identitas actor pada Me | Sumber ID: path REST untuk user target. Bentuk path REST: `[sesuai saran REST]` **Saran REST:** `GET {base_url}/users/{user_id}/permissions`. |
| `GetUserEffectivePermissions` | `application_code` | Filter aplikasi user target | Kosong mengikuti cakupan yang diizinkan backend | Perlu diwajibkan di REST atau tetap opsional: `[sesuai saran REST]` **Saran REST:** Opsional sebagai query pada endpoint permission yang sama: `GET {base_url}/users/{user_id}/permissions?application_code=clm`. Omitted mengikuti cakupan backend. |

### Evaluasi Aksi Internal Opsional

`EvaluateAccess` tetap tersedia sebagai kemampuan internal opsional untuk memeriksa satu aksi beserta alasan keputusan. Caller server menentukan `application_code` dan `permission_code` untuk aksi tersebut. Contract existing tetap berlaku: application code required, lowercase, maksimal 50; permission code required, exact, maksimal 150, namespace cocok, tanpa wildcard. Tidak diperlukan endpoint FE tambahan untuk perlindungan route.

RPC backend `GetMyEffectivePermissions`, `GetUserEffectivePermissions`, dan `EvaluateAccess` tetap dipertahankan. Validasi input, operasi administrasi user/role/aplikasi, sesi, audit, versioning, dan idempotency tetap mengikuti bagian masing-masing.

## 9. Pengelolaan Sesi

| Fitur/RPC | Field | Untuk apa | Acuan contract | Keputusan REST |
| --- | --- | --- | --- | --- |
| `ListMySessions` | Pagination | Membaca sesi sendiri | Owner berasal dari principal | Lihat aturan pagination **Saran REST:** `GET {base_url}/me/sessions?page=1&limit=20`. |
| `RevokeMySession` | `session_id` | Sesi sendiri yang dipilih | Required; UUID canonical; ownership diperiksa backend | Bentuk path REST: `[sesuai saran REST]` **Saran REST:** `DELETE {base_url}/me/sessions/{session_id}`, tanpa body. |
| `RevokeAllMySessions` | Tidak ada body | Mencabut seluruh sesi sendiri | Owner berasal dari principal | Tidak ada field bisnis **Saran REST:** `POST {base_url}/me/sessions/revoke-all`, tanpa body. |
| `ListUserSessions` | `user_id` | User pemilik sesi | Required; ID positif | Bentuk path REST: `[sesuai saran REST]` **Saran REST:** `GET {base_url}/users/{user_id}/sessions?page=1&limit=20`. |
| `RevokeUserSession` | `user_id`, `session_id` | User target dan sesi yang dicabut | ID positif; UUID canonical; sesi harus milik target | Bentuk path REST: `[sesuai saran REST]` **Saran REST:** `DELETE {base_url}/users/{user_id}/sessions/{session_id}`, tanpa body; header `Idempotency-Key` wajib. |
| `RevokeAllUserSessions` | `user_id` | User target | Required; ID positif | Bentuk path REST: `[sesuai saran REST]` **Saran REST:** `POST {base_url}/users/{user_id}/sessions/revoke-all`, tanpa body; header `Version` dan `Idempotency-Key` wajib. |
| `RevokeAllUserSessions` | `expected_user_version` | Versi entity user, bukan auth version | Required; positif int64 | Sumber versi: `[sesuai saran REST]` **Saran REST:** Header `Version: <version>` dari response detail user target terakhir. |

## 10. Pencarian, Pagination, dan Filter

Nama query REST untuk kemampuan baru masih usulan; nama route/query existing dipertahankan kecuali perubahan disepakati.

| Digunakan pada | Field | Untuk apa / acuan contract | Keputusan REST |
| --- | --- | --- | --- |
| Endpoint list | `page` | Nomor halaman | Jika dikirim integer 1–2147483647; omitted menjadi 1. Tidak dibatasi total_pages; halaman di luar hasil mengembalikan items kosong. |
| Endpoint list | `limit` | Jumlah item | Jika dikirim integer 1–100; omitted mengikuti default backend per endpoint. Tidak dibatasi total. Catalog/assignment/sesi/audit default 20; users mempertahankan default backend existing (10 pada source ORM workspace saat inspeksi). Contoh 20 merupakan nilai eksplisit. REST menolak 0, negatif, dan >100. |
| Endpoint list | `last_id` | Cursor opaque dari response sebelumnya; tidak diubah/ditebak sebagai ID numerik | Batas panjang REST: `[sesuai saran REST]` **Saran REST:** Query `last_id=<cursor>` dengan URL encoding; kirim persis nilai cursor dari response. |
| Users/role/permission/application/assignment jika RPC mendukung | `keyword` | Pencarian pada sumber yang ditentukan server | ASCII maksimal 200 byte sebelum/sesudah normalisasi. Trim/gabung whitespace mengikuti backend; whitespace-only berarti tanpa search. Kirim logical field keyword dan FULL_TEXT_TRIGRAM, bukan field fisik dari FE. |
| List users | `created_from` | Tanggal awal inklusif; existing `YYYY-MM-DD`, Jakarta | Pertahankan format atau perubahan yang disepakati: `[sesuai saran REST]` **Saran REST:** Query `created_from=2026-10-01`; pertahankan format tanggal existing. |
| List users | `created_to` | Tanggal akhir inklusif; REST mengubah ke batas awal hari berikutnya | Rentang maksimum: `[sesuai saran REST]` **Saran REST:** Query `created_to=2026-10-22`; pertahankan makna tanggal akhir inklusif existing. |
| List users | `role` | CSV role code numerik positif existing; berbeda dari role ID | Tetap CSV atau rencana additive lain: `[sesuai saran REST]` **Saran REST:** Query existing `role=1,2`; nilai adalah role code numerik existing, bukan role ID. |
| List users | `statuses` | Filter status akun | Cara kirim daftar dan label FE: `[sesuai saran REST]` **Saran REST:** Query CSV `statuses=ACTIVE,SUSPENDED`; kirim code backend, label hanya untuk tampilan FE. |
| List users | `application_code` | Filter aplikasi | Cara pemilihan: `[sesuai saran REST]` **Saran REST:** Query `application_code=clm`; nilai dipilih dari catalog aplikasi. |
| `ListRoles` | `application_code`, `statuses` | Filter aplikasi dan status role `ACTIVE`/`DISABLED` | Cara kirim daftar: `[sesuai saran REST]` **Saran REST:** `GET {base_url}/roles?application_code=clm&statuses=ACTIVE,DISABLED`. |
| `ListPermissions` | `application_code` | Filter catalog permission berdasarkan aplikasi | Required/optional di REST: `[opsional, sesuai saran REST]` **Saran REST:** `GET {base_url}/permissions?application_code=clm`; usulan opsional, omitted mengikuti cakupan backend. |
| `ListApplications` | `statuses` | Filter status aplikasi | Cara kirim daftar dan label FE: `[sesuai saran REST]` **Saran REST:** `GET {base_url}/applications?statuses=ACTIVE`; code status mengikuti enum backend. |
| `ListUserRoles` | `application_code`, `statuses` | Filter aplikasi dan status assignment `ACTIVE`/`REVOKED` | Cara kirim daftar: `[sesuai saran REST]` **Saran REST:** `GET {base_url}/users/{user_id}/roles?application_code=clm&statuses=ACTIVE,REVOKED`. |
| `ListUserApplicationAccess` | `application_code`, `statuses` | Filter aplikasi dan status akses `ACTIVE`/`REVOKED` | Cara kirim daftar: `[sesuai saran REST]` **Saran REST:** `GET {base_url}/users/{user_id}/applications?application_code=clm&statuses=ACTIVE,REVOKED`. |
| List audit/sesi | Pagination | Backend default page 1, limit 20, maksimal 100 | Default REST: `[sesuai saran REST]` **Saran REST:** Query `page=1&limit=20`; usulan tetap memakai default backend dan maksimum 100 untuk audit/sesi. |

### keyword pagination per feature
- Users: user.phone / user.email / user.name
- role: role.name / role.code
- permission: permission.code
- application: application.code / application.name
- assignment: role.name, example: "reviewer"

### Pagination mode / config
- Search mode : fulltext trigram
- List User :
  - Keyword : phone, email, name
  - Sort : id, email, status
  - Filter : statuses, role, application_code, created_from, created_to
  - Default sort : id DESC
- List Role :
  - Keyword : name, code
  - Sort : id, code, name, application_code, created_at, updated_at
  - Filter : application_code, statuses
  - Default sort : created_at DESC, updated_at DESC, id DESC
- List Permission :
  - Keyword : code
  - Sort : id, code, application_code
  - Filter : application_code
  - Default sort : id DESC
- List Application :
  - Keyword : code, name
  - Sort : id, code, name, status
  - Filter : statuses
  - Default sort : id DESC
- List Assignment (role milik satu user):
  - Scope : user_id dari path /users/{user_id}/roles
  - Keyword : role.name
  - Sort : id, code, name, application_code, status, granted_at, revoked_at
  - Filter : application_code, statuses
  - statuses : ACTIVE, REVOKED
  - Default sort : id ASC
  - Default statuses : ACTIVE dan REVOKED (histori, bukan permission efektif)


Konfigurasi di atas adalah target REST; default sort dikirim eksplisit ke backend. Multi-sort diakhiri ID unik; reset cursor ketika filter/keyword/sort berubah dan gunakan cursor persis dari response. Country code users, tanggal roles, dan exact code permissions ditunda sesuai [plan filter grpc-auth](../../grpc-auth/docs/list-filter-extension-plan.md); query tersebut ditolak pada tahap awal agar tidak diam-diam diabaikan. Field selection, search mode, dan sort ditentukan rest-orc per endpoint. Tidak semua RPC mendukung seluruh kemampuan `PageQuery`; audit tidak menerima fields/sort/search, dan sesi tidak menerima fields/search.

## 11. Audit Log

| Field | Untuk apa | Acuan contract | Keputusan REST |
| --- | --- | --- | --- |
| `actor_user_id` | Filter user pelaku historis | Opsional; jika hadir ID positif PostgreSQL INTEGER | Cara memilih user: `[sesuai saran REST]` **Saran REST:** Query `actor_user_id=123`; FE mengirim ID user yang dipilih untuk pencarian. |
| `actor_service` | Filter service pelaku historis | Maksimal 100 karakter | Daftar pilihan atau input bebas: `[sesuai saran REST]` **Saran REST:** Query `actor_service=<kode-service>` dengan URL encoding. |
| `action` | Filter jenis aktivitas | Maksimal 100; nilai tak dikenal menghasilkan hasil kosong | Daftar pilihan atau input bebas: `[sesuai saran REST]` **Saran REST:** Query `action=<kode-aksi>` dengan URL encoding. |
| `target_type` | Filter jenis objek | Maksimal 50 | Daftar pilihan atau input bebas: `[sesuai saran REST]` **Saran REST:** Query `target_type=<jenis-objek>` dengan URL encoding. |
| `target_id` | Filter identitas objek | Maksimal 100; membutuhkan `target_type` | Karakter dan format per jenis objek: `[sesuai saran REST]` **Saran REST:** Query `target_id=<id-objek>` bersama `target_type`. |
| `occurred_from` | Tanggal awal inklusif | REST YYYY-MM-DD dalam Asia/Jakarta | Dipetakan ke awal hari lokal, lalu UTC protobuf Timestamp. |
| `occurred_to` | Tanggal akhir inklusif | REST YYYY-MM-DD dalam Asia/Jakarta | Dipetakan ke awal hari berikutnya, lalu field backend occurred_before eksklusif. from > to ditolak; batas satu sisi mengikuti kemampuan backend. |

Filter pelaku audit hanya untuk pencarian; tidak mengubah identitas caller. Nama REST selalu xxx_from/xxx_to; occurred_before hanya nama field protobuf. from=to mencakup satu hari lokal penuh. Tanggal malformed ditolak sebelum RPC audit.

## 12. Header dan Field Teknis

| Field/header | Untuk apa | Sumber / aturan | Keputusan REST |
| --- | --- | --- | --- |
| `Authorization` | Access token pengguna | Dari request FE; diteruskan sebagai bearer metadata untuk operasi protected | Body/cookie/header policy: `[sesuai saran REST]` **Saran REST:** Header `Authorization: Bearer <access_token>` untuk operasi protected. FE membaca access token dari memory lalu memasangnya pada header. |
| `Idempotency-Key` | Menjaga retry tindakan yang sama | Diteruskan sebagai metadata `idempotency-key`; nonblank maksimal 200; key yang sama untuk retry payload yang sama | Cara FE menghasilkan/menyimpan key selama retry: `[sesuai saran REST]` **Saran REST:** Header `Idempotency-Key: <key>`; FE dapat membuat UUID sekali per tindakan dan mempertahankannya untuk retry payload yang sama. UUID adalah saran pembangkitan, bukan pola wajib backend. |
| `expected_user_version` | Mencegah update saling menimpa | Dari response terakhir; tidak otomatis diganti rest-orc saat conflict | Cara FE menangani conflict: `[sesuai saran REST]` **Saran REST:** Header `Version: <version>` dari user target; bila conflict, baca ulang target dan minta pengguna meninjau perubahan sebelum mengirim tindakan baru. |
| `expected_role_version` | Mencegah update role saling menimpa | Dari response terakhir; tidak otomatis diganti rest-orc saat conflict | Cara FE menangani conflict: `[sesuai saran REST]` **Saran REST:** Header `Version: <version>` dari role target; penanganan conflict sama seperti user. |
| `field_mask` | Menentukan field yang benar-benar berubah | Usulan dibentuk rest-orc dari presence body | Setujui atau tentukan bentuk lain: `[sesuai saran REST]` **Saran REST:** Tidak dikirim FE; rest-orc membentuknya setelah aturan presence/null dan allowlist body diterapkan. |
| `x-service-token` | Membuktikan identitas rest-orc ke grpc-auth | Dari environment server; dikirim melalui TLS; bukan input FE | Nama environment final saat implementasi: `[sesuai saran REST]` **Saran REST:** Tidak menjadi header input FE; rest-orc mengisinya dari konfigurasi server. |
| Locale/request ID/context | Korelasi, bahasa, deadline, cancellation | Dipropagasikan sampai downstream | Perubahan dari existing jika ada: `[sesuai saran REST]` **Saran REST:** Pertahankan header locale/request ID existing dan propagasi context; tidak menambah field body. |

Idempotency key wajib pada create user, change user status, create role, change role status, replace role permissions, grant/revoke role user, grant/revoke application access, revoke satu sesi user oleh admin, dan revoke seluruh sesi user oleh admin. Update user/role menerima key opsional.

## 13. Format Error dan Keputusan Umum

| Keputusan | Isian |
| --- | --- |
| Kumpulkan seluruh error field atau berhenti pada error pertama | `[kumpulkan error field untuk error yang merupakan validasi field request]` |
| Pesan error mengikuti locale existing | `[pesan error mengikuti header]` |
| Pesan required, minimum, maksimum, karakter tidak valid | Field errors berupa map field ke daftar pesan locale; contoh minimum %s karakter atau maksimal %s karakter. Tidak menaruh prefix field dua kali dalam pesan. |
| Pesan password kurang kompleks | `[tidak perlu]` |
| Pesan nomor telepon tidak valid | `[<field> : tidak valid]` |
| Unknown field ditolak/diabaikan per endpoint baru | `[abaikan]` |
| Definisi panjang untuk kebijakan baru: byte/karakter Unicode | Input teks baru ASCII-only sesuai allowlist per field; panjang byte setelah normalisasi yang disepakati. Non-ASCII ditolak tanpa dihapus/dikonversi. Login/current_password mengikuti credential existing; token opaque diteruskan persis. Response/data lama backend tidak divalidasi ulang sebagai input baru. |
| Penyampaian access/refresh token ke browser: body/cookie | Access token lewat body, disimpan dalam memory FE dan dikirim sebagai Bearer. Refresh token hanya cookie HttpOnly; rest-orc memetakannya ke field protobuf. Tidak memakai localStorage. Detail lifecycle di bawah. |
| Response register `PENDING` dan compatibility response existing | `[Pertahankan route, HTTP status sukses, envelope, dan field token existing dengan nilai "". Tambahkan registration_id dan registration_status: "PENDING". Register tidak membuat sesi atau menerbitkan access/refresh token. FE menampilkan pesan registrasi berhasil dan akun menunggu aktivasi. Bentuk response dipertahankan, tetapi FE yang sebelumnya otomatis login setelah register tetap perlu disesuaikan.]` |
| Mapping response login lengkap: token, refresh, session, expiry, restricted | Body data: token, token_type Bearer, session_id, access_expires_at, session_expires_at, must_change_password. Waktu RFC3339 UTC dari backend; refresh_token hanya cookie. Restricted ditandai must_change_password tanpa flag duplikat. Setelah change password/logout/revoke seluruh sesi sendiri berhasil, hapus credential browser dan minta login ulang. Cache-Control: no-store. |
| Label status/role/aplikasi untuk FE dan mapping ke code backend | PENDING=Menunggu aktivasi; ACTIVE=Aktif; SUSPENDED=Ditangguhkan; DEACTIVATED=Dinonaktifkan; DISABLED=Nonaktif; REVOKED=Dicabut. Label pilihan dibentuk rest-orc pada endpoint terpisah (bagian 16). FE mengirim value string/code, bukan label. Grant/revoke menerima string desimal role ID yang diparse ke uint64; filter role List User existing tetap CSV role code numerik. |

Allowlist teks bebas baru: description dan reason menerima ASCII printable 0x20–0x7E serta CR/LF; NUL, tab, DEL, dan kontrol lain ditolak. Keyword menerima ASCII printable dan whitespace yang dinormalisasi backend. Code/status/ID memakai pola typed masing-masing, bukan allowlist teks bebas. Validasi ASCII dilakukan sebelum normalisasi agar non-ASCII tidak diam-diam dihapus. Required/panjang description dan reason tetap mengikuti tabel feature.

Error menggunakan catalog existing dan shared response envelope. Dukungan mapping `FieldErrors` lokal ke response per field perlu ditambahkan; mapper existing sudah mendukung field errors yang berasal dari gRPC.

### Cookie dan Lifecycle Browser

- Access token tetap `data.token`, hanya disimpan dalam memory FE. Refresh token tidak masuk JSON browser atau localStorage. Transport cookie ini tidak mengubah contract RPC grpc-auth.
- Cookie refresh bernama `__Secure-refresh`, dengan `HttpOnly; Secure; SameSite=Lax; Path=/api/v1/auth`, tanpa Domain. Path mengikuti prefix API aktual dari konfigurasi startup. Masa berlaku tidak boleh melampaui session_expires_at backend. Seluruh konfigurasi runtime berasal dari environment dan didaftarkan di .env.example saat implementasi; token tidak disimpan di server rest-orc.
- Login/refresh memasang cookie dari refresh_token backend. Logout sukses, change password sukses, atau revoke seluruh sesi sendiri sukses menghapus cookie dengan nama/path yang sama serta menghapus access token memory. FE memakai flag is_current dari ListMySessions untuk mengakhiri tampilan sesi jika sesi tersebut dicabut; jangan mengarang login_required dari response RevokeMySession yang hanya memuat changed. Dependency failure tidak dianggap logout/refresh sukses.
- Pada mode cookie, validasi Origin terhadap allowlist dan gunakan signed double-submit CSRF token melalui cookie CSRF terpisah serta header X-CSRF-Token. Cookie CSRF boleh dibaca FE dan bukan refresh credential; token terikat pada digest refresh credential dari cookie dan ditandatangani server, tanpa menyimpan JWT/session secara lokal. Digest tidak dikirim sebagai credential pengganti dan raw refresh tidak dimasukkan dalam CSRF token. CSRF diperbarui bersama rotasi refresh. Sediakan GET /auth/csrf untuk memperoleh ulang token CSRF dari cookie refresh, dengan validasi Origin dan Cache-Control: no-store; endpoint ini tidak memerlukan bearer atau decode refresh token, sedangkan validitas sesi tetap diperiksa grpc-auth saat refresh/logout. Jangan mengembalikan refresh token melalui endpoint ini.
- Login juga memvalidasi Origin FE dan Content-Type JSON untuk mencegah login lintas origin yang tidak diizinkan. Penandatanganan CSRF menggunakan secret environment terpisah, tanpa default, sama antar-instance; tidak mengirim signing secret ke FE. Cookie CSRF memiliki Secure/SameSite/path yang konsisten dan dibersihkan bersama refresh cookie. Acuan pola: [OWASP signed double-submit](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html#signed-double-submit-cookie-recommended).
- Mode bearer logout tidak meneruskan cookie refresh ke RPC; mode cookie tidak meneruskan authorization metadata. Dua credential tidak pernah dikirim bersama ke grpc-auth. Header bearer malformed ditolak, bukan diabaikan.
- FE menggunakan credentials: include untuk login/refresh/logout dan pengambilan CSRF. CORS jika diperlukan mengizinkan origin FE yang eksplisit, credentials, Authorization, Content-Type, Version, Idempotency-Key, X-CSRF-Token, serta header locale/request ID existing; bukan wildcard origin. Deployment target menggunakan HTTPS dan FE/API dalam satu site. Cookie tidak otomatis diubah menjadi SameSite=None untuk deployment lintas site.
- Refresh tidak memakai retry otomatis. Request bisnis yang gagal setelah refresh tidak diulang otomatis jika merupakan mutasi; ikuti idempotency contract operasi tersebut.

### Status HTTP untuk Error Lokal dan Downstream

Keputusan: status HTTP ditentukan berdasarkan sumber dan jenis error. Validasi request oleh rest-orc sebelum call RPC bisnis menghasilkan **400 Bad Request**. Bila error berasal dari service dan detail error terstruktur menyertakan HTTP code yang valid, **status HTTP response mengikuti nilai tersebut**, termasuk ketika error memiliki field errors. Keberadaan field errors saja tidak menentukan status HTTP. Error yang tidak memiliki mapping yang dikenali menggunakan default **500 Internal Server Error**.

1. Decode error melalui grpcerror dari api-contracts. Wire grpc-auth saat ini menggunakan ErrorInfo.metadata.http_status, dengan metadata.code dan Reason; tidak ada JSON body HTTP dari RPC. Inilah sumber yang dimaksud sebagai http_code downstream. Jangan menambah field protobuf hanya untuk mengganti nama ini.
2. Nilai http_status yang valid pada rentang 400–599 menjadi status HTTP aktual, termasuk saat field errors tersedia. Contoh: email sudah digunakan dengan http_status=409 dari service tetap menghasilkan HTTP 409 Conflict meskipun memiliki field error email; jika service mengirim http_status=400, gunakan HTTP 400. Jangan memaksa seluruh field errors downstream menjadi 400 atau menyimpulkan status hanya dari nama field. Pertahankan code, field errors, dan pesan aman dari downstream sesuai locale yang diteruskan. Definisi registry bukan alasan menimpa status eksplisit downstream yang valid. Envelope code/errors/data existing tetap dipakai; jangan membocorkan metadata internal atau detail dependency.
3. Jika detail HTTP status tidak tersedia/invalid, pakai definisi catalog dari Reason. Jika catalog tidak dikenal, gunakan fallback gRPC: InvalidArgument→400, Unauthenticated→401, PermissionDenied→403, NotFound→404, AlreadyExists/Aborted→409, ResourceExhausted→429, FailedPrecondition→400, Unavailable→503, DeadlineExceeded→504, Canceled→499 bila koneksi masih bisa ditulis, selainnya→500. Ini fallback, bukan pengganti status eksplisit backend.
4. RetryInfo yang tersedia dipetakan ke Retry-After secara aman; tidak memicu retry otomatis. Error transport tanpa structured detail memakai fallback. Pesan unknown/internal disamarkan melalui catalog generik.
5. Validasi request lokal oleh rest-orc, seperti required, panjang minimum/maksimum, format, dan karakter yang tidak diizinkan, memakai HTTP 400 dan field errors lokal dengan format yang sama. Jika validasi gagal, hentikan alur sebelum memanggil RPC bisnis; panggilan authentication middleware pada route protected tetap diperhitungkan. Error lokal lainnya memakai mapping sesuai jenis error dari catalog/shared mapper; jika tidak dikenali, gunakan HTTP 500. Perubahan pemilihan status ini perlu diterapkan di internal/response; mapper existing belum memprioritaskan metadata.http_status.

Acceptance:

- Uji required, panjang, format, dan karakter tidak valid yang diperiksa rest-orc: HTTP 400, field errors sesuai locale, dan RPC bisnis tidak dipanggil.
- Uji email sudah digunakan dari service dengan http_status=409 dan field error email: HTTP 409. Uji field error dari service dengan http_status=400: HTTP 400.
- Uji metadata 409/429/503 dengan dan tanpa field errors. Pastikan status HTTP aktual sesuai detail downstream, termasuk ketika status gRPC lebih generik atau status catalog berbeda.
- Uji status malformed/di luar rentang atau tidak tersedia, Reason tidak terdaftar, pesan aman, locale, timeout/cancellation, serta RetryInfo sesuai urutan fallback. Uji error lokal dan downstream tanpa mapping yang dikenali menghasilkan HTTP 500.

## 14. Urutan Implementasi dan Verifikasi

| Tahap | Pekerjaan | Hasil |
| --- | --- | --- |
| 1 | Lengkapi keputusan pada tabel | Matriks aturan yang disepakati |
| 2 | Selaraskan versi api-contracts, TLS, service credential, dan metadata | Client kompatibel dengan backend; tidak bergantung pada workspace lokal saja |
| 3 | Register, login, profil, change password, dan Me | Validasi/normalisasi input serta mapping response; profil dan akses Me berasal dari satu RPC backend lokal; publikasi/pin contract diverifikasi terpisah |
| 4 | Refresh, logout, sesi sendiri/admin | Lifecycle sesi dan credential mode |
| 5 | Administrasi user, role, assignment, application access, permission, audit | Adapter REST sesuai contract backend |
| 6 | Test per fitur dan integration | Validasi argumen downstream, field mask, context, error, version, dan idempotency |

- Seluruh test buatan project berada di root `test/`.
- Uji success, invalid/edge case, normalisasi, dan dependency error.
- Pastikan validasi gagal tidak memanggil RPC bisnis; perhitungkan panggilan authentication middleware pada route protected.
- Jalankan targeted test, `go test ./...`, dan `go vet ./...`; race detector jika menyentuh concurrency/shared state.
- Pertahankan contract route/response existing kecuali perubahan telah disepakati.
- Permission/aplikasi berupa catalog read; CRUD definisinya dan `TesRPC` tidak masuk fitur FE pada plan ini.
- Endpoint options merupakan adapter REST ke RPC list existing; filter backend tertunda dikerjakan melalui plan terpisah dan tidak menjadi dependency tahap REST awal.

## 15. Ringkasan Saran Bentuk REST

`{base_url}` mencakup prefix API, misalnya `http://localhost:8080/api/v1`. Path pada tabel ditambahkan setelah base URL tersebut. Register, login, dan list users mempertahankan route existing; endpoint lain merupakan usulan untuk implementasi berikutnya.

### Konvensi Request

- Gunakan nama resource plural: `/users`, `/roles`, `/applications`, `/permissions`. `/me` berarti pengguna dari token, sedangkan `/users/{user_id}` berarti user target yang dipilih admin.
- ID target berada pada path; pagination, pencarian, dan filter berada pada query; data perubahan berada pada body JSON dengan `Content-Type: application/json`. Jangan mengulang ID path pada body.
- Gunakan GET untuk membaca, PATCH untuk perubahan sebagian field, dan PUT untuk mengganti seluruh daftar permission role. Request GET dan DELETE pada usulan ini tanpa body. Acuan: [HTTP semantics](https://www.rfc-editor.org/rfc/rfc9110.html#section-9.3), [PATCH](https://www.rfc-editor.org/rfc/rfc5789.html#section-2).
- Grant/revoke role dan aplikasi memakai POST dengan path aksi berbeda agar operasi bulk, alasan audit, versioning, dan satu pemanggilan RPC tetap jelas. Aksi `revoke-all` juga memakai POST.
- Operasi protected memakai `Authorization: Bearer <access_token>`. Register/login tidak memerlukan bearer; refresh memakai refresh token. Logout memakai tepat satu mode credential sesuai bagian 4.
- Kolom header tambahan di bawah tidak mengulang `Authorization` atau `Content-Type`. `Version` dipetakan ke field `expected_user_version` atau `expected_role_version` sesuai target operasi. Nilainya berasal dari response target terakhir, bukan penghitung buatan FE.
- `Version` adalah header usulan sesuai pilihan user; nama header HTTP tidak membedakan kapitalisasi. `version` dan `Version` dibaca sebagai header yang sama. Jangan otomatis memakai versi terbaru untuk mengulangi mutasi setelah conflict.
- Parameter query yang tidak digunakan boleh dihilangkan. Encode nilai query melalui URL encoder, terutama cursor, keyword, serta timezone bertanda `+`. Tanggal users/audit memakai YYYY-MM-DD, Asia/Jakarta; xxx_to inklusif dikonversi ke awal hari berikutnya yang eksklusif.
- Aturan validasi, presence, enum, response envelope, status HTTP, dan idempotency pada bagian sebelumnya tetap berlaku. Contoh request hanya menunjukkan bentuk transport; keputusan final pada tabel dan tambahan bagian 16–17 menjadi acuan.

### Authentication dan Profil Sendiri

| Fitur/RPC | Method | Path | Body JSON | Header tambahan |
| --- | --- | --- | --- | --- |
| Register | POST | `/users/create` (existing) | `email`, `password` | — |
| Login | POST | `/users/login` (existing) | `email`, `password` | — |
| `RefreshSession` | POST | `/auth/refresh` | Tanpa body credential; refresh dari cookie | X-CSRF-Token |
| `Logout` | POST | `/auth/logout` | Tanpa body credential | Bearer opsional; X-CSRF-Token untuk mode cookie |
| Bootstrap CSRF REST | GET | `/auth/csrf` | — | Origin FE diizinkan; cookie refresh; tidak perlu bearer |
| `Me` | GET | `/me` | — | — |
| `UpdateProfile` | PATCH | `/me/profile` | Field terpilih: `name`, `phone`, `country_code`, `address` | — |
| `ChangePassword` | POST | `/me/password` | `current_password`, `new_password` | — |

Browser mengirim cookie melalui credentials: include. FE tidak membaca refresh token. REST mengisi field RPC dari cookie dan menggantinya setelah refresh berhasil. FE mengoordinasikan satu refresh yang sedang berjalan termasuk antar-tab; rotasi tidak di-retry otomatis.

```http
POST /api/v1/auth/refresh
Cookie: __Secure-refresh=<opaque_refresh_token>
X-CSRF-Token: <csrf_token>

```

### User dan Catalog

| Fitur/RPC | Method | Path / contoh query | Body JSON | Header tambahan |
| --- | --- | --- | --- | --- |
| List users | GET | `/users?page=1&limit=20` (existing) | — | — |
| `GetUser` | GET | `/users/{user_id}` | — | — |
| `CreateUser` | POST | `/users` | `email`, `password`, `name`, `address`, `phone`, `country_code`; status ditetapkan PENDING | `Idempotency-Key` wajib |
| `UpdateUser` | PATCH | `/users/{user_id}/profile` | Field profil terpilih sesuai contract | `Version` wajib; `Idempotency-Key` opsional |
| `ChangeUserStatus` | PATCH | `/users/{user_id}/status` | `status`, `reason` | `Version`, `Idempotency-Key` wajib |
| `ListRoles` | GET | `/roles?application_code=clm&statuses=ACTIVE` | — | — |
| `GetRole` | GET | `/roles/{role_id}` | — | — |
| `CreateRole` | POST | `/roles` | `application_code`, `code`, `name`, `description`, `permission_codes` | `Idempotency-Key` wajib |
| `UpdateRole` | PATCH | `/roles/{role_id}` | `name` dan/atau `description` | `Version` wajib; `Idempotency-Key` opsional |
| `ChangeRoleStatus` | PATCH | `/roles/{role_id}/status` | `status`, `reason` | `Version`, `Idempotency-Key` wajib |
| `ReplaceRolePermissions` | PUT | `/roles/{role_id}/permissions` | `permissions: {codes: [...]}`, `reason` | `Version`, `Idempotency-Key` wajib |
| `ListPermissions` | GET | `/permissions?application_code=clm` | — | — |
| `ListApplications` | GET | `/applications?statuses=ACTIVE` | — | — |

Contoh perubahan status: `123` adalah user target dan `7` berasal dari `version` pada response detail user tersebut. Status dan alasan tetap divalidasi sesuai aturan backend.

```http
PATCH /api/v1/users/123/status
Authorization: Bearer <access_token_admin>
Content-Type: application/json
Version: 7
Idempotency-Key: <key_tindakan>

{"status":"SUSPENDED","reason":"Peninjauan akun"}
```

### Assignment dan Permission

| Fitur/RPC | Method | Path / contoh query | Body JSON | Header tambahan |
| --- | --- | --- | --- | --- |
| `ListUserRoles` | GET | `/users/{user_id}/roles?application_code=clm&statuses=ACTIVE` | — | — |
| `GrantUserRoles` | POST | `/users/{user_id}/roles/grant` | `role_ids`, `reason` | `Version`, `Idempotency-Key` wajib |
| `RevokeUserRoles` | POST | `/users/{user_id}/roles/revoke` | `role_ids`, `reason` | `Version`, `Idempotency-Key` wajib |
| `ListUserApplicationAccess` | GET | `/users/{user_id}/applications?statuses=ACTIVE` | — | — |
| `GrantApplicationAccess` | POST | `/users/{user_id}/applications/{application_code}/grant` | `role_ids`, `reason` | `Version`, `Idempotency-Key` wajib |
| `RevokeApplicationAccess` | POST | `/users/{user_id}/applications/{application_code}/revoke` | `reason` | `Version`, `Idempotency-Key` wajib |
| `GetUserEffectivePermissions` | GET | `/users/{user_id}/permissions?application_code=clm` | — | — |
| `GetMyEffectivePermissions` | — | Dibaca melalui `/me`; tidak dibuat endpoint FE terpisah | — | — |
| `EvaluateAccess` | — | Internal opsional; tidak dibuat endpoint FE | — | — |

`application_code` pada pembacaan permission merupakan filter query pada `/users/{user_id}/permissions`, bukan endpoint baru `/user-application/{user_id}`. Daftar aplikasi yang boleh diakses user dibaca melalui `/users/{user_id}/applications`.

Contoh pemberian beberapa role dalam satu tindakan:

```http
POST /api/v1/users/123/roles/grant
Authorization: Bearer <access_token_admin>
Content-Type: application/json
Version: 7
Idempotency-Key: <key_tindakan>

{"role_ids":["10","11"],"reason":"Penugasan reviewer"}
```

### Sesi dan Audit

| Fitur/RPC | Method | Path / contoh query | Body JSON | Header tambahan |
| --- | --- | --- | --- | --- |
| `ListMySessions` | GET | `/me/sessions?page=1&limit=20` | — | — |
| `RevokeMySession` | DELETE | `/me/sessions/{session_id}` | — | — |
| `RevokeAllMySessions` | POST | `/me/sessions/revoke-all` | — | — |
| `ListUserSessions` | GET | `/users/{user_id}/sessions?page=1&limit=20` | — | — |
| `RevokeUserSession` | DELETE | `/users/{user_id}/sessions/{session_id}` | — | `Idempotency-Key` wajib |
| `RevokeAllUserSessions` | POST | `/users/{user_id}/sessions/revoke-all` | — | `Version`, `Idempotency-Key` wajib |
| `ListAuditLogs` | GET | `/audit-logs?page=1&limit=20` | — | — |

Contoh audit: `/audit-logs?actor_user_id=123&occurred_from=2026-10-01&occurred_to=2026-10-23`. Rentang mencakup 1–23 Oktober 2026 di Asia/Jakarta; REST mengirim Timestamp backend occurred_from=2026-09-30T17:00:00Z dan occurred_before=2026-10-23T17:00:00Z.

## 16. Endpoint Pilihan FE

GetRoles berbeda dari ListRoles: ListRoles adalah daftar administrasi; GetRoles adalah adapter pilihan FE. Label dibentuk di handler/response mapper rest-orc karena merupakan kebutuhan tampilan REST. grpc-auth tetap mengembalikan data catalog terstruktur dan memeriksa permission; tidak perlu RPC baru hanya untuk format label.

| Nama REST | Method/path | Sumber | Value string | Label |
| --- | --- | --- | --- | --- |
| GetRoles | GET /roles/options | ListRoles | role ID dalam string desimal | code - description; jika description kosong/null gunakan code - name |
| GetPermissions | GET /permissions/options | ListPermissions | permission.code | code - description; description kosong cukup code |
| GetApplications | GET /applications/options | ListApplications | application.code | code - name (application tidak memiliki description) |
| GetStatuses | GET /statuses/options?resource=users | Allowlist code backend per resource | status code | Label Indonesia pada bagian 13 |

- Route /roles/options didaftarkan sebelum /roles/:role_id, demikian pula options catalog lain sebelum route parameter. GetRoles bukan GetRole yang membaca satu detail role.
- Endpoint options memakai page, limit, keyword dan filter yang didukung RPC sumber; metadata pagination/cursor dipertahankan. Jangan mengambil seluruh catalog dengan loop tersembunyi. Authorization dan namespace sama dengan RPC sumber; options tidak melewati pemeriksaan permission.
- FE menampilkan label dan mengirim value string. Role ID pada body endpoint grant/revoke baru berupa string desimal positif, diparse ke uint64 protobuf setelah validasi rentang 1–2147483647; bukan label atau string code. permission_codes dan application_code sudah string sesuai backend. Path ID juga diparse dari string desimal. Tidak menerima arbitrary object/label sebagai ID.
- GetRoles juga menyertakan code dan role_code sebagai string agar FE dapat membentuk filter role List User existing dari numeric role_code, tanpa mengubah semantik query role=1,2. ID role, numeric role_code, dan code string tidak saling dipertukarkan.
- GetStatuses memakai resource users, roles, applications, assignments, application_access, atau sessions; resource required dan nilai unknown ditolak. Users: PENDING/ACTIVE/SUSPENDED/DEACTIVATED; roles/applications: ACTIVE/DISABLED; assignments/application_access/sessions: ACTIVE/REVOKED. Endpoint protected; hanya memasok catalog code, bukan daftar transisi yang pasti diizinkan untuk suatu target. Expired session bukan code status baru.
- Label hanya untuk tampilan, tidak menjadi sumber identitas atau permission. Kode role baru pada CreateRole tetap string yang diisi FE; catalog existing tidak dapat menyediakan kode role yang belum dibuat. Input yang berformat label gabungan bukan code yang valid.

Contoh data GetRoles (dibungkus envelope sukses existing):

```json
{
  "items": [
    {
      "value": "10",
      "label": "clm.legal_reviewer - Reviewer legal kontrak",
      "code": "clm.legal_reviewer",
      "role_code": "3"
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
```

## 17. Permission Route dan Response

### Permission Endpoint

Semua RPC tetap memerlukan TLS dan service identity/namespace/method allowlist. Permission berikut berasal dari kemampuan grpc-auth; rest-orc memakai hasil Me pada request saat ini dan grpc-auth tetap memeriksa otorisasi RPC. Group admin existing untuk ListUsers dipertahankan sampai migrasi eksplisit ke permission gate tersedia; jangan memakai role name sebagai pengganti permission baru.

| Route/operasi | Credential atau permission |
| --- | --- |
| Register/login | Tanpa bearer user; service credential tetap wajib |
| Me | Bearer diri sendiri; restricted memperoleh profil minimum |
| UpdateProfile, sesi sendiri | Bearer diri sendiri; ownership dan tidak restricted |
| ChangePassword | Bearer dan current_password; restricted boleh |
| Refresh | Refresh dari cookie; tidak restricted |
| Logout | Tepat satu mode bearer/refresh; restricted boleh |
| ListUsers/GetUser/ListUserRoles/ListUserApplicationAccess/GetUserEffectivePermissions | iam.user.read |
| CreateUser / UpdateUser / ChangeUserStatus | iam.user.create / iam.user.update / iam.user.status.update sesuai operasi |
| GrantUserRoles / RevokeUserRoles | iam.user.role.grant / iam.user.role.revoke |
| ListRoles/GetRole/GetRoles options | iam.role.read |
| CreateRole | iam.role.create |
| UpdateRole/ChangeRoleStatus | iam.role.update |
| ReplaceRolePermissions | iam.role.permission.update |
| ListPermissions/ListApplications dan options masing-masing | iam.permission.read |
| GrantApplicationAccess | iam.user.application.grant AND iam.user.role.grant |
| RevokeApplicationAccess | iam.user.application.revoke AND iam.user.role.revoke |
| ListUserSessions | iam.session.read |
| RevokeUserSession/RevokeAllUserSessions | iam.session.revoke |
| ListAuditLogs | iam.audit.read |
| GetStatuses | Bearer aktif nonrestricted; hanya catalog code statis |
| GET /auth/csrf | Origin FE diizinkan dan cookie refresh; tidak mengembalikan profil/credential |

### Mapping Response

Envelope sukses/error existing code/message/data atau code/errors/data dipertahankan. Field baru tidak diambil langsung dari generated protobuf ke HTTP. Version baru diserialisasikan sebagai string desimal agar tidak kehilangan presisi int64 di browser; FE meneruskannya persis ke header Version. Tipe field ID/response existing tidak diubah; value pada options dan body pilihan baru berupa string.

| Operasi | Mapping data |
| --- | --- |
| Register | token tetap "", registration_id dari backend, registration_status=PENDING; tanpa cookie auth/session; aktivasi oleh admin |
| Login | token, token_type=Bearer, session_id, access_expires_at, session_expires_at, must_change_password; refresh token hanya Set-Cookie |
| Refresh | Mapping credential sama dengan login; ganti cookie dengan token hasil rotasi; tidak mengarang expiry atau memperpanjang sesi sendiri |
| Me | Profil/status/version/must_change_password/roles/application_access/permissions dari satu RPC; tidak concat ulang phone |
| List | items dan seluruh metadata pagination dipetakan; collection kosong items=[], total=0 dan total_pages=0 jika tidak ada hasil; jangan mengubah total menjadi nol hanya karena page/cursor tidak berisi item |
| Mutation | Field safe sesuai RPC: changed, version, user_id/role/application dan changed_role_ids bila tersedia; tidak menciptakan field yang tidak dikembalikan RPC |
| PATCH tanpa field efektif | data={"changed":false}, tanpa RPC mutasi/version buatan |
| Error service | Envelope error existing; status HTTP mengikuti detail http_status (bagian 13), code/pesan/field errors aman dipertahankan |

Timestamp dikonversi ke RFC3339 UTC; collection/map kosong dipetakan konsisten menjadi []/{} pada DTO baru. Restricted Me tetap tanpa privilege; FE tidak menganggap map kosong sebagai fallback saat dependency gagal. Response profil/detail/list lama tetap backward compatible.

Contoh login browser:

```json
{
  "code": "SS",
  "message": "Login Successfully",
  "data": {
    "token": "<access_token>",
    "token_type": "Bearer",
    "session_id": "9c9a94a2-7b1f-49d0-8536-4f061cfd85b7",
    "access_expires_at": "2026-10-23T04:00:00Z",
    "session_expires_at": "2026-10-24T03:00:00Z",
    "must_change_password": false
  }
}
```

HTTP response tersebut juga mengandung Set-Cookie refresh dan Cache-Control: no-store; nilai waktu/ID hanya ilustrasi, bukan TTL default. Contoh error: jika backend menyertakan http_status=409, HTTP response berstatus 409 dengan envelope code/errors/data dari error yang sama, bukan HTTP 200 yang hanya memuat angka 409 dalam JSON.

### Acceptance Tambahan

- Uji bootstrap CSRF, cookie attributes/path/expiry, rotasi, logout bearer bersama cookie otomatis, logout cookie tanpa bearer, bearer invalid tanpa fallback, refresh tanpa access token, serta dependency failure.
- Uji PATCH omitted/null/unknown/no-op; authentication/permission tetap berjalan walau tidak ada RPC mutasi.
- Uji ASCII allowlist, credential lama, panjang phone setelah normalisasi, pilihan label/value serta parsing string ID, numeric role_code compatibility, dan endpoint options yang tidak tertangkap route detail.
- Uji boundary tanggal audit (awal/akhir hari Asia/Jakarta), from=to/from>to, cursor saat filter/sort berubah, tie-breaker ID, metadata empty page, dan HTTP status downstream.

## Acuan Source

- [Handler users rest-orc](../internal/features/users/handler/http/handler.go)
- [Use case register](../internal/features/users/usecase/register.go)
- [Use case login](../internal/features/users/usecase/login.go)
- [Gateway users](../internal/features/users/gateway/authservice/gateway.go)
- [Shared auth client](../internal/client/authservice/client.go)
- [HTTP error mapper](../internal/response/error_mapper.go)
- [Spesifikasi user management](user-management-spec.md)
- [Validator password backend](../../grpc-auth/internal/repositories/domain/auth.go)
- [Validator current/new password](../../grpc-auth/internal/repositories/domain/password.go)
- [Validator country code](../../grpc-auth/internal/repositories/domain/country_code.go)
- [Verifikasi mode logout backend](../../grpc-auth/internal/sessions/logout.go)
- [Me snapshot backend](../../grpc-auth/docs/me-access-snapshot.md)
- [Encoder HTTP status dalam gRPC error](../../api-contracts/pkg/grpcerror/encode.go)
- [Plan filter tambahan backend](../../grpc-auth/docs/list-filter-extension-plan.md)

Inventaris/backend diverifikasi pada 23 Oktober 2026 terhadap source lokal grpc-auth dan api-contracts. Workspace bukan bukti versi published siap dipakai: pin rest-orc masih contract Agustus. Publikasi dan pembaruan dependency tidak dilakukan pada perubahan dokumen ini; setelah versi tersedia verifikasi dengan GOWORK=off.
