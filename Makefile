.PHONY: all build test clean install run setup completion

BINARY_NAME := tagfile.out

all: build

build:
	go build -o $(BINARY_NAME) ./cmd/cli

test: setup
	go test ./... -v

clean:
	rm -f $(BINARY_NAME)
	rm -rf .tagfile
	rm -rf ./dist

install:
	go install ./cmd/cli

run: build
	./$(BINARY_NAME)

setup: build
	bash scripts/setup_test_workspace.sh

completion: build
	./$(BINARY_NAME) completion zsh > /tmp/_tagfile && echo "source /tmp/_tagfile" ; \
	./$(BINARY_NAME) completion bash > /tmp/tagfile.bash && echo "source /tmp/tagfile.bash" ; \
	./$(BINARY_NAME) completion fish > /tmp/tagfile.fish && echo "cp /tmp/tagfile.fish ~/.config/fish/completions/"

build-linux:
	GOOS=linux GOARCH=amd64 go build -o dist/$(BINARY_NAME)-linux ./cmd/cli

build-windows:
	GOOS=windows GOARCH=amd64 go build -o dist/$(BINARY_NAME)-windows.exe ./cmd/cli
