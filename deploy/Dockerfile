# Stage 1: Build Go binary
FROM golang:1.25-alpine AS go-builder
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
RUN GOPROXY=https://goproxy.cn,direct go mod download
COPY . .
COPY web/dist ./cmd/engine/dist
ARG VERSION=dev
ARG COMMIT=unknown
RUN CGO_ENABLED=0 go build -ldflags "-X main.version=$VERSION -X main.commit=$COMMIT" -o /engine ./cmd/engine/

# Stage 2: Runtime
FROM alpine:latest
RUN apk add --no-cache ffmpeg ca-certificates tzdata
COPY --from=go-builder /engine /engine
COPY configs/config.example.yaml /config.yaml
EXPOSE 8080
ENTRYPOINT ["/engine", "-config", "/config.yaml"]
