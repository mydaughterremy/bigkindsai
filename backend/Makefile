MIGRATION_URL := $(if $(MIGRATION_URL),$(MIGRATION_URL),$(shell echo "mysql://root:root@localhost:3306/upstage?parseTime=true"))
DB_ENGINE := $(if $(DB_ENGINE),$(DB_ENGINE),mysql)


run-postgres:
	docker run --name postgres -p 5432:5432 -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=upstage -d postgres

run-mysql:
	docker run --name atlas-demo -e MYSQL_ROOT_PASSWORD=pass -e MYSQL_DATABASE=example -p 3306:3306 -d mysql:latest

diff:
	atlas migrate diff --env "$(DB_ENGINE)"

migrate:
	atlas migrate apply --env "$(DB_ENGINE)" --url "$(MIGRATION_URL)"

run:
	go run .

bootstrap-local-dev-stack:
	docker-compose up -d
	$(MAKE) migrate