FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /jynko-server ./cmd/server/main.go

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata \
    && addgroup -S jynko \
    && adduser -S -u 10001 -G jynko jynko

WORKDIR /app
COPY --from=builder /jynko-server ./jynko-server
RUN mkdir -p /app/data/attachments && chown -R jynko:jynko /app

USER jynko

ENV PORT=8080
ENV ENV=prd
EXPOSE 8080
CMD ["./jynko-server"]
