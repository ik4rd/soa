import json
import os
from urllib.request import Request, urlopen
from uuid import uuid4

base = os.getenv("API_URL", "http://localhost:8082")
suffix = uuid4().hex[:8]


def call(method, path, body=None, token=None):
    headers = {"Content-Type": "application/json"}

    if token:
        headers["Authorization"] = "Bearer " + token

    data = json.dumps(body).encode() if body is not None else None

    with urlopen(Request(base + path, data, headers, method=method)) as response:
        result = json.load(response)
        print(method, path, response.status, response.headers["X-Request-Id"])
        if not path.startswith("/auth/"):
            print(json.dumps(result, ensure_ascii=False, indent=2))
        return result


def login(role):
    body = {"username": role.lower() + "_" + suffix, "password": "demo-password"}
    call("POST", "/auth/register", dict(body, role=role))
    return call("POST", "/auth/login", body)["access_token"]


seller, buyer = login("SELLER"), login("USER")

p = call(
    "POST",
    "/products",
    {
        "name": "Книга",
        "price": 250,
        "stock": 5,
        "category": "books",
        "status": "ACTIVE",
    },
    seller,
)

o = call("POST", "/orders", {"items": [{"product_id": p["id"], "quantity": 2}]}, buyer)

call(
    "PUT",
    "/orders/" + o["id"],
    {"items": [{"product_id": p["id"], "quantity": 3}]},
    buyer,
)
call("POST", "/orders/" + o["id"] + "/cancel", token=buyer)
call("DELETE", "/products/" + p["id"], token=seller)

print("\nSELECT * FROM orders WHERE id = {!r};".format(o["id"]))
print("SELECT * FROM order_items WHERE order_id = {!r};".format(o["id"]))
print("SELECT * FROM products WHERE id = {!r};".format(p["id"]))
print("SELECT * FROM user_operations WHERE user_id = {!r};".format(o["user_id"]))
