.PHONY: all build test clean install run setup completion

BUILD_DIR := bin
DIST_DIR  := dist

CLI    := $(BUILD_DIR)/tagfile.exe
WEBUI  := $(BUILD_DIR)/tagfile-webui.exe
GUI    := $(BUILD_DIR)/tagfile-gui.exe
TEST   := $(BUILD_DIR)/test.exe

all: build

build:
	go build -o $(CLI) ./cmd/cli

test: setup
	go build -o $(TEST) ./cmd/test && $(TEST); rm -f $(TEST)

clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	rm -rf .tagfile
	rm -rf cmd/gui/build

install:
	go install ./cmd/cli

run: build
	./$(CLI)

setup: build
	bash scripts/setup_test_workspace.sh

completion: build
	./$(CLI) completion zsh > /tmp/_tagfile && echo "source /tmp/_tagfile" ; \
	./$(CLI) completion bash > /tmp/tagfile.bash && echo "source /tmp/tagfile.bash" ; \
	./$(CLI) completion fish > /tmp/tagfile.fish && echo "cp /tmp/tagfile.fish ~/.config/fish/completions/"

build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(DIST_DIR)/tagfile-linux ./cmd/cli

build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(DIST_DIR)/tagfile-windows.exe ./cmd/cli

build-webui:
	go build -o $(WEBUI) ./cmd/webui

run-webui: build-webui
	./$(WEBUI)

MPV_CFLAGS := -I/opt/homebrew/include -I/usr/local/include
MPV_LDFLAGS := -L/opt/homebrew/lib -L/usr/local/lib

build-gui:
	cd cmd/gui && CGO_CFLAGS="$(MPV_CFLAGS)" CGO_LDFLAGS="$(MPV_LDFLAGS)" \
		wails build -debug -o tagfile-gui.exe && \
		mkdir -p ../../$(BUILD_DIR) && mv build/bin/tagfile-gui.exe ../../$(BUILD_DIR)/

build-gui-prod:
	cd cmd/gui && CGO_CFLAGS="$(MPV_CFLAGS)" CGO_LDFLAGS="$(MPV_LDFLAGS)" \
		wails build -clean -o tagfile-gui.exe && \
		mkdir -p ../../$(BUILD_DIR) && mv build/bin/tagfile-gui.exe ../../$(BUILD_DIR)/

build-gui-macos:
	cd cmd/gui && CGO_CFLAGS="$(MPV_CFLAGS)" CGO_LDFLAGS="$(MPV_LDFLAGS)" \
		wails build -debug -o tagfile-gui && \
		mkdir -p ../../$(BUILD_DIR) && cp build/bin/TagFile.app/Contents/MacOS/tagfile-gui ../../$(BUILD_DIR)/tagfile-gui

build-gui-macos-prod:
	cd cmd/gui && CGO_CFLAGS="$(MPV_CFLAGS)" CGO_LDFLAGS="$(MPV_LDFLAGS)" \
		wails build -clean -o tagfile-gui && \
		bash ../../scripts/bundle-mpv-macos.sh build/bin/TagFile.app && \
		mkdir -p ../../$(BUILD_DIR) && cp build/bin/TagFile.app/Contents/MacOS/tagfile-gui ../../$(BUILD_DIR)/tagfile-gui

run-gui:
	./$(GUI)
