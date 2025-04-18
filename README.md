# 📬 blueshorts

**blueshorts** is a lightweight Go web application that exposes IMAP mail
folders as JSON feeds (JSON Feed spec v1.1).

## Features

- Reads configuration from a TOML file (`/data/config.toml`).
- Connects to IMAP server, selects folders, and fetches messages.
- Parses email bodies, exposing both **content_html** and **content_text**
  fields.
- Serves JSON feeds under `/feeds/{folder}.json?key={api_key}`.
- In-memory TTL cache for faster repeated requests.
- Runs in Docker with zero local Go installation.

## JSON Feed Spec

This application outputs feeds conforming to
[JSON Feed version 1.1](https://jsonfeed.org/version/1.1/). Each feed item
includes:

- `id`: unique message ID
- `title`: email subject
- `date_published`: email date
- `content_html` (optional): HTML body if available
- `content_text` (optional): plaintext body fallback

## Configuration

Create a TOML file at `/data/config.toml` (this is not currently configurable...
pull requests welcome!) with the following structure:

```toml
[server]
api_key = "your-secret-api-key"

[imap]
host     = "imap.example.com"
port     = 993
username = "you@example.com"
password = "hunter2"

[feeds]
news   = "Newsletters"
alerts = "Notifications/Alerts"
```

- **server.api_key**: required query parameter to authenticate API requests.
- **imap**: connection settings for your IMAP server.
- **feeds**: map of feed names to IMAP folder paths.

## Usage

### Docker

1. **Build the Docker image** (from project root):

   ```bash
   docker build -t blueshorts .
   ```

2. **Run the container**, mounting your config directory:

   ```bash
   docker run -d \
     -v "$(pwd)/data:/data:ro" \
     -p 8080:8080 \
     blueshorts
   ```

3. **Access a feed**:

   ```bash
   curl "http://localhost:8080/feeds/inbox.json?key=your-secret-api-key"
   ```

### Docker Compose

```yaml
services:
  blueshorts:
    image: ghcr.io/philtr/blueshorts:edge
    container_name: blueshorts
    restart: unless-stopped
    ports:
      - 8080:8080
    volumes:
      - ./data:/data
```

### Local Development (requires Go)

```bash
# install deps and run
go mod tidy
go run main.go
```

Then visit `http://localhost:8080/feeds/{feed}.json?key={api_key}`.

## Endpoints

- `GET /feeds/{feed}.json?key={api_key}`
  - **200**: JSON feed
  - **403**: missing or invalid API key
  - **404**: unknown feed name

## Caching

Feed data is cached in-memory for 5 minutes by default. Subsequent requests
within TTL serve cached results.

## Logging

Server logs are printed to stdout, including startup info, errors, and fetch
operations.

## License

MIT License — see [LICENSE](LICENSE) for details.
