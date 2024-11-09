# Paste69

[![CI](https://github.com/watzon/paste69/actions/workflows/ci.yml/badge.svg)](https://github.com/watzon/paste69/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/watzon/paste69)](https://goreportcard.com/report/github.com/watzon/paste69)
[![GoDoc](https://godoc.org/github.com/watzon/paste69?status.svg)](https://godoc.org/github.com/watzon/paste69)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A modern, Go-based pastebin service inspired by 0x0.st.

## Features

- File uploads and URL shortening
- Multiple storage providers (local and S3)
- Simple and clean API
- Docker support
- Configurable through environment variables

## Installation

### Prerequisites

- Go 1.21+
- The desire to live dangerously

### Local Setup

```bash
# Clone the repository
git clone https://github.com/watzon/paste69
cd paste69

# Install dependencies
go mod download

# Copy and configure the environment
cp .env.example .env
# Edit .env with your settings

# Run migrations and start the server
go run cmd/migrate/main.go
go run cmd/server/main.go
```

### Docker

Build and run locally:

```bash
docker build -v ./uploads:/app/uploads --tag paste69 ./docker
docker run -d -p 8080:8080 paste69
```

Or use the pre-built image:

```bash
docker pull ghcr.io/watzon/paste69:main
docker run -d -p 8080:8080 ghcr.io/watzon/paste69:main
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -am 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. 