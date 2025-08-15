.PHONY: run-dev down-dev build-dev clean-dev logs-dev restart-dev ps-dev migrate-up-dev migrate-down-dev rebuild-dev mock clean-mock test

info:
	docker-compose ps

run-dev:
	docker-compose -f docker-compose.dev.yml up

down-dev:
	docker-compose -f docker-compose.dev.yml down

build-dev:
	docker-compose -f docker-compose.dev.yml build

clean-dev:
	docker-compose -f docker-compose.dev.yml down --rmi all --volumes --remove-orphans

logs-dev:
	docker-compose -f docker-compose.dev.yml logs -f

restart-dev:
	docker-compose -f docker-compose.dev.yml restart

ps-dev:
	docker-compose -f docker-compose.dev.yml ps

migrate-up-dev:
	docker-compose -f docker-compose.dev.yml exec onyx-server migrate -path ./internal/db/migration -database postgres://user:password@localhost:5432/onyx?sslmode=disable up

migrate-down-dev:
	docker-compose -f docker-compose.dev.yml exec onyx-server migrate -path ./internal/db/migration -database postgres://user:password@localhost:5432/onyx?sslmode=disable down

sqlc:
	sqlc generate

rebuild-dev: clean-dev build-dev run-dev

mock-gen:
	mockery --all

clean-mock:
	rm -rf ./internal/mocks

test:
	go test -v -cover ./...
