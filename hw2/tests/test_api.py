import os
from concurrent.futures import ThreadPoolExecutor
from datetime import UTC, datetime, timedelta
from pathlib import Path
from uuid import uuid4

import httpx
import jwt
import psycopg
import pytest
import yaml
from jsonschema import RefResolver
from openapi_schema_validator import OAS30Validator
from openapi_spec_validator import validate
from psycopg.rows import dict_row

BASE = os.getenv("API_URL", "http://localhost:8082")
SPEC = yaml.safe_load(
    (Path(__file__).parents[1] / "openapi/marketplace.yaml").read_text()
)


def sql(query, args=()):
    with psycopg.connect(os.environ["DATABASE_URL"], row_factory=dict_row) as db:
        return (
            db.execute(query, args).fetchall()
            if query.lstrip().upper().startswith("SELECT")
            else db.execute(query, args).rowcount
        )


def call(method, path, token=None, body=None, status=200, code=None):
    response = httpx.request(
        method,
        BASE + path,
        json=body,
        headers={"Authorization": f"Bearer {token}"} if token else {},
        timeout=20,
    )
    assert response.status_code == status, response.text
    assert response.headers.get("X-Request-Id")
    data = response.json()
    if code:
        assert data["error_code"] == code, data

    import re

    endpoint = next(
        p
        for p in SPEC["paths"]
        if re.fullmatch(p.replace("{id}", "[^/]+"), path.split("?")[0])
    )
    schema = SPEC["paths"][endpoint][method.lower()]["responses"][str(status)]
    resolver = RefResolver.from_schema(SPEC)
    if "$ref" in schema:
        schema = resolver.resolve(schema["$ref"])[1]
    OAS30Validator(
        schema["content"]["application/json"]["schema"], resolver=resolver
    ).validate(data)
    return data


def account(role="USER"):
    username = "test_" + uuid4().hex
    user = call(
        "POST",
        "/auth/register",
        body={
            "username": username,
            "password": "password123",
            "role": "SELLER" if role == "SELLER" else "USER",
        },
        status=201,
    )
    if role == "ADMIN":
        sql("UPDATE users SET role='ADMIN' WHERE id=%s", (user["id"],))
    tokens = call(
        "POST", "/auth/login", body={"username": username, "password": "password123"}
    )
    return user["id"], tokens


@pytest.fixture
def actors():
    return {role: account(role) for role in ("USER", "SELLER", "ADMIN")}


def token(actors, role="USER"):
    return actors[role][1]["access_token"]


def product(actors, **kw):
    body = {
        "name": "Книга",
        "price": 100,
        "stock": 20,
        "category": "books",
        "status": "ACTIVE",
    } | kw
    return call("POST", "/products", token(actors, "SELLER"), body, status=201)


def items(p, q=1):
    return {"items": [{"product_id": p["id"], "quantity": q}]}


def age(user):
    sql(
        "UPDATE user_operations SET created_at=now()-interval '1 day' WHERE user_id=%s",
        (user,),
    )


def stock(p):
    return sql("SELECT stock FROM products WHERE id=%s", (p["id"],))[0]["stock"]


def promo(actors, **kw):
    now = datetime.now(UTC)
    body = {
        "code": "P_" + uuid4().hex[:10].upper(),
        "discount_type": "PERCENTAGE",
        "discount_value": 20,
        "min_order_amount": 100,
        "max_uses": 1,
        "valid_from": (now - timedelta(days=1)).isoformat(),
        "valid_until": (now + timedelta(days=1)).isoformat(),
    } | kw
    return call("POST", "/promo-codes", token(actors, "SELLER"), body, status=201)


def test_contract_and_auth(actors):
    validate(SPEC)
    call("GET", "/products", status=401, code="TOKEN_INVALID")
    call("GET", "/products", "garbage", status=401, code="TOKEN_INVALID")
    call(
        "POST",
        "/auth/refresh",
        body={"refresh_token": "bad"},
        status=401,
        code="REFRESH_TOKEN_INVALID",
    )
    call(
        "GET",
        "/products",
        actors["USER"][1]["refresh_token"],
        status=401,
        code="TOKEN_INVALID",
    )
    call(
        "POST",
        "/auth/refresh",
        body={"refresh_token": token(actors)},
        status=401,
        code="REFRESH_TOKEN_INVALID",
    )
    refreshed = call(
        "POST",
        "/auth/refresh",
        body={"refresh_token": actors["USER"][1]["refresh_token"]},
    )
    call("GET", "/products", refreshed["access_token"])
    claims = jwt.decode(token(actors), options={"verify_signature": False})
    claims["exp"] = datetime.now(UTC) - timedelta(seconds=1)
    expired = jwt.encode(claims, os.environ["JWT_SECRET"], algorithm="HS256")
    call("GET", "/products", expired, status=401, code="TOKEN_EXPIRED")
    call(
        "POST",
        "/auth/register",
        body={"username": "elevate", "password": "password123", "role": "ADMIN"},
        status=400,
        code="VALIDATION_ERROR",
    )


def test_products_roles_and_validation(actors):
    p = product(actors)
    for body in (
        {"price": 0},
        {"price": 0.001},
        {"price": "1"},
        {"stock": -1},
        {"stock": 1.5},
        {"name": ""},
        {"name": "x" * 256},
        {"description": "x" * 4001},
        {"category": ""},
        {"status": "WRONG"},
    ):
        request = dict(name="x", price=1, stock=1, category="x", status="ACTIVE") | body
        err = call(
            "POST",
            "/products",
            token(actors, "SELLER"),
            request,
            status=400,
            code="VALIDATION_ERROR",
        )
        assert err["details"]["fields"]
    call("GET", "/products?size=0", token(actors), status=400, code="VALIDATION_ERROR")
    call("GET", "/products?page=-1", token(actors), status=400, code="VALIDATION_ERROR")
    call(
        "GET", "/products/not-uuid", token(actors), status=400, code="VALIDATION_ERROR"
    )
    call(
        "GET",
        f"/products/{uuid4()}",
        token(actors),
        status=404,
        code="PRODUCT_NOT_FOUND",
    )
    call(
        "DELETE",
        f"/products/{p['id']}",
        token(actors),
        status=403,
        code="ACCESS_DENIED",
    )
    _, other = account("SELLER")
    call(
        "DELETE",
        f"/products/{p['id']}",
        other["access_token"],
        status=403,
        code="ACCESS_DENIED",
    )
    changed = call(
        "PUT",
        f"/products/{p['id']}",
        token(actors, "SELLER"),
        {
            "name": "Новая",
            "price": 150,
            "stock": 10,
            "category": "unique_" + p["id"],
            "status": "INACTIVE",
        },
    )
    assert (
        changed["created_at"] == p["created_at"]
        and changed["updated_at"] != p["updated_at"]
    )
    page = call(
        "GET",
        f"/products?category=unique_{p['id']}&status=INACTIVE&size=1",
        token(actors),
    )
    assert page["totalElements"] == 1 and page["items"][0]["id"] == p["id"]
    assert (
        call(
            "GET", f"/products?category=unique_{p['id']}&page=1&size=1", token(actors)
        )["items"]
        == []
    )
    call("DELETE", f"/products/{p['id']}", token(actors, "ADMIN"))
    assert (
        sql("SELECT status FROM products WHERE id=%s", (p["id"],))[0]["status"]
        == "ARCHIVED"
    )


def test_order_lifecycle_and_snapshot(actors):
    p = product(actors)
    user = token(actors)
    o = call("POST", "/orders", user, items(p, 2), status=201)
    assert o["total_amount"] == 200 and stock(p) == 18
    call("POST", "/orders", user, items(p), status=429, code="ORDER_LIMIT_EXCEEDED")
    age(actors["USER"][0])
    call("POST", "/orders", user, items(p), status=409, code="ORDER_HAS_ACTIVE")
    call(
        "PUT",
        f"/products/{p['id']}",
        token(actors, "SELLER"),
        {
            "name": "x",
            "price": 999,
            "stock": 18,
            "category": "books",
            "status": "ACTIVE",
        },
    )
    assert call("GET", f"/orders/{o['id']}", user)["total_amount"] == 200
    o = call("PUT", f"/orders/{o['id']}", user, items(p, 3))
    assert o["total_amount"] == 300 and stock(p) == 17
    call(
        "PUT",
        f"/orders/{o['id']}",
        user,
        items(p, 4),
        status=429,
        code="ORDER_LIMIT_EXCEEDED",
    )
    call(
        "POST",
        f"/orders/{o['id']}/status",
        token(actors, "ADMIN"),
        {"status": "PAID"},
        status=409,
        code="INVALID_STATE_TRANSITION",
    )
    for state in ["PAYMENT_PENDING", "PAID", "SHIPPED", "COMPLETED"]:
        call(
            "POST",
            f"/orders/{o['id']}/status",
            token(actors, "ADMIN"),
            {"status": state},
        )
    call(
        "POST",
        f"/orders/{o['id']}/cancel",
        user,
        status=409,
        code="INVALID_STATE_TRANSITION",
    )
    call(
        "PUT",
        f"/orders/{o['id']}",
        user,
        items(p),
        status=409,
        code="INVALID_STATE_TRANSITION",
    )
    assert (
        sql("SELECT count(*) AS n FROM order_items WHERE order_id=%s", (o["id"],))[0][
            "n"
        ]
        == 1
    )


def test_order_failures_and_rollback(actors):
    p = product(actors, stock=2)
    inactive = product(actors, status="INACTIVE")
    user = token(actors)
    call("POST", "/orders", user, items(inactive), status=409, code="PRODUCT_INACTIVE")
    call(
        "POST",
        "/orders",
        user,
        {"items": [{"product_id": str(uuid4()), "quantity": 1}]},
        status=404,
        code="PRODUCT_NOT_FOUND",
    )
    err = call(
        "POST", "/orders", user, items(p, 3), status=409, code="INSUFFICIENT_STOCK"
    )
    assert err["details"]["products"][0]["available"] == 2
    call(
        "POST",
        "/orders",
        user,
        items(p) | {"promo_code": "MISSING"},
        status=422,
        code="PROMO_CODE_INVALID",
    )
    assert stock(p) == 2
    assert (
        sql(
            "SELECT count(*) AS n FROM user_operations WHERE user_id=%s",
            (actors["USER"][0],),
        )[0]["n"]
        == 0
    )
    o = call("POST", "/orders", user, items(p), status=201)
    call(
        "PUT",
        f"/orders/{o['id']}",
        user,
        items(p, 3),
        status=409,
        code="INSUFFICIENT_STOCK",
    )
    assert (
        stock(p) == 1 and call("GET", f"/orders/{o['id']}", user)["total_amount"] == 100
    )
    _, stranger = account()
    for method, suffix, body in [
        ("GET", "", None),
        ("PUT", "", items(p)),
        ("POST", "/cancel", None),
    ]:
        call(
            method,
            f"/orders/{o['id']}" + suffix,
            stranger["access_token"],
            body,
            status=403,
            code="ORDER_OWNERSHIP_VIOLATION",
        )
    call(
        "GET",
        f"/orders/{o['id']}",
        token(actors, "SELLER"),
        status=403,
        code="ACCESS_DENIED",
    )
    call("GET", f"/orders/{uuid4()}", user, status=404, code="ORDER_NOT_FOUND")
    call("POST", f"/orders/{o['id']}/cancel", token(actors, "ADMIN"))
    assert stock(p) == 2
    call(
        "POST",
        f"/orders/{o['id']}/cancel",
        user,
        status=409,
        code="INVALID_STATE_TRANSITION",
    )
    assert stock(p) == 2


@pytest.mark.parametrize(
    "bad",
    [
        [],
        [{"product_id": str(uuid4()), "quantity": 0}],
        [{"product_id": str(uuid4()), "quantity": 1000}],
        [{"product_id": str(uuid4()), "quantity": 1}] * 51,
    ],
)
def test_order_validation(actors, bad):
    call(
        "POST",
        "/orders",
        token(actors),
        {"items": bad},
        status=400,
        code="VALIDATION_ERROR",
    )


def test_promos_recalculation_and_cancel(actors):
    p = product(actors)
    code = promo(actors, min_order_amount=200)
    user = token(actors)
    call(
        "POST",
        "/orders",
        user,
        items(p) | {"promo_code": code["code"]},
        status=422,
        code="PROMO_CODE_MIN_AMOUNT",
    )
    o = call(
        "POST", "/orders", user, items(p, 3) | {"promo_code": code["code"]}, status=201
    )
    assert o["total_amount"] == 240 and o["discount_amount"] == 60
    o = call("PUT", f"/orders/{o['id']}", user, items(p, 2))
    assert (
        o["total_amount"] == 160
    )  # max_uses=1: своё использование не мешает пересчёту.
    age(actors["USER"][0])
    o = call("PUT", f"/orders/{o['id']}", user, items(p))
    assert (
        o["promo_code_id"] is None
        and o["discount_amount"] == 0
        and o["total_amount"] == 100
    )
    assert (
        sql("SELECT current_uses FROM promo_codes WHERE id=%s", (code["id"],))[0][
            "current_uses"
        ]
        == 0
    )
    call("POST", f"/orders/{o['id']}/cancel", user)
    age(actors["USER"][0])
    o = call(
        "POST", "/orders", user, items(p, 2) | {"promo_code": code["code"]}, status=201
    )
    call(
        "POST",
        f"/orders/{o['id']}/status",
        token(actors, "ADMIN"),
        {"status": "PAYMENT_PENDING"},
    )
    call("POST", f"/orders/{o['id']}/cancel", user)
    assert (
        stock(p) == 20
        and sql("SELECT current_uses FROM promo_codes WHERE id=%s", (code["id"],))[0][
            "current_uses"
        ]
        == 0
    )


@pytest.mark.parametrize(
    "kind,value,expected",
    [("PERCENTAGE", 70, 30), ("PERCENTAGE", 71, 100), ("FIXED_AMOUNT", 500, 0)],
)
def test_discount_rules(actors, kind, value, expected):
    p = product(actors)
    code = promo(actors, discount_type=kind, discount_value=value)
    o = call(
        "POST",
        "/orders",
        token(actors),
        items(p) | {"promo_code": code["code"]},
        status=201,
    )
    assert o["total_amount"] == expected


def test_expired_promo_update_rolls_back(actors):
    p = product(actors)
    code = promo(actors)
    o = call(
        "POST",
        "/orders",
        token(actors),
        items(p) | {"promo_code": code["code"]},
        status=201,
    )
    sql("UPDATE promo_codes SET active=false WHERE id=%s", (code["id"],))
    call(
        "PUT",
        f"/orders/{o['id']}",
        token(actors),
        items(p, 2),
        status=422,
        code="PROMO_CODE_INVALID",
    )
    assert stock(p) == 19
    assert call("GET", f"/orders/{o['id']}", token(actors))["total_amount"] == 80


def test_concurrent_stock_and_promo(actors):
    p = product(actors, stock=1)
    buyers = [account()[1]["access_token"] for _ in range(2)]

    def order(t, body):
        return httpx.post(
            BASE + "/orders",
            headers={"Authorization": "Bearer " + t},
            json=body,
            timeout=20,
        )

    with ThreadPoolExecutor(2) as pool:
        responses = list(pool.map(lambda t: order(t, items(p)), buyers))
    assert sorted(r.status_code for r in responses) == [201, 409]
    assert stock(p) == 0
    p = product(actors)
    code = promo(actors)
    buyers = [account()[1]["access_token"] for _ in range(2)]
    with ThreadPoolExecutor(2) as pool:
        responses = list(
            pool.map(
                lambda t: order(t, items(p) | {"promo_code": code["code"]}), buyers
            )
        )
    assert sorted(r.status_code for r in responses) == [201, 422]
    assert stock(p) == 19
    assert (
        sql("SELECT current_uses FROM promo_codes WHERE id=%s", (code["id"],))[0][
            "current_uses"
        ]
        == 1
    )
    buyer = account()[1]["access_token"]
    with ThreadPoolExecutor(2) as pool:
        responses = list(pool.map(lambda _: order(buyer, items(p)), range(2)))
    assert sorted(r.status_code for r in responses) == [201, 429]


def test_catalog_priority_and_full_shortage_list(actors):
    p, q = product(actors, stock=0), product(actors, stock=1)
    body = {
        "items": [
            {"product_id": p["id"], "quantity": 2},
            {"product_id": q["id"], "quantity": 2},
        ]
    }
    err = call(
        "POST", "/orders", token(actors), body, status=409, code="INSUFFICIENT_STOCK"
    )
    assert len(err["details"]["products"]) == 2
    body["items"].append({"product_id": str(uuid4()), "quantity": 1})
    call("POST", "/orders", token(actors), body, status=404, code="PRODUCT_NOT_FOUND")
    assert stock(p) == 0 and stock(q) == 1
    call(
        "POST",
        "/orders",
        token(actors),
        {"items": items(q)["items"] * 2},
        status=400,
        code="VALIDATION_ERROR",
    )
    call(
        "POST",
        "/orders",
        token(actors),
        items(q) | {"promo_code": "bad"},
        status=400,
        code="VALIDATION_ERROR",
    )


@pytest.mark.parametrize("change", ["expired", "future", "inactive", "exhausted"])
def test_invalid_promo_conditions(actors, change):
    p = product(actors)
    now = datetime.now(UTC)
    options = {
        "expired": {
            "valid_from": (now - timedelta(days=2)).isoformat(),
            "valid_until": (now - timedelta(days=1)).isoformat(),
        },
        "future": {
            "valid_from": (now + timedelta(days=1)).isoformat(),
            "valid_until": (now + timedelta(days=2)).isoformat(),
        },
        "inactive": {"active": False},
        "exhausted": {},
    }
    code = promo(actors, **options[change])
    if change == "exhausted":
        sql("UPDATE promo_codes SET current_uses=max_uses WHERE id=%s", (code["id"],))
    call(
        "POST",
        "/orders",
        token(actors),
        items(p) | {"promo_code": code["code"]},
        status=422,
        code="PROMO_CODE_INVALID",
    )
    assert stock(p) == 20


def test_role_matrix_and_admin_order_access(actors):
    p = product(actors)
    body = {"name": "x", "price": 1, "stock": 1, "category": "x", "status": "ACTIVE"}
    call("POST", "/products", token(actors), body, status=403, code="ACCESS_DENIED")
    call(
        "POST",
        "/orders",
        token(actors, "SELLER"),
        items(p),
        status=403,
        code="ACCESS_DENIED",
    )
    call("POST", "/promo-codes", token(actors), {}, status=403, code="ACCESS_DENIED")
    o = call("POST", "/orders", token(actors), items(p), status=201)
    call("GET", f"/orders/{o['id']}", token(actors, "ADMIN"))
    call("PUT", f"/orders/{o['id']}", token(actors, "ADMIN"), items(p, 2))
    call(
        "POST",
        f"/orders/{o['id']}/status",
        token(actors),
        {"status": "PAYMENT_PENDING"},
        status=403,
        code="ACCESS_DENIED",
    )
    call("POST", f"/orders/{o['id']}/cancel", token(actors, "ADMIN"))
    assert stock(p) == 20
