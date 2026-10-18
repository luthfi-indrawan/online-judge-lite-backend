# 📄 API Contract & Specification — Online Judge Lite

## 1. Standar Umum API & Konvensi

- **Base URL:** `http://localhost:8080/api/v1`
- **Content-Type:** `application/json` untuk seluruh request
- **Autentikasi:** HTTP Bearer Token (`Authorization: Bearer <jwt_token>`)
- **Format Respons Berhasil:**

```json
{
  "success": true,
  "message": "Deskripsi pesan sukses",
  "data": {}, // Object atau Array
  "error": null
}
```

- **Format Respons Gagal (Error):**

```json
{
  "success": false,
  "message": "Deskripsi umum kesalahan",
  "data": null,
  "error": {
    "code": "ERROR_CODE",
    "details": "Detail teknis kesalahan / pesan validasi"
  }
}
```

---

## 2. Pemetaan Status Code & Kode Error

| Status HTTP          | Error Code                            | Deskripsi & Skenario                                                                                |
| -------------------- | ------------------------------------- | --------------------------------------------------------------------------------------------------- |
| `200 OK`             | -                                     | Request berhasil diproses.                                                                          |
| `201 Created`        | -                                     | Data/Resource baru berhasil ditambahkan ke database.                                                |
| `202 Accepted`       | -                                     | Request diterima dan masuk ke dalam antrean (_submission queue_).                                   |
| `400 Bad Request`    | `VALIDATION_ERROR`                    | Format JSON salah, field wajib kosong, atau tipe data mismatch.                                     |
| `401 Unauthorized`   | `UNAUTHORIZED` / `TOKEN_EXPIRED`      | JWT missing, tanda tangan token tidak valid, atau token kadaluarsa.                                 |
| `403 Forbidden`      | `FORBIDDEN_ACCESS` / `CONTEST_CLOSED` | Role tidak sesuai (bukan Admin) atau user mencoba submit di luar rentang `start_time` - `end_time`. |
| `404 Not Found`      | `RESOURCE_NOT_FOUND`                  | Data (User, Group, Problem, TestCase) tidak ditemukan.                                              |
| `409 Conflict`       | `DUPLICATE_ENTRY`                     | Username sudah digunakan oleh akun lain.                                                            |
| `500 Internal Error` | `INTERNAL_SERVER_ERROR`               | Terjadi kegagalan internal di server / basis data.                                                  |

---

## 3. Detail API Endpoints & Logic Flow

### 3.1. Autentikasi (`/auth`)

---

#### 3.1.1. Login User & Admin

Menerima kredensial pengguna, memvalidasi password dengan hash bcrypt, dan mengembalikan JWT Token.

- **Endpoint:** `POST /auth/login`
- **Akses:** Public (Tanpa Token)
- **Request Body:**

```json
{
  "username": "mahasiswa123",
  "password": "passwordSecret"
}
```

- **Alur & Logika Teknis (Flow & Logic):**

1. **Validasi Input:** BE mengecek apakah `username` dan `password` diisi.
2. **Query Database:** BE mencari record di tabel `users` berdasarkan `username`. Jika tidak ditemukan, kembalikan `401 Unauthorized`.
3. **Verifikasi Hash Password:** BE membandingkan `password` plain-text dari request dengan `password_hash` menggunakan pustaka `bcrypt`. Jika tidak cocok, kembalikan `401 Unauthorized`.
4. **Generate JWT:** BE membuat JWT token berisi klaim `user_id`, `username`, `role`, dan waktu kadaluarsa (misal: 24 jam).
5. **Respons:** Mengembalikan token JWT beserta ringkasan profil user.

- **Response (`200 OK`):**

```json
{
  "success": true,
  "message": "Login berhasil",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 2,
      "username": "mahasiswa123",
      "name": "Budi Santoso",
      "role": "USER"
    }
  },
  "error": null
}
```

---

#### 3.1.2. Get Current Authenticated User Profile

Mengambil data profil pengguna yang sedang aktif/login berdasarkan JWT token.

- **Endpoint:** `GET /auth/me`
- **Akses:** Authenticated (`ADMIN` & `USER`)

- **Headers:** `Authorization: Bearer <token>`
- **Alur & Logika Teknis (Flow & Logic):**

1. Middleware `AuthMiddleware` memvalidasi JWT token dari header request.

2. Ekstrak `user_id` dari payload token.
3. Query tabel `users` berdasarkan `id` (abaikan kolom `password_hash`).
4. Kirim respons data profil.

- **Response (`200 OK`):**

```json
{
  "success": true,
  "message": "Profil berhasil didapatkan",
  "data": {
    "id": 2,
    "username": "mahasiswa123",
    "name": "Budi Santoso",
    "role": "USER",
    "created_at": "2026-10-18T00:00:00Z"
  },
  "error": null
}
```

---

### 3.2. Manajemen User (`/admin/users`) — Admin Only

---

#### 3.2.1. Create Single User

Membuat akun mahasiswa/peserta baru secara individual.

- **Endpoint:** `POST /admin/users`
- **Akses:** `ADMIN`

- **Request Body:**

```json
{
  "username": "mhs_001",
  "name": "Ahmad Yani",
  "password": "Password123!",
  "role": "USER"
}
```

- **Alur & Logika Teknis (Flow & Logic):**

1. **Akses Kontrol:** Middleware memverifikasi bahwa `role` pengguna pada JWT adalah `ADMIN`.

2. **Validasi Unik:** Cek apakah `username` sudah ada di tabel `users`. Jika ada, kembalikan `409 Conflict`.
3. **Hashing Password:** Hash `password` menggunakan `bcrypt` dengan cost factor 10.
4. **Insert Data:** Simpan `username`, `name`, `password_hash`, dan `role` ke tabel `users`.

5. Kembalikan data user yang dibuat tanpa `password_hash`.

- **Response (`201 Created`):**

```json
{
  "success": true,
  "message": "Pengguna berhasil dibuat",
  "data": {
    "id": 3,
    "username": "mhs_001",
    "name": "Ahmad Yani",
    "role": "USER",
    "created_at": "2026-10-18T00:15:00Z"
  },
  "error": null
}
```

---

#### 3.2.2. Bulk Create Users

Membuat banyak akun peserta sekaligus (hasil impor CSV/Excel yang telah diproses oleh Frontend).

- **Endpoint:** `POST /admin/users/bulk`
- **Akses:** `ADMIN`

- **Request Body:**

```json
{
  "users": [
    {
      "username": "mhs_002",
      "name": "Siti Aminah",
      "password": "Password123!",
      "role": "USER"
    },
    {
      "username": "mhs_003",
      "name": "Dewi Sartika",
      "password": "Password123!",
      "role": "USER"
    }
  ]
}
```

- **Alur & Logika Teknis (Flow & Logic):**

1. BE membuka transaksi database (`BEGIN TRANSACTION`).
2. Untuk setiap user di dalam array:

- Cek apakah `username` sudah terdaftar. Jika duplikat, buat rollback transaksi atau lewatkan tergantung strategi error (default: rollback & gagal seluruhnya).
- Hash password masing-masing user.

- Masukkan record ke dalam tabel `users`.

3. `COMMIT` transaksi dan kembalikan total akun yang berhasil dibuat.

- **Response (`201 Created`):**

```json
{
  "success": true,
  "message": "Bulk create pengguna berhasil",
  "data": {
    "created_count": 2,
    "users": [
      { "id": 4, "username": "mhs_002", "name": "Siti Aminah" },
      { "id": 5, "username": "mhs_003", "name": "Dewi Sartika" }
    ]
  },
  "error": null
}
```

---

#### 3.2.3. Reset User Password

Mengubah password akun peserta oleh Admin tanpa memerlukan password lama.

- **Endpoint:** `PUT /admin/users/:id/reset-password`
- **Akses:** `ADMIN`

- **Request Params:** `id` (User ID yang akan di-reset)
- **Request Body:**

```json
{
  "new_password": "NewSecurePassword123!"
}
```

- **Alur & Logika Teknis (Flow & Logic):**

1. Cek keberadaan user berdasarkan `:id` di tabel `users`.
2. Hash `new_password` dengan `bcrypt`.
3. Update kolom `password_hash` dan `updated_at` pada record user tersebut.

- **Response (`200 OK`):**

```json
{
  "success": true,
  "message": "Password pengguna berhasil diperbarui",
  "data": null,
  "error": null
}
```

---

### 3.3. Manajemen Problem Group (`/problem-groups`)

---

#### 3.3.1. Get All Problem Groups

Mendapatkan daftar seluruh grup soal beserta status pengerjaannya relatif terhadap waktu server.

- **Endpoint:** `GET /problem-groups`
- **Akses:** Authenticated (`ADMIN` & `USER`)

- **Alur & Logika Teknis (Flow & Logic):**

1. Query seluruh record dari tabel `problem_groups`.

2. Hitung dynamic field `status` dengan membandingkan `CURRENT_TIMESTAMP` terhadap `start_time` dan `end_time`:

- Jika `NOW() < start_time` $\rightarrow$ `"UPCOMING"`

- Jika `start_time <= NOW() <= end_time` $\rightarrow$ `"ONGOING"`

- Jika `NOW() > end_time` $\rightarrow$ `"ENDED"`

3. Sertakan pula jumlah total soal (`total_problems`) pada masing-masing grup.

- **Response (`200 OK`):**

```json
{
  "success": true,
  "message": "Daftar grup soal berhasil didapatkan",
  "data": [
    {
      "id": 1,
      "title": "Praktikum Algoritma Minggu 1",
      "description": "Topik: Array dan Pointer",
      "start_time": "2026-10-18T08:00:00Z",
      "end_time": "2026-10-18T12:00:00Z",
      "status": "ONGOING",
      "total_problems": 3,
      "created_by": 1
    }
  ],
  "error": null
}
```

---

#### 3.3.2. Get Detail Problem Group

Mendapatkan informasi detail suatu grup soal beserta daftar soal yang ada di dalamnya.

- **Endpoint:** `GET /problem-groups/:id`
- **Akses:** Authenticated (`ADMIN` & `USER`)

- **Request Params:** `id` (Group Problem ID)
- **Alur & Logika Teknis (Flow & Logic):**

1. Fetch detail dari `problem_groups` berdasarkan ID. Jika tidak ada, return `404 Not Found`.
2. Jika peran pengguna yang mengakses adalah `USER` dan grup berstatus `"UPCOMING"`, kembalikan `403 Forbidden` (_Belum Waktunya Akses_).
3. Ambil seluruh soal dari tabel `problems` yang terikat pada `group_id` ini.

4. Untuk masing-masing soal, hitung `user_status` spesifik untuk user yang sedang login berdasarkan riwayat submission mereka (`AC`, `FIRST_BLOOD`, `ATTEMPTED`, `NOT_ATTEMPTED`).

- **Response (`200 OK`):**

```json
{
  "success": true,
  "message": "Detail grup soal berhasil didapatkan",
  "data": {
    "id": 1,
    "title": "Praktikum Algoritma Minggu 1",
    "description": "Topik: Array dan Pointer",
    "start_time": "2026-10-18T08:00:00Z",
    "end_time": "2026-10-18T12:00:00Z",
    "status": "ONGOING",
    "problems": [
      {
        "id": 10,
        "title": "A. Penjumlahan Matriks",
        "time_limit_ms": 1000,
        "memory_limit_mb": 128,
        "user_status": "AC"
      }
    ]
  },
  "error": null
}
```

---

#### 3.3.3. Create Problem Group

Membuat grup soal/kuis/sesi lab baru.

- **Endpoint:** `POST /problem-groups`
- **Akses:** `ADMIN`

- **Request Body:**

```json
{
  "title": "Kuis Tengah Semester Algoritma",
  "description": "Kuis wajib sesi reguler",
  "start_time": "2026-10-20T09:00:00Z",
  "end_time": "2026-10-20T11:00:00Z"
}
```

- **Alur & Logika Teknis (Flow & Logic):**

1. **Validasi Waktu:** BE mengecek apakah `end_time` > `start_time` (diperkuat juga oleh database constraint `chk_group_time`).

2. Ekstrak `created_by` dari token JWT milik Admin.
3. Simpan record ke tabel `problem_groups`.

- **Response (`201 Created`):**

```json
{
  "success": true,
  "message": "Grup soal berhasil dibuat",
  "data": {
    "id": 2,
    "title": "Kuis Tengah Semester Algoritma",
    "description": "Kuis wajib sesi reguler",
    "start_time": "2026-10-20T09:00:00Z",
    "end_time": "2026-10-20T11:00:00Z",
    "created_by": 1,
    "created_at": "2026-10-18T00:20:00Z"
  },
  "error": null
}
```

---

### 3.4. Manajemen Problem & Test Case (`/problems`)

---

#### 3.4.1. Create Problem (Mengirimkan Supabase Storage PDF Link)

Membuat soal baru di mana Frontend sudah mengunggah file PDF secara langsung ke Supabase Storage dan mengirimkan URL file tersebut (`pdf_url`) ke Backend.

- **Endpoint:** `POST /problems`
- **Akses:** `ADMIN`

- **Content-Type:** `application/json`
- **Request Body:**

```json
{
  "group_id": 1,
  "title": "Soal A - Reverse String",
  "pdf_url": "https://xyz.supabase.co/storage/v1/object/public/problems/pdf_10.pdf",
  "time_limit_ms": 1000,
  "memory_limit_mb": 128
}
```

- **Alur & Logika Teknis (Flow & Logic):**

1. Validasi keberadaan `group_id` pada tabel `problem_groups`.

2. Validasi format URL `pdf_url` (memastikan string URL valid dari Supabase Storage).
3. Masukkan data soal baru ke tabel `problems`.

- **Response (`201 Created`):**

```json
{
  "success": true,
  "message": "Soal berhasil dibuat",
  "data": {
    "id": 10,
    "group_id": 1,
    "title": "Soal A - Reverse String",
    "pdf_url": "https://xyz.supabase.co/storage/v1/object/public/problems/pdf_10.pdf",
    "time_limit_ms": 1000,
    "memory_limit_mb": 128,
    "created_at": "2026-10-18T00:30:00Z"
  },
  "error": null
}
```

---

#### 3.4.2. Get Detail Problem

Mendapatkan detail deskripsi soal (termasuk `pdf_url` untuk di-render di PDF Viewer Frontend) dan pasangan _Sample Test Cases_.

- **Endpoint:** `GET /problems/:id`
- **Akses:** Authenticated (`ADMIN` & `USER`)

- **Request Params:** `id` (Problem ID)
- **Alur & Logika Teknis (Flow & Logic):**

1. Query tabel `problems` join dengan `problem_groups`.

2. **Validasi Rentang Waktu (User Only):** Jika peran pengguna adalah `USER`, BE mengecek apakah waktu sekarang berada pada rentang `start_time` <= `NOW()` <= `end_time`. Jika tidak, kembalikan `403 Forbidden` (`CONTEST_CLOSED`).

3. **Filter Test Cases:** Ambil test cases dari tabel `test_cases` khusus yang bertanda `is_sample = TRUE`. Hidden test cases **TIDAK Boleh** dibocorkan ke peserta.

4. Ambil informasi `first_blood_user` jika sudah ada peserta yang memenangkan First Blood.

- **Response (`200 OK`):**

```json
{
  "success": true,
  "message": "Detail soal berhasil didapatkan",
  "data": {
    "id": 10,
    "group_id": 1,
    "title": "Soal A - Reverse String",
    "pdf_url": "https://xyz.supabase.co/storage/v1/object/public/problems/pdf_10.pdf",
    "time_limit_ms": 1000,
    "memory_limit_mb": 128,
    "first_blood_user": {
      "id": 2,
      "username": "mahasiswa123",
      "name": "Budi Santoso"
    },
    "sample_test_cases": [
      {
        "id": 1,
        "input_data": "hello\n",
        "expected_output": "olleh\n"
      }
    ]
  },
  "error": null
}
```

---

#### 3.4.3. Add Test Case to Problem

Menambahkan masukan _input_ dan _expected output_ ke dalam soal untuk evaluasi kompilator.

- **Endpoint:** `POST /problems/:id/test-cases`
- **Akses:** `ADMIN`

- **Request Params:** `id` (Problem ID)
- **Request Body:**

```json
{
  "input_data": "5\n1 2 3 4 5\n",
  "expected_output": "15\n",
  "is_sample": true
}
```

- **Alur & Logika Teknis (Flow & Logic):**

1. Verifikasi keberadaan `problem_id` pada tabel `problems`.

2. Simpan `input_data`, `expected_output`, dan penanda `is_sample` ke dalam tabel `test_cases`.

- **Response (`201 Created`):**

```json
{
  "success": true,
  "message": "Test case berhasil ditambahkan",
  "data": {
    "id": 101,
    "problem_id": 10,
    "input_data": "5\n1 2 3 4 5\n",
    "expected_output": "15\n",
    "is_sample": true
  },
  "error": null
}
```

---

### 3.5. Submissions & Execution Sandbox (`/submissions`)

---

#### 3.5.1. Submit Solution Code

Mengirimkan kode C/C++ untuk dievaluasi secara otomatis di Docker Container Sandbox.

- **Endpoint:** `POST /submissions`
- **Akses:** Authenticated (`USER` & `ADMIN`)

- **Request Body:**

```json
{
  "problem_id": 10,
  "language": "cpp", // 'c' atau 'cpp'
  "code": "#include <iostream>\nusing namespace std;\nint main() {\n  string s;\n  if(cin >> s) {\n    for(int i=s.length()-1; i>=0; i--) cout << s[i];\n    cout << endl;\n  }\n  return 0;\n}"
}
```

- **Alur & Logika Teknis (Flow & Logic):**

1. **Validasi Pengerjaan:** Check apakah waktu saat ini berada di rentang `start_time <= NOW() <= end_time` dari `problem_group` terkait. Jika di luar rentang, tolak request dengan `403 Forbidden`.

2. **Insert Initial Record:** BE memasukkan data submission awal ke tabel `submissions` dengan `verdict = 'PENDING'`.

3. **Queue / Worker Channel:** BE memasukkan ID submission ini ke dalam antrean _Worker Pool Go_ secara asinkron.
4. **Proses Asinkron Docker Sandbox Execution:**

- Worker mengubah status submission di DB menjadi `'RUNNING'`.

- Worker mengambil seluruh `test_cases` (termasuk hidden cases) milik `problem_id`.

- Worker menyiapkannya di dalam kontainer Docker ephemeral (misal berbasis `gcc:alpine` atau `g++`).

- **Isolasi Keamanan:** Docker dijalankan dengan flag `--network none` (tanpa internet), dibatasi RAM sesuai `memory_limit_mb`, dan diberi batas timeout eksekusi `time_limit_ms`.

- **Proses Evaluasi Verdict:**
- Jika kompilasi `gcc/g++` gagal $\rightarrow$ Status `'CE'`, simpan pesan log di `compile_output`.

- Jika eksekusi melebihi waktu $\rightarrow$ Status `'TLE'`.

- Jika program crash/segfault $\rightarrow$ Status `'RTE'`.

- Jika output tidak cocok dengan `expected_output` $\rightarrow$ Status `'WA'`.

- Jika seluruh test case menghasilkan output yang 100% cocok $\rightarrow$ Status `'AC'`.

- **Aturan Atomic First Blood:** Jika status yang diperoleh adalah `'AC'`, **Trigger Database (`trigger_check_first_blood`)** otomatis akan mencatatkan `user_id` ini ke kolom `first_blood_user_id` di tabel `problems` jika belum pernah terisi.

5. Kirim respons `202 Accepted` langsung ke FE setelah pesan sukses masuk ke antrean.

- **Response (`202 Accepted`):**

```json
{
  "success": true,
  "message": "Submission diterima dan sedang diproses dalam antrean",
  "data": {
    "submission_id": 501,
    "status": "PENDING",
    "submitted_at": "2026-10-18T00:40:00Z"
  },
  "error": null
}
```

---

#### 3.5.2. Get Submission Status / Polling

Digunakan oleh Frontend untuk melakukan polling berkala guna mengetahui hasil akhir eksekusi kode (_verdict_).

- **Endpoint:** `GET /submissions/:id`
- **Akses:** Authenticated (Pemilik Submission atau `ADMIN`)

- **Request Params:** `id` (Submission ID)
- **Alur & Logika Teknis (Flow & Logic):**

1. Ambil data submission dari tabel `submissions` berdasarkan ID.

2. Verifikasi kepemilikan submission (User hanya boleh membaca submission miliknya sendiri, kecuali Admin).

3. Kembalikan detail status `verdict`, `execution_time_ms`, dan `compile_output` (jika CE).

- **Response (`200 OK`):**

```json
{
  "success": true,
  "message": "Detail submission berhasil diambil",
  "data": {
    "id": 501,
    "user_id": 2,
    "problem_id": 10,
    "language": "cpp",
    "code": "#include <iostream>...",
    "verdict": "AC",
    "execution_time_ms": 12,
    "compile_output": "",
    "submitted_at": "2026-10-18T00:40:00Z"
  },
  "error": null
}
```

---

### 3.6. Scoreboard Matrix & Color Logic (`/scoreboards`)

---

#### 3.6.1. Get Group Scoreboard Matrix

Mengembalikan tabel matriks papan peringkat per grup soal beserta kalkulasi warna indikator visual sesuai kriteria PRD.

- **Endpoint:** `GET /scoreboards/groups/:group_id`
- **Akses:** Authenticated (`ADMIN` & `USER`)

- **Request Params:** `group_id` (Group Problem ID)
- **Alur & Logika Teknis (Flow & Logic - Kalkulasi Warna & Peringkat):**

1. Ambil seluruh daftar peserta (`users`) dan seluruh daftar soal (`problems`) di grup ini.

2. Ambil seluruh riwayat submission dari tabel `submissions` untuk grup soal tersebut.

3. **Algoritma Pewarnaan & Status Per Soal (`color_code`):**

- 🟢 **`DARK_GREEN` (First Blood):** Jika user memiliki setidaknya satu submission `verdict = 'AC'` DAN `user_id == problems.first_blood_user_id`.

- 🟢 **`LIGHT_GREEN` (Accepted):** Jika user memiliki setidaknya satu submission `verdict = 'AC'` TETAPI bukan pemenang First Blood.

- 🔴 **`RED` (Attempted / Wrong):** Jika user pernah mengirimkan jawaban (punya record submission) TETAPI tidak pernah mendapatkan status `'AC'`.

- ⚪ **`GRAY` (Not Attempted):** Jika user belum pernah mengirimkan jawaban sama sekali untuk soal ini.

4. **Pengurutan Peringkat (_Sorting Logic_):**

- Peringkat utama diurutkan berdasarkan `total_solved` terbanyak (jumlah soal berstatus AC) secara _descending_.
- Jika `total_solved` sama, urutkan berdasarkan waktu pencapaian `AC` terakhir yang paling cepat.

- **Response (`200 OK`):**

```json
{
  "success": true,
  "message": "Scoreboard berhasil didapatkan",
  "data": {
    "group_id": 1,
    "group_title": "Praktikum Algoritma Minggu 1",
    "problems": [
      { "id": 10, "title": "A" },
      { "id": 11, "title": "B" },
      { "id": 12, "title": "C" }
    ],
    "rankings": [
      {
        "rank": 1,
        "user_id": 2,
        "username": "mahasiswa123",
        "name": "Budi Santoso",
        "total_solved": 2,
        "submissions_status": {
          "10": {
            "verdict": "AC",
            "is_first_blood": true,
            "color_code": "DARK_GREEN",
            "attempt_count": 1
          },
          "11": {
            "verdict": "AC",
            "is_first_blood": false,
            "color_code": "LIGHT_GREEN",
            "attempt_count": 3
          },
          "12": {
            "verdict": "WA",
            "is_first_blood": false,
            "color_code": "RED",
            "attempt_count": 2
          }
        }
      },
      {
        "rank": 2,
        "user_id": 4,
        "username": "mhs_002",
        "name": "Siti Aminah",
        "total_solved": 0,
        "submissions_status": {
          "10": {
            "verdict": "NOT_ATTEMPTED",
            "is_first_blood": false,
            "color_code": "GRAY",
            "attempt_count": 0
          },
          "11": {
            "verdict": "NOT_ATTEMPTED",
            "is_first_blood": false,
            "color_code": "GRAY",
            "attempt_count": 0
          },
          "12": {
            "verdict": "NOT_ATTEMPTED",
            "is_first_blood": false,
            "color_code": "GRAY",
            "attempt_count": 0
          }
        }
      }
    ]
  },
  "error": null
}
```
