# Loyalty Service / Gophermart

Training backend project written in Go.

The service implements a REST API for a loyalty points system. Users can register, log in, upload order numbers, receive loyalty accruals from an external accrual service, check their balance, withdraw points, and view withdrawal history.

This is a training project, not a production system.

---

## Features

* User registration and login
* Password hashing with bcrypt
* JWT-based authentication
* PostgreSQL storage
* Order upload and ownership checks
* Order number validation with the Luhn algorithm
* User balance calculation
* Balance withdrawals with transaction-based consistency
* Withdrawal history
* Background processing of unprocessed orders
* Integration with an external accrual service
* Handling of external `429 Too Many Requests` responses with `Retry-After`
* Gzip compression and decompression middleware
* Request logging
* Unit tests, including handler tests with mocks

---

## Tech stack

* Go
* net/http
* chi router
* PostgreSQL
* pgx / pgxpool
* JWT
* bcrypt
* zap logger
* gomock

---

## API

| Method | Endpoint                     | Description             |
| ------ | ---------------------------- | ----------------------- |
| POST   | `/api/user/register`         | Register a new user     |
| POST   | `/api/user/login`            | Authenticate user       |
| POST   | `/api/user/orders`           | Upload order number     |
| GET    | `/api/user/orders`           | Get user orders         |
| GET    | `/api/user/balance`          | Get current balance     |
| POST   | `/api/user/balance/withdraw` | Withdraw loyalty points |
| GET    | `/api/user/withdrawals`      | Get withdrawal history  |

Protected endpoints require:

```http
Authorization: Bearer <token>
```

---

## Storage model

PostgreSQL is used as the main storage.

The service stores:

* users
* uploaded orders
* order processing statuses
* calculated accruals
* balance withdrawals

The database schema is initialized on application startup.

---

## Background processing

The service periodically loads unprocessed orders and requests their status from the external accrual service.

Supported external statuses:

* `REGISTERED`
* `PROCESSING`
* `INVALID`
* `PROCESSED`

Final statuses are:

* `INVALID`
* `PROCESSED`

If the accrual service responds with `429 Too Many Requests`, the service reads the `Retry-After` header and delays the next request.

---

## Balance logic

User balance is calculated as:

```text
processed order accruals - withdrawals
```

Withdrawals are written inside a PostgreSQL transaction:

1. calculate current available balance
2. reject the request if funds are insufficient
3. insert withdrawal record
4. commit transaction

---

## Configuration

Configuration is provided via flags or environment variables.

### Flags

```bash
-a   HTTP server address
-d   PostgreSQL connection string
-r   accrual service address
-k   JWT signing key
```

Example:

```bash
go run ./cmd/gophermart \
  -a localhost:8080 \
  -d "postgres://user:pass@localhost:5432/loyalty?sslmode=disable" \
  -r "http://localhost:8081" \
  -k "secret-key"
```

### Environment variables

```bash
RUN_ADDRESS=localhost:8080
DATABASE_URI=postgres://user:pass@localhost:5432/loyalty?sslmode=disable
ACCRUAL_SYSTEM_ADDRESS=http://localhost:8081
LOYALTY_KEY=secret-key
```

---

## Build

```bash
go build -o bin/gophermart ./cmd/gophermart
```

Run:

```bash
./bin/gophermart \
  -a localhost:8080 \
  -d "postgres://user:pass@localhost:5432/loyalty?sslmode=disable" \
  -r "http://localhost:8081" \
  -k "secret-key"
```

---

## Tests

```bash
go test ./...
```

---

## Limitations

* This is a training project.
* The schema is initialized directly by the application; there is no migration system.
* Observability is limited to logging; production metrics are not implemented.
* Background order processing is intentionally simple.
* The default JWT key is insecure and must be overridden outside local development.

---

## Notes for reviewers

The project demonstrates a typical REST backend with authentication, PostgreSQL persistence, order and balance business rules, transaction-based withdrawals, and integration with an external accrual service.

It is not intended to be presented as a production-grade system.
