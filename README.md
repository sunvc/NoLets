
# NoLetServer

![](https://status.wzs.app/api/badge/1/uptime/24?style=for-the-badge)

[English](./README_EN.md) | [日本語](./README_JP.md) | [한국어](./README_KR.md)

## 安装与运行

| App Store | Server Works  |
|--------|-------|
| [<img src="https://developer.apple.com/assets/elements/badges/download-on-the-app-store.svg" alt="Pushback App" height="40">](https://apps.apple.com/cn/app/id6615073345) | [![Deploy to Cloudflare Workers](https://deploy.workers.cloudflare.com/button)](https://github.com/sunvc/nolet-worker) |


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

- Docker Hub: `sunvc/nolet:latest`
- GitHub Container Registry: `ghcr.io/sunvc/nolet:latest`

您可以使用以下命令拉取镜像：

```bash
# 从Docker Hub拉取
docker pull sunvc/nolet:latest

# 或从GitHub Container Registry拉取
docker pull ghcr.io/sunvc/nolet:latest

docker run -d --name NoLet-server \
  -p 8080:8080 \
  -v ./data:/data \
  --restart=always \
  ghcr.io/sunvc/nolet:latest
```

#### 使用Docker Compose

项目根目录下的`compose.yaml`文件已配置好使用Docker镜像的环境：

```yaml
services:
  NoLetServer:
    image: ghcr.io/sunvc/nolet:latest
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
```

## 服务配置方式

服务可以通过以下三种方式配置，优先级从高到低：

1. **命令行参数**：启动时指定的参数，优先级最高
2. **环境变量**：系统环境变量，次优先级
3. **配置文件**：`config.yaml`文件或通过`--config`/`-c`参数指定的配置文件

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
   # 配置文件中的设置会被命令行参数覆盖
   ./NoLets -c /path/to/your/config.yaml --debug --addr 127.0.0.1:8080
   ```
