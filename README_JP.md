
# NoLetServer

![](https://status.wzs.app/api/badge/1/uptime/24?style=for-the-badge)

[中文](./README.md) | [English](./README_EN.md) | [한국어](./README_KR.md)

## インストールと実行

| App Store | Server Works  |
|--------|-------|
| [<img src="https://developer.apple.com/assets/elements/badges/download-on-the-app-store.svg" alt="Pushback App" height="40">](https://apps.apple.com/us/app/id6615073345) | [![Deploy to Cloudflare Workers](https://deploy.workers.cloudflare.com/button)](https://deploy.workers.cloudflare.com/?url=https://github.com/sunvc/NoLets-worker) |

### GitHub Releasesからダウンロード

GitHub Releasesページからプリコンパイルされたバイナリをダウンロードできます：

1. [GitHub Releases](https://github.com/sunvc/NoLetserver/releases) ページにアクセス
2. お使いのオペレーティングシステムとアーキテクチャに適したバージョンを選択：
   - Windows (amd64)
   - macOS (amd64, arm64)
   - Linux (amd64, arm64, mips64, mips64le)
   - FreeBSD (amd64, arm64)
3. ダウンロードしたファイルを解凍
4. 設定ファイルを作成（以下の設定手順を参照）
5. プログラムを実行：
   ```bash
   # Linux/macOS
   ./NoLets --config your_config.yaml
   
   # Windows
   NoLets.exe --config your_config.yaml
   ```

### Dockerの使用

#### Dockerイメージ

このプロジェクトでは、以下のDockerイメージアドレスを提供しています：

- Docker Hub: `sunvc/nolet:latest`
- GitHub Container Registry: `ghcr.io/sunvc/nolet:latest`

以下のコマンドでイメージをプルできます：

```bash
# Docker Hubからプル
docker pull sunvc/nolet:latest

# または、GitHub Container Registryからプル
docker pull ghcr.io/sunvc/nolet:latest

docker run -d --name NoLet-server \
  -p 8080:8080 \
  -v ./data:/data \
  --restart=always \
  ghcr.io/sunvc/nolet:latest
```

#### Docker Composeの使用

プロジェクトのルートディレクトリにある`compose.yaml`ファイルは、Dockerイメージを使用するように既に設定されています：

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

以下のコマンドでサービスを起動します：

```bash
docker-compose up -d
```

## 設定ファイル

プロジェクト内の`config.yaml`は設定ファイルの例にすぎません。**ユーザーは自分で設定ファイルを作成して指定する必要があります**。`--config`または`-c`パラメータを使用して設定ファイルのパスを指定できます。

### 設定ファイルの構造

```yaml
system:
  user: ""                         # 基本認証ユーザー名
  password: ""                     # 基本認証パスワード
  push_password: ""                # グループプッシュパスワード
  addr: "0.0.0.0:8080"             # サーバーリスニングアドレス
  url_prefix: "/"                  # サービスURLプレフィックス
  data: "./data"                   # データストレージディレクトリ
  name: "NoLets"                   # サービス名
  dsn: ""                          # MySQL DSN接続文字列
  cert: ""                         # TLS証明書パス
  key: ""                          # TLS証明書秘密鍵パス
  reduce_memory_usage: false       # メモリ使用量を削減（CPU消費量が増加）
  proxy_header: ""                 # HTTPヘッダーのリモートIPアドレスソース
  max_batch_push_count: -1         # バッチプッシュの最大数、-1は無制限
  max_apns_client_count: 1         # APNsクライアント接続の最大数
  max_device_key_arr_length: 10    # キーリストの最大数
  concurrency: 262144              # 最大同時接続数（256 * 1024）
  read_timeout: 3s                 # 読み取りタイムアウト
  write_timeout: 3s                # 書き込みタイムアウト
  idle_timeout: 10s                # アイドルタイムアウト
  admins: []                       # 管理者IDリスト
  debug: true                      # デバッグモードを有効にする
  expired: 0                       # 音声の有効期限（秒）
  icp_info: ""                     # ICP登録情報
  time_zone: "UTC"                 # タイムゾーン設定

apple:
  apnsPrivateKey: ""               # APNs秘密鍵の内容またはパス
  topic: ""                        # APNs Topic
  keyID: ""                        # APNs Key ID
  teamID: ""                       # APNs Team ID
  develop: false                   # APNs開発環境を有効にする
```

## サービス設定方法

サービスは以下の3つの方法で設定でき、優先順位は高いものから低いものへ：

1. **コマンドラインパラメータ**：起動時に指定されるパラメータ、最優先
2. **環境変数**：システム環境変数、次の優先順位
3. **設定ファイル**：`config.yaml`ファイルまたは`--config`/`-c`パラメータで指定された設定ファイル

### コマンドラインパラメータと環境変数

| パラメータ | 環境変数 | 説明 | デフォルト値 |
|------|----------|------|--------|
| `--addr` | `NOLET_SERVER_ADDRESS` | サーバー監視アドレス | `0.0.0.0:8080` |
| `--url-prefix` | `NOLET_SERVER_URL_PREFIX` | サービスURLプレフィックス | `/` |
| `--dir` | `NOLET_SERVER_DATA_DIR` | サーバーデータ保存ディレクトリ | `./data` |
| `--dsn` | `NOLET_SERVER_DSN` | MySQL DSN（user:pass@tcp(host)/dbname） |  |
| `--cert` | `NOLET_SERVER_CERT` | サーバーTLS証明書 |  |
| `--key` | `NOLET_SERVER_KEY` | サーバーTLS証明書キー |  |
| `--reduce-memory-usage` | `NOLET_SERVER_REDUCE_MEMORY_USAGE` | メモリ使用量を削減（CPU消費が増加） | `false` |
| `--user`, `-u` | `NOLET_SERVER_BASIC_AUTH_USER` | ベーシック認証ユーザー名 |  |
| `--password`, `-p` | `NOLET_SERVER_BASIC_AUTH_PASSWORD` | ベーシック認証パスワード |  |
| `--push-password` | `NOLET_PUSH_PASSWORD` | プッシュ認証パスワード |  |
| `--sign-key`, `--sk` | `NOLET_SIGN_KEY` | アプリ署名キー |  |
| `--proxy-header` | `NOLET_SERVER_PROXY_HEADER` | プロキシヘッダーのリモートIPアドレスフィールド |  |
| `--max-batch-push-count` | `NOLET_SERVER_MAX_BATCH_PUSH_COUNT` | 最大バッチプッシュ数、`-1`は無制限 | `-1` |
| `--max-apns-client-count`, `--max` | `NOLET_SERVER_MAX_APNS_CLIENT_COUNT` | 最大APNsクライアント接続数 | `1` |
| `--max-device-key-arr-length` | `NOLET_CONCURRENCY` | 最大デバイスキーリスト長 | `10` |
| `--concurrency` | `NOLET_SERVER_CONCURRENCY` | 最大同時接続数 | `262144` |
| `--read-timeout` | `NOLET_SERVER_READ_TIMEOUT` | リクエスト読み取りタイムアウト時間 | `3s` |
| `--write-timeout` | `NOLET_SERVER_WRITE_TIMEOUT` | レスポンス書き込みタイムアウト時間 | `3s` |
| `--idle-timeout` | `NOLET_SERVER_IDLE_TIMEOUT` | Keep-Aliveアイドルタイムアウト時間 | `10s` |
| `--debug` | `NOLET_DEBUG` | デバッグモードを有効化 | `false` |
| `--voice` | `NOLET_VOICE` | 音声サポートを有効化 | `false` |
| `--auths` | `NOLET_AUTHS` | 認証IDリスト |  |
| `--apns-private-key` | `NOLET_APPLE_APNS_PRIVATE_KEY` | APNs秘密鍵パス | 組み込みデフォルト |
| `--topic` | `NOLET_APPLE_TOPIC` | APNsトピック | `me.uuneo.Meoworld` |
| `--key-id` | `NOLET_APPLE_KEY_ID` | APNsキーID | `BNY5GUGV38` |
| `--team-id` | `NOLET_APPLE_TEAM_ID` | APNsチームID | `FUWV6U942Q` |
| `--develop`, `--dev` | `NOLET_APPLE_DEVELOP` | APNs開発環境を使用 | `false` |
| `--Expired`, `--ex` | `NOLET_EXPIRED_TIME` | 音声有効期限（秒） | `120` |
| `--ICP`, `--icp` | `NOLET_ICP_INFO` | ICP登録情報 |  |
| `--config`, `-c` |  | 設定ファイルパス |  |
| `--proxy-download`, `--dp` | `NOLET_PROXY_DOWNLOAD` | プロキシダウンロードを有効化 | `false` |
| `--export-path`, `--dc` | `NOLET_EXPORT_PATH` | データベースエクスポートパス |  |
| `--import-path`, `--dl` | `NOLET_IMPORT_PATH` | データベースインポートパス |  |
| `--build-test` |  | ビルドテストモード |  |

### 設定ファイルの使用

1. 自分の設定ファイルを作成：
   - プロジェクト内の`config.yaml`例を参考に自分の設定ファイルを作成
   - 設定ファイルに必要な設定項目が含まれていることを確認

2. 設定ファイルパスを指定：
   ```bash
   ./NoLets --config /path/to/your/config.yaml
   # または省略形を使用
   ./NoLets -c /path/to/your/config.yaml
   ```

3. 設定ファイルとコマンドラインパラメータの混合使用：
   ```bash
   # 設定ファイル内の設定はコマンドラインパラメータによって上書きされます
   ./NoLets -c /path/to/your/config.yaml --debug --addr 127.0.0.1:8080
   ```