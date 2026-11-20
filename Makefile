.DEFAULT_GOAL := build

.PHONY: dev test build db-migrate web-build

load_env = set -a; if [ -f .env ]; then . ./.env || exit; fi; set +a

web/node_modules/.installed: web/package.json web/package-lock.json
	cd web && npm ci
	touch $@

web-build: web/node_modules/.installed
	cd web && npm run build

build: web-build
	go build -o bin/science-swap .

dev: build
	@$(load_env); ADDR="$${ADDR:-127.0.0.1:8080}" exec ./bin/science-swap

test: web-build
	cd web && npm test
	@$(load_env); go test -count=1 ./...

db-migrate: build
	@$(load_env); exec ./bin/science-swap migrate
