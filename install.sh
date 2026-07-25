#!/usr/bin/env bash
#
# NoLet Server 一键安装脚本
#
# 使用:
#   curl -fsSL https://raw.githubusercontent.com/sunvc/nolets/main/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/sunvc/nolets/main/install.sh | bash -s -- --dir /opt/nolet --port 8080
#
# 通过环境变量控制:
#   NOLET_DIR=/opt/nolet NOLET_PORT=8080 NOLET_IMAGE=sunvc/nolets:latest \
#   NOLET_SIGN_KEY=xxx NOLET_AUTHS='["uid1","uid2"]' TZ=Asia/Shanghai \
#     bash -c "$(curl -fsSL https://raw.githubusercontent.com/sunvc/nolets/main/install.sh)"

set -euo pipefail

# ---------- 默认参数 ----------
# macOS 上 Docker Desktop 默认只共享 /Users /Volumes /private /tmp,
# 因此默认目录随平台切换,避免出现 "path is not shared from the host"。
if [ -z "${NOLET_DIR:-}" ]; then
    if [ "$(uname -s)" = "Darwin" ]; then
        NOLET_DIR="${HOME}/.nolet"
    else
        NOLET_DIR="/opt/nolet"
    fi
fi
NOLET_PORT="${NOLET_PORT:-8080}"
NOLET_IMAGE="${NOLET_IMAGE:-ghcr.io/sunvc/nolets:latest}"
NOLET_SIGN_KEY="${NOLET_SIGN_KEY:-}"
NOLET_AUTHS="${NOLET_AUTHS:-}"
NOLET_TZ="${TZ:-Asia/Shanghai}"
NOLET_CONTAINER="${NOLET_CONTAINER:-NoLets}"
NOLET_UNINSTALL=0

# ---------- 输出辅助 ----------
if [ -t 1 ]; then
    C_RED='\033[0;31m'; C_GREEN='\033[0;32m'; C_YELLOW='\033[0;33m'
    C_BLUE='\033[0;34m'; C_BOLD='\033[1m'; C_OFF='\033[0m'
else
    C_RED=; C_GREEN=; C_YELLOW=; C_BLUE=; C_BOLD=; C_OFF=
fi

info()  { printf "${C_BLUE}==>${C_OFF} %s\n"  "$*"; }
ok()    { printf "${C_GREEN}✓${C_OFF}  %s\n"  "$*"; }
warn()  { printf "${C_YELLOW}!${C_OFF}  %s\n" "$*"; }
die()   { printf "${C_RED}✗${C_OFF}  %s\n" "$*" >&2; exit 1; }

# ---------- 参数解析 ----------
while [ $# -gt 0 ]; do
    case "$1" in
        --dir)        NOLET_DIR="$2"; shift 2 ;;
        --port)       NOLET_PORT="$2"; shift 2 ;;
        --image)      NOLET_IMAGE="$2"; shift 2 ;;
        --sign-key)   NOLET_SIGN_KEY="$2"; shift 2 ;;
        --auths)      NOLET_AUTHS="$2"; shift 2 ;;
        --tz)         NOLET_TZ="$2"; shift 2 ;;
        --name)       NOLET_CONTAINER="$2"; shift 2 ;;
        --uninstall)  NOLET_UNINSTALL=1; shift ;;
        -h|--help)
            sed -n '2,15p' "$0" | sed 's/^# \{0,1\}//'
            exit 0 ;;
        *) die "未知参数: $1" ;;
    esac
done

# ---------- 权限 / 平台检测 ----------
require_root() {
    # macOS 上 Docker Desktop 由用户会话管理,不需要 root
    if [ "$(uname -s)" = "Darwin" ]; then
        SUDO=""
        return
    fi
    if [ "$(id -u)" -ne 0 ]; then
        if command -v sudo >/dev/null 2>&1; then
            SUDO="sudo"
        else
            die "需要 root 权限运行 (或安装 sudo)"
        fi
    else
        SUDO=""
    fi
}

detect_os() {
    OS="$(uname -s)"
    ARCH="$(uname -m)"
    case "$OS" in
        Linux)  ;;
        Darwin) warn "macOS 上仅支持已安装 Docker Desktop 的场景" ;;
        *) die "不支持的操作系统: $OS" ;;
    esac
    case "$ARCH" in
        x86_64|amd64|aarch64|arm64) ;;
        *) warn "未测试的架构: $ARCH,继续尝试" ;;
    esac
}

# ---------- 卸载 ----------
do_uninstall() {
    info "停止并移除容器 ${NOLET_CONTAINER}"
    if command -v docker >/dev/null 2>&1; then
        $SUDO docker rm -f "$NOLET_CONTAINER" >/dev/null 2>&1 || true
    fi
    if [ -d "$NOLET_DIR" ]; then
        warn "保留数据目录: $NOLET_DIR (如需彻底删除请手动 rm -rf)"
    fi
    ok "已卸载"
    exit 0
}

# ---------- 安装 Docker ----------
install_docker() {
    if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
        ok "Docker 已安装: $(docker --version)"
        return
    fi
    info "安装 Docker (使用官方脚本 get.docker.com)"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL https://get.docker.com | $SUDO sh
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- https://get.docker.com | $SUDO sh
    else
        die "找不到 curl 或 wget,无法安装 Docker"
    fi

    if command -v systemctl >/dev/null 2>&1; then
        $SUDO systemctl enable --now docker || true
    fi

    command -v docker >/dev/null 2>&1 || die "Docker 安装失败"
    ok "Docker 安装完成"
}

# ---------- 生成 compose.yaml ----------
write_compose() {
    info "创建工作目录: $NOLET_DIR"
    $SUDO mkdir -p "$NOLET_DIR/data"

    local compose_file="$NOLET_DIR/compose.yaml"
    if [ -f "$compose_file" ]; then
        warn "已存在 $compose_file,备份为 ${compose_file}.bak"
        $SUDO cp "$compose_file" "${compose_file}.bak"
    fi

    info "写入 $compose_file"
    $SUDO tee "$compose_file" >/dev/null <<EOF
services:
  NoLets:
    image: ${NOLET_IMAGE}
    container_name: ${NOLET_CONTAINER}
    restart: always
    pid: host
    ports:
      - "${NOLET_PORT}:8080"
    ulimits:
      nofile:
        soft: 65535
        hard: 65535
    volumes:
      - ./data:/app/data
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - NOLET_SIGN_KEY=${NOLET_SIGN_KEY}
      - TZ=${NOLET_TZ}
      - NOLET_AUTHS=${NOLET_AUTHS}
EOF
}

# ---------- 拉起服务 ----------
start_service() {
    info "拉取镜像 ${NOLET_IMAGE}"
    $SUDO docker pull "$NOLET_IMAGE"

    # 优先使用 compose 插件;否则回退到 docker run
    if $SUDO docker compose version >/dev/null 2>&1; then
        info "使用 docker compose 启动"
        ( cd "$NOLET_DIR" && $SUDO docker compose up -d )
    elif command -v docker-compose >/dev/null 2>&1; then
        info "使用 docker-compose 启动"
        ( cd "$NOLET_DIR" && $SUDO docker-compose up -d )
    else
        warn "未检测到 docker compose 插件,使用 docker run 启动"
        $SUDO docker rm -f "$NOLET_CONTAINER" >/dev/null 2>&1 || true
        $SUDO docker run -d \
            --name "$NOLET_CONTAINER" \
            --restart always \
            --pid host \
            --ulimit nofile=65535:65535 \
            -p "${NOLET_PORT}:8080" \
            -v "${NOLET_DIR}/data:/app/data" \
            -v /var/run/docker.sock:/var/run/docker.sock \
            -e "NOLET_SIGN_KEY=${NOLET_SIGN_KEY}" \
            -e "TZ=${NOLET_TZ}" \
            -e "NOLET_AUTHS=${NOLET_AUTHS}" \
            "$NOLET_IMAGE"
    fi
}

# ---------- 健康检查 ----------
health_check() {
    info "等待服务就绪..."
    local url="http://127.0.0.1:${NOLET_PORT}/health"
    local i
    for i in $(seq 1 20); do
        if command -v curl >/dev/null 2>&1 && curl -fsS "$url" >/dev/null 2>&1; then
            ok "健康检查通过: $url"
            return 0
        fi
        sleep 1
    done
    warn "健康检查超时,请手动检查:  docker logs ${NOLET_CONTAINER}"
}

# ---------- 打印摘要 ----------
summary() {
    local ip
    ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
    [ -z "$ip" ] && ip="<server-ip>"

    printf "\n"
    printf "${C_BOLD}NoLet Server 已启动${C_OFF}\n"
    printf "  访问地址 : http://%s:%s/\n" "$ip" "$NOLET_PORT"
    printf "  容器名称 : %s\n" "$NOLET_CONTAINER"
    printf "  数据目录 : %s/data\n" "$NOLET_DIR"
    printf "  Compose  : %s/compose.yaml\n" "$NOLET_DIR"
    printf "\n"
    printf "常用命令:\n"
    printf "  查看日志 : docker logs -f %s\n" "$NOLET_CONTAINER"
    printf "  重启服务 : docker restart %s\n" "$NOLET_CONTAINER"
    printf "  停止服务 : docker stop %s\n" "$NOLET_CONTAINER"
    printf "  更新镜像 : cd %s && docker compose pull && docker compose up -d\n" "$NOLET_DIR"
    printf "  卸载     : curl -fsSL <本脚本 URL> | bash -s -- --uninstall\n"
    printf "\n"
}

# ---------- 主流程 ----------
main() {
    printf "${C_BOLD}NoLet Server Installer${C_OFF}\n"
    require_root
    detect_os

    if [ "$NOLET_UNINSTALL" -eq 1 ]; then
        do_uninstall
    fi

    install_docker
    write_compose
    start_service
    health_check
    summary
}

main "$@"
