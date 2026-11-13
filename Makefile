.PHONY: swagger build run test coverage

SWAG ?= $(shell go env GOPATH)/bin/swag

swagger:
	@test -f $(SWAG) || go install github.com/swaggo/swag/cmd/swag@v1.16.4
	cd cmd/app && $(SWAG) init \
		-g main.go \
		-o ../../docs \
		--parseDependency \
		--parseInternal \
		--dir .,../../internal/transport/http/v1/handler,../../internal/transport/http/v1/dto

build: swagger
	go build -o bin/app ./cmd/app

run: build
	./bin/app

test:
	go test ./...

coverage:
	go test ./... -coverprofile=./coverage.out
	go tool cover -func=./coverage.out