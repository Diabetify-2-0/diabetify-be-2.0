# Diabetify BE

Backend API service untuk sistem Diabetify.

## Current Phase

Backend ini menjalankan API utama Diabetify dengan:

- PostgreSQL sebagai database aplikasi.
- RabbitMQ untuk job prediksi ML dan counterfactual.
- Redis untuk penyimpanan sementara hasil what-if.
- JWT authentication dan role-based access untuk endpoint tertentu.
- Swagger documentation.
- Health endpoints untuk deployment.

Mode deployment Tugas Akhir yang direkomendasikan adalah `USE_SHARDING=false`
dengan satu PostgreSQL instance. Mode sharding tersedia untuk eksperimen, tetapi
single database lebih sederhana dan cukup untuk jumlah user TA.

## Folder Structure

```text
diabetify-be/
  cmd/
    main.go
    seed/
  database/
  docs/
  internal/
    cache/
    config/
    controllers/
    dto/
    httpx/
    middleware/
    ml/
    models/
    openai/
    repository/
    services/
    utils/
  routes/
  tests/
  docker-compose.yml
  Dockerfile
  Jenkinsfile
  go.mod
```

## Required Environment

Minimum environment untuk local/deployment:

```env
PORT=8080
APP_ENV=dev
JWT_SECRET_KEY=change-me

USE_SHARDING=false
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=diabetify
DB_SSLMODE=disable

RABBITMQ_URL=amqp://admin:password123@localhost:5672/
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0

ALLOWED_ORIGINS=http://localhost:5173,http://localhost:3000
MLOPS_SERVICE_URL=http://localhost:8000
```

Optional integrations:

```env
SMTP_HOST=
SMTP_PORT=
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_SENDER=

GOOGLE_KEY=
GOOGLE_SECRET=
GOOGLE_CALLBACK_URL=

OPENAI_API_KEY=
```

Untuk `APP_ENV=production`, gunakan `JWT_SECRET_KEY` dan kredensial RabbitMQ
non-default.

## Local Setup

Jalankan dependency lokal:

```powershell
docker compose up -d diabetify-db rabbitmq redis
```

Salin env dan jalankan backend:

```powershell
Copy-Item .env.example .env
go run ./cmd
```

Backend akan tersedia di:

- API: `http://localhost:8080`
- Swagger: `http://localhost:8080/swagger/index.html`
- Liveness: `http://localhost:8080/health/live`
- Readiness: `http://localhost:8080/health/ready`

## Docker Deployment

Jalankan backend dan dependency:

```powershell
docker compose up --build -d
```

Compose menyediakan:

- Backend API: `http://localhost:8080`
- PostgreSQL: `localhost:5432`
- RabbitMQ: `amqp://admin:password123@localhost:5672/`
- RabbitMQ UI: `http://localhost:25672`
- Redis: `localhost:6379`

Container backend berjalan sebagai non-root user.

## Worker Integration

Backend memakai RabbitMQ queue berikut:

- Prediction request: `ml.prediction.request`
- Prediction response: `ml.prediction.response`
- Counterfactual request: `ml.cf.request`
- Counterfactual response: `ml.cf.response`

Untuk menjalankan worker `diabetify-ml` dan `diabetify-cf` secara lokal:

```powershell
$env:RABBITMQ_URL = "amqp://admin:password123@localhost:5672/"
```

Counterfactual payload divalidasi ketat di backend sebelum dikirim ke worker.
Payload minimal harus berisi `instance.features` dan `constraints`.

## Health Endpoints

- `GET /health/live`
  Mengecek proses backend hidup.

- `GET /health/ready`
  Mengecek kesiapan dependency utama: database, Redis, prediction worker/RabbitMQ,
  dan counterfactual service/RabbitMQ.

Endpoint debug masih tersedia untuk development:

- `GET /debug/stats`
- `GET /debug/jobs`
- `GET /debug/counterfactual`
- `GET /debug/database` atau `/debug/shards`

Jangan expose endpoint debug ke publik pada deployment final.

## Quality Commands

```powershell
go test ./...
go vet ./...
docker compose config
```

Catatan: `docker compose config` akan menampilkan environment hasil resolusi.
Untuk log publik/CI, gunakan env non-secret atau `.env.example`.

Quality gate juga tersedia di:

- `.github/workflows/quality.yml`
- `Jenkinsfile`

Pipeline menjalankan `go test`, `go vet`, dan `docker compose config`.

## Deployment Notes

Urutan deployment TA yang disarankan:

1. Jalankan PostgreSQL, RabbitMQ, dan Redis.
2. Pastikan `.env` backend sudah berisi konfigurasi dependency.
3. Jalankan `go test ./...` dan `go vet ./...`.
4. Jalankan `docker compose config` untuk validasi compose.
5. Jalankan `docker compose up --build -d`.
6. Cek `/health/live` dan `/health/ready`.
7. Jalankan worker `diabetify-ml` dan `diabetify-cf`.

Untuk scope Tugas Akhir, gunakan:

```env
USE_SHARDING=false
```

Mode ini cukup untuk user sedikit dan lebih mudah diverifikasi saat demo.
