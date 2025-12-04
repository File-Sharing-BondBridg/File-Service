FROM golang:1.25.2 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .

# Datadog settings
ENV DD_ENV=staging \
    DD_SERVICE=file-service \
    DD_VERSION=1.0.0 \
    DD_TRACE_ENABLED=true \
    DD_AGENT_HOST=datadog-agent \
    DD_TRACE_AGENT_PORT=8126

EXPOSE 8080
CMD ["./main"]
