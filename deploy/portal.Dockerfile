FROM --platform=$BUILDPLATFORM node:20-alpine AS spa
WORKDIR /spa
COPY portal/package.json portal/package-lock.json* ./
RUN npm install --no-audit --no-fund
COPY portal ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=spa /pkg/portal/server/dist ./pkg/portal/server/dist
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w" \
    -o portal ./cmd/portal/...

FROM gcr.io/distroless/static:nonroot@sha256:963fa6c544fe5ce420f1f54fb88b6fb01479f054c8056d0f74cc2c6000df5240
WORKDIR /
COPY --from=builder /workspace/portal /portal
USER 65532:65532
ENTRYPOINT ["/portal"]
