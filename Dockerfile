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
FROM golang:1.23-alpine AS backend-builder
WORKDIR /app/backend

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /jynko-server ./cmd/server/main.go

# Stage 3: Minimal Production Container
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=backend-builder /jynko-server ./server
COPY --from=frontend-builder /app/frontend/dist/jynko-frontend/browser ./web

ENV PORT=8080
ENV STATIC_DIR=/app/web
ENV ENV=production
EXPOSE 8080

CMD ["./server"]
