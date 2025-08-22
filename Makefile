dc = docker compose -f compose.dev.yml

.PHONY: run-dev down-dev build-dev clean-dev logs-dev restart-dev ps-dev migrate-up-dev migrate-down-dev rebuild-dev mock clean-mock test

info:
	$(dc) ps

run-dev:
	$(dc) up

down-dev:
	$(dc) down

build-dev:
	$(dc) build

clean-dev:
	$(dc) down --rmi all --volumes --remove-orphans

logs-dev:
	$(dc) logs -f

restart-dev:
	$(dc) restart

ps-dev:
	$(dc) ps

migrate-up-dev:
	$(dc) exec interview-backend-server migrate -path ./internal/db/migration -database postgres://user:password@localhost:5432/interview?sslmode=disable up

migrate-down-dev:
	$(dc) exec interview-backend-server migrate -path ./internal/db/migration -database postgres://user:password@localhost:5432/interview?sslmode=disable down

sqlc:
	sqlc generate

rebuild-dev: clean-dev build-dev run-dev

mock-gen:
	mockery --all

clean-mock:
	rm -rf ./internal/mocks

test:
	go test -v -cover ./...
