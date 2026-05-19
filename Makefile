.PHONY: test build backend frontend dev docker-build docker-up docker-down

GO_TEST_ENV=GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin

test:
	$(GO_TEST_ENV) go test ./...
	npm test

build: backend frontend

backend:
	$(GO_TEST_ENV) go test ./...
	GOOS=darwin GOARCH=arm64 go build -o bin/feed-puller ./cmd/feed-puller

frontend:
	npm run build

dev:
	npm run dev

docker-build:
	docker build -t feed-puller:local .

docker-build-amd64:
	docker build --platform linux/amd64 -t feed-puller:amd64 .

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down
