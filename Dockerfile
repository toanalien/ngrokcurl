# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o file-transfer .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget tar bash

WORKDIR /app

# Copy binary
COPY --from=builder /app/file-transfer .

# Install ngrok
RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then NGROK_ARCH="amd64"; \
    elif [ "$ARCH" = "aarch64" ]; then NGROK_ARCH="arm64"; \
    else NGROK_ARCH="amd64"; fi && \
    wget -q https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-linux-${NGROK_ARCH}.tgz && \
    tar xzf ngrok-v3-stable-linux-${NGROK_ARCH}.tgz && \
    mv ngrok /usr/local/bin/ && \
    rm ngrok-v3-stable-linux-${NGROK_ARCH}.tgz

# Create data directory
RUN mkdir -p /data

# Create startup script
RUN cat > /app/start.sh << 'EOF'
#!/bin/bash
echo "🚀 Starting File Transfer Service..."
./file-transfer &
sleep 2
echo "🌐 Starting ngrok..."
ngrok http 8080 --authtoken=$NGROK_AUTHTOKEN --log=stdout 2>&1 | tee /tmp/ngrok.log &
sleep 3
URL=$(grep -o 'url=https://[^ ]*' /tmp/ngrok.log 2>/dev/null | head -1 | cut -d'=' -f2 || echo "")
if [ ! -z "$URL" ]; then
    echo ""
    echo "✅ Ready! Public URL: $URL"
    echo ""
    echo "Upload: curl $URL/upload -F \"file=@file.pdf\""
    echo ""
fi
tail -f /tmp/ngrok.log
EOF

RUN chmod +x /app/start.sh

EXPOSE 8080 4040

CMD ["/app/start.sh"]