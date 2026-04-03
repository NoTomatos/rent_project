# GoRent API

API для сервиса аренды автомобилей с ролевой моделью, JWT-аутентификацией и финансовой аналитикой.

## Технологии

- **Go 1.21** - язык программирования
- **Gin** - веб-фреймворк
- **PostgreSQL 15** - база данных
- **Docker** - контейнеризация
- **JWT** - аутентификация

## Установка и запуск

### Через Docker (рекомендуется)

```bash
git clone <repository-url>
cd gorrent

cp .env.example .env

make docker-up

make migrate-up