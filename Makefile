.PHONY: build lint deps dev_deps generate clean work dev

BUILDER := "unknown"
VERSION := "unknown"

ifeq ($(origin API_BUILDER),undefined)
	BUILDER = $(shell git config --get user.name);
else
	BUILDER = ${API_BUILDER};
endif

ifeq ($(origin API_VERSION),undefined)
	VERSION = $(shell git rev-parse HEAD);
else
	VERSION = ${API_VERSION};
endif

build:
	GOOS=linux GOARCH=amd64 go build -v -ldflags "-X 'main.Version=${VERSION}' -X 'main.Unix=$(shell date +%s)' -X 'main.User=${BUILDER}'" -o out/api cmd/*.go

lint:
	golangci-lint run --go=1.18
	yarn prettier --check .

format:
	gofmt -s -w .
	yarn prettier --write .

deps:
	go install github.com/swaggo/swag/cmd/swag@v1.8.1
	go install github.com/99designs/gqlgen@v0.17.5
	go mod download

dev_deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	yarn

generate: 
	swag init --dir internal/rest/v3 -g v3.go -o internal/rest/v3/docs & swag init --dir internal/rest/v2 -g v2.go -o internal/rest/v2/docs
	gqlgen --config ./gqlgen.v3.yml & gqlgen --config ./gqlgen.v2.yml

test:
	go test -count=1 -cover -parallel $$(nproc) -race ./...

clean:
	rm -rf \
		out \
		internal/gql/v2/gen/generated/generated-gqlgen.go \
		internal/gql/v2/gen/model/models-gqlgen.go \
		internal/gql/v3/gen/generated/generated-gqlgen.go \
		internal/gql/v3/gen/model/models-gqlgen.go \
		internal/rest/v2/docs \
		internal/rest/v3/docs \
		node_modules

work:
	echo -e "go 1.18\n\nuse (\n\t.\n\t../Common\n\t../message-queue/go\n\t../image-processor/go\n\t../CompactDisc\n)" > go.work
	go mod tidy

dev:
	go run cmd/main.go
