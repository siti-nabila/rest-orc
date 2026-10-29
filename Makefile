APP_NAME := rest-orc
BIN_DIR := bin
MAIN_FILE := cmd/api/main.go
BINARY := $(BIN_DIR)/$(APP_NAME)

.PHONY: all build run clean

all: clean run

build:
	@echo "🚧 Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BINARY) $(MAIN_FILE)

run: build
	@echo "🚀 Running $(APP_NAME)..."
	./$(BINARY)

clean:
	@echo "🧹 Cleaning build..."
	@rm -f $(BINARY)
