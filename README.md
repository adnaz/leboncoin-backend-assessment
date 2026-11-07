# Way of working

This repository is read-only, click on "Use this template" => "Create a new repository" and work on your copy just as you would on a live one. \
Please make sure the first commit on your repo is the fresh copy of the original repository, please start your work from the second commit only. \
Once you are satisfied with your work, please send your public repository url to the hiring manager.

# Hiring instructions

Please refer to [these instructions](INSTRUCTIONS.md).

## Assessment items

One pull request per item, squash-merged in order. Each PR body has the reasoning, the alternatives and what was measured on the compose stack.

| # | Item | PR |
|---|------|----|
| 1 | Fix the `GetAdByID` usecase | [#1](https://github.com/adnaz/leboncoin-backend-assessment/pull/1) |
| 2 | `DELETE /ads/{id}` endpoint, usecase and dao method | [#2](https://github.com/adnaz/leboncoin-backend-assessment/pull/2) |
| 3 | `username` in `GET /ads/{id}` from the user service | [#3](https://github.com/adnaz/leboncoin-backend-assessment/pull/3) |
| 4 | `price_cents` → `price` as a float: [the answer](docs/price-as-float.md) | [#4](https://github.com/adnaz/leboncoin-backend-assessment/pull/4) |
| 5 | Bulk import: one transaction per batch, idempotency proposed | [#5](https://github.com/adnaz/leboncoin-backend-assessment/pull/5) |
| 6 | Quota of 5 ads per rolling 24 h on `POST /ads` | [#6](https://github.com/adnaz/leboncoin-backend-assessment/pull/6) |

Improvements after the six items:

| Improvement | PR |
|-------------|----|
| CI (lint, tidy, build, tests with race), lint config, green test baseline, `main` protected by the `ci` check | [#7](https://github.com/adnaz/leboncoin-backend-assessment/pull/7) |
| Nickname cache and cap on identity calls in flight for `GET /ads/{id}` | [#8](https://github.com/adnaz/leboncoin-backend-assessment/pull/8) |
| `401` instead of `500` on an invalid `Authorization` header, checked once in the middleware | [#9](https://github.com/adnaz/leboncoin-backend-assessment/pull/9) |

What I would do next, in order:

- **Pagination on `GET /ads`.** The list runs `SELECT … FROM ads WHERE …` with no `LIMIT` and no `ORDER BY`. Fine with 100 seed rows; on a real table every call reads and serialises the whole table, on the most-read endpoint, and the row order is whatever the database returns. `limit`/`offset` plus `has_more`, newest first, with an index on `(created_at DESC, id DESC)` so "newest first, first 20" is an index read, not a sort. Opt-in: a client that sends no parameters keeps getting everything.
- **One JSON error body for every error.** Today `400`, `401`, `404`, `429` and `500` are bare status codes with an empty body. One small shape such as `{"status": 400, "title": "invalid body", "detail": "price_cents must be positive"}`, written by one helper every handler calls. The `429` would then also say when the user can publish again, which is what a mobile app shows.
- **Read, write and idle timeouts on the shared HTTP server.** `common/httpserver` sets only `ReadHeaderTimeout`. Without `ReadTimeout` a client sending a body slowly holds a connection and a goroutine for as long as it likes; without `WriteTimeout` a client reading slowly does the same; without `IdleTimeout` unused keep-alive connections stay open. Three fields on `http.Server`, and the users API gets them too.

# Ads API

A small Go HTTP API around a single domain: *ads* (classified ads).

The API needs an up-and-running postgres database, which can be launched via the provided Docker Compose file.

## Requirements

- Go 1.26 or later
- Docker with the Compose plugin

## Launching the stack

```bash
docker compose up -d
```

This starts PostgreSQL with the schema and seed data already applied, as well as the `ads` api and the `users` api the `ads` api asks for owner nicknames; the `ads` api serves without it (`username` is then `null`).

## Model

There are 2 entities in this sample:
- ad (title, price, etc...)
- user (id and nickname only)

## API

| Method | Path          | Auth | Description                                                                       |
|--------|---------------|------|-----------------------------------------------------------------------------------|
| `POST` | `/ads`        | yes | Create an ad (at most 5 per rolling 24 h, then `429` with `Retry-After`)            |
| `GET` | `/ads/{id}`   | no | Get a single ad, with its owner's username (`null` if the identity service cannot answer; cached 60 s) |
| `DELETE` | `/ads/{id}`   | yes | Delete one of your ads (`404` if it is not yours or already deleted) |
| `GET` | `/ads`        | no | List ads (filterable by `title`, `user_id`, `min_price_cents`, `max_price_cents`) |
| `POST` | `/ads/import` | yes | Bulk ad import                                                                    |

```bash
# Health check
curl -s -v http://localhost:8080/health

# List ads (with optional filters)
curl -s 'http://localhost:8080/ads?title=bike&min_price_cents=1000'

# Get a single ad
curl -s http://localhost:8080/ads/1

# Create an ad (requires Authorization header)
curl -s -X POST http://localhost:8080/ads \
  -H 'Authorization: 00000000-0000-0000-0000-000000000001' \
  -H 'Content-Type: application/json' \
  -d '{"title":"Mountain bike","price_cents":12000,"photo_url":"https://example.com/photos/bike.jpg"}'
```

## Authentication

To pass authentication information to a call, simply add the `Authorization` header with the user ID in plain text. A missing or invalid user ID is a `401`.

It will be extracted by the service.

```
Authorization: 00000000-0000-0000-0000-000000000001
```

## Tests

To launch all the tests:

```shell
go test ./...
```
