.PHONY: test lint up down

test:
	go test -race ./...

lint:
	golangci-lint run

up:
	docker compose up -d --build

down: # wipes the database volume
	docker compose down -v
