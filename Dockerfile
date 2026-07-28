# syntax=docker/dockerfile:1
ARG GO_VERSION=1.26.5
FROM golang:${GO_VERSION}-bookworm AS build

WORKDIR /src
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -trimpath -buildvcs=false -ldflags="-s -w \
  -X github.com/iuoow/OpenDroneOps/internal/buildinfo.Version=${VERSION} \
  -X github.com/iuoow/OpenDroneOps/internal/buildinfo.Commit=${COMMIT} \
  -X github.com/iuoow/OpenDroneOps/internal/buildinfo.BuildTime=${BUILD_TIME}" \
  -o /out/server ./cmd/server && \
  go build -trimpath -buildvcs=false -ldflags="-s -w \
  -X github.com/iuoow/OpenDroneOps/internal/buildinfo.Version=${VERSION} \
  -X github.com/iuoow/OpenDroneOps/internal/buildinfo.Commit=${COMMIT} \
  -X github.com/iuoow/OpenDroneOps/internal/buildinfo.BuildTime=${BUILD_TIME}" \
  -o /out/worker ./cmd/worker && \
  go build -trimpath -buildvcs=false -ldflags="-s -w \
  -X github.com/iuoow/OpenDroneOps/internal/buildinfo.Version=${VERSION} \
  -X github.com/iuoow/OpenDroneOps/internal/buildinfo.Commit=${COMMIT} \
  -X github.com/iuoow/OpenDroneOps/internal/buildinfo.BuildTime=${BUILD_TIME}" \
  -o /out/migrate ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot AS server
COPY --from=build /out/server /app/server
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]

FROM gcr.io/distroless/static-debian12:nonroot AS worker
COPY --from=build /out/worker /app/worker
USER nonroot:nonroot
ENTRYPOINT ["/app/worker"]

FROM gcr.io/distroless/static-debian12:nonroot AS migrate
COPY --from=build /out/migrate /app/migrate
USER nonroot:nonroot
ENTRYPOINT ["/app/migrate"]
