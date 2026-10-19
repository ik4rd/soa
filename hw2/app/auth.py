import hashlib
import hmac
import os
from datetime import UTC, datetime, timedelta
from uuid import UUID, uuid4

import jwt

from .common import fail

SECRET = os.environ["JWT_SECRET"]


def password_hash(password, salt=None):
    salt = salt or os.urandom(16).hex()
    digest = hashlib.scrypt(
        password.encode(), salt=bytes.fromhex(salt), n=16384, r=8, p=1
    ).hex()

    return salt + ":" + digest


def password_matches(password, stored):
    return hmac.compare_digest(password_hash(password, stored.split(":")[0]), stored)


def tokens(user):
    now = datetime.now(UTC)

    def token(kind, lifetime):
        return jwt.encode(
            {
                "sub": str(user["id"]),
                "role": user["role"],
                "type": kind,
                "iat": now,
                "exp": now + lifetime,
                "jti": str(uuid4()),
                "iss": "marketplace",
                "aud": "marketplace",
            },
            SECRET,
            algorithm="HS256",
        )

    return {
        "access_token": token("access", timedelta(minutes=20)),
        "refresh_token": token("refresh", timedelta(days=14)),
        "token_type": "bearer",
    }


def decode(token, kind="access"):
    try:
        data = jwt.decode(
            token,
            SECRET,
            algorithms=["HS256"],
            issuer="marketplace",
            audience="marketplace",
            options={"require": ["sub", "role", "type", "exp", "iat", "jti"]},
        )

        UUID(data["sub"])

        if data["type"] != kind or data["role"] not in ("USER", "SELLER", "ADMIN"):
            raise ValueError()

        return data
    except (jwt.InvalidTokenError, ValueError, TypeError) as exc:
        code = (
            "REFRESH_TOKEN_INVALID"
            if kind == "refresh"
            else (
                "TOKEN_EXPIRED"
                if isinstance(exc, jwt.ExpiredSignatureError)
                else "TOKEN_INVALID"
            )
        )

        fail(401, code, "Токен недействителен или истёк.")
