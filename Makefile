local_dev:
	@docker compose -f ./infra/docker-compose.yml up -d

local_down:
	@docker compose -f ./infra/docker-compose.yml down

local_reset:
	@docker compose -f ./infra/docker-compose.yml down -v
