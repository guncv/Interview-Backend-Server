dc = docker compose -f compose.dev.yml
dc-prod = docker compose -f compose.prod.yml

.PHONY: run-dev down-dev build-dev clean-dev logs-dev restart-dev ps-dev migrate-up-dev migrate-down-dev rebuild-dev mock clean-mock test swagger-gen

info:
	$(dc) ps

run-dev:
	$(dc) up

run-prod:
	$(dc-prod) up

build-prod:
	$(dc-prod) build

build-dev:
	$(dc) build

clean-dev:
	$(dc) down --rmi all --volumes --remove-orphans

clean-prod:
	$(dc-prod) down --rmi all --volumes --remove-orphans

migrate-up-dev:
	$(dc) exec interview-backend-server migrate -path ./internal/db/migration -database postgres://user:password@localhost:5432/interview?sslmode=disable up

migrate-down-dev:
	$(dc) exec interview-backend-server migrate -path ./internal/db/migration -database postgres://user:password@localhost:5432/interview?sslmode=disable down

sqlc:
	sqlc generate

rebuild-dev: clean-dev build-dev run-dev

rebuild-prod: clean-prod build-prod run-prod

mock-gen:
	mockery --all

clean-mock:
	rm -rf ./internal/mocks

test:
	go test -v -cover ./...

swagger-gen:
	swag init -g cmd/server/main.go -o ./docs --outputTypes json --parseVendor --parseDependency

