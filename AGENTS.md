# AirTrak Agent Notes

## Scope
- The active codebase is the Go backend in `backend/`; run Go, Buf, sqlc, Goose, and lint commands from that directory.
- `README.md` describes the current implementation stage; `airtrak_initial_plan.md` is the broader plan, not all implemented behavior.
- Current real entrypoints are `backend/cmd/server` for the API and `backend/cmd/migrate` for Goose migrations.

## Architecture
- Keep business logic in internal services such as `internal/auth`; Connect handlers in `internal/api` should stay thin adapters.
- Connect RPC + Protobuf is the primary API. Add proto files under `backend/proto/...` and regenerate into `backend/gen/`.
- Database access is hand-written SQL in `backend/db/queries` plus sqlc-generated pgx code in `backend/internal/store`.
- Do not hand-edit generated code under `backend/gen` or `backend/internal/store`; change proto, SQL, migrations, or config and regenerate.

## Commands
- Install local tools: `cd backend && make tools`.
- Generate all code: `cd backend && make generate`.
- Generate only Protobuf/Connect: `cd backend && PATH="$(go env GOPATH)/bin:$PATH" buf generate`.
- Generate only sqlc: `cd backend && sqlc generate`.
- Run tests: `cd backend && go test ./...` or one package like `go test ./internal/auth -v`.
- Build: `cd backend && go build ./...`.
- Lint/format check: `cd backend && golangci-lint run`.
- Validate migrations: `cd backend && goose -dir db/migrations validate`.

## Database
- Migrations use Goose SQL files in `backend/db/migrations`; create new ones with `cd backend && make migrate-create name=short_name`.
- Apply migrations with `cd backend && go run ./cmd/migrate up`; it reads `APP_DATABASE_URL` or defaults to local app Postgres.
- The current auth migration uses Postgres `uuidv7()` defaults, so use Postgres 18 or another setup that provides `uuidv7()`.
- `docker-compose.yml` currently starts the app Postgres on localhost `5432`; the WhatsApp DB URL in `.env.example` is future-facing and not backed by compose yet.

## Tests
- Fast tests must not require Postgres; the auth integration test is skipped unless `AIRTRAK_TEST_DATABASE_URL` is set.
- Run the optional integration test with `cd backend && AIRTRAK_TEST_DATABASE_URL='postgres://airtrak:airtrak@localhost:5432/airtrak?sslmode=disable' go test ./internal/auth -v`.

## Style And Gotchas
- `golangci-lint` enables `gofumpt` with extra rules, `lll`, `wsl_v5`, full `govet`, and full `gocritic`; expect strict blank-line and struct-layout feedback.
- Go changes are not complete until `gofumpt -w` has been applied and `golangci-lint run` returns `0 issues`.
- `backend/lefthook.yml` exists, but still verify manually with `buf lint`, `sqlc generate`, `go test ./...`, `go build ./...`, and `golangci-lint run`.
- The auth model intentionally uses `user_credentials` for both `email_password` now and `whatsapp_jid` later; preserve that 1-to-many credential design.
- `users` and `user_credentials` use `deleted_at` soft deletes; auth lookup queries should filter out deleted rows.
