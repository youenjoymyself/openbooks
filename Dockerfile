# Web assets are platform independent, so always build them on the build platform.
FROM --platform=$BUILDPLATFORM node:24-alpine AS web
WORKDIR /web
COPY server/app/package.json server/app/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY server/app/ ./
RUN npm run build

# Cross compile the Go binary for the target platform.
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /web/dist ./server/app/dist
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GOARM=${TARGETVARIANT#v} \
    go build -trimpath -ldflags="-s -w" -o /out/openbooks ./cmd/openbooks

FROM gcr.io/distroless/static AS app
WORKDIR /app
COPY --from=build /out/openbooks .

EXPOSE 80
VOLUME [ "/books" ]
ENV BASE_PATH=/

ENTRYPOINT ["./openbooks", "server", "--dir", "/books", "--port", "80"]
