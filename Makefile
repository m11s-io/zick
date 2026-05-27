BIN := bin/zick

.PHONY: build test lint clean docs

build:
	go build -o $(BIN) ./cmd/zick

test:
	go test ./...

lint:
	golangci-lint run

docs:
	go run ./cmd/docgen -out ./docs/cli -frontmatter

clean:
	rm -rf bin/
