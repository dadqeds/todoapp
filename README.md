# todoapp

Список задач в виде Telegram Mini App: Go + PostgreSQL, веб-интерфейс встроен в бинарник.

Сервер слушает два адреса:

| Адрес | Для чего | Вход |
| --- | --- | --- |
| `http://localhost:5050` (`HTTP_ADDR`) | Публичный: сюда смотрит туннель, мини-апп открывается из Telegram | Только с `initData` Telegram |
| `http://localhost:5051` (`AUTH_LOCAL_ADDR`) | Локальный: открыть в браузере на своём компьютере | Без Telegram, от имени `AUTH_LOCAL_TELEGRAM_ID`, права администратора |

Swagger всегда доступен на локальном адресе: `http://localhost:5051/swagger/index.html`.

> Локальный порт нельзя пробрасывать наружу: любой, кто до него достучится, получит права администратора. В `docker-compose.yaml` оба порта привязаны к `127.0.0.1`.

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

## Telegram

1. Создайте бота у [@BotFather](https://t.me/BotFather) и положите токен в `AUTH_TELEGRAM_BOT_TOKEN`.
2. Узнайте свой Telegram id (например, у [@userinfobot](https://t.me/userinfobot)) и укажите его в `AUTH_LOCAL_TELEGRAM_ID` и `AUTH_ADMIN_TELEGRAM_IDS`.
3. Telegram открывает мини-аппы только по HTTPS. Поднимите туннель на публичный адрес, например:
   ```bash
   cloudflared tunnel --url http://localhost:5050
   ```
4. В @BotFather: `/mybots` → бот → *Bot Settings* → *Menu Button* и укажите HTTPS-адрес туннеля.
5. Для приглашений в общие списки там же включите *Configure Mini App* (главный мини-апп бота) с тем же адресом и укажите имя бота в `AUTH_TELEGRAM_BOT_USERNAME`. Ссылка вида `t.me/<бот>?startapp=join_<код>` открывает главный мини-апп, а он добавляет человека в список.

Как это работает: Telegram передаёт мини-аппу строку `initData`, подписанную ключом бота. Фронт отправляет её в каждом запросе в заголовке `Authorization: tma <initData>`, сервер проверяет подпись и срок (`AUTH_INIT_DATA_MAX_AGE`). При первом входе пользователь создаётся автоматически.

Права:
- обычный пользователь видит и меняет свои задачи и задачи общих списков, в которых участвует; остальные для него не существуют (404). Статистика — по задачам, которые он создал;
- в общем списке участники добавляют, отмечают и правят задачи; переименовать, удалить список и управлять приглашениями может только владелец;
- администратор (`AUTH_ADMIN_TELEGRAM_IDS` и локальный адрес) видит всё и управляет пользователями.

Чтобы привязать уже существующего пользователя (и его задачи) к своему Telegram, до первого входа выполните:

```sql
UPDATE todoapp.users SET telegram_id = <ваш id> WHERE id = <id пользователя>;
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
| `db-backup` | Сохранить копию базы в `out/backups/` (хранятся последние `BACKUP_KEEP`, по умолчанию 30) |
| `db-restore file=…` | Восстановить базу из копии (спросит подтверждение) |
| `test`, `vet`, `lint`, `check` | Тесты, `go vet`, golangci-lint, vet + тесты |

## Резервные копии

```bash
make db-backup
make db-restore file=out/backups/todoapp-2026-10-05_19-35-09.dump
```

Копии лежат в `out/backups/` в формате `pg_dump --format=custom`. Восстановление заменяет все текущие данные содержимым копии в одной транзакции: если что-то пойдёт не так, база останется как была.

Делать копию раз в день автоматически (macOS/Linux, `crontab -e`):

```
0 3 * * * cd /path/to/todoapp && make db-backup >> out/backups/cron.log 2>&1
```

## Конфигурация

Все параметры задаются переменными окружения, пример в [.env.example](.env.example).

| Переменная | По умолчанию | Описание |
| --- | --- | --- |
| `HTTP_ADDR` | — (обязательна) | Адрес HTTP-сервера, например `:5050` |
| `HTTP_SHUTDOWN_TIMEOUT` | `30s` | Время на graceful shutdown |
| `HTTP_READ_HEADER_TIMEOUT`, `HTTP_READ_TIMEOUT`, `HTTP_WRITE_TIMEOUT`, `HTTP_IDLE_TIMEOUT` | `5s`, `15s`, `30s`, `60s` | Таймауты `http.Server` |
| `HTTP_CORS_ALLOWED_ORIGINS` | пусто | Origin-ы через запятую. UI отдаётся тем же сервером, поэтому CORS по умолчанию выключен |
| `HTTP_SWAGGER_ENABLED` | `false` | Swagger на публичном адресе (на локальном включён всегда) |
| `POSTGRES_HOST`, `POSTGRES_PORT` | —, `5432` | Адрес БД |
| `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` | — | Учётные данные |
| `POSTGRES_SSLMODE` | `disable` | `sslmode` подключения |
| `POSTGRES_TIMEOUT` | — | Таймаут одной операции с БД |
| `POSTGRES_MAX_CONNS`, `POSTGRES_MIN_CONNS`, `POSTGRES_MAX_CONN_LIFETIME`, `POSTGRES_MAX_CONN_IDLE_TIME` | дефолты pgxpool | Настройки пула |
| `AUTH_TELEGRAM_BOT_TOKEN` | — | Токен бота. Без него публичный API отвечает 401 |
| `AUTH_TELEGRAM_BOT_USERNAME` | пусто | Имя бота без `@` для ссылок-приглашений |
| `AUTH_INIT_DATA_MAX_AGE` | `24h` | Срок жизни `initData` |
| `AUTH_ADMIN_TELEGRAM_IDS` | пусто | Telegram id администраторов через запятую |
| `AUTH_LOCAL_ADDR` | пусто (в compose `:5051`, в `make todoapp-run` `127.0.0.1:5051`) | Локальный адрес без Telegram |
| `AUTH_LOCAL_TELEGRAM_ID` | — (обязателен, если задан `AUTH_LOCAL_ADDR`) | От чьего имени работает локальный адрес |
| `LOGGER_LEVEL` | `DEBUG` | Уровень логирования |
| `LOGGER_FOLDER` | — | Папка для файлов логов |

## API

| Метод | Путь | Описание |
| --- | --- | --- |
| `GET` | `/me` | Текущий пользователь и `is_admin` |
| `GET` | `/users?limit&offset` | Список пользователей (админ) |
| `POST` | `/users` | Создать пользователя (админ) |
| `GET` / `PATCH` | `/users/{id}` | Получить / изменить себя (админ — любого) |
| `DELETE` | `/users/{id}` | Удалить (админ). Пользователь с задачами → `409` |
| `GET` | `/lists` | Мои списки со счётчиками задач; «Личное» создаётся автоматически |
| `POST` | `/lists` | Создать список (`title`, `color`: green, violet, coral, blue, pink, amber) |
| `PATCH` / `DELETE` | `/lists/{id}` | Изменить / удалить список вместе с задачами (только владелец). «Личное» удалить нельзя → `409` |
| `POST` / `DELETE` | `/lists/{id}/invite` | Выдать новую ссылку-приглашение / выключить её (только владелец) |
| `POST` | `/lists/join` | Вступить в список по коду из ссылки |
| `DELETE` | `/lists/{id}/members/{user_id}` | Выйти из списка самому или исключить участника (владелец) |
| `GET` | `/tasks?list_id&user_id&limit&offset` | Задачи: сначала невыполненные по ближайшему сроку (`user_id` только для админа) |
| `POST` | `/tasks` | Создать задачу: `list_id` (по умолчанию «Личное»), `due_at`, `due_all_day` (`author_user_id` только для админа) |
| `GET` / `PATCH` / `DELETE` | `/tasks/{id}` | Получить / изменить / удалить свою задачу |
| `GET` | `/statistics?user_id&from&to` | Статистика (`from`/`to` в формате `YYYY-MM-DD`; `user_id` только для админа) |

Повтор задачи — `repeat`: `{"kind": "daily" | "weekly" | "monthly" | "yearly", "weekdays": [1, 4]}` (дни недели только для `weekly`, 1 = пн). Требует срока. При выполнении повторяющейся задачи создаётся следующая с новым сроком (по часовому поясу пользователя, `timezone` в профиле — мини-апп присылает его с телефона), у выполненной повтор снимается. Месячный повтор помнит день: задача на 31-е в феврале встанет на 28-е, а в марте вернётся на 31-е.

Срок задачи — `due_at` с часовым поясом (RFC 3339). Срок «на весь день» (`due_all_day: true`) передаётся как конец этого дня по времени пользователя. В статистике `tasks_on_time_rate` — доля выполненных задач со сроком, закрытых не позже срока; `lists` — разбивка по спискам.

`limit` по умолчанию 50, максимум 500. `PATCH` поддерживает три состояния поля: не передано — не меняется, значение — обновляется, `null` — очищается (для nullable-полей). Изменения защищены оптимистичной блокировкой по `version`: при конкурентном изменении вернётся `409`.

Все запросы к API на публичном адресе требуют заголовок `Authorization: tma <initData>`, без него — `401`.

Ошибки возвращаются в формате `{"error": "...", "message": "..."}`. Для `5xx` детали ошибки не раскрываются клиенту и пишутся только в лог.

## Структура

```
cmd/todoapp          точка входа, Dockerfile
internal/core        домен, логгер, пул БД, HTTP-инфраструктура
internal/features    users, tasks, statistics, web: transport → service → repository
migrations           SQL-миграции (golang-migrate)
public               фронтенд (index.html + assets/app.css, assets/app.js), встраивается через go:embed
docs                 сгенерированная Swagger-спецификация
```
