# Subscriptions Service

REST-сервис для агрегации данных об онлайн-подписках пользователей.

## Стек

- **Go** + [gin](https://github.com/gin-gonic/gin) — HTTP
- **PostgreSQL** + [pgx/v5](https://github.com/jackc/pgx) — хранение (чистый SQL)
- [golang-migrate](https://github.com/golang-migrate/migrate) — миграции (применяются автоматически при старте)
- [cleanenv](https://github.com/ilyakaznacheev/cleanenv) — конфигурация (yaml + переопределение через env/.env)
- **slog** — структурированное логирование (text для local, JSON для prod)
- [swaggo/swag](https://github.com/swaggo/swag) — swagger-документация
- docker compose — запуск

## Быстрый старт

```bash
make compose-up      # собрать и поднять сервис + postgres
```

Сервис доступен на `http://localhost:8080`:

- Swagger UI — http://localhost:8080/swagger/index.html
- Health-check — http://localhost:8080/health

Остановить: `make compose-down`.

### Локальный запуск без Docker

```bash
cp .env.example .env # при необходимости поправить значения
make compose-up      # либо поднять только postgres любым другим способом
make run
```

## Конфигурация

Базовые значения — в `config.yaml`, любые из них переопределяются переменными
окружения (`.env` подхватывается автоматически): `HTTP_PORT`, `POSTGRES_HOST`,
`POSTGRES_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `APP_ENV` и др.
Полный список — в `.env.example` и `internal/config/config.go`.
Пароль БД в yaml не хранится — только env.

## API

Базовый путь — `/api/v1`. Даты передаются строками в формате `MM-YYYY`.

| Метод  | Путь                  | Описание |
|--------|-----------------------|----------|
| POST   | `/subscriptions`      | Создать подписку |
| GET    | `/subscriptions`      | Список (фильтры `user_id`, `service_name`; `limit`/`offset`) |
| GET    | `/subscriptions/{id}` | Получить по id |
| PUT    | `/subscriptions/{id}` | Полностью обновить |
| DELETE | `/subscriptions/{id}` | Удалить |
| GET    | `/subscriptions/cost` | Суммарная стоимость за период (`from`, `to` обязательны; фильтры `user_id`, `service_name`) |

### Примеры

```bash
# создать подписку
curl -X POST http://localhost:8080/api/v1/subscriptions \
  -H 'Content-Type: application/json' \
  -d '{"service_name":"Yandex Plus","price":400,"user_id":"60601fee-2bf1-4721-ae6f-7636e79a0cba","start_date":"07-2025"}'

# суммарная стоимость за период
curl 'http://localhost:8080/api/v1/subscriptions/cost?from=01-2025&to=12-2025&user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba'
```

### Семантика подсчёта стоимости

Для каждой подписки, пересекающейся с периодом, месячная цена умножается на число
месяцев пересечения (границы включительно). Подписка без `end_date` считается
бессрочной и учитывается до конца запрошенного периода.

```
Период: 01-2025..06-2025
Подписка A: 400₽, 02-2025..04-2025 → 3 × 400 = 1200
Подписка B: 300₽, с 05-2025 (бессрочная) → 2 × 300 = 600
Итого: 1800
```

## Разработка

```bash
make test             # юнит-тесты (-race)
make test-integration # интеграционные тесты (нужен запущенный postgres)
make lint             # golangci-lint
make swag             # перегенерировать swagger в api/
make build            # бинарник в bin/
```

## Структура проекта

```
api/                      — сгенерированная swagger-спецификация
build/                    — Dockerfile
cmd/subscriptions/        — точка входа
docs/                     — дизайн-документ
env/docker-compose/       — docker-compose
env/migrations/           — SQL-миграции
internal/app/             — сборка зависимостей, graceful shutdown
internal/config/          — конфигурация
internal/logger/          — настройка slog
internal/models/          — доменные модели и валидация
internal/repository/      — слой хранения (pgx)
internal/service/         — бизнес-логика
internal/transport/http/  — роутер, хендлеры, DTO, middleware
tests/integration/        — интеграционные тесты
```

Слои связаны через интерфейсы, объявленные на стороне потребителя:
`transport → service → repository`.
