.PHONY: all build test clean install run setup

BINARY_NAME := tagfile.out

all: build

build:
	go build -o $(BINARY_NAME) ./cmd/cli

test: setup
	go build -o test.out ./cmd/test && ./test.out; rm -f test.out

clean:
	rm -f $(BINARY_NAME)
	rm -f .tag_file.db
	rm -rf ./dist

install:
	go install ./cmd/cli

run: build
	./$(BINARY_NAME)

setup: build
	bash scripts/setup_test_workspace.sh

build-linux:
	GOOS=linux GOARCH=amd64 go build -o dist/$(BINARY_NAME)-linux ./cmd/cli

build-windows:
	GOOS=windows GOARCH=amd64 go build -o dist/$(BINARY_NAME)-windows.exe ./cmd/cli
