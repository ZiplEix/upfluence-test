# Build stage
FROM golang:alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates git

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o /app/bin/server main.go

# Production stage
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata \
    && addgroup -S appgroup && adduser -S -G appgroup appuser

COPY --from=builder /app/bin/server /app/bin/server

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/bin/server"]
