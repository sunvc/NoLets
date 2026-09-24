
# NoLetServer

![](https://status.wzs.app/api/badge/1/uptime/24?style=for-the-badge)

[中文](./README.md)

## Installation and Running

### Server Works
[![Deploy to Cloudflare Workers](https://deploy.workers.cloudflare.com/button)](https://github.com/sunvc/nolets-worker)

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

harmony:
  project_id: ""            # HarmonyOS AGC project ID (used in push URL)
  key_id: ""                # Service account key ID (JWT kid)
  private_key: ""           # Service account RSA private key (PEM content, real newlines required)
  sub_account: ""           # Service / sub account (JWT iss)
  client_id: ""             # App client ID (used for message revocation)
  auth_uri: "https://oauth-login.cloud.huawei.com/oauth2/v3/authorize"
  token_uri: "https://oauth-login.cloud.huawei.com/oauth2/v3/token"
  auth_provider_cert_uri: "https://oauth-login.cloud.huawei.com/oauth2/v3/certs"
  client_cert_uri: "https://oauth-login.cloud.huawei.com/oauth2/v3/x509?client_id="
  develop: false            # HarmonyOS test push
```

## Service Configuration Methods

The service can be configured in three ways. When `--config`/`-c` is used, priority from high to low is:

1. **Configuration file**: keys present in the file have the highest priority (even empty-string values override environment variables)
2. **Command-line parameters**: Parameters specified at startup
3. **Environment variables**: System environment variables

Without `-c`, priority is: command-line parameters > environment variables > built-in defaults.

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
| `--hm-project-id` | `NOLET_HM_PROJECT_ID` | HarmonyOS AGC project ID (push URL) |  |
| `--hm-key-id` | `NOLET_HM_KEY_ID` | HarmonyOS service account key ID (JWT kid) |  |
| `--hm-private-key` | `NOLET_HM_PRIVATE_KEY` | HarmonyOS service account RSA private key (PEM content, real newlines required) |  |
| `--hm-sub-account` | `NOLET_HM_SUB_ACCOUNT` | HarmonyOS service / sub account (JWT iss) |  |
| `--hm-client-id` | `NOLET_HM_CLIENT_ID` | HarmonyOS app client ID (message revocation) |  |
| `--hm-auth-uri` | `NOLET_HM_AUTH_URI` | HarmonyOS OAuth authorize URI | `https://oauth-login.cloud.huawei.com/oauth2/v3/authorize` |
| `--hm-token-uri` | `NOLET_HM_TOKEN_URI` | HarmonyOS OAuth token URI (JWT aud) | `https://oauth-login.cloud.huawei.com/oauth2/v3/token` |
| `--hm-auth-provider-cert-uri` | `NOLET_HM_AUTH_PROVIDER_CERT_URI` | HarmonyOS auth provider cert URI | `https://oauth-login.cloud.huawei.com/oauth2/v3/certs` |
| `--hm-client-cert-uri` | `NOLET_HM_CLIENT_CERT_URI` | HarmonyOS client cert URI | `https://oauth-login.cloud.huawei.com/oauth2/v3/x509?client_id=` |
| `--hm-develop` | `NOLET_HM_DEVELOP` | HarmonyOS test push | `false` |
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
   # Note: keys present in the config file override command-line parameters;
   # only keys missing from the file fall back to parameter values.
   ./NoLets -c /path/to/your/config.yaml --debug --addr 127.0.0.1:8080
   ```

## HarmonyOS Push Kit

In addition to Apple APNs, the server supports HarmonyOS Push Kit. Push requests are routed automatically by the user's device OS — no extra switch is needed, just configure the HarmonyOS credentials.

### Authentication Flow

1. Sign a JWT with the service account RSA private key using **PS256**: `kid` is the key ID, `iss` is the sub account, `aud` is the token URI; valid for 1 hour (cached and refreshed ahead of expiry);
2. Normal pushes call `https://push-api.cloud.huawei.com/v3/{projectID}/messages:send` with `Authorization: Bearer <JWT>`;
3. Silent background pushes (no notification content) call `messages:revoke`, which uses the **Client ID** in the URL instead.

### Obtaining Credentials (AppGallery Connect)

| Field | Where to find it |
|-------|------------------|
| Project ID `project_id` | AGC → Project settings → General → Project ID |
| Client ID `client_id` | AGC → Project settings → General → App information → Client ID |
| Key ID `key_id` | AGC → Users and permissions → Service accounts → create/view key |
| Private key `private_key` | PEM file downloaded when creating the service account key |
| Sub account `sub_account` | The corresponding account under AGC → Users and permissions → Service accounts |

### Notes

- **Private key newlines**: the PEM must contain real newlines. When injecting via environment variable, use single quotes and paste the multi-line content directly; `\n` inside double quotes is not interpreted by the shell, so the program receives literal backslash-n and PEM parsing fails.
- **Variable case**: the test-push variable is the all-uppercase `NOLET_HM_DEVELOP`. Environment variable names are case-sensitive on Linux.
- **Together with a config file**: keys written under the harmony section of a `-c` config file (including empty strings) override same-named environment variables. To configure purely via environment variables, do not use `-c`.
- **Test push**: with `--hm-develop` (or system `--debug`), push requests are sent as test messages (`TestMessage`).

