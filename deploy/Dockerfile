# Stage 1: Build frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.25-alpine AS go-builder
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
COPY --from=frontend-builder /app/web/dist ./cmd/engine/dist
RUN CGO_ENABLED=0 go build -ldflags "-X main.version=$VERSION -X main.commit=$COMMIT" -o /engine ./cmd/engine/

# Stage 3: Runtime
FROM alpine:3.20
RUN apk add --no-cache ffmpeg ca-certificates tzdata
COPY --from=go-builder /engine /engine
COPY configs/config.example.yaml /config.yaml
EXPOSE 8080
ENTRYPOINT ["/engine", "-config", "/config.yaml"]
