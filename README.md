# File Transfer Service

Simple HTTP file transfer service with ngrok integration - like transfer.sh.

## Quick Start

### 1. Build
```bash
docker build -t file-transfer .
```

### 2. Get Ngrok Token
Get your free token: https://dashboard.ngrok.com/get-started/your-authtoken

### 3. Run
```bash
docker run -it --rm \
  -p 8080:8080 -p 4040:4040 \
  -v $(pwd)/data:/data \
  -e NGROK_AUTHTOKEN=your_token_here \
  file-transfer
```

The public URL will be displayed automatically!

## Usage

### Upload
```bash
curl https://your-url.ngrok-free.app/upload -F "file=@myfile.pdf"
```

Response:
```json
{"id":"abc123","filename":"myfile.pdf","size":1024,"url":"https://your-url.ngrok-free.app/files/abc123"}
```

### Download

**With original filename preserved:**
```bash
curl https://your-url.ngrok-free.app/files/abc123 -O -J
```

**Or specify output filename:**
```bash
curl https://your-url.ngrok-free.app/files/abc123 -o myfile.pdf
```

**Or open in browser:**
```
https://your-url.ngrok-free.app/files/abc123
```

The original filename is automatically preserved when downloading!

## File Storage

Files are stored in `./data` directory with format: `{id}_{original_filename}`

Example: `abc123def456_document.pdf`

```bash
ls -lh data/
```

When downloading, the original filename is automatically restored.

## Run in Background

```bash
docker run -d \
  -p 8080:8080 -p 4040:4040 \
  -v $(pwd)/data:/data \
  -e NGROK_AUTHTOKEN=your_token \
  --name file-transfer \
  file-transfer

# View logs to get URL
docker logs file-transfer

# Stop
docker stop file-transfer && docker rm file-transfer
```

## Without Docker

```bash
# Start server
go run main.go

# In another terminal
ngrok http 8080 --authtoken your_token
```

## Features

- Simple HTTP API (Go stdlib only)
- Ngrok integration for public access
- Unique IDs for each file
- **Original filenames preserved on download**
- 100MB file size limit
- Local storage with Docker volume
- Web interface at root URL

## API

- `GET /` - Web interface
- `POST /upload` - Upload file (multipart/form-data)
- `GET /files/{id}` - Download file
- `GET /health` - Health check

## Configuration

Environment variables:
- `NGROK_AUTHTOKEN` - Required for ngrok
- Files stored in `/data` inside container

## Notes

- Use HTTPS with ngrok URLs (not HTTP)
- Files persist in mounted volume
- No built-in expiration
- For personal/testing use