# Article Service

Microservice untuk use case *post article* pada Test Backend Sharing Vision.
Ditulis dengan Go dan MySQL, dengan pembagian lapisan handler → service → repository.

## Kebutuhan

- Go 1.22 atau lebih baru
- MySQL 8. Kalau tidak ada di lokal, `docker compose up -d` sudah menyiapkannya.

## Menjalankan

Cara paling singkat, service dan database sekaligus:

```bash
docker compose up -d --build
```

Compose menunggu MySQL sehat lebih dulu, baru menyalakan service. Setelah itu
`http://localhost:8080` sudah siap dipakai.

Service ini tidak menyimpan state di memori, jadi beberapa instance bisa
dijalankan berdampingan di belakang load balancer. Ubah dulu baris `ports` pada
service `api` di `docker-compose.yml` menjadi rentang `"8080-8082:8080"`, lalu:

```bash
docker compose up -d --scale api=3
```

Ketiganya memakai database yang sama dan tetap mendengarkan port 8080 di dalam
container masing-masing; yang berbeda hanya port di host.

Kalau ingin menjalankan service langsung dari sumber dan hanya databasenya yang
di container:

```bash
cp .env.example .env     # lalu isi DB_PASSWORD dengan root
go mod tidy
docker compose up -d mysql
go run ./cmd/api
```

Kredensial bawaan pada `.env.example` cocok dengan `docker-compose.yml`, kecuali
`DB_PASSWORD` yang perlu diisi `root`. Kalau memakai MySQL sendiri, sesuaikan isinya.

## Migrasi

Migrasi memakai golang-migrate. Berkas SQL-nya ada di `migrations/` dan ikut
ditempel ke binary lewat `go:embed`, jadi migrasi tetap jalan walau foldernya
tidak ikut disalin ke server.

```bash
go run ./cmd/migrate up        # buat database article + tabel posts
go run ./cmd/migrate down      # kembalikan seluruh migrasi
go run ./cmd/migrate version   # lihat versi yang aktif
```

golang-migrate tidak bisa membuat database, hanya isinya, jadi
`CREATE DATABASE IF NOT EXISTS article` dijalankan lebih dulu oleh runner-nya.

Karena `AUTO_MIGRATE=true`, migrasi juga jalan otomatis setiap `cmd/api` dinyalakan.
Berkas yang sama tetap kompatibel dengan CLI resmi golang-migrate:

```bash
migrate -path migrations -database "mysql://root:root@tcp(127.0.0.1:3306)/article" up
```

Untuk membuat tabelnya secara manual tanpa migrate, tersedia `docs/schema.sql`.

## Tabel posts

| Kolom | Tipe | Catatan |
|---|---|---|
| `id` | `INT` | auto increment, primary key |
| `title` | `VARCHAR(200)` | |
| `content` | `TEXT` | |
| `category` | `VARCHAR(100)` | |
| `created_date` | `TIMESTAMP` | diisi saat insert |
| `updated_date` | `TIMESTAMP` | diperbarui saat update |
| `status` | `VARCHAR(100)` | `publish`, `draft`, atau `thrash` |

## Endpoint

| URL | Method | Keterangan |
|---|---|---|
| `/health`, `/health/live` | GET | proses hidup atau tidak |
| `/health/ready` | GET | siap menerima trafik, ikut mengecek database |
| `/article/` | POST | membuat article baru |
| `/article/{limit}/{offset}` | GET | daftar article dengan paging |
| `/article/{id}` | GET | satu article |
| `/article/{id}` | POST, PUT, PATCH | mengubah article |
| `/article/{id}` | DELETE | menghapus article |
| `/article/{id}/delete` | POST | hapus untuk klien yang hanya mendukung POST |

Spesifikasi menuliskan `POST /article/{id}` untuk dua keperluan sekaligus, yaitu
mengubah dan menghapus. Satu pasangan URL dan method hanya bisa punya satu arti,
jadi POST dipakai untuk mengubah, sementara hapus lewat POST diberi jalur sendiri
di `/article/{id}/delete`.

Contoh membuat article:

```bash
curl -X POST http://localhost:8080/article/ \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Panduan Membangun REST API Article dengan Golang",
    "content": "Isi article minimal 200 karakter ...",
    "category": "Teknologi",
    "status": "publish"
  }'
```

Response-nya berisi article yang tersimpan beserta id dan kedua timestamp-nya:

```json
{
  "id": 1,
  "title": "Panduan Membangun REST API Article dengan Golang",
  "content": "Isi article minimal 200 karakter ...",
  "category": "Teknologi",
  "status": "publish",
  "created_date": "2023-08-21T10:15:00+07:00",
  "updated_date": "2023-08-21T10:15:00+07:00"
}
```

Endpoint daftar mengembalikan array dengan bentuk yang sama, dan hapus
mengembalikan `{}`.

Pemisahan liveness dan readiness mengikuti kebiasaan orchestrator. `/health/live`
sengaja tidak menyentuh database, karena database yang sedang mati bukan alasan
untuk me-restart service. `/health/ready` melakukan ping ke database dan membalas
`503` bila tidak terjangkau, supaya instance itu dikeluarkan dari load balancer
sampai koneksinya pulih.

## Validasi

Dijalankan sebelum membuat maupun mengubah article:

- `title` wajib diisi, 20 sampai 200 karakter
- `content` wajib diisi, minimal 200 karakter
- `category` wajib diisi, 3 sampai 100 karakter
- `status` wajib diisi, salah satu dari `publish`, `draft`, `thrash`

Status diseragamkan ke huruf kecil, jadi `Publish` dari klien tersimpan sebagai
`publish`. Payload yang tidak lolos dibalas `422` dengan rincian per field:

```json
{
  "message": "validasi gagal",
  "errors": {
    "title": "title minimal 20 karakter",
    "content": "content minimal 200 karakter",
    "category": "category minimal 3 karakter",
    "status": "status harus salah satu dari: publish, draft, thrash"
  }
}
```

Kode status lain yang dipakai: `201` saat article dibuat, `400` untuk parameter
URL atau body yang tidak bisa dibaca, `404` untuk id yang tidak ada, `405` untuk
method yang tidak tersedia, dan `500` untuk kesalahan tak terduga.

## Postman

`docs/Article-Service.postman_collection.json` berisi seluruh endpoint beserta satu
contoh payload yang sengaja dibuat gagal validasi. Variable `base_url` bawaannya
`http://localhost:8080`, dan `article_id` terisi otomatis dari response request
Create Article.

## Struktur

```
cmd/api                 menyalakan service
cmd/migrate             perintah migrasi dari terminal
internal/model          struct Article, wakil satu baris tabel posts
internal/dto            bentuk request dan response JSON
internal/validation     aturan validasi soal nomor 4
internal/repository     perintah SQL ke tabel posts
internal/service        aturan bisnis: validasi dan pembatasan paging
internal/handler        membaca request HTTP, memanggil service
internal/router         daftar rute
internal/response       penulisan response JSON
internal/apperr         error yang membawa status HTTP
internal/config         membaca environment dan menyusun alamat koneksi
internal/database       koneksi MySQL dan migrasi
migrations              berkas SQL migrasi
docs                    Postman collection dan skema manual
```

Alurnya satu arah: `handler` → `service` → `repository`. Handler hanya mengurus
HTTP, service memegang validasi dan aturan paging, repository memegang SQL.
Karena `repository` dan `service` berupa interface, keduanya bisa diganti tiruan
saat menulis unit test tanpa menyentuh database sungguhan.

Isi tiap berkas sengaja dibuat lugas. Routing memakai `net/http` bawaan Go 1.22,
yang sudah bisa mencocokkan method dan mengambil potongan URL seperti `{id}`,
jadi tidak perlu pustaka router tambahan. Validasi ditulis dengan percabangan
biasa, tanpa pustaka validator. Dependensi luar hanya tiga: driver MySQL,
golang-migrate, dan godotenv.

## Konfigurasi

| Variable | Bawaan | Keterangan |
|---|---|---|
| `APP_PORT` | `8080` | port HTTP |
| `AUTO_MIGRATE` | `true` | jalankan migrasi saat service menyala |
| `DB_HOST` | `127.0.0.1` | |
| `DB_PORT` | `3306` | |
| `DB_USER` | `root` | |
| `DB_PASSWORD` | kosong | |
| `DB_NAME` | `article` | |
| `DEFAULT_LIMIT` | `10` | dipakai bila limit yang diminta `<= 0` |
| `MAX_LIMIT` | `100` | batas atas limit |
