VERSION ?= dev
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
  -X github.com/averagenative/axis-movies/internal/version.Version=$(VERSION) \
  -X github.com/averagenative/axis-movies/internal/version.Commit=$(COMMIT) \
  -X github.com/averagenative/axis-movies/internal/version.Date=$(DATE)

.PHONY: build test vet fmt fmt-check lint run docker tidy clean check install-hooks

install-hooks:
	git config core.hooksPath scripts/hooks
	chmod +x scripts/hooks/*
	@echo "git hooks installed (pre-push runs 'make check')"

# Local quality gate — run before committing/pushing. There is no CI; this is it.
check: fmt-check vet test build
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || \
		echo "note: golangci-lint not installed, skipping (go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest)"
	@echo "check: OK"

fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then echo "not gofmt-clean:"; echo "$$unformatted"; exit 1; fi

build:
	go build -trimpath -ldflags="$(LDFLAGS)" -o bin/axis-movies ./cmd/axis-movies

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

lint:
	golangci-lint run

run: build
	./bin/axis-movies

docker:
	docker build -t axis-movies:$(VERSION) .

tidy:
	go mod tidy

clean:
	rm -rf bin
