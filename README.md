# Subscription Service

REST API для управления подписками пользователей на различные сервисы.

**Что это?**  
Микросервис для отслеживания онлайн-подписок. Позволяет добавлять, редактировать, удалять подписки и подсчитывать расходы за выбранный период.

## Основные возможности

- CRUD операции над подписками
- Подсчет затрат за период с фильтрацией
- Поиск подписок по цене и названию
- Продление существующих подписок
- Swagger документация из коробки

---

## Быстрый старт

**Что нужно:**

- Docker & Docker Compose  
- Go 1.23+ (если запускать локально)  

**Запуск:**

1. Клонируем репозиторий и переходим в директорию:


```bash git clone <repo>```

```bash cd EffectiveTask```


# Копируем и настраиваем .env:

``cp .env.example .env
``

редактируем .env под себя


**Поднимаем все контейнеры:**

*make up*
Сервис будет доступен по адресу: http://localhost:8080

Swagger UI: http://localhost:8080/swagger/index.html

**Примеры использования**


*Создать подписку:*


```bash
curl -X POST http://localhost:8080/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "service_name": "Spotify Premium",
    "price": 500,
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "start_date": "01-2026",
    "end_date": "12-2026"
  }'```
*Посмотреть все подписки пользователя:*

```bash

curl "http://localhost:8080/subscriptions?user_id=550e8400-e29b-41d4-a716-446655440000"

```
*Посчитать расходы за период:*
```bash

curl "http://localhost:8080/subscriptions/total?user_id=550e8400-e29b-41d4-a716-446655440000&from=01-2026&to=12-2026"
```
*Продлить подписку:*
```bash

curl -X PUT http://localhost:8080/subscriptions/1/extend \
  -H "Content-Type: application/json" \
  -d '{
    "end_date": "12-2027",
    "price": 600
  }'
```
**Структура проекта**
```bash

cmd/app/          # Точка входа
internal/
  handler/        # HTTP handlers
  service/        # Бизнес-логика
  repository/     # Работа с БД
  domain/         # Модели данных
  middleware/     # HTTP middleware
migrations/       # SQL миграции
docs/             # Swagger документация
```
**Архитектура**
Классическая слоеная архитектура: Handler → Service → Repository → PostgreSQL

Каждый слой занимается своей задачей и не вмешивается в логику других слоев.

**Технологии**
Go 1.23 — основной язык

PostgreSQL 16 — база данных

Docker & Docker Compose — контейнеризация

golang-migrate — миграции БД

slog — структурированное логирование

Swagger — документация API

**Полезные команды**
```bash

make up       # Запустить все в Docker
make down     # Остановить контейнеры
make run      # Запустить локально (нужна БД)
make swag     # Обновить Swagger документацию
```


**Переменные окружения**


```bash DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=subscription_db
DB_SSL_MODE=disable

SERVER_PORT=8080
LOG_LEVEL=debug
```

**Особенности**

- Даты хранятся в формате MM-YYYY (месяц-год)

- Цены только в рублях, без копеек

- Подписка без end_date считается активной бессрочно

- Один пользователь не может иметь две активные подписки на один сервис

- Нельзя продлить подписку в прошлое