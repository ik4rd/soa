import os
from datetime import UTC, datetime
from decimal import ROUND_HALF_UP, Decimal

from psycopg import sql

from . import auth
from .common import connect, fail

INTERVAL = float(os.getenv("ORDER_INTERVAL_MINUTES", "1"))
CENT = Decimal("0.01")


def insert(db, table, values):
    return db.execute(
        sql.SQL("INSERT INTO {} ({}) VALUES ({}) RETURNING *").format(
            sql.Identifier(table),
            sql.SQL(",").join(map(sql.Identifier, values)),
            sql.SQL(",").join(sql.Placeholder() for _ in values),
        ),
        list(values.values()),
    ).fetchone()


def update(db, table, id, values):
    return db.execute(
        sql.SQL("UPDATE {} SET {} WHERE id=%s RETURNING *").format(
            sql.Identifier(table),
            sql.SQL(",").join(
                sql.SQL("{}=%s").format(sql.Identifier(k)) for k in values
            ),
        ),
        [*values.values(), id],
    ).fetchone()


def register(request, body):
    with connect() as db:
        row = db.execute(
            "INSERT INTO users(username,password_hash,role) VALUES (%s,%s,%s) "
            "ON CONFLICT(username) DO NOTHING RETURNING id,username,role",
            (body.username, auth.password_hash(body.password), body.role),
        ).fetchone()

        if not row:
            fail(409, "USERNAME_TAKEN", "Имя пользователя занято.")

        return row


def login(request, body):
    with connect() as db:
        user = db.execute(
            "SELECT * FROM users WHERE username=%s", (body.username,)
        ).fetchone()

        if not user or not auth.password_matches(body.password, user["password_hash"]):
            fail(401, "TOKEN_INVALID", "Неверный логин или пароль.")

        return auth.tokens(user)


def refresh(request, body):
    data = auth.decode(body.refresh_token, "refresh")
    with connect() as db:
        user = db.execute("SELECT * FROM users WHERE id=%s", (data["sub"],)).fetchone()

        if not user:
            fail(401, "REFRESH_TOKEN_INVALID", "Пользователь не найден.")

        return auth.tokens(user)


def product(db, id):
    row = db.execute("SELECT * FROM products WHERE id=%s FOR UPDATE", (id,)).fetchone()

    if not row:
        fail(404, "PRODUCT_NOT_FOUND", "Товар не найден.")

    return row


def product_owner(request, row):
    if (
            request.state.user["role"] != "ADMIN"
            and str(row["seller_id"]) != request.state.user["sub"]
    ):
        fail(403, "ACCESS_DENIED", "Товар принадлежит другому продавцу.")


def create_product(request, body):
    with connect() as db:
        return insert(
            db, "products", dict(body.model_dump(), seller_id=request.state.user["sub"])
        )


def get_product(request, id):
    with connect() as db:
        return product(db, id)


def list_products(request, page, size, status, category):
    with connect() as db:
        db.execute("SET TRANSACTION ISOLATION LEVEL REPEATABLE READ")
        params = (status, status, category, category)
        where = " WHERE (%s::text IS NULL OR status=%s) AND (%s::text IS NULL OR category=%s)"
        total = db.execute(
            "SELECT count(*) AS n FROM products" + where, params
        ).fetchone()["n"]
        rows = db.execute(
            "SELECT * FROM products"
            + where
            + " ORDER BY created_at,id LIMIT %s OFFSET %s",
            (*params, size, page * size),
        ).fetchall()
        return {"items": rows, "totalElements": total, "page": page, "size": size}


def update_product(request, id, body):
    with connect() as db:
        product_owner(request, product(db, id))
        return update(db, "products", id, body.model_dump())


def delete_product(request, id):
    with connect() as db:
        product_owner(request, product(db, id))
        return update(db, "products", id, {"status": "ARCHIVED"})


def create_promo(request, body):
    if body.valid_until < body.valid_from:
        fail(
            400,
            "VALIDATION_ERROR",
            "Неверный период действия.",
            {"valid_until": "Раньше valid_from."},
        )
    with connect() as db:
        db.execute(
            "SELECT pg_advisory_xact_lock(hashtextextended(%s, 0))", (body.code,)
        )
        if db.execute(
                "SELECT id FROM promo_codes WHERE code=%s", (body.code,)
        ).fetchone():
            fail(409, "PROMO_CODE_EXISTS", "Промокод уже существует.")
        return insert(db, "promo_codes", body.model_dump())


def lock_user(db, user_id):
    db.execute("SELECT id FROM users WHERE id=%s FOR UPDATE", (user_id,)).fetchone()


def rate_limit(db, user_id, operation):
    recent = db.execute(
        "SELECT 1 FROM user_operations WHERE user_id=%s AND operation_type=%s "
        "AND created_at > clock_timestamp() - %s * interval '1 minute' LIMIT 1",
        (user_id, operation, INTERVAL),
    ).fetchone()
    if recent:
        fail(429, "ORDER_LIMIT_EXCEEDED", "Повторите операцию позже.")


def record(db, user_id, operation):
    insert(db, "user_operations", {"user_id": user_id, "operation_type": operation})


def order(db, request, id):
    row = db.execute("SELECT * FROM orders WHERE id=%s", (id,)).fetchone()
    if not row:
        fail(404, "ORDER_NOT_FOUND", "Заказ не найден.")
    if (
            request.state.user["role"] != "ADMIN"
            and str(row["user_id"]) != request.state.user["sub"]
    ):
        fail(
            403, "ORDER_OWNERSHIP_VIOLATION", "Заказ принадлежит другому пользователю."
        )
    lock_user(db, row["user_id"])
    return db.execute("SELECT * FROM orders WHERE id=%s FOR UPDATE", (id,)).fetchone()


def order_response(db, row):
    items = db.execute(
        "SELECT id,product_id,quantity,price_at_order FROM order_items "
        "WHERE order_id=%s ORDER BY product_id",
        (row["id"],),
    ).fetchall()
    return dict(row, items=items)


def get_order(request, id):
    with connect() as db:
        return order_response(db, order(db, request, id))


def state(row, allowed):
    if row["status"] not in allowed:
        fail(
            409,
            "INVALID_STATE_TRANSITION",
            "Операция недоступна в текущем состоянии.",
            {"status": row["status"]},
        )


def reserve(db, items, old=()):
    wanted = {item.product_id: item.quantity for item in items}
    if len(wanted) != len(items):
        fail(
            400,
            "VALIDATION_ERROR",
            "Товар повторяется.",
            {"items": "product_id должны быть уникальны."},
        )
    ids = sorted(set(wanted) | {i["product_id"] for i in old})
    rows = db.execute(
        "SELECT * FROM products WHERE id=ANY(%s) ORDER BY id FOR UPDATE", (ids,)
    ).fetchall()
    products = {p["id"]: p for p in rows}
    for item in old:
        products[item["product_id"]]["stock"] += item["quantity"]
    for id in wanted:
        if id not in products:
            fail(404, "PRODUCT_NOT_FOUND", "Товар не найден.", {"product_id": str(id)})
        if products[id]["status"] != "ACTIVE":
            fail(409, "PRODUCT_INACTIVE", "Товар неактивен.", {"product_id": str(id)})
    shortages = [
        {"product_id": str(id), "requested": q, "available": products[id]["stock"]}
        for id, q in wanted.items()
        if products[id]["stock"] < q
    ]
    if shortages:
        fail(409, "INSUFFICIENT_STOCK", "Недостаточно товара.", {"products": shortages})
    for id in ids:
        update(db, "products", id, {"stock": products[id]["stock"] - wanted.get(id, 0)})
    prices = {i["product_id"]: i["price_at_order"] for i in old}
    return [
        {
            "product_id": id,
            "quantity": q,
            "price_at_order": prices.get(id, products[id]["price"]),
        }
        for id, q in wanted.items()
    ]


def discount(db, promo, subtotal, existing=False):
    now = datetime.now(UTC)
    if (
            not promo
            or not promo["active"]
            or not promo["valid_from"] <= now <= promo["valid_until"]
            or (promo["current_uses"] - int(existing) >= promo["max_uses"])
    ):
        fail(422, "PROMO_CODE_INVALID", "Промокод недействителен.")
    if subtotal < promo["min_order_amount"]:
        if existing:
            db.execute(
                "UPDATE promo_codes SET current_uses=current_uses-1 WHERE id=%s",
                (promo["id"],),
            )
            return None, Decimal(0)
        fail(422, "PROMO_CODE_MIN_AMOUNT", "Сумма меньше минимальной для промокода.")
    if promo["discount_type"] == "PERCENTAGE":
        amount = (
            subtotal * promo["discount_value"] / 100
            if promo["discount_value"] <= 70
            else Decimal(0)
        )
    else:
        amount = min(promo["discount_value"], subtotal)
    return promo["id"], amount.quantize(CENT, rounding=ROUND_HALF_UP)


def save_items(db, id, items):
    db.execute("DELETE FROM order_items WHERE order_id=%s", (id,))
    for item in items:
        insert(db, "order_items", dict(item, order_id=id))


def amount(items):
    subtotal = sum((i["price_at_order"] * i["quantity"] for i in items), Decimal(0))
    if subtotal > Decimal("9999999999.99"):
        fail(
            400,
            "VALIDATION_ERROR",
            "Сумма заказа слишком велика.",
            {"items": "Сумма превышает DECIMAL(12,2)."},
        )
    return subtotal


def create_order(request, body):
    user_id = request.state.user["sub"]
    with connect() as db:
        lock_user(db, user_id)
        rate_limit(db, user_id, "CREATE_ORDER")
        if db.execute(
                "SELECT id FROM orders WHERE user_id=%s AND status IN ('CREATED','PAYMENT_PENDING')",
                (user_id,),
        ).fetchone():
            fail(409, "ORDER_HAS_ACTIVE", "У пользователя уже есть активный заказ.")
        items = reserve(db, body.items)
        subtotal, promo_id, deduction = amount(items), None, Decimal(0)
        if body.promo_code:
            promo = db.execute(
                "SELECT * FROM promo_codes WHERE code=%s FOR UPDATE", (body.promo_code,)
            ).fetchone()
            promo_id, deduction = discount(db, promo, subtotal)
            db.execute(
                "UPDATE promo_codes SET current_uses=current_uses+1 WHERE id=%s",
                (promo_id,),
            )
        row = insert(
            db,
            "orders",
            {
                "user_id": user_id,
                "status": "CREATED",
                "promo_code_id": promo_id,
                "total_amount": subtotal - deduction,
                "discount_amount": deduction,
            },
        )
        save_items(db, row["id"], items)
        record(db, user_id, "CREATE_ORDER")
        return order_response(db, row)


def update_order(request, id, body):
    with connect() as db:
        row = order(db, request, id)
        state(row, ["CREATED"])
        rate_limit(db, row["user_id"], "UPDATE_ORDER")
        old = db.execute(
            "SELECT * FROM order_items WHERE order_id=%s", (id,)
        ).fetchall()
        items = reserve(db, body.items, old)
        subtotal, promo_id, deduction = amount(items), row["promo_code_id"], Decimal(0)
        if promo_id:
            promo = db.execute(
                "SELECT * FROM promo_codes WHERE id=%s FOR UPDATE", (promo_id,)
            ).fetchone()
            promo_id, deduction = discount(db, promo, subtotal, existing=True)
        row = update(
            db,
            "orders",
            id,
            {
                "total_amount": subtotal - deduction,
                "discount_amount": deduction,
                "promo_code_id": promo_id,
            },
        )
        save_items(db, id, items)
        record(db, row["user_id"], "UPDATE_ORDER")
        return order_response(db, row)


def cancel_order(request, id):
    with connect() as db:
        row = order(db, request, id)
        state(row, ["CREATED", "PAYMENT_PENDING"])
        old = db.execute(
            "SELECT * FROM order_items WHERE order_id=%s", (id,)
        ).fetchall()
        reserve(db, [], old)
        if row["promo_code_id"]:
            db.execute(
                "UPDATE promo_codes SET current_uses=current_uses-1 WHERE id=%s",
                (row["promo_code_id"],),
            )
        return order_response(db, update(db, "orders", id, {"status": "CANCELED"}))


def transition_order(request, id, body):
    transitions = {
        "CREATED": "PAYMENT_PENDING",
        "PAYMENT_PENDING": "PAID",
        "PAID": "SHIPPED",
        "SHIPPED": "COMPLETED",
    }
    with connect() as db:
        row = order(db, request, id)
        if transitions.get(row["status"]) != body.status:
            fail(409, "INVALID_STATE_TRANSITION", "Недопустимый переход состояния.")
        return order_response(db, update(db, "orders", id, {"status": body.status}))
