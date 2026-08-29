local_dev:
	@docker compose -f ./infra/docker-compose.yml up -d

bootstrap_cassandra:
	@sh ./scripts/bootstrap-cassandra.sh
