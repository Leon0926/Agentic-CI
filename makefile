BINARY  := reviewd
BIN_DIR := bin

.PHONY: build test vet run clean

build:
	go build -o $(BIN_DIR)/$(BINARY) ./cmd/reviewd

test:
	go test -race ./...

vet:
	go vet ./...

run: build
	./$(BIN_DIR)/$(BINARY) run

clean:
	rm -rf $(BIN_DIR)