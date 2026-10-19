FROM python:3.13.12-slim AS runtime
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir --disable-pip-version-check -r requirements.txt
COPY . .
RUN python generate.py
USER 10001:10001
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=10 CMD ["python", "-c", "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8080/health', timeout=2)"]
CMD ["python", "-m", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080", "--no-access-log"]

FROM python:3.13.12-slim AS tests
WORKDIR /app
COPY requirements-test.txt requirements.txt ./
RUN pip install --no-cache-dir --disable-pip-version-check -r requirements-test.txt
COPY . .
CMD ["python", "-m", "pytest", "-q", "tests"]
