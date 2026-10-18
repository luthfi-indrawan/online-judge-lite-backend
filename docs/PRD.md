# 📄 Product Requirement Document (PRD)

**Nama Proyek:** Online Judge (Binus-style) Lite

---

## 1. Executive Summary & Goals

Aplikasi ini adalah platform **Online Judge** berbasis web yang dirancang khusus untuk lingkungan akademik/kompetisi (serupa BINUS SOCS). System ini memungkinkan **Admin/Dosen/Tutor** untuk mengelola grup soal (_contests/lab sessions_), mengunggah deskripsi soal berbasis file PDF, menentukan jadwal pengerjaan, serta mendaftarkan pasangan _Input Case_ dan _Expected Output Case_.

**User/Mahasiswa** dapat mengakses grup soal aktif, mengunduh/membaca file PDF deskripsi soal, menulis & mengirimkan kode C/C++ (_submission_), mendapatkan umpan balik kelayakan kode secara otomatis (_automated checking case_), serta memantau papan peringkat (_scoreboard_) interaktif per grup soal.

---

## 2. Tech Stack & Infrastructure

- **Frontend:** React.js, Tailwind CSS, Code Editor (Monaco Editor / CodeMirror), PDF Viewer (`react-pdf` / Embedded Viewer).
- **Backend:** Go (Golang) — Gin / Fiber framework.
- **Database & Auth:** PostgreSQL + JWT (JSON Web Tokens) / Session Auth.
- **Storage:** Supabase Storage (Object Storage untuk file PDF deskripsi soal).
- **Execution Engine / Sandbox:** **Docker Container (GCC/G++ Alpine Container)** — Isolasi proses kompilasi & eksekusi kode C/C++ secara aman dan efisien.

---

## 3. User Roles & Access Control

### A. Admin / Lecturer / Tutor

- **No Public Registration:** Tidak ada pendaftaran mandiri. Akun Admin dibuat langsung via _seeder_ database / konfigurasi awal.
- **User Management:** Membuat akun user/mahasiswa secara manual atau _bulk import_.
- **Group & Problem Management:** Membuat grup soal, mengatur jadwal (buka/tutup), mengunggah PDF soal ke Supabase Storage, serta menentukan test cases (_input_ & _expected output_).

### B. User / Student

- **Authenticated Access:** Login menggunakan kredensial yang dibuatkan oleh Admin.
- **Problem Execution & Scoreboard:** Mengakses grup soal aktif, membaca PDF soal, melakukan submission kode C/C++ berkali-kali selama rentang waktu aktif, serta memantau posisi di Scoreboard.

---

## 4. Detailed Feature Specifications

### 4.1. Admin Management & Problem Setup

1. **User Management:**

- Admin dapat menambahkan akun peserta secara individu atau _bulk create_.
- Admin berwenang melakukan reset password akun peserta.

2. **Group Problem Management (Contest / Practice Session):**

- **Fields:** Judul Grup, Deskripsi, `Start Time` (Waktu Buka), `End Time` (Waktu Tutup).
- Akses pengerjaan soal dan _submission_ hanya diizinkan apabila waktu server berada di dalam rentang `Start Time` $\le$ `Current Time` $\le$ `End Time`.

3. **Problem & Test Case Management:**

- Admin membuat soal di dalam Group Problem tertentu.
- **Upload Deskripsi Soal:** File PDF diunggah langsung ke **Supabase Storage**, link CDN disimpan pada database.
- **Test Cases Setup:**
- Admin menginput satu atau lebih pasangan _Input Case_ dan _Expected Output Case_.
- Batasan Eksekusi: _Time Limit_ (misal: `1000 ms`) dan _Memory Limit_ (misal: `128 MB`).

---

### 4.2. Docker-based Compilation & Code Runner Engine

Setiap submission koding dari user diproses menggunakan mekanisme **Docker Sandbox Isolation**:

1. **Execution Flow:**

- User mengirimkan kode program C atau C++ melalui API backend Go.
- Backend Go memasukkan job ke _Submission Queue/Worker Pool_.
- Worker menjalankan Docker Container sementara (_ephemeral container_) berbasis Linux minimalis yang dilengkapi kompilator `gcc` / `g++`.

2. **Container Security & Resource Restrictions:**

- **Network Disabled:** Docker dijalankan dengan flag `--network none` (tanpa akses internet) untuk mencegah eksploitasi atau _Remote Code Execution_ (RCE).
- **Memory & CPU Limits:** Kontainer dibatasi penggunaan RAM dan CPU-nya untuk mencegah _Resource Exhaustion_.
- **Execution Timeout:** Jika eksekusi melebihi _Time Limit_, kontainer akan dipaksa mati (_killed_) dan mengembalikan status `TLE`.
- **Automated Destruction:** Kontainer langsung dihapus (_rm_) segera setelah eksekusi selesai.

3. **Verdict Output System:**

| Status Verdict                | Deskripsi Teknikal                                                                                |
| ----------------------------- | ------------------------------------------------------------------------------------------------- |
| **Accepted (AC)**             | Output program 100% sama dengan _Expected Output_.                                                |
| **Wrong Answer (WA)**         | Program berjalan sukses tanpa error, tetapi output tidak cocok dengan _Expected Output_.          |
| **Compile Error (CE)**        | Proses kompilasi `gcc`/`g++` gagal. Pesan error dari compiler disimpan untuk ditampilkan ke user. |
| **Time Limit Exceeded (TLE)** | Waktu eksekusi program melebihi batas waktu (_Time Limit_) yang ditentukan.                       |
| **Runtime Error (RTE)**       | Program mengalami crash saat berjalan (misal: _Segmentation Fault_, _Division by Zero_).          |

---

### 4.3. User Experience & Code Submission

1. **Dashboard & Group Access:**

- User melihat daftar grup soal beserta indikator status (Belum Dibuka, Sedang Berlangsung, atau Sudah Ditutup).

2. **Problem Reader & Embedded Workspace:**

- User membuka soal dan membaca file PDF deskripsi yang dipanggil langsung dari Supabase Storage.
- User menulis/menempelkan kode C/C++ pada kode editor terintegrasi.

3. **Multi-Submission:**

- User diperbolehkan mengirimkan (_submit_) jawaban berkali-kali tanpa batasan jumlah percobaan hingga batas waktu pengerjaan berakhir.

---

### 4.4. Scoreboard & Color Logic

Scoreboard dikalkulasi secara terpisah per Group Problem dan diperbarui secara _real-time_ atau via polling otomatis.

#### Indikator Warna Status Soal per User:

| Status Soal                      | Indikator Warna                  | Logic / Kriteria                                                                                                                              |
| -------------------------------- | -------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| **First Blood / First Accepted** | 🟢 **Hijau Tua (Dark Green)**    | User pertama di seluruh grup yang berhasil mendapatkan verdict `Accepted` (AC) untuk soal tersebut.                                           |
| **Accepted**                     | 🟢 **Hijau biasa (Light Green)** | User berhasil mendapatkan verdict `Accepted` (AC), tetapi bukan yang pertama.                                                                 |
| **Attempted / Wrong**            | 🔴 **Merah (Red)**               | User pernah melakukan submission untuk soal tersebut, namun belum pernah berhasil mendapatkan verdict `Accepted` (masih WA / TLE / CE / RTE). |
| **Not Attempted**                | ⚪ **Abu-abu / Netral**          | User belum pernah mengirimkan jawaban untuk soal tersebut.                                                                                    |

---

## 5. Database Schema (PostgreSQL)

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE user_role AS ENUM ('ADMIN', 'USER');

CREATE TYPE programming_language AS ENUM ('c', 'cpp');

CREATE TYPE submission_verdict AS ENUM (
    'PENDING',
    'RUNNING',
    'AC',
    'WA',
    'CE',
    'TLE',
    'RTE'
);



CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role user_role NOT NULL DEFAULT 'USER',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE problem_groups (
    id SERIAL PRIMARY KEY,
    title VARCHAR(150) NOT NULL,
    description TEXT,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    created_by INT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_group_time CHECK (end_time > start_time)
);

CREATE TABLE problems (
    id SERIAL PRIMARY KEY,
    group_id INT NOT NULL REFERENCES problem_groups(id) ON DELETE CASCADE,
    title VARCHAR(150) NOT NULL,
    pdf_url TEXT NOT NULL,
    time_limit_ms INT DEFAULT 1000 NOT NULL CHECK (time_limit_ms > 0),
    memory_limit_mb INT DEFAULT 128 NOT NULL CHECK (memory_limit_mb > 0),
    first_blood_user_id INT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE test_cases (
    id SERIAL PRIMARY KEY,
    problem_id INT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    input_data TEXT NOT NULL,
    expected_output TEXT NOT NULL,
    is_sample BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE submissions (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    problem_id INT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    language programming_language NOT NULL,
    code TEXT NOT NULL,
    verdict submission_verdict NOT NULL DEFAULT 'PENDING',
    execution_time_ms INT DEFAULT 0,
    compile_output TEXT,
    submitted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);


CREATE INDEX idx_submissions_problem_user ON submissions(problem_id, user_id, verdict);

CREATE INDEX idx_submissions_submitted_at ON submissions(submitted_at DESC);

CREATE INDEX idx_problem_groups_schedule ON problem_groups(start_time, end_time);


CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$ BEGIN NEW.updated_at = CURRENT_TIMESTAMP; RETURN NEW; END; $$ LANGUAGE plpgsql;

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_problem_groups_updated_at
    BEFORE UPDATE ON problem_groups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_problems_updated_at
    BEFORE UPDATE ON problems
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();



CREATE OR REPLACE FUNCTION check_first_blood()
RETURNS TRIGGER AS $$
DECLARE
    v_first_blood_id INT;
BEGIN
    IF NEW.verdict = 'AC' THEN
        SELECT first_blood_user_id INTO v_first_blood_id
        FROM problems
        WHERE id = NEW.problem_id;

        IF v_first_blood_id IS NULL THEN
            UPDATE problems
            SET first_blood_user_id = NEW.user_id
            WHERE id = NEW.problem_id;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_check_first_blood
    AFTER INSERT OR UPDATE OF verdict ON submissions
    FOR EACH ROW
    WHEN (NEW.verdict = 'AC')
    EXECUTE FUNCTION check_first_blood();


INSERT INTO users (username, name, password_hash, role)
VALUES ('admin', 'System Administrator', '$2a$10$76.k508/bIn58.2I1EwR6e.sVj2sJj4M3.52u0QYf6r4Y1Y5A8Rie', 'ADMIN');
```
