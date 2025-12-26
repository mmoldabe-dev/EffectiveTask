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

```bash
git clone https://github.com/mmoldabe-dev/EffectiveTask
cd EffectiveTask
```

2. Копируем и настраиваем .env:

```bash
cp .env.example .env
# редактируем .env под себя
```

3. Поднимаем все контейнеры:

```bash
make up
```

Сервис будет доступен по адресу: **http://localhost:8080**

Swagger UI: **http://localhost:8080/swagger/index.html**

---

## Примеры использования
[Ссылка на Postman колекцию](
https://mmoldabe.postman.co/workspace/mmoldabe's-Workspace~1b088653-c10e-4e7c-82ef-cfda7dbe17a5/collection/43867812-4f31fca1-910e-4dd0-b6fb-457da0c99fe1?action=share&creator=43867812&active-environment=43867812-ec333e4b-6914-4534-92ad-cc98cf1a30c8)
### *Создать подписку*

```bash
curl -X POST http://localhost:8080/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "service_name": "Spotify Premium",
    "price": 500,
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "start_date": "01-2026",
    "end_date": "12-2026"
  }'
```

**Ответ:**
```json
{
  "id": 1
}
```

---

### *Получить подписку по ID*

```bash
curl http://localhost:8080/subscriptions/1
```

**Ответ:**
```json
{
  "id": 1,
  "service_name": "Spotify Premium",
  "price": 500,
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "start_date": "01-2026",
  "end_date": "12-2026"
}
```

---

### *Посмотреть все подписки пользователя*

```bash
curl "http://localhost:8080/subscriptions?user_id=550e8400-e29b-41d4-a716-446655440000"
```

**С фильтрами:**
```bash
curl "http://localhost:8080/subscriptions?user_id=550e8400-e29b-41d4-a716-446655440000&service_name=Spotify&min_price=100&max_price=1000&limit=10&offset=0"
```

**Ответ:**
```json
[
  {
    "id": 1,
    "service_name": "Spotify Premium",
    "price": 500,
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "start_date": "01-2026",
    "end_date": "12-2026"
  }
]
```

---

### *Посчитать расходы за период*

```bash
curl "http://localhost:8080/subscriptions/total?user_id=550e8400-e29b-41d4-a716-446655440000&from=01-2026&to=12-2026"
```

**С фильтром по сервису:**
```bash
curl "http://localhost:8080/subscriptions/total?user_id=550e8400-e29b-41d4-a716-446655440000&from=01-2026&to=12-2026&service_name=Spotify Premium"
```

**Ответ:**
```json
{
  "total_cost": 6000,
  "details": [
    "Spotify Premium: 6000"
  ],
  "period": {
    "from": "01-2026",
    "to": "12-2026"
  }
}
```

---

### *Продлить подписку*

```bash
curl -X PUT http://localhost:8080/subscriptions/1/extend \
  -H "Content-Type: application/json" \
  -d '{
    "end_date": "12-2027",
    "price": 600
  }'
```

**Ответ:**
```json
{
  "status": "success"
}
```

---

### *Удалить подписку*

```bash
curl -X DELETE http://localhost:8080/subscriptions/1
```

**Ответ:**
```json
{
  "status": "deleted"
}
```

---

## Структура проекта

```
├── cmd/app/          # Точка входа
├── internal/
│   ├── handler/      # HTTP handlers
│   ├── service/      # Бизнес-логика
│   ├── repository/   # Работа с БД
│   ├── domain/       # Модели данных
│   └── middleware/   # HTTP middleware
├── migrations/       # SQL миграции
└── docs/             # Swagger документация
```

---

## Архитектура

Классическая слоеная архитектура:

**Handler → Service → Repository → PostgreSQL**

Каждый слой занимается своей задачей и не вмешивается в логику других слоев.

---

## Технологии

- **Go 1.23** — основной язык
- **PostgreSQL 16** — база данных
- **Docker & Docker Compose** — контейнеризация
- **golang-migrate** — миграции БД
- **slog** — структурированное логирование
- **Swagger** — документация API

---

## Полезные команды

```bash
make up       # Запустить все в Docker
make down     # Остановить контейнеры
make run      # Запустить локально (нужна БД)
make swag     # Обновить Swagger документацию
```

---

## Переменные окружения

Основные настройки в `.env`:

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=subscription_db
DB_SSL_MODE=disable

SERVER_PORT=8080
LOG_LEVEL=debug
```

---

## API Endpoints

| Метод | Endpoint | Описание |
|-------|----------|----------|
| POST | `/subscriptions` | Создать подписку |
| GET | `/subscriptions/{id}` | Получить подписку по ID |
| DELETE | `/subscriptions/{id}` | Удалить подписку |
| GET | `/subscriptions` | Список подписок с фильтрами |
| GET | `/subscriptions/total` | Посчитать расходы за период |
| PUT | `/subscriptions/{id}/extend` | Продлить подписку |

---

## Особенности

- Даты хранятся в формате **MM-YYYY** (месяц-год)
- Цены только в рублях, без копеек
- Подписка без `end_date` считается активной бессрочно
- Один пользователь не может иметь две активные подписки на один сервис
- Нельзя продлить подписку в прошлое
- При расчете расходов за будущий период выдается предупреждение

---

### P.S. : При разработке статистики я реализовал комбинированный подход. Система считает расходы на основе текущих данных, но также поддерживает расчет на будущие периоды, если подписки активны. Просто не мог опредлиться как правильнее будет