.PHONY: run build test test-coverage migrate-up migrate-down docker-up docker-down clean deps swagger seed help

APP_NAME=gorrent

run:
	go run cmd/api/main.go

build:
	go build -o bin/$(APP_NAME) cmd/api/main.go

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

migrate-up:
	migrate -path internal/database/migrations -database "postgresql://gorrent_user:gorrent_password@localhost:5432/gorrent_db?sslmode=disable" up

migrate-down:
	migrate -path internal/database/migrations -database "postgresql://gorrent_user:gorrent_password@localhost:5432/gorrent_db?sslmode=disable" down

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-restart:
	docker-compose down
	docker-compose up -d

clean:
	rm -rf bin/
	rm -rf coverage.out coverage.html
	go clean

deps:
	go mod download
	go mod tidy

swagger:
	swag init -g cmd/api/main.go

seed:
	go run scripts/seed.go

reset-db:
	make docker-down
	make docker-up
	sleep 5
	make migrate-up

help:
	@echo "Доступные команды:"
	@echo ""
	@echo "Запуск:"
	@echo "  make run           - Запустить приложение"
	@echo "  make build         - Собрать бинарный файл"
	@echo ""
	@echo "Тестирование:"
	@echo "  make test          - Запустить тесты"
	@echo "  make test-coverage - Запустить тесты с покрытием"
	@echo ""
	@echo "База данных:"
	@echo "  make migrate-up    - Выполнить миграции"
	@echo "  make migrate-down  - Откатить миграции"
	@echo "  make reset-db      - Сбросить БД"
	@echo "  make seed          - Заполнить тестовыми данными"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-up     - Запустить контейнеры"
	@echo "  make docker-down   - Остановить контейнеры"
	@echo "  make docker-restart- Перезапустить контейнеры"
	@echo ""
	@echo "Утилиты:"
	@echo "  make deps          - Установить зависимости"
	@echo "  make swagger       - Сгенерировать Swagger"
	@echo "  make clean         - Очистить бинарные файлы"
	@echo "  make help          - Показать эту помощь"