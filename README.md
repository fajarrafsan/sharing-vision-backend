# Article Service

Pengerjaan Test Backend Sharing Vision, use case post article. Go + MySQL.

## Menjalankan

Butuh Docker. Service dan databasenya sekaligus:

```bash
docker compose up -d --build
```

Tunggu sekitar 20 detik, lalu cek `http://localhost:8080/health/ready`.

Kalau mau jalan langsung dari kode dan hanya databasenya di container:

```bash
cp .env.example .env
go mod tidy
docker compose up -d mysql
go run ./cmd/api
```

Isi `DB_PASSWORD` di `.env` dengan `root` supaya cocok dengan `docker-compose.yml`.

## Migrasi

Pakai golang-migrate, berkas SQL-nya di `migrations/` dan ikut di-embed ke binary.

```bash
go run ./cmd/migrate up
go run ./cmd/migrate down
```

golang-migrate tidak bisa membuat database, hanya mengisinya, jadi
`CREATE DATABASE IF NOT EXISTS article` dijalankan lebih dulu oleh runner-nya.
Karena `AUTO_MIGRATE=true`, migrasi juga jalan otomatis tiap service menyala.

Untuk membuat tabelnya manual tanpa migrate, ada `docs/schema.sql`.

## Endpoint

| Method | URL | |
|---|---|---|
| POST | `/article/` | membuat article |
| GET | `/article/{limit}/{offset}` | daftar article dengan paging |
| GET | `/article/{id}` | satu article |
| POST, PUT, PATCH | `/article/{id}` | mengubah article |
| DELETE | `/article/{id}` | menghapus article |
| POST | `/article/{id}/delete` | hapus, untuk klien yang tidak mendukung DELETE |
| GET | `/health`, `/health/ready` | status service |

```bash
curl -X POST http://localhost:8080/article/ \
  -H "Content-Type: application/json" \
  -d '{"title":"Panduan Membangun REST API Article dengan Golang","content":"...","category":"Teknologi","status":"publish"}'
```

`/health/ready` ikut mengecek koneksi database dan membalas 503 kalau tidak
terjangkau, sedangkan `/health` cukup menandakan prosesnya hidup.

## Validasi

- `title` wajib, 20 sampai 200 karakter
- `content` wajib, minimal 200 karakter
- `category` wajib, 3 sampai 100 karakter
- `status` wajib, salah satu dari `publish`, `draft`, `thrash`

Payload yang tidak lolos dibalas 422 beserta rinciannya:

```json
{
  "message": "validasi gagal",
  "errors": { "title": "title minimal 20 karakter" }
}
```

Status huruf besar tetap diterima dan disimpan sebagai huruf kecil.

## Postman

Import `docs/Article-Service.postman_collection.json` lalu jalankan lewat Run
collection. Urutan requestnya sudah disusun supaya bisa dijalankan sekali jalan
dari atas ke bawah, karena id hasil Create dipakai request berikutnya.

Tanpa membuka Postman:

```bash
npx newman run docs/Article-Service.postman_collection.json
```

## Struktur

```
cmd/api              menyalakan service
cmd/migrate          perintah migrasi
internal/model       struct Article
internal/dto         bentuk request dan response
internal/validation  aturan validasi
internal/repository  query ke tabel posts
internal/service     validasi dan pembatasan paging
internal/handler     handler HTTP
internal/router      daftar rute
internal/response    penulisan response JSON
internal/apperr      error yang membawa status HTTP
internal/config      pembacaan environment
internal/database    koneksi MySQL dan migrasi
```

Alurnya handler ke service ke repository. Routing memakai `net/http` bawaan
Go 1.22 yang sudah bisa mencocokkan method dan membaca `{id}` dari URL, jadi
tidak perlu pustaka router tambahan. Dependensi luar hanya tiga: driver MySQL,
golang-migrate, dan godotenv.

## Konfigurasi

Semua lewat environment, daftar lengkapnya ada di `.env.example`.

## Catatan

Tiga hal yang saya putuskan sendiri karena soalnya kurang konsisten:

Response ikut memuat `id` dan kedua timestamp. Di soal, response hanya berisi
title, content, category, dan status, tapi tanpa `id` klien tidak punya cara
memanggil `/article/<id>` yang juga diminta soal.

`POST /article/<id>` di soal dipakai untuk mengubah sekaligus menghapus. Satu
pasangan URL dan method hanya bisa punya satu arti, jadi POST saya pakai untuk
mengubah, sedangkan hapus lewat POST saya letakkan di `/article/<id>/delete`.

Soal menulis status `Publish | Draft | Thrash` pada tabel kolom, tapi
`"publish"`, `"draft"`, `"thrash"` pada aturan validasi. Keduanya saya terima
dan disimpan sebagai huruf kecil.
