.PHONY: test clean deps lint

# Go parameters
GOCMD=go
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod

all: deps test

test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

test-cv:
	@echo "Running CV tests..."
	$(GOTEST) -v ./pkg/vision/cv/...

test-ocr:
	@echo "Running OCR tests..."
	$(GOTEST) -v ./pkg/vision/ocr/...

test-vision:
	@echo "Running vision tests..."
	$(GOTEST) -v ./pkg/vision/...

clean:
	@echo "Cleaning..."
	$(GOCLEAN)

deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

lint:
	@echo "Linting..."
	golangci-lint run ./...

# Install development dependencies
dev-deps:
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Download OCR models
download-models:
	@echo "Downloading OCR models..."
	@mkdir -p models
	@if [ ! -d "models/paddle_weights" ]; then \
		git clone https://huggingface.co/getcharzp/go-ocr models/go-ocr-models; \
		mv models/go-ocr-models/* models/; \
		rm -rf models/go-ocr-models; \
	fi

help:
	@echo "Available targets:"
	@echo "  test           - Run all tests"
	@echo "  test-cv        - Run CV module tests"
	@echo "  test-ocr       - Run OCR module tests"
	@echo "  test-vision    - Run vision module tests"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Download dependencies"
	@echo "  lint           - Run linter"
	@echo "  dev-deps       - Install development dependencies"
	@echo "  download-models - Download OCR models"
