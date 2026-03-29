FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod main.go ./
RUN go build -o proxy .

FROM ollama/ollama:latest
RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*
COPY --from=builder /build/proxy /usr/local/bin/proxy
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENV OLLAMA_MODELS=/data
ENTRYPOINT ["/entrypoint.sh"]
