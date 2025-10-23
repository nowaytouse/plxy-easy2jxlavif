SHELL := /bin/bash

.PHONY: all build fmt tidy test-mini release clean

all: build

fmt:
	@echo "==> go fmt"
	@go fmt ./...

tidy:
	@echo "==> go mod tidy"
	@go mod tidy

build: fmt tidy
	@echo "==> building easymode"
	@go build -o ./easymode ./cmd/easymode

test-mini: build
	@echo "==> running mini test (dry-run)"
	@./easymode -input mini_test -output mini_test/_out_jxl -type jxl -dry-run -report json -report-path mini_test/_report_jxl_dry.json || true
	@echo "==> running mini test (JXL)"
	@./easymode -input mini_test -output mini_test/_out_jxl -type jxl -workers 2 -report json -report-path mini_test/_report_jxl.json || true
	@echo "==> running mini test (AVIF)"
	@./easymode -input mini_test -output mini_test/_out_avif -type avif -workers 2 -report json -report-path mini_test/_report_avif.json || true

release: build
	@echo "==> release artifact at ./easymode"

clean:
	@rm -f ./easymode
	@rm -rf mini_test/_out_* mini_test/_report_* || true


