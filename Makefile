.PHONY: build dev test clean

WAILS_DIR := cmd/desktop

build:
	cd $(WAILS_DIR) && wails build

dev:
	cd $(WAILS_DIR) && wails dev

test:
	go test ./...

clean:
	rm -rf $(WAILS_DIR)/build/bin
	rm -rf $(WAILS_DIR)/frontend/dist
	rm -f desktop
