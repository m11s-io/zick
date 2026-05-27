BIN := bin/zick

.PHONY: build test lint clean

build:
	go build -o $(BIN) ./cmd/zick

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/
