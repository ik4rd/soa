import json
import re
import time
from datetime import UTC, datetime
from pathlib import Path
from uuid import uuid4

import yaml
from fastapi import FastAPI, Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse
from starlette.exceptions import HTTPException

from . import auth
from .common import APIError, connect, fail
from .generated.models import Error
from .generated.routes import router

spec = yaml.safe_load(
    (Path(__file__).parents[1] / "openapi/marketplace.yaml").read_text()
)
app = FastAPI(title="Marketplace")
app.openapi = lambda: spec
app.include_router(router)
permissions = [
    (
        method.upper(),
        re.compile("^" + path.replace("{id}", "[^/]+") + "$"),
        op["x-roles"],
    )
    for path, methods in spec["paths"].items()
    for method, op in methods.items()
]


def error_response(status, body):
    return JSONResponse(
        Error.model_validate(body).model_dump(mode="json"), status_code=status
    )


@app.exception_handler(APIError)
async def api_error(request, exc):
    return error_response(exc.status, exc.body)


@app.exception_handler(RequestValidationError)
async def validation_error(request, exc):
    errors = [
        {"field": ".".join(map(str, e["loc"])), "message": e["msg"]}
        for e in exc.errors()
    ]

    return error_response(
        400,
        {
            "error_code": "VALIDATION_ERROR",
            "message": "Ошибка входных данных.",
            "details": {"fields": errors},
        },
    )


@app.exception_handler(HTTPException)
async def http_error(request, exc):
    code = {404: "NOT_FOUND", 405: "METHOD_NOT_ALLOWED"}.get(
        exc.status_code, "VALIDATION_ERROR"
    )

    return error_response(
        exc.status_code, {"error_code": code, "message": str(exc.detail)}
    )


def redact(value):
    if isinstance(value, dict):
        return {
            key: "***"
            if any(s in key.lower() for s in ("password", "token", "secret"))
            else redact(v)
            for key, v in value.items()
        }

    if isinstance(value, list):
        return [redact(v) for v in value]

    return value


@app.middleware("http")
async def audit(request: Request, call_next):
    start, request_id = time.perf_counter(), str(uuid4())
    timestamp = datetime.now(UTC).isoformat()
    request.state.user = None
    body = None

    if request.method in ("POST", "PUT", "DELETE"):
        raw = await request.body()
        try:
            body = redact(json.loads(raw)) if raw else None
        except (ValueError, UnicodeError):
            body = "[invalid JSON]"

    try:
        for method, path, roles in permissions:
            if request.method == method and path.fullmatch(request.url.path) and roles:
                scheme, _, token = request.headers.get("Authorization", "").partition(
                    " "
                )
                if scheme.lower() != "bearer" or not token:
                    fail(401, "TOKEN_INVALID", "Требуется Bearer token.")
                request.state.user = auth.decode(token)
                if request.state.user["role"] not in roles:
                    fail(403, "ACCESS_DENIED", "Недостаточно прав.")
                break
        response = await call_next(request)
    except APIError as exc:
        response = error_response(exc.status, exc.body)
    except Exception:
        response = error_response(
            500,
            {"error_code": "INTERNAL_ERROR", "message": "Внутренняя ошибка сервиса."},
        )

    response.headers["X-Request-Id"] = request_id
    entry = {
        "request_id": request_id,
        "method": request.method,
        "endpoint": request.url.path,
        "status_code": response.status_code,
        "duration_ms": round((time.perf_counter() - start) * 1000, 3),
        "user_id": request.state.user["sub"] if request.state.user else None,
        "timestamp": timestamp,
    }

    if request.method in ("POST", "PUT", "DELETE"):
        entry["request_body"] = body
    print(json.dumps(entry, ensure_ascii=False), flush=True)

    return response


@app.get("/health", include_in_schema=False)
def health():
    with connect() as db:
        db.execute("SELECT 1")

    return {"status": "ok"}
