# Warta

Backend platform artikel dan berita. Go + MySQL.

- Akun dengan tiga role: `admin`, `author`, `reader`
- Login JWT dengan refresh token yang dirotasi setiap dipakai
- Artikel dengan status `draft`, `published`, `archived`, slug otomatis, kategori, dan tag
- Isi artikel dalam Markdown, gambar sampul yang bisa diunggah, dan perkiraan waktu baca
- Pencarian, filter, pengurutan (termasuk "populer"), dan paging dengan total data
- Komentar, suka, dan bookmark pembaca, serta hitungan dibaca per hari
- Statistik dashboard untuk penulis dan admin
- Dokumentasi OpenAPI dengan Swagger UI di `/docs`

## Menjalankan

Butuh Docker. Service dan databasenya sekaligus:

```bash
docker compose up -d --build
```

Tunggu sekitar 20 detik, lalu buka `http://localhost:8080/docs`. Akun admin
bawaan untuk mencoba: `admin@warta.local` / `admin12345`. Nilai bawaan ini
hanya untuk komputer sendiri; service mencatat peringatan di log saat
memakainya. Untuk server sungguhan, lihat [Menjalankan di production](#menjalankan-di-production).

Kalau mau jalan langsung dari kode dan hanya databasenya di container:

```bash
cp .env.example .env
docker compose up -d mysql
go run ./cmd/api
```

Di `.env`, isi `DB_PASSWORD=root` supaya cocok dengan `docker-compose.yml`,
isi `JWT_SECRET` (minimal 32 karakter, misalnya dari `openssl rand -base64 48`),
dan isi `ADMIN_PASSWORD`.

## Akun dan role

| Role | Bisa |
|---|---|
| `reader` | membaca, berkomentar, mengubah dan menghapus komentarnya sendiri |
| `author` | semua yang reader bisa, ditambah menulis artikel dan mengelola artikelnya sendiri |
| `admin` | semuanya, termasuk kategori, tag, role user, dan artikel serta komentar siapa pun |

Register selalu menghasilkan `reader`. Admin menaikkan role lewat
`PATCH /api/v1/users/{id}/role`. Admin tidak bisa mengubah role akunnya sendiri,
jadi selalu tersisa minimal satu admin.

Akun admin pertama dibuat saat service menyala dari `ADMIN_EMAIL` dan
`ADMIN_PASSWORD`. Bila email itu sudah terdaftar, akunnya dinaikkan menjadi
admin tanpa mengubah password-nya. Email admin dianggap sudah terverifikasi.

### Verifikasi email dan lupa password

Register mengirim email berisi tautan `APP_URL/verify-email?token=...`.
Halaman frontend itu meneruskan token ke `POST /auth/verify-email`. Tautan
berlaku 24 jam dan hanya sekali pakai; `POST /auth/resend-verification`
mengirim yang baru (paling cepat sekali per menit) dan membatalkan yang lama.

Bila `REQUIRE_EMAIL_VERIFICATION` aktif (bawaan di production), akun yang
emailnya belum terverifikasi tidak bisa berkomentar atau melaporkan komentar
(403 dengan kode `email_not_verified`); membaca, menyukai, dan menyimpan
artikel tetap bisa. Field `email_verified` di data akun menunjukkan statusnya.
Akun yang sudah ada sebelum fitur ini dianggap terverifikasi.

Lupa password: `POST /auth/forgot-password` dengan `email` mengirim tautan
`APP_URL/reset-password?token=...` yang berlaku 1 jam. Jawabannya selalu 204,
terdaftar atau tidak emailnya, supaya endpoint ini tidak bisa dipakai menebak
akun. `POST /auth/reset-password` dengan `token` dan `new_password` mengganti
password, mencabut semua sesi, dan sekaligus menandai email terverifikasi.

Seperti refresh token, di database hanya tersimpan hash token email. Email
dikirim lewat SMTP (`SMTP_HOST` dan kawan-kawan) di latar belakang, jadi
permintaan tidak menunggu server email. Tanpa `SMTP_HOST`, isi email hanya
dicatat ke log service, cukup untuk mencoba di komputer sendiri: salin
tautannya dari log.

### Token

Login dan register mengembalikan:

```json
{
  "data": {
    "access_token": "eyJhbGciOi...",
    "token_type": "Bearer",
    "expires_in": 900,
    "refresh_token": "g26hKLlInBVK...",
    "user": { "id": 2, "name": "Budi", "email": "budi@warta.id", "role": "reader", "email_verified": false }
  }
}
```

Kirim access token di header `Authorization: Bearer <token>`. Umurnya pendek
(bawaan 15 menit). Setelah kedaluwarsa, tukar refresh token di
`POST /api/v1/auth/refresh` untuk mendapat pasangan token baru.

Refresh token hanya berlaku sekali. Bila refresh token yang sudah ditukar
dipakai lagi, kemungkinan besar token itu bocor, jadi semua sesi akun itu
dicabut dan penggunanya harus login ulang. Ganti password juga mencabut semua
sesi. Di database hanya tersimpan hash SHA-256 dari refresh token.

Role ikut tersimpan di access token. Perubahan role oleh admin berlaku setelah
pengguna melakukan refresh.

## Endpoint

Semua di bawah `/api/v1`. Kolom Akses: `-` publik, `login` semua role,
`author` untuk author dan admin.

| Method | URL | Akses | |
|---|---|---|---|
| POST | `/auth/register` | - | membuat akun reader |
| POST | `/auth/login` | - | login |
| POST | `/auth/refresh` | - | menukar refresh token |
| POST | `/auth/logout` | - | mencabut refresh token |
| POST | `/auth/verify-email` | - | verifikasi email dengan token dari email |
| POST | `/auth/resend-verification` | login | kirim ulang email verifikasi |
| POST | `/auth/forgot-password` | - | minta tautan reset password |
| POST | `/auth/reset-password` | - | password baru dengan token dari email |
| GET, PATCH | `/me` | login | data akun sendiri, ganti nama |
| PUT | `/me/password` | login | ganti password |
| GET | `/me/articles` | login | artikel sendiri, semua status |
| GET | `/me/bookmarks` | login | artikel yang disimpan |
| GET | `/stats` | author | statistik dashboard, `?days=7..90` |
| POST | `/uploads` | author | unggah gambar sampul (multipart, field `image`) |
| GET | `/users` | admin | daftar user, filter `q` dan `role` |
| GET | `/users/{id}` | admin | detail user |
| PATCH | `/users/{id}/role` | admin | ubah role |
| GET | `/categories` | - | semua kategori beserta jumlah artikel terbit |
| GET | `/categories/{id atau slug}` | - | detail kategori |
| POST | `/categories` | admin | buat kategori |
| PUT, DELETE | `/categories/{id}` | admin | ubah, hapus kategori |
| GET | `/tags` | - | daftar tag, cari dengan `q` |
| POST | `/tags` | admin | buat tag |
| PUT, DELETE | `/tags/{id}` | admin | ganti nama, hapus tag |
| GET | `/articles` | - | daftar artikel |
| GET | `/articles/{id atau slug}` | - | detail artikel |
| POST | `/articles` | author | tulis artikel |
| PUT | `/articles/{id}` | penulis, admin | ganti seluruh isi |
| PATCH | `/articles/{id}` | penulis, admin | ubah sebagian |
| DELETE | `/articles/{id}` | penulis, admin | hapus beserta komentarnya |
| GET | `/articles/{id}/comments` | - | komentar, dari yang terlama |
| POST | `/articles/{id}/comments` | login | berkomentar |
| PATCH | `/comments/{id}` | penulis komentar | ubah komentar |
| POST | `/comments/{id}/report` | login | laporkan komentar |
| GET | `/moderation/comments` | admin | antrean komentar yang dilaporkan atau disembunyikan |
| POST | `/moderation/comments/{id}` | admin | `approve` atau `hide` |
| DELETE | `/comments/{id}` | penulis komentar, admin | hapus komentar |
| PUT, DELETE | `/articles/{id}/like` | login | suka, batal suka |
| PUT, DELETE | `/articles/{id}/bookmark` | login | simpan, batal simpan |
| POST | `/articles/{id}/view` | - | catat artikel dibaca |

Di luar `/api/v1`: `GET /health`, `GET /health/ready` (ikut mengecek database),
`GET /docs` (Swagger UI), `GET /api/v1/openapi.yaml`, `GET /uploads/{nama}`
(gambar yang diunggah), serta `GET /sitemap.xml` dan `GET /feed.xml` (RSS).

Sitemap berisi beranda, kategori yang punya artikel, dan artikel terbit; RSS
berisi 20 artikel terbit terbaru. Semua tautan di dalamnya menuju frontend
(`APP_URL`), dan keduanya di-cache 15 menit. Frontend meneruskan
`/sitemap.xml` dan `/feed.xml` di domainnya ke sini.

Rincian lengkap setiap request dan response ada di `api/openapi.yaml`.

### Daftar artikel

```
GET /api/v1/articles?q=golang&category=teknologi&tag=backend&author=2&sort=newest&page=1&per_page=10
```

| Parameter | |
|---|---|
| `q` | cari di judul dan isi, tidak peka huruf besar |
| `category`, `tag` | slug kategori atau tag |
| `author` | id penulis |
| `status` | `draft`, `published`, `archived`, atau `all`. Tanpa parameter ini hanya artikel terbit yang tampil. Selain `published`, hanya admin. |
| `sort` | `newest` (bawaan), `oldest`, `title`, `updated`, `popular` |
| `page`, `per_page` | paging, `per_page` dipangkas ke `MAX_PER_PAGE` |

```json
{
  "data": [
    {
      "id": 1,
      "title": "Panduan Membangun REST API dengan Golang",
      "slug": "panduan-membangun-rest-api-dengan-golang",
      "excerpt": "Artikel ini membahas cara menyusun REST API dengan Go dari nol…",
      "status": "published",
      "author": { "id": 2, "name": "Budi" },
      "category": { "id": 1, "name": "Teknologi", "slug": "teknologi" },
      "tags": [{ "id": 1, "name": "golang", "slug": "golang" }],
      "comment_count": 3,
      "published_at": "2026-09-28T05:00:00Z",
      "created_at": "2026-09-27T10:00:00Z",
      "updated_at": "2026-09-28T05:00:00Z"
    }
  ],
  "meta": { "page": 1, "per_page": 10, "total": 1, "total_pages": 1 }
}
```

Daftar berisi cuplikan (`excerpt`); isi lengkap (`content`) hanya ada di detail.

### Menulis artikel

```bash
curl -X POST http://localhost:8080/api/v1/articles \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Panduan Membangun REST API dengan Golang","content":"...","category_id":1,"tags":["golang","backend"],"status":"draft"}'
```

- `title` wajib, 20 sampai 200 karakter
- `content` wajib, 200 sampai 100.000 karakter
- `category_id` wajib, kategorinya harus ada
- `tags` opsional, paling banyak 10, masing-masing 2 sampai 50 karakter. Tag
  dirapikan menjadi huruf kecil, yang ganda dibuang, dan yang belum ada dibuat
  otomatis.
- `status` wajib, salah satu dari `draft`, `published`, `archived`
- `cover_image` opsional, path hasil `POST /api/v1/uploads`

`content` ditulis dalam Markdown dan disimpan apa adanya; frontend yang
merendernya. Cuplikan di daftar artikel dibersihkan dari sintaks Markdown.

`PUT` mewajibkan semua field. `PATCH` hanya mengubah field yang dikirim, jadi
menerbitkan draft cukup dengan `{"status": "published"}`.

### Gambar sampul

```bash
curl -X POST http://localhost:8080/api/v1/uploads \
  -H "Authorization: Bearer $TOKEN" \
  -F image=@sampul.png
# {"data":{"url":"/uploads/3f2a9c0d4e5b6a7c8d9e0f1a2b3c4d5e.png"}}
```

JPEG, PNG, WebP, atau GIF, paling besar `MAX_UPLOAD_MB` (bawaan 2 MB). Format
dikenali dari isi berkas, bukan dari nama atau header yang dikirim klien, dan
nama berkasnya selalu dibuat acak oleh server. Berkas disimpan di `UPLOAD_DIR`;
di Docker folder itu memakai volume `warta_uploads`.

### Suka, bookmark, dan dibaca

`PUT /articles/{id}/like` dan `/bookmark` bersifat idempoten: menyukai dua kali
tetap satu suka. Responsnya keadaan terbaru, `{"liked", "bookmarked",
"like_count"}`. Detail artikel juga membawa `liked` dan `bookmarked` untuk
pembaca yang login.

`POST /articles/{id}/view` dipanggil halaman baca. Pembaca yang sama (akun,
atau IP bila belum login) hanya dihitung sekali per 30 menit, dan penulis yang
membuka artikelnya sendiri tidak dihitung. Hitungan disimpan per hari untuk
grafik di dashboard.

Urutan `popular` menimbang jumlah dibaca, suka (x5), dan komentar (x3).

### Statistik

`GET /api/v1/stats?days=30` mengembalikan total artikel per status, dibaca,
suka, komentar, dan bookmark; aktivitas per hari (dibaca, komentar, artikel
terbit) dengan hari kosong tetap diisi nol; dan lima artikel terpopuler. Admin
melihat semua artikel ditambah jumlah akun per role, author hanya artikelnya
sendiri. Tanggal dihitung dalam UTC.

## Aturan yang perlu diketahui

**Artikel yang belum terbit hanya terlihat oleh penulisnya dan admin.** Bagi
yang lain, responsnya 404, bukan 403, supaya keberadaan draft tidak bocor.

**Slug** dibuat dari judul dan dijamin unik dengan akhiran `-2`, `-3`, dan
seterusnya. Selama artikel belum pernah terbit, slug ikut berubah bila judulnya
diubah. Setelah terbit slug tidak berubah lagi, karena tautannya mungkin sudah
tersebar. Slug tidak pernah seluruhnya angka, sehingga `/articles/{id atau slug}`
tidak pernah salah tebak.

**`published_at`** diisi saat artikel pertama kali terbit dan tidak berubah
walau artikel kemudian diarsipkan dan diterbitkan lagi.

**Komentar** hanya bisa ditambahkan ke artikel yang sudah terbit. Admin bisa
menghapus komentar siapa pun, tapi tidak mengubah kata-katanya.

**Perlindungan spam komentar:**

- Setiap akun paling banyak `COMMENT_RATE_LIMIT` komentar per menit (bawaan 5,
  termasuk mengubah dan melaporkan komentar). Batasnya per akun, jadi tidak
  bisa diakali dengan berganti IP.
- Komentar paling banyak berisi 2 tautan dan tidak boleh seluruhnya huruf
  kapital (422).
- Komentar yang sama persis dari akun yang sama di artikel yang sama dalam 10
  menit ditolak (409).
- Pembaca bisa melaporkan komentar orang lain dengan alasan `spam`, `abusive`,
  atau `other`, satu laporan per akun. Bila laporan dari akun berbeda mencapai
  `COMMENT_HIDE_THRESHOLD` (bawaan 3), komentar disembunyikan otomatis: tidak
  tampil dan tidak ikut dihitung di `comment_count` maupun statistik.
- Admin meninjau antrean di `GET /moderation/comments`: `approve` menampilkan
  kembali komentar dan menghapus laporannya, `hide` menyembunyikannya, dan
  `DELETE /comments/{id}` menghapus permanen.

**Kategori** yang masih dipakai artikel tidak bisa dihapus (409). Menghapus tag
melepasnya dari semua artikel.

**Token yang dikirim tapi tidak valid** selalu ditolak dengan 401, termasuk di
endpoint publik, supaya klien tahu harus refresh alih-alih diam-diam
diperlakukan sebagai pengunjung anonim.

**Register, login, refresh, verifikasi email, dan reset password dibatasi**
`AUTH_RATE_LIMIT` permintaan per menit per IP (bawaan 20). Upload gambar dibatasi `UPLOAD_RATE_LIMIT` per menit
per akun (bawaan 10). Kelebihannya dibalas 429 dengan header `Retry-After`.

**IP pengunjung** dipakai untuk batas login dan hitungan dibaca. Bila service
berada di balik reverse proxy (Nginx, load balancer), semua koneksi datang dari
proxy itu. Daftarkan IP proxy di `TRUSTED_PROXIES`, maka IP asli dibaca dari
`X-Forwarded-For`. Header itu hanya dipercaya dari proxy yang terdaftar, dan
dibaca dari kanan sehingga entri palsu dari klien diabaikan. Tanpa
`TRUSTED_PROXIES`, header itu diabaikan sama sekali.

## Bentuk response

Sukses: `{"data": ...}`, ditambah `meta` untuk daftar berhalaman. Hapus dan
logout membalas 204 tanpa body.

Gagal:

```json
{
  "error": {
    "code": "validation_failed",
    "message": "validasi gagal",
    "fields": { "title": "title minimal 20 karakter" }
  }
}
```

| Status | `code` |
|---|---|
| 400 | `bad_request`: body atau parameter tidak bisa dibaca |
| 401 | `unauthorized`: token tidak ada, tidak valid, atau kedaluwarsa |
| 403 | `forbidden`: role atau kepemilikan tidak mengizinkan |
| 404 | `not_found` |
| 405 | `method_not_allowed` |
| 409 | `conflict` |
| 413 | `payload_too_large`: body lebih dari 1 MB |
| 422 | `validation_failed`, rinciannya per field di `fields` |
| 429 | `too_many_requests` |
| 500 | `internal_error` |

Setiap response membawa header `X-Request-ID` yang juga tercatat di log, untuk
menelusuri satu permintaan.

## Migrasi

Pakai golang-migrate, berkas SQL-nya di `migrations/` dan ikut di-embed ke binary.

```bash
go run ./cmd/migrate up
go run ./cmd/migrate down
go run ./cmd/migrate goto 5
go run ./cmd/migrate version
```

golang-migrate tidak bisa membuat database, hanya mengisinya, jadi
`CREATE DATABASE IF NOT EXISTS` dijalankan lebih dulu oleh runner-nya. Dengan
`AUTO_MIGRATE=true`, migrasi juga jalan otomatis tiap service menyala.

Migrasi `000001` adalah tabel `posts` dari versi awal project ini. Migrasi
`000006` mengubahnya menjadi `articles` tanpa membuang data: kategori teks
dipindah ke tabel `categories`, status `publish`/`thrash` menjadi
`published`/`archived`, setiap post diberi slug, dan waktu buat serta ubah yang
asli dipertahankan. Post lama belum punya penulis, jadi semuanya diberikan ke
akun `arsip@warta.local` yang tidak bisa dipakai login. Migrasi `000009` dan
`000010` menambahkan sampul, hitungan dibaca, suka, dan bookmark; `000011`
menambahkan laporan dan penyembunyian komentar.

Untuk membuat skema manual tanpa migrate, ada `docs/schema.sql`.

## Test

```bash
make test               # unit test, tanpa database
make db-up              # MySQL dari docker compose
make test-integration   # semua test, termasuk end-to-end terhadap MySQL
```

Test end-to-end di `internal/app` membuat database sendiri dengan nama acak,
menjalankan migrasi naik, turun, lalu naik lagi, kemudian menguji seluruh alur
lewat HTTP. Ada juga test yang mengisi tabel `posts` versi awal lalu memastikan
migrasi mengubahnya dengan benar. Test ini dilewati bila `TEST_DB_HOST` kosong.

GitHub Actions (`.github/workflows/ci.yml`) menjalankan gofmt, `go vet`, semua
test dengan MySQL, dan build image Docker.

### Postman

Import `docs/Warta.postman_collection.json` lalu jalankan lewat Run
collection. Urutannya sudah disusun dari login admin sampai pembersihan, dan
email akun dibuat unik setiap Run sehingga bisa dijalankan berulang. Sesuaikan
variable `admin_email` dan `admin_password` bila tidak memakai nilai bawaan.

Tanpa membuka Postman:

```bash
npx newman run docs/Warta.postman_collection.json
```

Satu Run memakai 7 permintaan ke endpoint auth. Menjalankannya lebih dari dua kali
dalam semenit akan terkena rate limit.

## Struktur

```
cmd/api              menyalakan service
cmd/migrate          perintah migrasi
api                  spesifikasi OpenAPI (di-embed)
migrations           berkas SQL migrasi (di-embed)
internal/app         merangkai semua lapisan menjadi http.Handler
internal/router      daftar rute dan middleware per rute
internal/middleware  request id, log, recover, CORS, JWT, role, rate limit
internal/clientip    IP pengunjung di balik reverse proxy
internal/handler     handler HTTP
internal/service     aturan bisnis: validasi, hak akses, slug, token
internal/repository  query MySQL
internal/auth        JWT, refresh token, bcrypt, identitas pemanggil
internal/validation  aturan validasi
internal/dto         bentuk request dan response
internal/model       struct domain
internal/pagination  membaca page dan per_page
internal/slug        membuat slug
internal/storage     menyimpan dan memeriksa gambar yang diunggah
internal/mail        pengiriman email lewat SMTP dan templatnya
internal/apperr      error yang membawa status HTTP
internal/response    penulisan response JSON
internal/logging     logger slog dengan request id
internal/config      pembacaan environment
internal/database    koneksi MySQL dan migrasi
```

Alurnya handler ke service ke repository. Routing memakai `net/http` bawaan Go
yang sudah bisa mencocokkan method dan membaca `{id}` dari URL, jadi tidak
perlu pustaka router tambahan. Dependensi luar: driver MySQL, golang-migrate,
godotenv, golang-jwt, dan `x/crypto` untuk bcrypt.

## Konfigurasi

Semua lewat environment, daftar lengkapnya ada di `.env.example`. Yang wajib
hanya `JWT_SECRET`. Service menolak menyala dan menyebutkan semua nilai yang
salah sekaligus bila konfigurasinya tidak valid.

`CORS_ORIGINS` bawaannya `*`. Untuk penggunaan sungguhan, isi dengan origin
frontend saja, dipisahkan koma bila lebih dari satu.

`APP_URL` adalah alamat frontend (bawaan `http://localhost:5173`), dipakai
untuk tautan di email, sitemap, dan RSS.

## Menjalankan di production

Pakai `docker-compose.prod.yml` di atas `docker-compose.yml`. Override ini
menyalakan `APP_ENV=production`, memakai user database sendiri (bukan root),
dan menutup port MySQL dari luar server.

1. Buat `.env` di server, jangan di-commit:

   ```env
   MYSQL_ROOT_PASSWORD=<acak>
   DB_PASSWORD=<acak>
   JWT_SECRET=<hasil: openssl rand -base64 48>
   ADMIN_EMAIL=admin@domainmu.id
   ADMIN_PASSWORD=<minimal 12 karakter>
   CORS_ORIGINS=https://domain-frontend-mu.id
   APP_URL=https://domain-frontend-mu.id
   TRUSTED_PROXIES=<IP reverse proxy, bila ada>
   SMTP_HOST=smtp.penyedia-email.com
   SMTP_PORT=587
   SMTP_USERNAME=<dari penyedia email>
   SMTP_PASSWORD=<dari penyedia email>
   MAIL_FROM=Warta <noreply@domainmu.id>
   ```

   SMTP bisa dari penyedia email transaksional mana pun (Brevo, Mailgun,
   Amazon SES, Resend, dan sejenisnya). Port 465 memakai TLS langsung, port
   lain memakai STARTTLS. Pastikan domain `MAIL_FROM` sudah diatur SPF dan
   DKIM-nya di penyedia itu supaya email tidak masuk spam.

2. Jalankan:

   ```bash
   docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
   ```

Compose menolak jalan bila salah satu nilai wajib kosong. Service juga menolak
menyala dalam mode production bila `JWT_SECRET` masih nilai contoh,
`ADMIN_PASSWORD` kurang dari 12 karakter atau nilai contoh, `DB_PASSWORD`
kosong atau nilai contoh, `CORS_ORIGINS` masih `*`, `APP_URL` bukan https,
atau `SMTP_HOST` kosong.

Image MySQL hanya membuat user dan password saat volume datanya masih kosong.
Bila volume `warta_mysql_data` sudah pernah dipakai dengan pengaturan lama,
buat user `warta` secara manual atau mulai dari volume baru.

Yang tetap perlu disiapkan sendiri: HTTPS di reverse proxy, backup rutin
database dan volume `warta_uploads`, serta alamat backend untuk
`VITE_API_URL` di frontend.

## Catatan

Endpoint versi awal (`/article/...`) sudah diganti seluruhnya oleh `/api/v1`.
Frontend-nya ada di
[warta-frontend](https://github.com/fajarrafsan/warta-frontend) dan sudah
memakai API ini.

Access token sengaja tidak dicek ke database di setiap permintaan supaya
ringan. Akibatnya perubahan role atau penonaktifan akun baru terasa setelah
access token habis (paling lama `ACCESS_TOKEN_TTL`). Karena itu umurnya dibuat
pendek.

Rate limit dan pencegah hitungan dibaca ganda disimpan di memori, jadi bila
service dijalankan beberapa instance, keduanya berlaku per instance; untuk itu
dibutuhkan penyimpanan bersama seperti Redis. Gambar
sampul disimpan di disk lokal; untuk beberapa instance, folder upload harus
dibagi bersama (misalnya volume jaringan) atau diganti penyimpanan objek.

Gambar yang diunggah tapi tidak jadi dipakai artikel tidak dihapus otomatis.

Pencarian memakai `LIKE`, cukup untuk ribuan artikel. Bila datanya tumbuh jauh
lebih besar, langkah berikutnya adalah indeks FULLTEXT atau mesin pencari
terpisah.
