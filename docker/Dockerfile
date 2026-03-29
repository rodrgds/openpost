# Build stage for frontend
FROM oven/bun:1 AS frontend-builder

WORKDIR /app

# Copy the entire project to get correct relative paths for svelte.config.js
COPY web/package.json web/bun.lock* ./web/
RUN cd web && bun install --frozen-lockfile

COPY web/ ./web/
RUN cd web && bun run build

# Build stage for backend
FROM golang:1.24-alpine AS backend-builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
# Copy frontend build output from the correct location
COPY --from=frontend-builder /app/backend/cmd/openpost/public ./cmd/openpost/public

RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o openpost ./cmd/openpost

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata sqlite

RUN adduser -D -u 1000 openpost

WORKDIR /app

COPY --from=backend-builder /app/openpost .

RUN mkdir -p /data/media /data/db && chown -R openpost:openpost /data

USER openpost

ENV OPENPOST_DB_PATH=/data/db/openpost.db
ENV OPENPOST_MEDIA_PATH=/data/media
ENV OPENPOST_PORT=8080

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

CMD ["./openpost"]