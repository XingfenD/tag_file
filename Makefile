.PHONY: all build test clean install run

BINARY_NAME := tagfile.out

all: build

build:
	go build -o $(BINARY_NAME) ./cmd/cli

test:
	go test ./... -v

clean:
	rm -f $(BINARY_NAME)
	rm -rf ./dist

install:
	go install ./cmd/cli

run: build
	./$(BINARY_NAME)

build-linux:
	GOOS=linux GOARCH=amd64 go build -o dist/$(BINARY_NAME)-linux ./cmd/cli

build-windows:
	GOOS=windows GOARCH=amd64 go build -o dist/$(BINARY_NAME)-windows.exe ./cmd/cli
