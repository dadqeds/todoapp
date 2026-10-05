-include .env
export

PROJECT_ROOT := $(shell pwd)
export PROJECT_ROOT

.PHONY: env-up env-down env-cleanup env-port-forward env-port-close \
	migrate-create migrate-up migrate-down migrate-action \
	logs-cleanup todoapp-run todoapp-deploy todoapp-undeploy \
	swagger-gen test vet lint check ps db-backup db-restore

env-up:
	@docker compose up -d todoapp-postgres

env-down:
	@docker compose down todoapp-postgres

env-cleanup:
	@read -p "Очистить все volume файлы окружения? Опасность утери данных. [y/N]: " ans; \
	if [ "$$ans" = "y" ]; then \
		docker compose down todoapp-postgres port-forwarder && \
		rm -rf ${PROJECT_ROOT}/out/pgdata && \
		echo "Файлы окружения очищены"; \
	else \
		echo "Очистка окружения отменена"; \
	fi

env-port-forward:
	@docker compose up -d port-forwarder

env-port-close:
	@docker compose down port-forwarder

migrate-create:
	@if [ -z "$(seq)" ]; then \
		echo "Отсутствует необходимый параметр seq. Пример: make migrate-create seq=init"; \
		exit 1; \
	fi; \
	docker compose run --rm todoapp-postgres-migrate \
		create \
		-ext sql \
		-dir /migrations \
		-seq "$(seq)"

migrate-up:
	@$(MAKE) --no-print-directory migrate-action action=up

migrate-down:
	@$(MAKE) --no-print-directory migrate-action action=down

migrate-action:
	@if [ -z "$(action)" ]; then \
		echo "Отсутствует необходимый параметр action. Пример: make migrate-action action=up"; \
		exit 1; \
	fi; \
	docker compose run --rm todoapp-postgres-migrate \
		-path /migrations \
		-database "postgres://$${POSTGRES_USER}:$${POSTGRES_PASSWORD}@todoapp-postgres:5432/$${POSTGRES_DB}?sslmode=disable" \
		"$(action)"

BACKUP_DIR := ${PROJECT_ROOT}/out/backups
# Сколько последних копий хранить; старые удаляются после нового бэкапа.
BACKUP_KEEP ?= 30

db-backup:
	@mkdir -p ${BACKUP_DIR}
	@file=${BACKUP_DIR}/todoapp-$$(date +%Y-%m-%d_%H-%M-%S).dump; \
	docker compose exec -T todoapp-postgres \
		pg_dump -U "$${POSTGRES_USER}" -d "$${POSTGRES_DB}" --format=custom > "$$file" && \
	echo "Копия сохранена: $$file ($$(du -h "$$file" | cut -f1))" || \
	{ rm -f "$$file"; echo "Не удалось сделать копию"; exit 1; }
	@ls -1t ${BACKUP_DIR}/todoapp-*.dump 2>/dev/null | tail -n +$$(( ${BACKUP_KEEP} + 1 )) | xargs -r rm -f

db-restore:
	@if [ -z "$(file)" ]; then \
		echo "Укажите файл копии. Пример: make db-restore file=out/backups/todoapp-2026-10-05_19-30-00.dump"; \
		echo "Доступные копии:"; ls -1t ${BACKUP_DIR}/todoapp-*.dump 2>/dev/null | head -5; \
		exit 1; \
	fi; \
	if [ ! -f "$(file)" ]; then echo "Файл не найден: $(file)"; exit 1; fi; \
	read -p "Заменить текущие данные базы копией $(file)? Текущие данные будут потеряны. [y/N]: " ans; \
	if [ "$$ans" = "y" ]; then \
		docker compose exec -T todoapp-postgres \
			pg_restore -U "$${POSTGRES_USER}" -d "$${POSTGRES_DB}" --clean --if-exists --no-owner --single-transaction < "$(file)" && \
		echo "База восстановлена из $(file)"; \
	else \
		echo "Восстановление отменено"; \
	fi

logs-cleanup:
	@read -p "Очистить все log файлы? Опасность утери логов. [y/N]: " ans; \
	if [ "$$ans" = "y" ]; then \
		rm -rf ${PROJECT_ROOT}/out/logs && \
		echo "Файлы логов очищены"; \
	else \
		echo "Очистка логов отменена"; \
	fi

todoapp-run:
	@LOGGER_FOLDER=${PROJECT_ROOT}/out/logs \
	POSTGRES_HOST=localhost \
	AUTH_LOCAL_ADDR=$${AUTH_LOCAL_ADDR:-127.0.0.1:5051} \
	go run ${PROJECT_ROOT}/cmd/todoapp

todoapp-deploy:
	@docker compose up -d --build todoapp

todoapp-undeploy:
	@docker compose down todoapp

swagger-gen:
	@docker compose run --rm swagger \
		init \
		-g cmd/todoapp/main.go \
		-o docs \
		--parseInternal \
		--parseDependency

test:
	@go test -race ./...

vet:
	@go vet ./...

lint:
	@golangci-lint run ./...

check: vet test

ps:
	@docker compose ps
