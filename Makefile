dc = docker compose -f compose.dev.yml
dc-uat = docker compose -f compose.uat.yml

.PHONY: run-dev down-dev build-dev clean-dev logs-dev restart-dev ps-dev migrate-up-dev migrate-down-dev rebuild-dev mock clean-mock test swagger-gen

info:
	$(dc) ps

run-dev:
	$(dc) up

run-uat:
	$(dc-uat) up

build-uat:
	$(dc-uat) build

build-dev:
	$(dc) build

clean-dev:
	$(dc) down --rmi all --volumes --remove-orphans

clean-uat:
	$(dc-uat) down --rmi all --volumes --remove-orphans

migrate-up-dev:
	$(dc) exec interview-backend-server migrate -path ./internal/db/migration -database postgres://user:password@localhost:5432/interview?sslmode=disable up

migrate-down-dev:
	$(dc) exec interview-backend-server migrate -path ./internal/db/migration -database postgres://user:password@localhost:5432/interview?sslmode=disable down

sqlc:
	sqlc generate

rebuild-dev: clean-dev build-dev run-dev

rebuild-uat: clean-uat build-uat run-uat

mock-gen:
	mockery --all

clean-mock:
	rm -rf ./internal/mocks

test:
	go test -v -cover ./...

swagger-gen:
	swag init -g cmd/server/main.go -o ./docs --outputTypes json --parseVendor --parseDependency

