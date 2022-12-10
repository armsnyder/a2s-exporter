default: build test

lint:
	golangci-lint run

fix:
	golangci-lint run --fix

build:
	go mod tidy
	go build ./...

test:
	go test -v -race ./...
