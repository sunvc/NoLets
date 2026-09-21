
# NoLetServer

![](https://status.wzs.app/api/badge/1/uptime/24?style=for-the-badge)

[English](./README_EN.md)

## 安装与运行

| App Store | Server Works  |
|--------|-------|
| [<img src="https://developer.apple.com/assets/elements/badges/download-on-the-app-store.svg" alt="Pushback App" height="40">](https://apps.apple.com/cn/app/id6615073345) | [![Deploy to Cloudflare Workers](https://deploy.workers.cloudflare.com/button)](https://github.com/sunvc/nolets-worker) |


### 一键安装 (推荐)

Linux / macOS (需已安装 Docker,Linux 未安装时脚本会自动安装 Docker):

```bash
curl -fsSL https://raw.githubusercontent.com/sunvc/nolets/main/install.sh | bash
```

脚本会:

1. 检查 / 安装 Docker(macOS 请先自行安装 Docker Desktop)
2. 在工作目录写入 `compose.yaml`(Linux 默认 `/opt/nolet`,macOS 默认 `~/.nolet`)
3. 拉取 `ghcr.io/sunvc/nolets:latest` 并通过 `docker compose` 启动容器 `NoLets`
4. 轮询 `http://127.0.0.1:8080/health` 做健康检查

#### 常用示例

```bash
# 自定义安装目录、端口,并注入签名密钥与授权 ID 列表
curl -fsSL https://raw.githubusercontent.com/sunvc/nolets/main/install.sh | bash -s -- \
    --dir /opt/nolet \
    --port 8080 \
    --sign-key "your-sign-key" \
    --auths '["uid1","uid2"]'

# 通过环境变量传参
NOLET_PORT=9090 NOLET_SIGN_KEY=xxx TZ=Asia/Shanghai \
    bash -c "$(curl -fsSL https://raw.githubusercontent.com/sunvc/nolets/main/install.sh)"

# 卸载 (仅移除容器,保留数据目录)
curl -fsSL https://raw.githubusercontent.com/sunvc/nolets/main/install.sh | bash -s -- --uninstall
```

#### 所有参数

| 参数 | 环境变量 | 说明 | 默认值 |
|------|----------|------|--------|
| `--dir` | `NOLET_DIR` | 工作目录(compose.yaml 与 data 存放位置) | Linux `/opt/nolet` · macOS `~/.nolet` |
| `--port` | `NOLET_PORT` | 宿主机映射端口 | `8080` |
| `--image` | `NOLET_IMAGE` | Docker 镜像 | `ghcr.io/sunvc/nolets:latest` |
| `--sign-key` | `NOLET_SIGN_KEY` | 加密签名密钥 |  |
| `--auths` | `NOLET_AUTHS` | 管理员 ID 列表,如 `'["uid1","uid2"]'` |  |
| `--tz` | `TZ` | 时区 | `Asia/Shanghai` |
| `--name` | `NOLET_CONTAINER` | 容器名 | `NoLets` |
| `--uninstall` |  | 移除容器 |  |

#### 常用运维命令

```bash
docker logs -f NoLets                              # 查看日志
docker restart NoLets                              # 重启
cd /opt/nolet && docker compose pull && docker compose up -d   # 更新到最新镜像
```

### 从GitHub Releases下载

您可以从GitHub Releases页面下载预编译的二进制文件：

1. 访问 [GitHub Releases](https://github.com/sunvc/NoLetserver/releases) 页面
2. 根据您的操作系统和架构选择合适的版本下载：
   - Windows (amd64)
   - macOS (amd64, arm64)
   - Linux (amd64, arm64, mips64, mips64le)
   - FreeBSD (amd64, arm64)
3. 解压下载的文件
4. 创建配置文件（参考下方配置说明）
5. 运行程序：
   ```bash
   # Linux/macOS
   ./NoLets --config your_config.yaml
   
   # Windows
   NoLets.exe --config your_config.yaml
   ```

   常用参数：
   - `--addr`: 服务器监听地址，默认为0.0.0.0:8080
   - `--url-prefix`: 服务URL前缀，默认为/
   - `--dir`: 数据存储目录，默认为./data
   - `--dsn`: MySQL数据库连接字符串
   - `--debug`: 启用调试模式
   - `--config, -c`: 指定配置文件路径

### 使用Docker

#### Docker 镜像

本项目提供了以下Docker镜像地址：

- Docker Hub: `sunvc/nolets:latest`
- GitHub Container Registry: `ghcr.io/sunvc/nolets:latest`

您可以使用以下命令拉取镜像：

```bash
# 从Docker Hub拉取
docker pull sunvc/nolets:latest

# 或从GitHub Container Registry拉取
docker pull ghcr.io/sunvc/nolets:latest

docker run -d --name NoLet-server \
  -p 8080:8080 \
  -v ./data:/data \
  --restart=always \
  ghcr.io/sunvc/nolets:latest
```

#### 使用Docker Compose

项目根目录下的`compose.yaml`文件已配置好使用Docker镜像的环境：

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

运行以下命令启动服务：

```bash
docker-compose up -d
```

## 配置文件

项目中的`config.yaml`仅作为配置文件示例，**用户需要自己创建并指定配置文件**进行服务配置。可以使用`--config`或`-c`参数指定配置文件路径。

### 配置文件结构

```yaml
system:
  user: ""                         # 基础认证用户名
  password: ""                     # 基础认证密码
  push_password: ""                # 群组推送密码
  addr: "0.0.0.0:8080"             # 服务器监听地址
  url_prefix: "/"                  # 服务URL前缀
  data: "./data"                   # 数据存储目录
  name: "NoLets"                   # 服务名称
  dsn: ""                          # MySQL DSN连接字符串
  cert: ""                         # TLS证书路径
  key: ""                          # TLS证书私钥路径
  reduce_memory_usage: false       # 降低内存占用（增加CPU消耗）
  proxy_header: ""                 # HTTP头中远程IP地址来源
  max_batch_push_count: -1         # 批量推送最大数量，-1表示无限制
  max_apns_client_count: 1         # 最大APNs客户端连接数
  max_device_key_arr_length: 10    # 最大key列表数量
  concurrency: 262144              # 最大并发连接数（256 * 1024）
  read_timeout: 3s                 # 读取超时时间
  write_timeout: 3s                # 写入超时时间
  idle_timeout: 10s                # 空闲超时时间
  admins: [ ]                      # 管理员ID列表
  debug: true                      # 启用调试模式
  expired: 0                       # 语音过期时间（秒）
  icp_info: ""                     # ICP备案信息
  time_zone: "UTC"                 # 时区设置

apple:
  apnsPrivateKey: ""               # APNs私钥内容或路径
  topic: ""                        # APNs Topic
  keyID: ""                        # APNs Key ID
  teamID: ""                       # APNs Team ID
  develop: false                   # 启用APNs开发环境

harmony:
  project_id: ""                  # 鸿蒙AGC项目ID（推送URL使用）
  key_id: ""                      # 服务账号Key ID（JWT kid）
  private_key: ""                 # 服务账号RSA私钥（PEM内容，需真实换行）
  sub_account: ""                 # 服务账号/子账号（JWT iss）
  client_id: ""                   # 应用Client ID（撤销推送时使用）
  auth_uri: "https://oauth-login.cloud.huawei.com/oauth2/v3/authorize"
  token_uri: "https://oauth-login.cloud.huawei.com/oauth2/v3/token"
  auth_provider_cert_uri: "https://oauth-login.cloud.huawei.com/oauth2/v3/certs"
  client_cert_uri: "https://oauth-login.cloud.huawei.com/oauth2/v3/x509?client_id="
  develop: false                  # 鸿蒙测试推送
```

## 服务配置方式

服务可以通过以下三种方式配置。使用 `--config`/`-c` 时，优先级从高到低为：

1. **配置文件**：配置文件中出现的键优先级最高（即使值为空字符串也会覆盖环境变量）
2. **命令行参数**：启动时指定的参数
3. **环境变量**：系统环境变量

不使用 `-c` 时，优先级为：命令行参数 > 环境变量 > 内置默认值。

### 命令行参数和环境变量

| 参数 | 环境变量 | 说明 | 默认值 |
|------|----------|------|--------|
| `--addr` | `NOLET_SERVER_ADDRESS` | 服务器监听地址 | `0.0.0.0:8080` |
| `--url-prefix` | `NOLET_SERVER_URL_PREFIX` | 服务 URL 前缀 | `/` |
| `--dir` | `NOLET_SERVER_DATA_DIR` | 服务器数据存储目录 | `./data` |
| `--dsn` | `NOLET_SERVER_DSN` | MySQL DSN（user:pass@tcp(host)/dbname） |  |
| `--cert` | `NOLET_SERVER_CERT` | 服务器 TLS 证书 |  |
| `--key` | `NOLET_SERVER_KEY` | 服务器 TLS 证书密钥 |  |
| `--reduce-memory-usage` | `NOLET_SERVER_REDUCE_MEMORY_USAGE` | 降低内存使用（会增加 CPU 消耗） | `false` |
| `--user`, `-u` | `NOLET_SERVER_BASIC_AUTH_USER` | 基本认证用户名 |  |
| `--password`, `-p` | `NOLET_SERVER_BASIC_AUTH_PASSWORD` | 基本认证密码 |  |
| `--push-password` | `NOLET_PUSH_PASSWORD` | 推送认证密码 |  |
| `--sign-key`, `--sk` | `NOLET_SIGN_KEY` | 应用签名密钥 |  |
| `--proxy-header` | `NOLET_SERVER_PROXY_HEADER` | 代理头中远程 IP 地址字段 |  |
| `--max-batch-push-count` | `NOLET_SERVER_MAX_BATCH_PUSH_COUNT` | 最大批量推送数量，`-1` 表示无限制 | `-1` |
| `--max-apns-client-count`, `--max` | `NOLET_SERVER_MAX_APNS_CLIENT_COUNT` | 最大 APNs 客户端连接数 | `1` |
| `--max-device-key-arr-length` | `NOLET_CONCURRENCY` | 最大设备 Key 列表长度 | `10` |
| `--concurrency` | `NOLET_SERVER_CONCURRENCY` | 最大并发连接数 | `262144` |
| `--read-timeout` | `NOLET_SERVER_READ_TIMEOUT` | 读取请求超时时间 | `3s` |
| `--write-timeout` | `NOLET_SERVER_WRITE_TIMEOUT` | 响应写入超时时间 | `3s` |
| `--idle-timeout` | `NOLET_SERVER_IDLE_TIMEOUT` | Keep-Alive 空闲超时时间 | `10s` |
| `--debug` | `NOLET_DEBUG` | 启用调试模式 | `false` |
| `--voice` | `NOLET_VOICE` | 启用语音支持 | `false` |
| `--auths` | `NOLET_AUTHS` | 授权 ID 列表 |  |
| `--apns-private-key` | `NOLET_APPLE_APNS_PRIVATE_KEY` | APNs 私钥路径 | 内置默认值 |
| `--topic` | `NOLET_APPLE_TOPIC` | APNs Topic | `me.uuneo.Meoworld` |
| `--key-id` | `NOLET_APPLE_KEY_ID` | APNs Key ID | `BNY5GUGV38` |
| `--team-id` | `NOLET_APPLE_TEAM_ID` | APNs Team ID | `FUWV6U942Q` |
| `--develop`, `--dev` | `NOLET_APPLE_DEVELOP` | 使用 APNs 开发环境 | `false` |
| `--hm-project-id` | `NOLET_HM_PROJECT_ID` | 鸿蒙 AGC 项目 ID（推送 URL） |  |
| `--hm-key-id` | `NOLET_HM_KEY_ID` | 鸿蒙服务账号 Key ID（JWT kid） |  |
| `--hm-private-key` | `NOLET_HM_PRIVATE_KEY` | 鸿蒙服务账号 RSA 私钥（PEM 内容，需真实换行） |  |
| `--hm-sub-account` | `NOLET_HM_SUB_ACCOUNT` | 鸿蒙服务账号/子账号（JWT iss） |  |
| `--hm-client-id` | `NOLET_HM_CLIENT_ID` | 鸿蒙应用 Client ID（撤销推送时使用） |  |
| `--hm-auth-uri` | `NOLET_HM_AUTH_URI` | 鸿蒙 OAuth 授权地址 | `https://oauth-login.cloud.huawei.com/oauth2/v3/authorize` |
| `--hm-token-uri` | `NOLET_HM_TOKEN_URI` | 鸿蒙 OAuth Token 地址（JWT aud） | `https://oauth-login.cloud.huawei.com/oauth2/v3/token` |
| `--hm-auth-provider-cert-uri` | `NOLET_HM_AUTH_PROVIDER_CERT_URI` | 鸿蒙授权方证书地址 | `https://oauth-login.cloud.huawei.com/oauth2/v3/certs` |
| `--hm-client-cert-uri` | `NOLET_HM_CLIENT_CERT_URI` | 鸿蒙客户端证书地址 | `https://oauth-login.cloud.huawei.com/oauth2/v3/x509?client_id=` |
| `--hm-develop` | `NOLET_HM_DEVELOP` | 鸿蒙测试推送 | `false` |
| `--Expired`, `--ex` | `NOLET_EXPIRED_TIME` | 语音过期时间（秒） | `120` |
| `--ICP`, `--icp` | `NOLET_ICP_INFO` | ICP 备案信息 |  |
| `--config`, `-c` |  | 配置文件路径 |  |
| `--proxy-download`, `--dp` | `NOLET_PROXY_DOWNLOAD` | 启用代理下载 | `false` |
| `--export-path`, `--dc` | `NOLET_EXPORT_PATH` | 导出数据库路径 |  |
| `--import-path`, `--dl` | `NOLET_IMPORT_PATH` | 导入数据库路径 |  |
| `--build-test` |  | 构建测试模式 |  |


### 使用配置文件

1. 创建自己的配置文件：
   - 参考项目中的`config.yaml`示例创建自己的配置文件
   - 确保配置文件包含所需的配置项

2. 指定配置文件路径：
   ```bash
    ./NoLets --config /path/to/your/config.yaml
    # 或使用简写
    ./NoLets -c /path/to/your/config.yaml
    ```

3. 配置文件与命令行参数混合使用：
   ```bash
   # 注意：配置文件中写出的键会覆盖命令行参数，仅未在文件中出现的键使用参数值
   ./NoLets -c /path/to/your/config.yaml --debug --addr 127.0.0.1:8080
   ```

## 鸿蒙推送（HarmonyOS Push Kit）

除 Apple APNs 外，服务端同时支持 HarmonyOS Push Kit。推送时会按用户设备的 OS 类型自动分流，无需额外开关，配置好鸿蒙凭据即可。

### 认证流程

1. 使用服务账号的 RSA 私钥以 **PS256** 算法签名 JWT：`kid` 为 Key ID，`iss` 为子账号，`aud` 为 Token URI，有效期 1 小时（程序会缓存并提前刷新）；
2. 普通推送调用 `https://push-api.cloud.huawei.com/v3/{项目ID}/messages:send`，请求头携带 `Authorization: Bearer <JWT>`；
3. 后台静默推送（无通知内容）调用 `messages:revoke` 撤销消息，此时 URL 中使用 **Client ID**。

### 凭据获取（AppGallery Connect）

| 配置项 | 获取位置 |
|--------|----------|
| 项目 ID `project_id` | AGC →「项目设置」→「常规」→ 项目 ID |
| Client ID `client_id` | AGC →「项目设置」→「常规」→ 应用信息 → Client ID |
| Key ID `key_id` | AGC →「用户与访问」→「服务账号」→ 创建/查看密钥 |
| 私钥 `private_key` | 创建服务账号密钥时下载的 PEM 文件内容 |
| 子账号 `sub_account` | AGC →「用户与访问」→「服务账号」中对应账号 |

### 注意事项

- **私钥换行**：PEM 必须包含真实换行。通过环境变量注入时使用单引号并直接粘贴多行内容；写在双引号里的 `\n` 不会被 shell 转义，程序收到的是字面的反斜杠加 n，会导致 PEM 解析失败。
- **环境变量大小写**：测试推送开关变量为全大写的 `NOLET_HM_DEVELOP`，Linux 下环境变量大小写敏感。
- **与配置文件同时使用**：`-c` 指定的配置文件中 harmony 段写出的键（包括空字符串）会覆盖同名环境变量；若希望完全由环境变量配置，请勿使用 `-c`。
- **测试推送**：开启 `--hm-develop` 或系统 `--debug` 后，推送请求会以测试消息（`TestMessage`）发送。

