# Subscriptions Service — дизайн

REST-сервис для агрегации данных об онлайн-подписках пользователей.

## Стек

- Go, **gin** — HTTP-роутер
- **pgx/v5 (pgxpool)** + чистый SQL — доступ к PostgreSQL
- **golang-migrate** — миграции
- **cleanenv** — конфигурация (yaml + переопределение через env)
- **slog** — логирование (text для local, JSON для prod)
- **swaggo/swag** — Swagger-документация
- **docker compose** — запуск сервиса и БД

## Архитектура

Классическая трёхслойка, слои связаны через интерфейсы:

```
transport/http (gin handlers) → service (бизнес-логика) → repository (pgx, SQL)
```

- `handler` знает только про HTTP: биндинг, коды ответов, JSON-ошибки;
- `service` — валидация бизнес-правил и подсчёт стоимости;
- `repository` — SQL-запросы, скрыт за интерфейсом (мокается в тестах service).

## Структура каталогов

```
api/                      — сгенерированная swagger-спецификация
build/                    — Dockerfile
cmd/subscriptions/        — main.go
docs/                     — документация
env/docker-compose/       — docker-compose.yml
env/migrations/           — SQL-миграции
internal/app/             — сборка зависимостей, graceful shutdown
internal/config/          — конфигурация
internal/logger/          — настройка slog
internal/models/          — доменные модели
internal/repository/      — интерфейс + postgres-реализация
internal/service/         — бизнес-логика
internal/transport/http/  — роутер, хендлеры, middleware
scripts/                  — вспомогательные скрипты
tests/                    — интеграционные тесты
```

## API

| Метод  | Путь                        | Описание                                    |
|--------|-----------------------------|---------------------------------------------|
| POST   | `/api/v1/subscriptions`     | Создать подписку                            |
| GET    | `/api/v1/subscriptions/{id}`| Получить подписку                           |
| PUT    | `/api/v1/subscriptions/{id}`| Обновить подписку                           |
| DELETE | `/api/v1/subscriptions/{id}`| Удалить подписку                            |
| GET    | `/api/v1/subscriptions`     | Список с фильтрами `user_id`, `service_name`, пагинация `limit`/`offset` |
| GET    | `/api/v1/subscriptions/cost`| Суммарная стоимость за период `from`/`to` (`MM-YYYY`), фильтры `user_id`, `service_name` |

Даты передаются строками `MM-YYYY`, в БД хранятся как `date` (первое число месяца).

## Семантика подсчёта стоимости

Для каждой подписки считается число месяцев пересечения её срока действия
с запрошенным периодом и умножается на месячную цену:

```
Период запроса: 01-2025..06-2025
Подписка A: 400₽, 02-2025..04-2025 → 3 × 400 = 1200
Подписка B: 300₽, 05-2025..∞      → 2 × 300 = 600
Итого: 1800
```

Подписка без `end_date` считается активной бессрочно.

## Этапы разработки

1. `stage-1-skeleton` — структура, конфиг, slog, gin-сервер, graceful shutdown, health-check, Makefile, golangci-lint
2. `stage-2-db` — docker-compose (postgres), миграции, модель, repository
3. `stage-3-crudl` — service + CRUDL-хендлеры, валидация, обработка ошибок
4. `stage-4-cost` — ручка агрегации стоимости
5. `stage-5-final` — swagger, Dockerfile, сервис в compose, README
