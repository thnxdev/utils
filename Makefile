SRC = $(shell find . -type f -name '*.go' -not -path "./database/*")
REGEN_DIRS = $(shell find . -name '*.go' | xargs grep -l //go:generate | xargs dirname | sort | uniq)

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: regen
regen: $(patsubst %, %/.go-generate, $(REGEN_DIRS)) ## Regenerate all code.

.PHONY: migrate
migrate: ## Create all databases and apply migrations
	$(MAKE) -C server/db migrate

.PHONY:
dev: ## Run hot reload dev server
	reflex -d fancy -c reflex.conf

.PHONY:
reset:
	rm -rf tmp/dbs/*.sql

.PHONY: regen
regen: $(patsubst %, %/.go-generate, $(REGEN_DIRS)) ## Regenerate all code.
	$(MAKE) -C database regen

.PHONY: lint
lint: ## Lint the code.
	golangci-lint run

.PHONY:
fmt: ## Format all code.
	@gofmt -l -w $(SRC)

# Generate rules for each directory that has files with go:generate directives.
define GO_GENERATE_RULE
$1/.go-generate: $1/*.go
	go generate -x $1
	touch $1/.go-generate
endef

$(foreach dir,$(REGEN_DIRS),$(eval $(call GO_GENERATE_RULE,$(dir))))