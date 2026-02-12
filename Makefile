.PHONY: run build build-frontend dev clean test

DATA_DIR ?= ./data

run: build
	./bin/apricot --data-dir=$(DATA_DIR)

build: build-frontend
	@mkdir -p bin
	go build -o bin/apricot ./cmd/server

build-frontend:
	cd web && npm install && npm run build
	@mkdir -p internal/api/dist
	cp -r web/dist/* internal/api/dist/

dev:
	@echo "Starting React dev server and Go backend..."
	@cd web && npm run dev &
	@which air > /dev/null 2>&1 || go install github.com/air-verse/air@latest
	air

clean:
	rm -rf bin/
	rm -rf tmp/
	rm -rf web/dist/
	rm -rf web/node_modules/
	rm -rf internal/api/dist/

test:
	go test ./...
	cd web && npm test -- --run
