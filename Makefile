BIN     := bin/zick
INSTALL := $(HOME)/.local/bin/zick

.PHONY: build install test lint clean docs

build:
	go build -o $(BIN) ./cmd/zick

install: build
	cp $(BIN) $(INSTALL)

test:
	go test ./...

lint:
	golangci-lint run

docs:
	go run ./cmd/docgen -out ./docs/cli -frontmatter

clean:
	rm -rf bin/
