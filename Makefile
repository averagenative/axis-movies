VERSION ?= dev
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
  -X github.com/averagenative/axis-movies/internal/version.Version=$(VERSION) \
  -X github.com/averagenative/axis-movies/internal/version.Commit=$(COMMIT) \
  -X github.com/averagenative/axis-movies/internal/version.Date=$(DATE)

.PHONY: build test vet fmt lint run docker tidy clean

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
