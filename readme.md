# Avito.Kitchen

MVP backend-сервис для оформления заказов из ресторанов.

Проект реализован на Go и PostgreSQL. Основной API взаимодействует с отдельным сервисом ресторана через HTTP.

## Архитектура

Проект состоит из основного API и отдельного сервиса ресторана.

Основной API отвечает за работу с ресторанами, меню и заказами. Данные хранятся в PostgreSQL.

Restaurant Service запускается отдельно и используется для передачи созданного заказа ресторану. Взаимодействие между сервисами происходит по HTTP.

C4-диаграмма архитектуры находится в `docs/c4.puml`.

Customer Journey Map находится в `docs/cjm.puml`.

OpenAPI-спецификация находится в `docs/openapi.yaml`.

## Основные пользовательские сценарии

### 1. Просмотр ресторанов

Клиент получает список ресторанов:

GET /restaurants

### 2. Просмотр информации о ресторане

GET /restaurants/{id}

### 3. Просмотр меню ресторана

GET /restaurants/{id}/menu

### 4. Создание заказа

Клиент передаёт идентификатор ресторана и список товаров:

POST /orders

Пример запроса:

{
  "restaurant_id": 1,
  "items": [
    {
      "menu_item_id": 1,
      "quantity": 2
    },
    {
      "menu_item_id": 2,
      "quantity": 1
    }
  ]
}

При создании заказа API:

1. Проверяет существование ресторана.
2. Проверяет существование товаров.
3. Проверяет доступность товаров.
4. Получает актуальные цены из базы данных.
5. Рассчитывает итоговую стоимость заказа.
6. Создаёт заказ и его позиции в одной транзакции PostgreSQL.
7. Передаёт заказ в Restaurant Service.

### 5. Получение заказа

GET /orders/{id}

Возвращает информацию о заказе, его позиции, стоимость и текущий статус.

## Модель данных

### restaurants

Таблица ресторанов.

Поля:

- id
- name
- address
- status
- created_at

### menu_items

Таблица позиций меню.

Поля:

- id
- restaurant_id
- name
- description
- price
- available
- created_at

### orders

Таблица заказов.

Поля:

- id
- restaurant_id
- status
- total_price
- created_at

### order_items

Таблица позиций заказа.

Поля:

- id
- order_id
- menu_item_id
- name
- price
- quantity

Название и цена сохраняются непосредственно в order_items. Это необходимо для сохранения истории заказа: если цена или название товара изменятся в меню, уже созданный заказ останется неизменным.

SQL-миграция находится в:

migrations/001_init.sql

## API

## API

Основные эндпоинты:

- `GET /health` — проверка работоспособности API.
- `GET /restaurants` — получение списка ресторанов.
- `GET /restaurants/{id}` — получение информации о ресторане.
- `GET /restaurants/{id}/menu` — получение меню ресторана.
- `POST /orders` — создание заказа.
- `GET /orders/{id}` — получение информации о заказе.

Полное описание API находится в `docs/openapi.yaml`.

## Restaurant Service

Restaurant Service является отдельным сервисом, который запускается в собственном Docker-контейнере.

После создания заказа основной API отправляет запрос:

POST /orders в Restaurant Service.

Restaurant Service принимает информацию о заказе и возвращает статус его принятия.

Таким образом, взаимодействие между основным API и рестораном происходит через HTTP, что позволяет независимо развивать и масштабировать сервисы.

## Запуск проекта

Для запуска проекта необходим Docker Desktop.

Запуск всех сервисов:

docker compose up --build

После запуска доступны:

- Main API: http://localhost:8080
- Restaurant Service: http://localhost:8081
- PostgreSQL: localhost:5432

Проверка работоспособности API:

curl http://localhost:8080/health

Ожидаемый ответ:

{
  "status": "ok"
}

## Проверка API

Получение ресторанов:

curl http://localhost:8080/restaurants

Получение меню:

curl http://localhost:8080/restaurants/1/menu

Создание заказа:

curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "restaurant_id": 1,
    "items": [
      {
        "menu_item_id": 1,
        "quantity": 2
      },
      {
        "menu_item_id": 2,
        "quantity": 1
      }
    ]
  }'

Получение заказа:

curl http://localhost:8080/orders/1

## Структура проекта

avito-kitchen/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── database/
│   │   └── database.go
│   └── repository/
│       ├── restaurant.go
│       ├── menu.go
│       └── order.go
├── migrations/
│   └── 001_init.sql
├── restaurant_service/
│   ├── main.go
│   └── Dockerfile
├── docs/
│   ├── openapi.yaml
│   ├── cjm.puml
│   └── c4.puml
├── Dockerfile
├── docker-compose.yml
├── .golangci.yml
├── go.mod
├── go.sum
└── README.md

## Технологии

- Go
- PostgreSQL
- Docker
- Docker Compose
- HTTP / JSON
- OpenAPI
- PlantUML

## Ограничения MVP

Проект является MVP и намеренно не включает:

- аутентификацию и авторизацию;
- оплату;
- полноценную систему доставки;
- геотрекинг курьера;
- полноценную панель управления рестораном;
- брокер сообщений;
- сложную систему управления статусами заказа.

Взаимодействие с Restaurant Service выполняется синхронно по HTTP.

В текущей реализации заказ сначала сохраняется в PostgreSQL, после чего отправляется в Restaurant Service. Если Restaurant Service недоступен, клиент получает ошибку 502, однако уже созданный заказ остаётся в базе данных.

Для production-решения данное взаимодействие можно улучшить с помощью retry-механизма, очереди сообщений или паттерна Outbox.

## Документация

В директории `docs` находятся:

- `openapi.yaml` — OpenAPI-спецификация API;
- `cjm.puml` — Customer Journey Map;
- `c4.puml` — C4-диаграмма архитектуры.

## Результат

Реализованный MVP позволяет:

1. Получать список ресторанов.
2. Получать информацию о конкретном ресторане.
3. Получать меню ресторана.
4. Создавать заказ.
5. Проверять наличие товаров и актуальные цены.
6. Рассчитывать стоимость заказа.
7. Сохранять заказ в PostgreSQL.
8. Получать созданный заказ.
9. Передавать заказ в отдельный Restaurant Service.
10. Запускать всю систему через Docker Compose.