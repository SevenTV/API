all: api linux

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

linux:
	GOOS=linux GOARCH=amd64 go build -v -ldflags "-X 'main.Version=${VERSION}' -X 'main.Unix=$(shell date +%s)' -X 'main.User=${BUILDER}'" -o bin/rest .

lint:
	staticcheck ./...
	go vet ./...
	golangci-lint run --go=1.18
	yarn prettier --write .

deps: go_installs
	go mod download
	yarn

go_installs:
	go install honnef.co/go/tools/cmd/staticcheck@2022.1
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/99designs/gqlgen@v0.17.5

api: 
	swag init --dir rest/v3 -g v3.go -o rest/v3/docs & swag init --dir rest/v2 -g v2.go -o rest/v2/docs
	gqlgen --config ./gqlgen.v3.yml & gqlgen --config ./gqlgen.v2.yml
