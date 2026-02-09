.PHONY: build build-web test lint fmt vet check conformance setup clean tidy e2e

BINARY := build/hubspot

build-web:
	cd web && npm ci && npm run build

build: build-web
	go build -o $(BINARY) ./cmd/hubspot

test:
	go test -race -count=1 ./...

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run ./...

fmt:
	@test -z "$$(gofmt -l .)" || (echo "files not formatted:"; gofmt -l .; exit 1)

vet:
	go vet ./...

conformance:
	go test -v -count=1 ./tests/conformance/...

check: fmt vet lint test

setup: tidy
	git config core.hooksPath .githooks
	@echo "hooks activated and tools installed"

clean:
	rm -rf build/

tidy:
	go mod tidy

e2e:
	cd web && npx playwright test --config=playwright.config.ts
