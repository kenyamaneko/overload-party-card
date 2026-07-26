.PHONY: build test test-integration vet fmt run tidy generate help

APP    := overload-party-card
MODULE := github.com/kenyamaneko/$(APP)

build: ## Build Docker image
	docker build -t $(APP) .

test: ## Run unit tests
	go test ./... -count=1 -race

test-integration: ## Run unit + integration tests (Testcontainers で Postgres を起動するので Docker 必須)
	go test -tags=integration ./... -count=1 -race

vet: ## Run go vet
	go vet ./...

tidy: ## Tidy dependencies
	go mod tidy

fmt: ## Format code
	gofmt -s -w .

run: ## Run card server locally (ENV=dev inline; PORT / DATABASE_CONN / GOOGLE_CLOUD_PROJECT_ID / PLAYER_ONBOARDED_SUBSCRIPTION / CARD_PACK_PURCHASED_SUBSCRIPTION must be exported)
	ENV=dev go run ./cmd/server

generate: ## Regenerate all SSoT-derived outputs (契約型 / schema doc / カード / プロダクト・施策 / カードパック)。oapi-codegen / openapi-typescript / NSwag が必要
	scripts/generate_types.sh
	python3 scripts/generate_schema_doc.py
	python3 scripts/generate_cards.py
	python3 scripts/generate_products.py
	python3 scripts/generate_card_packs.py

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
