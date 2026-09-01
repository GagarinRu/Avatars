# Avatars — сервис аватарок

![Go](https://img.shields.io/badge/Go-1.25-blue?logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17-orange?logo=postgresql)
![RabbitMQ](https://img.shields.io/badge/RabbitMQ-3.12-orange?logo=rabbitmq)
![MinIO](https://img.shields.io/badge/MinIO-S3-red)
![Docker](https://img.shields.io/badge/Docker-Compose-blue?logo=docker)

## Описание

Avatars — HTTP API для загрузки, просмотра и удаления пользовательских аватарок. Загрузка принимается синхронно, обработка изображения (валидация, ресайз, thumbnail) выполняется асинхронно через **RabbitMQ**. Файлы хранятся в **S3-совместимом MinIO**, метаданные — в **PostgreSQL**.

## Основные функции

**API (`avatars-api`):**
- загрузка аватарки `POST /api/avatars/{user_id}` → `202 Accepted`;
- получение метаданных и presigned URL `GET /api/avatars/{user_id}`;
- удаление `DELETE /api/avatars/{user_id}`;
- health-check `GET /health`, проверка БД `GET /ping`.

**Worker (`avatars-worker`):**
- потребление задач из очереди `avatars.processing`;
- идемпотентная обработка (таблица `processed_messages`);
- retry через TTL+DLX (`avatars.retry.30s` → `avatars.retry.2m` → DLQ);
- QoS `prefetch=1`;
- валидация JPEG/PNG/WebP, ресайз до 512px, thumbnail 128px;
- сохранение в S3: `avatars/{user_id}/original.png`, `avatars/{user_id}/thumb.png`.

## Технологический стек

- **Go** — API и worker.
- **PostgreSQL** — метаданные аватарок и ключи идемпотентности.
- **RabbitMQ** — асинхронная обработка (DLX, retry, DLQ).
- **MinIO** — S3-хранилище файлов.
- **net/http ServeMux** — HTTP router (Go 1.22+).
- **golang-migrate** — миграции схемы БД.
- **aws-sdk-go** — S3 API для MinIO.
- **zap** — структурированное логирование.
- **Docker Compose** — локальная инфраструктура.

## Принцип работы

1. Клиент отправляет `multipart/form-data` с полем `file` на `POST /api/avatars/{user_id}`.
2. API сохраняет исходник в S3 (`staging/...`), записывает метаданные со статусом `pending`, публикует сообщение в RabbitMQ.
3. Worker забирает сообщение, проверяет идемпотентность, обрабатывает изображение и загружает результат в S3.
4. Статус в БД меняется на `ready`; клиент получает presigned URL через `GET`.
5. При ошибке worker отправляет сообщение в retry-очередь (30s, 2m), затем в DLQ.

## Требования

- Go 1.25+
- Docker и Docker Compose

## Инструкция по запуску

### 1. Подготовка

```bash
git clone <repo-url>
cd avatars
cp .env.example .env
```

### 2. Запуск инфраструктуры и приложения

```bash
docker compose up -d --build
```

Сервисы:
- **Веб-UI:** http://localhost:8080/
- API: `http://localhost:8080`
- RabbitMQ UI: `http://localhost:15672` (guest/guest)
- MinIO Console: `http://localhost:9001` (minioadmin/minioadmin)

### 3. Веб-интерфейс

Откройте в браузере: http://localhost:8080/

- укажите `user_id`;
- выберите файл и нажмите «Загрузить»;
- «Обновить» — статус и превью;
- «Удалить» — удаление аватарки.

### 4. Проверка API

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ping
```

### 5. Загрузка аватарки

```bash
curl -X POST -F "file=@avatar.png" http://localhost:8080/api/avatars/user-1
```

Ожидаемый ответ (`202`):

```json
{
  "user_id": "user-1",
  "status": "pending",
  "message_id": "..."
}
```

### 6. Получение результата

```bash
curl http://localhost:8080/api/avatars/user-1
```

После обработки worker:

```json
{
  "user_id": "user-1",
  "status": "ready",
  "content_type": "image/png",
  "original_url": "...",
  "thumbnail_url": "..."
}
```

### 7. Удаление

```bash
curl -X DELETE http://localhost:8080/api/avatars/user-1
```

## Локальный запуск без Docker (API + worker)

```bash
docker compose up -d avatars_db rabbitmq minio minio_init

go run ./cmd/api
go run ./cmd/worker
```

## API

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/avatars/{user_id}` | Загрузка файла (`file`) → `202` |
| GET | `/api/avatars/{user_id}` | Метаданные + URL (если `ready`) |
| DELETE | `/api/avatars/{user_id}` | Удаление → `204` |
| GET | `/health` | Liveness |
| GET | `/ping` | Проверка PostgreSQL |

## Сборка и тесты

```bash
make build
make test
make lint
make cover
```

## Структура проекта

```
cmd/api/           — HTTP API
cmd/worker/        — RabbitMQ consumer
internal/
  config/          — конфигурация
  handler/         — HTTP handlers
  logger/          — логирование
  models/          — доменные модели
  objectstore/     — S3 клиент
  processor/       — обработка изображений
  queue/           — RabbitMQ
  storage/         — PostgreSQL
  worker/          — бизнес-логика worker
  web/             — веб-интерфейс
migrations/        — SQL-миграции
scripts/minio-init/ — init-образ: создание S3 bucket при старте compose
```


## Автор

[Evgeny Kudryashov](https://github.com/GagarinRu)
