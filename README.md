# AirTrak

AirTrak is an open-source platform for managing airsoft games. The project is planned as a WhatsApp-first backend where players and admins can manage fields, games, registrations, and group setup through a bot, while the same domain operations are also exposed through a typed Connect RPC API.

The long-term goal is a modular Go backend that can support WhatsApp, web, mobile, and other chat adapters without duplicating business logic.

## Architecture Direction

- Backend language: Go
- Database: PostgreSQL
- Primary API: Connect RPC with Protobuf
- Database access: sqlc-generated Go from hand-written SQL
- PostgreSQL driver/runtime: pgx
- Migrations: Goose
- Initial chat integration target: WhatsApp through whatsmeow
- Initial auth model: normal email/password login with JWT sessions, designed to support WhatsApp JID credentials later

The project follows a modular monolith approach. Domain/application services should remain the center of the system, while Connect RPC, WhatsApp, and future clients act as adapters.

## Current Stage

The repository is in the initial backend implementation stage.

Implemented so far:

- Backend Go module under `backend/`.
- Buf configuration for Protobuf and Connect Go generation.
- Auth Protobuf service at `backend/proto/airtrak/auth/v1/auth.proto`.
- Generated Protobuf and Connect Go code under `backend/gen/`.
- PostgreSQL auth migration under `backend/db/migrations/`.
- sqlc query definitions under `backend/db/queries/`.
- sqlc-generated pgx query layer under `backend/internal/store/`.
- Auth service with register, login, JWT authentication, session validation, and logout.
- User credential model that supports `email_password` now and `whatsapp_jid` later.
- API server with Connect `AuthService`, `/healthz`, and `/readyz`.
- Goose migration runner under `backend/cmd/migrate`.
- Fast unit tests and optional PostgreSQL integration tests.

Not implemented yet:

- WhatsApp bot integration.
- Groups, fields, games, and registrations domain services.
- Permission layer.
- i18n/localized bot replies.
- Outbox worker.
- Parser, fixed commands, LLM provider, and conversation state.
- Frontend/mobile clients.

## Auth Model

The first slice uses these core tables:

- `users`: application users.
- `user_credentials`: one-to-many credentials for a user.
- `user_sessions`: server-side session records backing JWT access tokens.

Credential types currently planned:

- `email_password`: normalized email plus bcrypt password hash.
- `whatsapp_jid`: WhatsApp JID with no password hash, for future WhatsApp identity lookup and account linking.

IDs use PostgreSQL `uuidv7()` defaults. The current migration expects PostgreSQL 18 or another database setup that provides `uuidv7()`.

## Development

From the backend directory:

```sh
cd backend
```

Install tools:

```sh
make tools
```

Generate code:

```sh
make generate
```

Run tests:

```sh
make test
```

Run migrations:

```sh
go run ./cmd/migrate up
```

Run the API server:

```sh
go run ./cmd/server
```

Optional PostgreSQL integration test:

```sh
AIRTRAK_TEST_DATABASE_URL='postgres://airtrak:airtrak@localhost:5432/airtrak?sslmode=disable' go test ./internal/auth -v
```

## Useful Checks

```sh
cd backend
buf lint
sqlc generate
goose -dir db/migrations validate
go test ./...
go build ./...
go vet ./...
golangci-lint run
```

## Next Steps

1. Add the core domain schema and services for WhatsApp identities, groups, group members, fields, games, registrations, outbox, audit events, pending actions, and conversation state.
2. Implement a permission layer covering owner, admin, and member roles.
3. Expose group, field, game, and registration operations through Connect RPC.
4. Add i18n support for `pt-BR` and `en` bot/API messages where user-facing text is needed.
5. Implement fixed command parsing for `/setup`, `/games`, `/join`, `/leave`, and game administration flows.
6. Add WhatsApp integration through whatsmeow, using WhatsApp JIDs to resolve or create credentials.
7. Add outbox worker support for reliable WhatsApp replies.
8. Add LLM parser abstraction after deterministic command flows are working.

See `docs/airtrak_initial_plan.md` for the full MVP architecture and `docs/current_stage.md` for the current implementation checklist.
