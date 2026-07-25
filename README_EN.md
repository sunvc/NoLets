
# NoLetServer

![](https://status.wzs.app/api/badge/1/uptime/24?style=for-the-badge)

[中文](./README.md)

## Installation and Running

| App Store                                                                                                                                                             | Server Works  |
|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------|
| [<img src="https://developer.apple.com/assets/elements/badges/download-on-the-app-store.svg" alt="Aero App" height="40">](https://apps.apple.com/us/app/id6615073345) | [![Deploy to Cloudflare Workers](https://deploy.workers.cloudflare.com/button)](https://deploy.workers.cloudflare.com/?url=https://github.com/sunvc/NoLets-worker) |

### One-line Install (Recommended)

Linux / macOS (Docker required; the script auto-installs Docker on Linux):

```bash
curl -fsSL https://raw.githubusercontent.com/sunvc/nolets/main/install.sh | bash
```

The script will:

1. Check / install Docker (on macOS please install Docker Desktop first)
2. Write a `compose.yaml` into the working dir (Linux: `/opt/nolet`, macOS: `~/.nolet`)
3. Pull `ghcr.io/sunvc/nolets:latest` and start the `NoLets` container via `docker compose`
4. Health-check `http://127.0.0.1:8080/health`

#### Examples

```bash
# Custom dir, port, sign key and admin auth list
curl -fsSL https://raw.githubusercontent.com/sunvc/nolets/main/install.sh | bash -s -- \
    --dir /opt/nolet \
    --port 8080 \
    --sign-key "your-sign-key" \
    --auths '["uid1","uid2"]'

# Or via environment variables
NOLET_PORT=9090 NOLET_SIGN_KEY=xxx TZ=Asia/Shanghai \
    bash -c "$(curl -fsSL https://raw.githubusercontent.com/sunvc/nolets/main/install.sh)"

# Uninstall (removes the container, keeps the data directory)
curl -fsSL https://raw.githubusercontent.com/sunvc/nolets/main/install.sh | bash -s -- --uninstall
```

#### Flags

| Flag | Env | Description | Default |
|------|-----|-------------|---------|
| `--dir` | `NOLET_DIR` | Working directory (compose.yaml + data) | Linux `/opt/nolet` · macOS `~/.nolet` |
| `--port` | `NOLET_PORT` | Host port | `8080` |
| `--image` | `NOLET_IMAGE` | Docker image | `ghcr.io/sunvc/nolets:latest` |
| `--sign-key` | `NOLET_SIGN_KEY` | Sign key |  |
| `--auths` | `NOLET_AUTHS` | Admin ID list, e.g. `'["uid1","uid2"]'` |  |
| `--tz` | `TZ` | Timezone | `Asia/Shanghai` |
| `--name` | `NOLET_CONTAINER` | Container name | `NoLets` |
| `--uninstall` |  | Remove the container |  |

### Download from GitHub Releases

You can download pre-compiled binaries from the GitHub Releases page:

1. Visit the [GitHub Releases](https://github.com/sunvc/NoLetserver/releases) page
2. Choose the appropriate version for your operating system and architecture:
   - Windows (amd64)
   - macOS (amd64, arm64)
   - Linux (amd64, arm64, mips64, mips64le)
   - FreeBSD (amd64, arm64)
3. Extract the downloaded file
4. Create a configuration file (refer to the configuration instructions below)
5. Run the program:
   ```bash
   # Linux/macOS
   ./NoLets --config your_config.yaml
   
   # Windows
   NoLets.exe --config your_config.yaml
   ```

   Common parameters:
   - `--addr`: Server listening address, default is 0.0.0.0:8080
   - `--url-prefix`: Service URL prefix, default is /
   - `--dir`: Data storage directory, default is ./data
   - `--dsn`: MySQL database connection string
   - `--debug`: Enable debug mode
   - `--config, -c`: Specify configuration file path

### Using Docker

#### Docker Image

This project provides the following Docker image addresses:

- Docker Hub: `sunvc/nolets:latest`
- GitHub Container Registry: `ghcr.io/sunvc/nolets:latest`

You can pull the image using the following command:

```bash
# Pull from Docker Hub
docker pull sunvc/nolets:latest

# Or pull from GitHub Container Registry
docker pull ghcr.io/sunvc/nolets:latest

docker run -d --name NoLet-server \
  -p 8080:8080 \
  -v ./data:/data \
  --restart=always \
  ghcr.io/sunvc/nolets:latest
```

#### Using Docker Compose

The `compose.yaml` file in the project root directory is already configured to use the Docker image:

```yaml
services:
  NoLetServer:
    image: ghcr.io/sunvc/nolets:latest
    container_name: NoLets
    restart: always
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
```

Run the following command to start the service:

```bash
docker-compose up -d
```

## Configuration File

The `config.yaml` in the project is only a configuration file example. **Users need to create and specify their own configuration file** for service configuration. You can use the `--config` or `-c` parameter to specify the configuration file path.

### Configuration File Structure

```yaml
system:
  user: ""                  # Basic authentication username
  password: ""              # Basic authentication password
  push_password: ""         # Group Push Password
  addr: "0.0.0.0:8080"      # Server listening address
  url_prefix: "/"           # Service URL prefix
  data: "./data"            # Data storage directory
  name: "NoLets"            # Service name
  dsn: ""                   # MySQL DSN connection string
  cert: ""                  # TLS certificate path
  key: ""                   # TLS certificate private key path
  reduce_memory_usage: false # Reduce memory usage (increases CPU consumption)
  proxy_header: ""          # Remote IP address source in HTTP header
  max_batch_push_count: -1  # Maximum number of batch pushes, -1 means no limit
  max_apns_client_count: 1  # Maximum number of APNs client connections
  max_device_key_arr_length: 10    # maximum number of key lists
  concurrency: 262144       # Maximum number of concurrent connections (256 * 1024)
  read_timeout: 3s          # Read timeout
  write_timeout: 3s         # Write timeout
  idle_timeout: 10s         # Idle timeout
  admins: []                # Administrator ID list
  debug: true               # Enable debug mode
  expired: 0                # Voice expiration time (seconds)
  icp_info: ""              # ICP filing information
  time_zone: "UTC"          # Time zone setting

apple:
  apnsPrivateKey: ""        # APNs private key content or path
  topic: ""                 # APNs Topic
  keyID: ""                 # APNs Key ID
  teamID: ""                # APNs Team ID
  develop: false            # Enable APNs development environment
```

## Service Configuration Methods

The service can be configured in the following three ways, with priority from high to low:

1. **Command-line parameters**: Parameters specified at startup, highest priority
2. **Environment variables**: System environment variables, second priority
3. **Configuration file**: `config.yaml` file or configuration file specified via `--config`/`-c` parameter

### Command-line Parameters and Environment Variables


| Parameter | Environment Variable | Description | Default |
|------|----------|------|--------|
| `--addr` | `NOLET_SERVER_ADDRESS` | Server listening address | `0.0.0.0:8080` |
| `--url-prefix` | `NOLET_SERVER_URL_PREFIX` | Service URL prefix | `/` |
| `--dir` | `NOLET_SERVER_DATA_DIR` | Server data storage directory | `./data` |
| `--dsn` | `NOLET_SERVER_DSN` | MySQL DSN (user:pass@tcp(host)/dbname) |  |
| `--cert` | `NOLET_SERVER_CERT` | Server TLS certificate |  |
| `--key` | `NOLET_SERVER_KEY` | Server TLS certificate key |  |
| `--reduce-memory-usage` | `NOLET_SERVER_REDUCE_MEMORY_USAGE` | Reduce memory usage (increases CPU consumption) | `false` |
| `--user`, `-u` | `NOLET_SERVER_BASIC_AUTH_USER` | Basic authentication username |  |
| `--password`, `-p` | `NOLET_SERVER_BASIC_AUTH_PASSWORD` | Basic authentication password |  |
| `--push-password` | `NOLET_PUSH_PASSWORD` | Push authentication password |  |
| `--sign-key`, `--sk` | `NOLET_SIGN_KEY` | Application signature key |  |
| `--proxy-header` | `NOLET_SERVER_PROXY_HEADER` | Remote IP address field in proxy header |  |
| `--max-batch-push-count` | `NOLET_SERVER_MAX_BATCH_PUSH_COUNT` | Maximum batch push count, `-1` means unlimited | `-1` |
| `--max-apns-client-count`, `--max` | `NOLET_SERVER_MAX_APNS_CLIENT_COUNT` | Maximum APNs client connections | `1` |
| `--max-device-key-arr-length` | `NOLET_CONCURRENCY` | Maximum device key list length | `10` |
| `--concurrency` | `NOLET_SERVER_CONCURRENCY` | Maximum concurrent connections | `262144` |
| `--read-timeout` | `NOLET_SERVER_READ_TIMEOUT` | Read request timeout | `3s` |
| `--write-timeout` | `NOLET_SERVER_WRITE_TIMEOUT` | Response write timeout | `3s` |
| `--idle-timeout` | `NOLET_SERVER_IDLE_TIMEOUT` | Keep-Alive idle timeout | `10s` |
| `--debug` | `NOLET_DEBUG` | Enable debug mode | `false` |
| `--voice` | `NOLET_VOICE` | Enable voice support | `false` |
| `--auths` | `NOLET_AUTHS` | Authorization ID list |  |
| `--apns-private-key` | `NOLET_APPLE_APNS_PRIVATE_KEY` | APNs private key path | Built-in default |
| `--topic` | `NOLET_APPLE_TOPIC` | APNs Topic | `me.uuneo.Meoworld` |
| `--key-id` | `NOLET_APPLE_KEY_ID` | APNs Key ID | `BNY5GUGV38` |
| `--team-id` | `NOLET_APPLE_TEAM_ID` | APNs Team ID | `FUWV6U942Q` |
| `--develop`, `--dev` | `NOLET_APPLE_DEVELOP` | Use APNs development environment | `false` |
| `--Expired`, `--ex` | `NOLET_EXPIRED_TIME` | Voice expiration time (seconds) | `120` |
| `--ICP`, `--icp` | `NOLET_ICP_INFO` | ICP filing information |  |
| `--config`, `-c` |  | Configuration file path |  |
| `--proxy-download`, `--dp` | `NOLET_PROXY_DOWNLOAD` | Enable proxy download | `false` |
| `--export-path`, `--dc` | `NOLET_EXPORT_PATH` | Export database path |  |
| `--import-path`, `--dl` | `NOLET_IMPORT_PATH` | Import database path |  |
| `--build-test` |  | Build test mode |  |

### Using Configuration File

1. Create your own configuration file:
   - Create your own configuration file referring to the `config.yaml` example in the project
   - Ensure the configuration file contains the required configuration items

2. Specify the configuration file path:
   ```bash
   ./NoLets --config /path/to/your/config.yaml
   # Or use the shorthand
   ./NoLets -c /path/to/your/config.yaml
   ```

3. Mixed use of configuration file and command-line parameters:
   ```bash
   # Settings in the configuration file will be overridden by command-line parameters
   ./NoLets -c /path/to/your/config.yaml --debug --addr 127.0.0.1:8080
   ```

