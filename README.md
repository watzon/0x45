# Paste69

<div align="center">
    <img src="https://github.com/watzon/0x45/blob/main/public/images/0x45-og.png?raw=true" alt="0x45" width="300">
</div>

[![CI](https://github.com/watzon/0x45/actions/workflows/ci.yml/badge.svg)](https://github.com/watzon/0x45/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/watzon/0x45)](https://goreportcard.com/report/github.com/watzon/0x45)
[![GoDoc](https://godoc.org/github.com/watzon/0x45?status.svg)](https://godoc.org/github.com/watzon/0x45)
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
git clone https://github.com/watzon/0x45.git
cd paste69

# Install dependencies
go mod download

# Configure settings
vim config/config.yaml

# Start the server (migrations will be run automatically)
go run cmd/server/main.go
```

Alternatively for speedier local development, you can install the [air](https://github.com/cosmtrek/air) package and run:

```bash
air
```

### Docker

Build and run locally:

```bash
docker build -v ./uploads:/app/uploads --tag 0x45 ./docker
docker run -d -p 3000:3000 0x45
```

Or use the pre-built image:

```bash
docker pull ghcr.io/watzon/0x45:main
docker run -d -p 3000:3000 ghcr.io/watzon/0x45:main
```

## Configuration

Configuration is done through the `config/config.yaml` file. See the [example config](config/config.yaml) for more information on the available options. When running in a docker container it might be more convenient to set the environment variables instead, here is a list of available options:

### Database Configuration
Controls the database connection settings.

| Environment Variable | Description                              | Default    |
| -------------------- | ---------------------------------------- | ---------- |
| 0X_DATABASE_DRIVER   | Database driver to use (sqlite/postgres) | sqlite     |
| 0X_DATABASE_HOST     | Database host                            | localhost  |
| 0X_DATABASE_PORT     | Database port                            | 5432       |
| 0X_DATABASE_USER     | Database username                        | ""         |
| 0X_DATABASE_PASSWORD | Database password                        | ""         |
| 0X_DATABASE_NAME     | Database name                            | paste69.db |
| 0X_DATABASE_SSLMODE  | SSL mode for postgres                    | disable    |

### Storage Configuration
Configure one or more storage backends for file storage. Multiple backends can be configured using numbered environment variables (0-9).

| Environment Variable     | Description                                 | Default   |
| ------------------------ | ------------------------------------------- | --------- |
| 0X_STORAGE_0_NAME        | First storage backend name                  | local     |
| 0X_STORAGE_0_TYPE        | First storage type (local/s3)               | local     |
| 0X_STORAGE_0_DEFAULT     | First storage is default                    | true      |
| 0X_STORAGE_0_PATH        | First local storage path                    | ./uploads |
| 0X_STORAGE_0_S3_BUCKET   | First S3 bucket name                        | ""        |
| 0X_STORAGE_0_S3_REGION   | First S3 region                             | ""        |
| 0X_STORAGE_0_S3_KEY      | First S3 access key                         | ""        |
| 0X_STORAGE_0_S3_SECRET   | First S3 secret key                         | ""        |
| 0X_STORAGE_0_S3_ENDPOINT | First S3 endpoint                           | ""        |
| 0X_STORAGE_1_NAME        | Second storage backend name                 | ""        |
| ...                      | (and so on for STORAGE_1 through STORAGE_9) |           |

### Server Configuration
Core server settings and behavior.

| Environment Variable      | Description                  | Default |
| ------------------------- | ---------------------------- | ------- |
| 0X_SERVER_ADDRESS         | Server listen address        | :3000   |
| 0X_SERVER_BASE_URL        | Base URL for the server      | ""      |
| 0X_SERVER_MAX_UPLOAD_SIZE | Maximum upload size in bytes | 5242880 |
| 0X_SERVER_PREFORK         | Enable prefork mode          | false   |
| 0X_SERVER_SERVER_HEADER   | Server header value          | Paste69 |
| 0X_SERVER_APP_NAME        | Application name             | Paste69 |

### Cleanup Configuration
Settings for automatic content cleanup.

| Environment Variable       | Description                 | Default |
| -------------------------- | --------------------------- | ------- |
| 0X_SERVER_CLEANUP_ENABLED  | Enable automatic cleanup    | true    |
| 0X_SERVER_CLEANUP_INTERVAL | Cleanup interval in seconds | 3600    |
| 0X_SERVER_CLEANUP_MAX_AGE  | Maximum age for content     | 168h    |

### Rate Limiting Configuration
Controls rate limiting behavior.

| Environment Variable                     | Description                 | Default |
| ---------------------------------------- | --------------------------- | ------- |
| 0X_SERVER_RATE_LIMIT_GLOBAL_ENABLED      | Enable global rate limiting | true    |
| 0X_SERVER_RATE_LIMIT_GLOBAL_RATE         | Global requests per second  | 100.0   |
| 0X_SERVER_RATE_LIMIT_GLOBAL_BURST        | Global burst size           | 50      |
| 0X_SERVER_RATE_LIMIT_PER_IP_ENABLED      | Enable per-IP rate limiting | true    |
| 0X_SERVER_RATE_LIMIT_PER_IP_RATE         | Per-IP requests per second  | 1.0     |
| 0X_SERVER_RATE_LIMIT_PER_IP_BURST        | Per-IP burst size           | 5       |
| 0X_SERVER_RATE_LIMIT_USE_REDIS           | Use Redis for rate limiting | false   |
| 0X_SERVER_RATE_LIMIT_IP_CLEANUP_INTERVAL | IP cleanup interval         | 1h      |

### SMTP Configuration
Email sending configuration.

| Environment Variable | Description               | Default |
| -------------------- | ------------------------- | ------- |
| 0X_SMTP_ENABLED      | Enable SMTP functionality | false   |
| 0X_SMTP_HOST         | SMTP server host          | ""      |
| 0X_SMTP_PORT         | SMTP server port          | 587     |
| 0X_SMTP_USERNAME     | SMTP username             | ""      |
| 0X_SMTP_PASSWORD     | SMTP password             | ""      |
| 0X_SMTP_FROM         | From email address        | ""      |
| 0X_SMTP_FROM_NAME    | From name                 | Paste69 |
| 0X_SMTP_STARTTLS     | Use STARTTLS              | true    |

### Redis Configuration
Redis connection settings.

| Environment Variable | Description           | Default        |
| -------------------- | --------------------- | -------------- |
| 0X_REDIS_ENABLED     | Enable Redis          | false          |
| 0X_REDIS_ADDRESS     | Redis server address  | localhost:6379 |
| 0X_REDIS_PASSWORD    | Redis password        | ""             |
| 0X_REDIS_DB          | Redis database number | 0              |

### Retention Configuration
Content retention policy settings.

| Environment Variable          | Description                        | Default |
| ----------------------------- | ---------------------------------- | ------- |
| 0X_RETENTION_NO_KEY_MIN_AGE   | Minimum retention days without key | 7.0     |
| 0X_RETENTION_NO_KEY_MAX_AGE   | Maximum retention days without key | 128.0   |
| 0X_RETENTION_WITH_KEY_MIN_AGE | Minimum retention days with key    | 30.0    |
| 0X_RETENTION_WITH_KEY_MAX_AGE | Maximum retention days with key    | 730.0   |
| 0X_RETENTION_POINTS           | Number of retention curve points   | 50      |


## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -am 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. 