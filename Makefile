default: build test

lint:
	golangci-lint run

fix:
	golangci-lint run --fix

build:
	go build ./...

test:
	go test -v -race ./...
