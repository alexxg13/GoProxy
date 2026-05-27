# GoProxy

Reverse-proxy с IP-фильтрацией, rate limiting, кэшированием и админ-API.

## Swagger (swaggo)

Документация генерируется из аннотаций в `cmd/app/main.go` и `internal/transport/http/v1/handler/`.

```bash
make swagger
```

После запуска приложения UI доступен по адресу:

- http://localhost:8080/swagger/index.html

Сгенерированные файлы: `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`.

## Запуск

```bash
make build
# backend должен слушать PROXY_BACKEND_URL (по умолчанию http://127.0.0.1:3000)
python3 -m http.server 3000 &
./bin/app -env .env
```

Прокси: http://localhost:8080 → backend из `PROXY_BACKEND_URL`.

Если в логах `upstream error` / status 502 — backend не запущен или неверный URL.

## Тесты

```bash
make test
```

## Grafana

```bash
docker compose up --build
```

- Grafana: http://localhost:3001
- Login/password: `admin` / `admin`
- Dashboard: `GoProxy / GoProxy Overview`
- Prometheus: http://localhost:9090
