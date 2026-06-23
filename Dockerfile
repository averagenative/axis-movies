# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.26-alpine AS build
WORKDIR /src

# Cache modules first.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown
ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath \
    -ldflags="-s -w \
      -X github.com/averagenative/axis-movies/internal/version.Version=${VERSION} \
      -X github.com/averagenative/axis-movies/internal/version.Commit=${COMMIT} \
      -X github.com/averagenative/axis-movies/internal/version.Date=${DATE}" \
    -o /out/axis-movies ./cmd/axis-movies

# ---- runtime stage ----
FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/axis-movies /usr/local/bin/axis-movies
EXPOSE 7878
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/axis-movies"]
