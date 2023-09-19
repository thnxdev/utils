.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY :regen
regen: Makefile $(wildcard */db.go */query.sql.go */models.go) ## Rebuild the sqlc database access layers

db.go query.sql.go models.go: Makefile sqlc.yaml $$(wildcard queries/*.sql) $$(wildcard schema/*.sql)
	sqlc generate

.PHONY: migrate
migrate: ## Create all databases and apply migrations
	for i in */schema; do \
		db="td-$$(dirname $$i)"; \
		createdb "$$db" || true; \
		(cd "$$i" && $ goose postgres "dbname=$$db sslmode=disable" up); \
	done

.PHONY: drop
drop: ## Drop all databases
	for i in */schema; do \
		db="td-$$(dirname $$i)"; \
		dropdb "$$db"; \
	done