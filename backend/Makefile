.PHONY: tools generate proto sqlc tidy build run test up migrate-up migrate-down migrate-status migrate-create

tools:
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/evilmartians/lefthook@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install mvdan.cc/gofumpt@latest
	lefthook install

generate: proto sqlc

proto:
	PATH="$$(go env GOPATH)/bin:$$PATH" buf generate

sqlc:
	sqlc generate

tidy:
	go mod tidy

build:
	go build ./...

run:
	go run ./cmd/server

test:
	go test ./...

up:
	docker-compose up -d --build

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-status:
	go run ./cmd/migrate status

migrate-create:
	goose -dir db/migrations create $(name) sql
