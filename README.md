# Diabetify BE

Backend API service for Diabetify.

## Runtime
The backend owns the local infrastructure for full-app development:

- PostgreSQL for application data
- RabbitMQ for ML prediction and counterfactual jobs
- Redis for short-lived what-if results

ML and CF run as separate workers and connect to this backend RabbitMQ broker.

## Local Run
Run infrastructure first:

```powershell
docker compose up -d diabetify-db rabbitmq redis
```

Then run the backend on the host:

```powershell
Copy-Item .env.example .env
go run ./cmd
```

## Docker
Run backend and infrastructure together:

```powershell
docker compose up --build
```

This exposes:

- Backend API: `http://localhost:8080`
- RabbitMQ: `amqp://admin:password123@localhost:5672/`
- RabbitMQ UI: `http://localhost:15672`
- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`

## Worker Integration
Run `diabetify-ml` and `diabetify-cf` workers with:

```powershell
$env:RABBITMQ_URL = "amqp://admin:password123@localhost:5672/"
```

Prediction jobs use `ml.prediction.request` and `ml.prediction.response`.
Counterfactual jobs use `ml.cf.request` and `ml.cf.response`.

## Quality Checks
```powershell
go test ./...
go vet ./...
docker compose config
```
