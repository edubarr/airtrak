# AirTrak Current Stage

This checklist reflects the repository state today. The broader MVP plan is in `airtrak_initial_plan.md`.

## Foundation

- [x] Go backend module in `backend/`.
- [x] API server entrypoint at `backend/cmd/server`.
- [x] Goose migration runner at `backend/cmd/migrate`.
- [x] PostgreSQL access through `pgx`.
- [x] Hand-written SQL queries under `backend/db/queries`.
- [x] sqlc-generated pgx query layer under `backend/internal/store`.
- [x] Buf-powered Protobuf and Connect Go generation.
- [x] Generated code under `backend/gen` and `backend/internal/store`.
- [x] Root `README.md` and `AGENTS.md` for project and agent workflow context.
- [ ] CI workflow for Buf, sqlc, Goose validation, Go tests, build, and lint.
- [ ] Docker/dev setup that also covers future WhatsApp storage needs.

## Users And Auth

- [x] Auth Protobuf service with `Register`, `Login`, `GetMe`, and `Logout`.
- [x] Health Protobuf service with `Healthz` and `Readyz`.
- [x] Connect RPC handlers for auth routes.
- [x] Email/password registration with bcrypt password hashes.
- [x] Email/password login.
- [x] JWT access tokens backed by `user_sessions`.
- [x] Bearer-token Connect auth interceptor.
- [x] `users`, `user_credentials`, and `user_sessions` tables.
- [x] `uuidv7()` defaults for auth table IDs.
- [x] `deleted_at` soft deletes for `users` and `user_credentials`.
- [x] 1-to-many credential model for `email_password` now and `whatsapp_jid` later.
- [x] Active auth lookups filter deleted users and credentials.
- [x] Fast auth unit tests.
- [x] Optional PostgreSQL auth integration test gated by `AIRTRAK_TEST_DATABASE_URL`.
- [ ] Account linking flow between email/password users and WhatsApp JIDs.
- [ ] Password reset or recovery flow.
- [ ] User profile update flow.
- [ ] Explicit soft-delete/deactivate user API.

## Groups

- [ ] Group, group member, group role, and group settings migrations.
- [ ] Group sqlc queries.
- [ ] Group service for setup, get, list, locale update, and admin management.
- [ ] Owner, admin, and member permission checks.
- [ ] GroupService Protobuf and Connect handlers.
- [ ] WhatsApp `/setup` flow to bind a WhatsApp group to an AirTrak group.
- [ ] Group audit events.

## Fields

- [ ] Field migrations and sqlc queries.
- [ ] Field service for create, list, update, and soft delete.
- [ ] FieldService Protobuf and Connect handlers.
- [ ] Field management bot commands.
- [ ] Field permission checks.

## Games

- [ ] Game, slot type, and game status migrations.
- [ ] Game sqlc queries.
- [ ] Game service for create, list, get, cancel, and code generation.
- [ ] GameService Protobuf and Connect handlers.
- [ ] Game creation bot flow.
- [ ] Game listing bot flow.
- [ ] Game cancellation bot flow.
- [ ] Game audit events.

## Registrations

- [ ] Registration and waitlist migrations.
- [ ] Registration sqlc queries.
- [ ] Registration service for join, leave, list players, slot type, capacity, and waitlist behavior.
- [ ] RegistrationService Protobuf and Connect handlers.
- [ ] `/join`, `/leave`, and `/players` bot commands.
- [ ] Registration audit events.

## WhatsApp Bot

- [ ] WhatsApp identity migration and sqlc queries.
- [ ] WhatsApp identity service for create/get by JID.
- [ ] `whatsmeow` client integration.
- [ ] Incoming message persistence and idempotency.
- [ ] Message filtering for supported chat types.
- [ ] BotService operational API for status, QR, and reconnect.
- [ ] Fake WhatsApp event tests that do not require real WhatsApp.

## Parser And I18n

- [ ] Fixed command parser for `/setup`, `/games`, `/join`, `/leave`, `/players`, create game, cancel game, and locale config.
- [ ] Intent resolver that turns parsed commands into domain service calls.
- [ ] `pt-BR` and `en` translations for user-facing bot replies.
- [ ] Parser golden tests for `pt-BR` and `en` fixed commands.
- [ ] LLM provider abstraction and mock provider.
- [ ] Natural-language parser after deterministic command flows are stable.

## Outbox And Workflow State

- [ ] Outgoing message/outbox migration and sqlc queries.
- [ ] Outbox repository/service for queued WhatsApp replies.
- [ ] Outbox worker and WhatsApp sender adapter with retry behavior.
- [ ] Pending action and conversation state migrations.
- [ ] Conversation state service for multi-step flows and confirmations.
- [ ] Outbox worker tests with a fake sender.

## API And Docs

- [x] `POST /airtrak.auth.v1.AuthService/Register`.
- [x] `POST /airtrak.auth.v1.AuthService/Login`.
- [x] `POST /airtrak.auth.v1.AuthService/GetMe`.
- [x] `POST /airtrak.auth.v1.AuthService/Logout`.
- [x] `POST /airtrak.health.v1.HealthService/Healthz`.
- [x] `POST /airtrak.health.v1.HealthService/Readyz`.
- [x] Bruno collection files for currently available routes under `docs/bruno/AirTrak`.
- [ ] API tests using generated Connect clients.
- [ ] Bruno requests for future group, field, game, registration, and bot operations.
- [ ] Keep Connect handlers thin; business rules should stay in internal services.

## Verification Baseline

Before considering backend changes complete, run from `backend/`:

```sh
buf lint
sqlc generate
goose -dir db/migrations validate
go test ./...
go build ./...
golangci-lint run
```

`golangci-lint run` must return `0 issues`.
