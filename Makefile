.PHONY: build publisher event-listener test lint fmt infra infra-down integration-test

build:
	go build -o bin/server ./cmd/server

event-listener:
	go run ./cmd/event_listener/main.go

publisher:
ifndef INTERVAL
	$(error INTERVAL is required. Usage: make publisher INTERVAL=2)
endif
	go run ./cmd/publisher/main.go $(INTERVAL)

run:
	go run ./cmd/server/main.go

test:
	go test ./... -v -count=1

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

infra:
	docker compose up -d --build

infra-down:
	docker compose down -v

integration-test:
	./scripts/integration_test.sh
