#!/bin/sh

# DEBIAN

set -e


# 本地编译上传
# CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./Server/main  main.go || echo "编译linux版本失败"
# scp ./Server/main root@wzs.app:/root/main

#---------------------------------------------------------------------------------------------

# Linux setup 必须 root
[ "$EUID" -ne 0 ] && { echo "请使用 root 运行"; exit 1; }

echo ">> 备份 sysctl.conf"
cp -n /etc/sysctl.conf /etc/sysctl.conf.bak 2>/dev/null || true

echo ">> 写入 sysctl 优化"

cat > /etc/sysctl.d/99-custom.conf <<EOF
vm.swappiness = 10

net.core.somaxconn = 65535
net.core.netdev_max_backlog = 16384
net.ipv4.tcp_max_syn_backlog = 8192
net.ipv4.tcp_syncookies = 1

net.ipv4.ip_local_port_range = 10240 65535

fs.file-max = 2097152
EOF

sysctl --system >/dev/null

echo ">> 设置 nofile 限制"

# limits.conf（避免重复）
if ! grep -q "soft nofile 65535" /etc/security/limits.conf; then
cat >> /etc/security/limits.conf <<EOF

* soft nofile 65535
* hard nofile 65535
EOF
fi

# systemd 限制
mkdir -p /etc/systemd/system.conf.d
cat > /etc/systemd/system.conf.d/99-nofile.conf <<EOF
[Manager]
DefaultLimitNOFILE=65535
EOF

systemctl daemon-reexec

echo ">> 完成"
echo "验证: ulimit -n && sysctl net.core.somaxconn"

#---------------------------------------------------------------------------------------------

# 安装Docker

# Add Docker's official GPG key:
sudo apt update
sudo apt install ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc

# Add the repository to Apt sources:
sudo tee /etc/apt/sources.list.d/docker.sources <<EOF
Types: deb
URIs: https://download.docker.com/linux/debian
Suites: $(. /etc/os-release && echo "$VERSION_CODENAME")
Components: stable
Signed-By: /etc/apt/keyrings/docker.asc
EOF

sudo apt update

sudo apt install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

#---------------------------------------------------------------------------------------------
# 安装 Nginx

sudo apt update

sudo apt install curl gnupg2 ca-certificates lsb-release debian-archive-keyring

curl https://nginx.org/keys/nginx_signing.key | gpg --dearmor \
    | sudo tee /usr/share/keyrings/nginx-archive-keyring.gpg >/dev/null

echo "deb [signed-by=/usr/share/keyrings/nginx-archive-keyring.gpg] \
https://nginx.org/packages/debian `lsb_release -cs` nginx" \
    | sudo tee /etc/apt/sources.list.d/nginx.list

echo -e "Package: *\nPin: origin nginx.org\nPin: release o=nginx\nPin-Priority: 900\n" \
    | sudo tee /etc/apt/preferences.d/99nginx

sudo apt update
sudo apt install nginx
#---------------------------------------------------------------------------------------------

# Acme
# https://console.cloud.google.com/apis/library/publicca.googleapis.com?project=temporal-genius-1919810
# Google cloud shell:  gcloud beta publicca external-account-keys create

# https://dash.cloudflare.com/
# https://dash.cloudflare.com/profile/api-tokens

CERTDIR="/etc/nginx/cert"
DOMAIN="wzs.app"
USEREMAIL="uuneox@gmail.com"
GOOGLEKID=""
GOOGLEKEY=""
export CF_Token=""
export CF_Account_ID=""

curl https://get.acme.sh | sh -s email="${USEREMAIL}"


~/.acme.sh/acme.sh --set-default-ca --server google

~/.acme.sh/acme.sh --register-account -m "${USEREMAIL}" --server google \
        --eab-kid "${GOOGLEKID}" \
        --eab-hmac-key ${GOOGLEKEY}

~/.acme.sh/acme.sh --issue --dns dns_cf -d "${DOMAIN}" -d "*.${DOMAIN}"

mkdir -p "${CERTDIR}"

~/.acme.sh/acme.sh --install-cert -d "${DOMAIN}" \
--key-file       "${CERTDIR}/${DOMAIN}.key"  \
--fullchain-file "${CERTDIR}/${DOMAIN}.crt" \
--reloadcmd     "nginx -s reload"

#---------------------------------------------------------------------------------------------