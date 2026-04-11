# Build stage for frontend
FROM oven/bun:1 AS frontend-builder

WORKDIR /app

# Copy the entire project to get correct relative paths for svelte.config.js
COPY frontend/package.json frontend/bun.lock* ./frontend/
RUN cd frontend && bun install --frozen-lockfile

COPY frontend/ ./frontend/
COPY assets/ ./assets/
RUN cd frontend && bun run build

# Build stage for backend
FROM golang:1.25-alpine AS backend-builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download

COPY backend/ ./backend/
# Copy frontend build output from the correct location
COPY --from=frontend-builder /app/backend/cmd/openpost/public ./backend/cmd/openpost/public

RUN cd backend && CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o openpost ./cmd/openpost

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata sqlite

RUN adduser -D -u 1000 openpost

WORKDIR /app

COPY --from=backend-builder /app/backend/openpost .

RUN mkdir -p /data/media /data/db && chown -R openpost:openpost /data

USER openpost

ENV OPENPOST_DB_PATH=/data/db/openpost.db
ENV OPENPOST_MEDIA_PATH=/data/media
ENV OPENPOST_PORT=8080

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

CMD ["./openpost"]