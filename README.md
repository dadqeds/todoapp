# todoapp

REST API для задач и пользователей на Go + PostgreSQL со встроенным веб-интерфейсом.

- UI: `http://localhost:5050/`
- API: `http://localhost:5050/api/v1`
- Swagger: `http://localhost:5050/swagger/index.html` (выключается `HTTP_SWAGGER_ENABLED=false`)

## Быстрый старт

Нужны Docker и (для локального запуска) Go 1.25+.

```bash
cp .env.example .env        # и поменять пароль
make env-up                 # PostgreSQL
make migrate-up             # миграции
make todoapp-deploy         # приложение в Docker
```

Локально без Docker-образа приложения:

```bash
make env-up env-port-forward migrate-up
make todoapp-run
```

## Make-цели

| Цель | Что делает |
| --- | --- |
| `env-up` / `env-down` | Поднять / остановить PostgreSQL |
| `env-port-forward` | Пробросить PostgreSQL на `127.0.0.1:5432` для локального запуска |
| `migrate-up` / `migrate-down` | Применить / откатить миграции |
| `migrate-create seq=name` | Создать новую пару миграций |
| `todoapp-run` | Запустить приложение локально (`go run`) |
| `todoapp-deploy` / `todoapp-undeploy` | Собрать и запустить / остановить контейнер |
| `swagger-gen` | Перегенерировать `docs/` |
| `test`, `vet`, `lint`, `check` | Тесты, `go vet`, golangci-lint, vet + тесты |

## Конфигурация

Все параметры задаются переменными окружения, пример в [.env.example](.env.example).

| Переменная | По умолчанию | Описание |
| --- | --- | --- |
| `HTTP_ADDR` | — (обязательна) | Адрес HTTP-сервера, например `:5050` |
| `HTTP_SHUTDOWN_TIMEOUT` | `30s` | Время на graceful shutdown |
| `HTTP_READ_HEADER_TIMEOUT`, `HTTP_READ_TIMEOUT`, `HTTP_WRITE_TIMEOUT`, `HTTP_IDLE_TIMEOUT` | `5s`, `15s`, `30s`, `60s` | Таймауты `http.Server` |
| `HTTP_CORS_ALLOWED_ORIGINS` | пусто | Origin-ы через запятую. UI отдаётся тем же сервером, поэтому CORS по умолчанию выключен |
| `HTTP_SWAGGER_ENABLED` | `true` | Отдавать ли Swagger UI |
| `POSTGRES_HOST`, `POSTGRES_PORT` | —, `5432` | Адрес БД |
| `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` | — | Учётные данные |
| `POSTGRES_SSLMODE` | `disable` | `sslmode` подключения |
| `POSTGRES_TIMEOUT` | — | Таймаут одной операции с БД |
| `POSTGRES_MAX_CONNS`, `POSTGRES_MIN_CONNS`, `POSTGRES_MAX_CONN_LIFETIME`, `POSTGRES_MAX_CONN_IDLE_TIME` | дефолты pgxpool | Настройки пула |
| `LOGGER_LEVEL` | `DEBUG` | Уровень логирования |
| `LOGGER_FOLDER` | — | Папка для файлов логов |

## API

| Метод | Путь | Описание |
| --- | --- | --- |
| `GET` | `/users?limit&offset` | Список пользователей |
| `POST` | `/users` | Создать пользователя |
| `GET` / `PATCH` / `DELETE` | `/users/{id}` | Получить / изменить / удалить. Удаление пользователя с задачами → `409` |
| `GET` | `/tasks?user_id&limit&offset` | Список задач |
| `POST` | `/tasks` | Создать задачу |
| `GET` / `PATCH` / `DELETE` | `/tasks/{id}` | Получить / изменить / удалить |
| `GET` | `/statistics?user_id&from&to` | Статистика (`from`/`to` в формате `YYYY-MM-DD`) |

`limit` по умолчанию 50, максимум 500. `PATCH` поддерживает три состояния поля: не передано — не меняется, значение — обновляется, `null` — очищается (для nullable-полей). Изменения защищены оптимистичной блокировкой по `version`: при конкурентном изменении вернётся `409`.

Ошибки возвращаются в формате `{"error": "...", "message": "..."}`. Для `5xx` детали ошибки не раскрываются клиенту и пишутся только в лог.

## Структура

```
cmd/todoapp          точка входа, Dockerfile
internal/core        домен, логгер, пул БД, HTTP-инфраструктура
internal/features    users, tasks, statistics, web: transport → service → repository
migrations           SQL-миграции (golang-migrate)
public               фронтенд, встраивается в бинарник через go:embed
docs                 сгенерированная Swagger-спецификация
```
