# Flight Booking

Два сервиса:

1. Booking предоставляет REST API
2. Flight — gRPC API

У каждого своя PostgreSQL, Flight кеширует ответы в Redis с репликой и тремя Sentinel.

## Запуск

```sh
docker-compose up --build
```

REST доступен на `localhost:8080`, gRPC — на `localhost:50051`.

В миграциях есть демонстрационные рейсы SVO -> LED на 1 и 2 декабря 2026 года. Дата фильтруется в UTC. Все суммы — целые копейки RUB.

```sh
curl 'http://localhost:8080/flights?origin=SVO&destination=LED&date=2026-12-01'
curl http://localhost:8080/flights/11111111-1111-4111-8111-111111111111
curl -X POST http://localhost:8080/bookings \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"user-1","flight_id":"11111111-1111-4111-8111-111111111111","passenger_name":"Иван Иванов","passenger_email":"ivan@example.org","seat_count":2}'
curl 'http://localhost:8080/bookings?user_id=user-1'
```

Рейсы сериализуются по Protobuf JSON: 
- `Timestamp` — строка RFC 3339, 
- статусы — имена enum, 
- `price_minor` типа int64 — строка.

Ответы бронирований используют обычный JSON, `total_price_minor` — число. Поиск возвращает объект `{"flights": [...]}`, список бронирований — массив. Ошибки имеют вид `{"error":"..."}`.

## Проверки

```sh
make generate
make check
make integration
```

## Отказоустойчивость вручную

Остановка Flight:

```sh
docker compose stop flight
```

Проверка Redis:

```sh
docker compose exec sentinel-1 redis-cli -p 26379 SENTINEL get-master-addr-by-name flights
docker compose stop redis-master
docker compose exec sentinel-1 redis-cli -p 26379 SENTINEL get-master-addr-by-name flights
docker compose exec redis-replica redis-cli ROLE
```

## Настройки

| Переменная | По умолчанию | Назначение |
| --- | --- | --- |
| `GRPC_API_KEY` | `local-development-key` в Compose | Ключ в metadata `x-api-key`; обязателен для обоих приложений |
| `CACHE_TTL` | `5m` | TTL каждого значения; допустимо 5–10 минут |
| `RPC_TIMEOUT` | `2s` | Дедлайн отдельной попытки, ограниченный дедлайном вызывающего |
| `RETRY_MAX_ATTEMPTS` | `3` | Общее число попыток, от 1 до 3 |
| `RETRY_INITIAL_BACKOFF` | `100ms` | Задержки перед повторами: 100ms, 200ms |
| `CB_FAILURE_THRESHOLD` | `5` | Подряд неуспешных логических RPC в окне |
| `CB_WINDOW` | `30s` | Окно сброса счётчика ошибок |
| `CB_OPEN_TIMEOUT` | `15s` | Пауза перед единственным пробным RPC |

[Схема данных, гарантии и соответствие требованиям](docs/design.md)
