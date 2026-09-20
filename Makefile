.PHONY: build dev test clean

WAILS_DIR := cmd/desktop

build:
	cd $(WAILS_DIR) && wails build

dev:
	cd $(WAILS_DIR) && wails dev

test:
	go test ./...

# frontend/dist/.gitkeep must survive: go:embed in cmd/desktop needs at least
# one file there, or `go build ./...` fails on a clean tree.
clean:
	rm -rf $(WAILS_DIR)/build/bin
	-find $(WAILS_DIR)/frontend/dist -mindepth 1 ! -name .gitkeep -delete
	rm -f desktop
