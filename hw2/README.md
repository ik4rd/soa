## Запуск

```bash
docker compose -f hw2/compose.yaml up --build -d --wait
curl -i http://localhost:8082/health
```

Сборка генерирует модели и маршруты из openapi/marketplace.yaml, Swagger доступен на http://localhost:8082/docs, исходный контракт — на `/openapi.json`.

Проверка обновлений и телеметрия Flyway отключены, чтобы миграции не зависели от доступа к внешним сервисам.

```bash
docker compose -f hw2/compose.yaml ps
docker compose -f hw2/compose.yaml logs api migrate
docker compose -f hw2/compose.yaml down
```

## Проверка

```bash
python3 hw2/demo.py
docker compose -f hw2/compose.yaml exec db psql -U marketplace
```

Сценарий регистрирует продавца и покупателя, создаёт товар и заказ, меняет количество, отменяет заказ и архивирует товар.

Полный прогон:

```bash
docker compose -f hw2/compose.yaml --profile test run --build --rm tests
```

## Контракт и код

generate.py запускает `datamodel-code-generator` для Pydantic-моделей и генерирует типизированные маршруты FastAPI по `operationId`. DTO вручную не написаны.

Для локальной генерации:

```bash
python3 -m venv hw2/.venv
hw2/.venv/bin/pip install -r hw2/requirements.txt
PATH="$PWD/hw2/.venv/bin:$PATH" hw2/.venv/bin/python hw2/generate.py
```

## Заказы и конкурентные запросы

Одна операция заказа — одна транзакция.

Сначала проверяется весь каталог, затем все остатки. Ошибка `INSUFFICIENT_STOCK` перечисляет все дефицитные позиции. При обновлении старый резерв учитывается как доступный остаток, изменения сохраняются только после проверки нового состава.

Деньги хранятся в `DECIMAL(12,2)` и вычисляются через `Decimal`. Скидка округляется до копеек по `ROUND_HALF_UP`. При обновлении заказа сохранённые товары сохраняют `price_at_order`, новые получают текущую цену.

Процентная скидка выше 70% даёт полную цену, фиксированная скидка ограничена суммой заказа. При пересчёте собственное использование промокода не считается новым. Если сумма упала ниже минимума, привязка и скидка снимаются, счётчик уменьшается, а истёкший или отключённый промокод возвращает `PROMO_CODE_INVALID` с откатом обновления.

## Доступ и логи

Регистрация разрешает роли `USER` и `SELLER`. Для администратора нужно зарегистрировать пользователя и назначить роль через БД, затем снова войти:

```sql
UPDATE users SET role = 'ADMIN' WHERE username = 'admin';
```

Продавец меняет только свои товары, покупатель работает только со своими заказами, администратор — с любыми.
