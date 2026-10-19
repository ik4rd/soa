import os
from decimal import Decimal

import psycopg
from psycopg.rows import dict_row
from pydantic import BaseModel, field_serializer, field_validator


class APIModel(BaseModel):
    @field_validator("*", mode="before", check_fields=False)
    @classmethod
    def decimal_input(cls, value, info):
        if cls.model_fields[info.field_name].annotation is Decimal and type(
                value
        ) not in (int, float, Decimal):
            raise ValueError("Ожидается JSON number.")

        return value

    @field_serializer("*", check_fields=False, when_used="json")
    def json_number(self, value):
        return float(value) if isinstance(value, Decimal) else value


class APIError(Exception):
    def __init__(self, status, code, message, details=None):
        self.status = status
        self.body = {"error_code": code, "message": message, "details": details}


def fail(status, code, message, details=None):
    raise APIError(status, code, message, details)


def connect():
    return psycopg.connect(os.environ["DATABASE_URL"], row_factory=dict_row)
