# config-service

[![CI](https://github.com/vyacheslav-nuykin/config-service/actions/workflows/ci.yml/badge.svg)](https://github.com/vyacheslav-nuykin/config-service/actions/workflows/ci.yml)

A small configuration service for my infrastructure. Stores service
configs in PostgreSQL and serves them over HTTP.

Built with Go and `net/http` (no frameworks). Migrations run automatically
on startup.

## Endpoints

| Method   | Path                              | Description                    |
|----------|-----------------------------------|--------------------------------|
| `GET`    | `/health`                         | Health check (pings database)  |
| `GET`    | `/info`                           | Service name and version       |
| `GET`    | `/config/{namespace}`             | List all configs in namespace  |
| `GET`    | `/config/{namespace}/{key}`       | Get one config value           |
| `POST`   | `/config/{namespace}/{key}`       | Create or update a config      |
| `DELETE` | `/config/{namespace}/{key}`       | Delete a config                |

## Validation rules

- `namespace`: 1–64 chars, `a-z`, `0-9`, `-`, `_`
- `key`: 1–128 chars, `a-z`, `A-Z`, `0-9`, `_`, `-`, `.`
- `value`: 0–4096 chars

Invalid input returns `400` with a JSON body:

```json
{
  "error": "validation failed",
  "field": "namespace",
  "reason": "contains invalid characters"
}
```

## Quick start

Requires Docker.

```bash
git clone https://github.com/vyacheslav-nuykin/config-service.git
cd config-service
docker compose up -d
```

The service starts on `http://localhost:8080`. Postgres starts on `5432`.

## Examples

Create a config:

```bash
curl -X POST http://localhost:8080/config/webhook-service/PORT \
  -H "Content-Type: application/json" \
  -d '{"value":"9000"}'
```

Read it back:

```bash
curl http://localhost:8080/config/webhook-service/PORT
# {"key":"PORT","namespace":"webhook-service","value":"9000"}
```

List everything for a namespace:

```bash
curl http://localhost:8080/config/webhook-service
# {"configs":{"PORT":"9000"},"namespace":"webhook-service"}
```

Delete:

```bash
curl -X DELETE http://localhost:8080/config/webhook-service/PORT
```

## Local development

Requires Go 1.25+ and Docker.

```bash
# Start only the database
docker compose up -d postgres

# Run the service
go run ./cmd/server
```

Defaults match the docker-compose config. To override, set environment
variables before `go run`, for example:

```bash
DATABASE_URL=postgres://user:pass@host:5432/db go run ./cmd/server
```

## Tests

```bash
go test ./internal/... -cover
```

Tests require a running Postgres. Use the one from `docker compose` or set
`TEST_DATABASE_URL` to a different instance.

## Project structure

```
config-service/
├── cmd/server/           # Entry point
├── internal/
│   ├── api/              # HTTP handlers, middleware, validation
│   └── storage/          # PostgreSQL layer, migrations
├── migrations/           # SQL migrations (embedded in binary)
├── Dockerfile
└── docker-compose.yml
```

## Stack

- Go 1.25
- PostgreSQL 16
- `pgx/v5` for the database
- `golang-migrate` for migrations
- Standard `net/http` — no web framework

## License

MIT — see [LICENSE](LICENSE).
