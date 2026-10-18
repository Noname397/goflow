.PHONY: test race docker-up docker-down

run:
	go run ./cmd/goflow

test:
	go test ./...

race:
	go test -race ./...

docker-up:
	docker compose up -d postgres

docker-down:
	docker compose down -v
