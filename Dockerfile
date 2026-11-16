# ==============================================================================
# Jynko Unified Production Container
# Multi-stage build combining Angular SPA and Go HTTP Server
# ==============================================================================

# Stage 1: Build Angular Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build -- --configuration=production

# Stage 2: Build Go Backend
FROM golang:1.27-alpine AS backend-builder
WORKDIR /app/backend

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /jynko-server ./cmd/server/main.go

# Stage 3: Minimal Non-Root Production Container
FROM alpine:3.20
ARG BUILD_SHA=""
ARG BUILD_BRANCH=""
RUN apk --no-cache add ca-certificates tzdata \
    && addgroup -S jynko \
    && adduser -S -u 10001 -G jynko jynko

WORKDIR /app
COPY --from=backend-builder /jynko-server ./server
COPY --from=frontend-builder /app/frontend/dist/jynko-frontend/browser ./web
RUN mkdir -p /app/data/attachments && chown -R jynko:jynko /app

USER jynko

ENV PORT=8080
ENV STATIC_DIR=/app/web
ENV ENV=prd
ENV BUILD_SHA=${BUILD_SHA}
ENV BUILD_BRANCH=${BUILD_BRANCH}
EXPOSE 8080

CMD ["./server"]
