
# NoLetServer

![](https://status.wzs.app/api/badge/1/uptime/24?style=for-the-badge)

[中文](./README.md) | [English](./README_EN.md) | [日本語](./README_JP.md)

## 설치 및 실행

| App Store | Server Works  |
|--------|-------|
| [<img src="https://developer.apple.com/assets/elements/badges/download-on-the-app-store.svg" alt="Pushback App" height="40">](https://apps.apple.com/us/app/id6615073345) | [![Deploy to Cloudflare Workers](https://deploy.workers.cloudflare.com/button)](https://deploy.workers.cloudflare.com/?url=https://github.com/sunvc/NoLets-worker) |

### GitHub Releases에서 다운로드

GitHub Releases 페이지에서 미리 컴파일된 바이너리를 다운로드할 수 있습니다:

1. [GitHub Releases](https://github.com/sunvc/NoLetserver/releases) 페이지 방문
2. 운영 체제 및 아키텍처에 맞는 버전 선택:
   - Windows (amd64)
   - macOS (amd64, arm64)
   - Linux (amd64, arm64, mips64, mips64le)
   - FreeBSD (amd64, arm64)
3. 다운로드한 파일 압축 해제
4. 구성 파일 생성(아래 구성 지침 참조)
5. 프로그램 실행:
   ```bash
   # Linux/macOS
   ./NoLets --config your_config.yaml
   
   # Windows
   NoLets.exe --config your_config.yaml
   ```

### Docker 사용

#### Docker 이미지

이 프로젝트는 다음 Docker 이미지 주소를 제공합니다:

- Docker Hub: `sunvc/nolet:latest`
- GitHub Container Registry: `ghcr.io/sunvc/nolet:latest`

다음 명령으로 이미지를 가져올 수 있습니다:

```bash
# Docker Hub에서 가져오기
docker pull sunvc/nolet:latest

# 또는 GitHub Container Registry에서 가져오기
docker pull ghcr.io/sunvc/nolet:latest

docker run -d --name NoLet-server \
  -p 8080:8080 \
  -v ./data:/data \
  --restart=always \
  ghcr.io/sunvc/nolet:latest
```

#### Docker Compose 사용

프로젝트 루트 디렉토리의 `compose.yaml` 파일은 이미 Docker 이미지를 사용하도록 구성되어 있습니다:

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

다음 명령으로 서비스를 시작합니다:

```bash
docker-compose up -d
```

## 구성 파일

프로젝트의 `config.yaml`은 구성 파일의 예시일 뿐입니다. **사용자는 자신의 구성 파일을 생성하고 지정해야 합니다**. `--config` 또는 `-c` 매개변수를 사용하여 구성 파일 경로를 지정할 수 있습니다.

### 구성 파일 구조

```yaml
system:
  user: ""                         # 기본 인증 사용자 이름
  password: ""                     # 기본 인증 비밀번호
  push_password: ""                # 그룹 푸시 비밀번호
  addr: "0.0.0.0:8080"             # 서버 리스닝 주소
  url_prefix: "/"                  # 서비스 URL 접두사
  data: "./data"                   # 데이터 저장 디렉토리
  name: "NoLets"                   # 서비스 이름
  dsn: ""                          # MySQL DSN 연결 문자열
  cert: ""                         # TLS 인증서 경로
  key: ""                          # TLS 인증서 개인 키 경로
  reduce_memory_usage: false       # 메모리 사용량 감소(CPU 사용량 증가)
  proxy_header: ""                 # HTTP 헤더의 원격 IP 주소 소스
  max_batch_push_count: -1         # 배치 푸시 최대 수, -1은 무제한
  max_apns_client_count: 1         # APNs 클라이언트 연결 최대 수
  max_device_key_arr_length: 10    # 키 목록의 최대 수
  concurrency: 262144              # 최대 동시 연결 수(256 * 1024)
  read_timeout: 3s                 # 읽기 타임아웃
  write_timeout: 3s                # 쓰기 타임아웃
  idle_timeout: 10s                # 유휴 타임아웃
  admins: []                       # 관리자 ID 목록
  debug: true                      # 디버그 모드 활성화
  expired: 0                       # 음성 만료 시간(초)
  icp_info: ""                     # ICP 등록 정보
  time_zone: "UTC"                 # 시간대 설정
       
apple:
  apnsPrivateKey: ""        # APNs 개인 키 내용 또는 경로
  topic: ""                 # APNs Topic
  keyID: ""                 # APNs Key ID
  teamID: ""                # APNs Team ID
  develop: false            # APNs 개발 환경 활성화
```

## 서비스 구성 방법

서비스는 다음 3가지 방법으로 구성할 수 있으며, 우선순위는 높은 것부터 낮은 것 순입니다:

1. **명령줄 매개변수**: 시작 시 지정된 매개변수, 최우선
2. **환경 변수**: 시스템 환경 변수, 다음 우선순위
3. **구성 파일**: `config.yaml` 파일 또는 `--config`/`-c` 매개변수로 지정된 구성 파일

### 명령줄 매개변수 및 환경 변수

| 매개변수 | 환경 변수 | 설명 | 기본값 |
|------|----------|------|--------|
| `--addr` | `NOLET_SERVER_ADDRESS` | 서버 수신 주소 | `0.0.0.0:8080` |
| `--url-prefix` | `NOLET_SERVER_URL_PREFIX` | 서비스 URL 접두사 | `/` |
| `--dir` | `NOLET_SERVER_DATA_DIR` | 서버 데이터 저장 디렉토리 | `./data` |
| `--dsn` | `NOLET_SERVER_DSN` | MySQL DSN (user:pass@tcp(host)/dbname) |  |
| `--cert` | `NOLET_SERVER_CERT` | 서버 TLS 인증서 |  |
| `--key` | `NOLET_SERVER_KEY` | 서버 TLS 인증서 키 |  |
| `--reduce-memory-usage` | `NOLET_SERVER_REDUCE_MEMORY_USAGE` | 메모리 사용량 감소 (CPU 사용량 증가) | `false` |
| `--user`, `-u` | `NOLET_SERVER_BASIC_AUTH_USER` | 기본 인증 사용자 이름 |  |
| `--password`, `-p` | `NOLET_SERVER_BASIC_AUTH_PASSWORD` | 기본 인증 비밀번호 |  |
| `--push-password` | `NOLET_PUSH_PASSWORD` | 푸시 인증 비밀번호 |  |
| `--sign-key`, `--sk` | `NOLET_SIGN_KEY` | 앱 서명 키 |  |
| `--proxy-header` | `NOLET_SERVER_PROXY_HEADER` | 프록시 헤더의 원격 IP 주소 필드 |  |
| `--max-batch-push-count` | `NOLET_SERVER_MAX_BATCH_PUSH_COUNT` | 최대 배치 푸시 수량, `-1`은 무제한 | `-1` |
| `--max-apns-client-count`, `--max` | `NOLET_SERVER_MAX_APNS_CLIENT_COUNT` | 최대 APNs 클라이언트 연결 수 | `1` |
| `--max-device-key-arr-length` | `NOLET_CONCURRENCY` | 최대 디바이스 키 목록 길이 | `10` |
| `--concurrency` | `NOLET_SERVER_CONCURRENCY` | 최대 동시 연결 수 | `262144` |
| `--read-timeout` | `NOLET_SERVER_READ_TIMEOUT` | 읽기 요청 시간 초과 | `3s` |
| `--write-timeout` | `NOLET_SERVER_WRITE_TIMEOUT` | 응답 쓰기 시간 초과 | `3s` |
| `--idle-timeout` | `NOLET_SERVER_IDLE_TIMEOUT` | Keep-Alive 유휴 시간 초과 | `10s` |
| `--debug` | `NOLET_DEBUG` | 디버그 모드 활성화 | `false` |
| `--voice` | `NOLET_VOICE` | 음성 지원 활성화 | `false` |
| `--auths` | `NOLET_AUTHS` | 인증 ID 목록 |  |
| `--apns-private-key` | `NOLET_APPLE_APNS_PRIVATE_KEY` | APNs 개인 키 경로 | 기본값 포함 |
| `--topic` | `NOLET_APPLE_TOPIC` | APNs Topic | `me.uuneo.Meoworld` |
| `--key-id` | `NOLET_APPLE_KEY_ID` | APNs Key ID | `BNY5GUGV38` |
| `--team-id` | `NOLET_APPLE_TEAM_ID` | APNs Team ID | `FUWV6U942Q` |
| `--develop`, `--dev` | `NOLET_APPLE_DEVELOP` | APNs 개발 환경 사용 | `false` |
| `--Expired`, `--ex` | `NOLET_EXPIRED_TIME` | 음성 만료 시간 (초) | `120` |
| `--ICP`, `--icp` | `NOLET_ICP_INFO` | ICP 등록 정보 |  |
| `--config`, `-c` |  | 구성 파일 경로 |  |
| `--proxy-download`, `--dp` | `NOLET_PROXY_DOWNLOAD` | 프록시 다운로드 활성화 | `false` |
| `--export-path`, `--dc` | `NOLET_EXPORT_PATH` | 데이터베이스 내보내기 경로 |  |
| `--import-path`, `--dl` | `NOLET_IMPORT_PATH` | 데이터베이스 가져오기 경로 |  |
| `--build-test` |  | 빌드 테스트 모드 |  |

### 구성 파일 사용

1. 자신의 구성 파일 생성:
   - 프로젝트의 `config.yaml` 예시를 참조하여 자신의 구성 파일 생성
   - 구성 파일에 필요한 구성 항목이 포함되어 있는지 확인

2. 구성 파일 경로 지정:
   ```bash
   ./NoLets --config /path/to/your/config.yaml
   # 또는 약식 사용
   ./NoLets -c /path/to/your/config.yaml
   ```

3. 구성 파일과 명령줄 매개변수 혼합 사용:
   ```bash
   # 구성 파일의 설정은 명령줄 매개변수에 의해 재정의됩니다
   ./NoLets -c /path/to/your/config.yaml --debug --addr 127.0.0.1:8080
   ```