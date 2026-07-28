#!/usr/bin/env bash
set -euo pipefail

REPOSITORY="xuanwolei/file-guard"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/file-guard"
CONFIG_FILE="${CONFIG_DIR}/config.ini"
SERVICE_FILE="/etc/systemd/system/file-guard.service"
SERVICE_NAME="file-guard.service"

VERSION=""
START_AFTER_INSTALL=0
UNINSTALL=0
PURGE=0

usage() {
  cat <<'EOF'
用法：
  sudo bash install.sh [--version <版本>] [--start]
  sudo bash install.sh --uninstall [--purge]

选项：
  --version <版本>  安装指定 Release 标签，例如 1.1.0；默认安装最新版本。
  --start           安装完成后立即启动服务。
  --uninstall       删除二进制文件和 systemd 服务。
  --purge           仅与 --uninstall 一起使用，同时删除 /etc/file-guard。
  -h, --help        显示帮助。
EOF
}

fail() {
  echo "错误：$*" >&2
  exit 1
}

need_command() {
  command -v "$1" >/dev/null 2>&1 || fail "缺少命令：$1"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      [[ $# -ge 2 ]] || fail "--version 需要版本号"
      VERSION="$2"
      shift 2
      ;;
    --start)
      START_AFTER_INSTALL=1
      shift
      ;;
    --uninstall)
      UNINSTALL=1
      shift
      ;;
    --purge)
      PURGE=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "未知参数：$1"
      ;;
  esac
done

[[ "${EUID}" -eq 0 ]] || fail "请使用 root 或 sudo 执行"

if [[ "${PURGE}" -eq 1 && "${UNINSTALL}" -ne 1 ]]; then
  fail "--purge 必须与 --uninstall 一起使用"
fi

if [[ "${UNINSTALL}" -eq 1 ]]; then
  if command -v systemctl >/dev/null 2>&1; then
    systemctl disable --now "${SERVICE_NAME}" 2>/dev/null || true
  fi
  rm -f "${SERVICE_FILE}" "${INSTALL_DIR}/file-guard"
  if [[ "${PURGE}" -eq 1 ]]; then
    rm -rf "${CONFIG_DIR}"
  fi
  if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload
  fi
  echo "file-guard 已卸载。"
  [[ "${PURGE}" -eq 1 ]] || echo "配置保留在：${CONFIG_FILE}"
  exit 0
fi

need_command uname
need_command sha256sum
need_command systemctl

case "$(uname -s)" in
  Linux) ;;
  *) fail "仅支持 Linux 系统" ;;
esac

case "$(uname -m)" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) fail "不支持的 CPU 架构：$(uname -m)" ;;
esac

if command -v curl >/dev/null 2>&1; then
  DOWNLOAD=(curl -fsSL --retry 3 --retry-delay 1)
elif command -v wget >/dev/null 2>&1; then
  DOWNLOAD=(wget -qO-)
else
  fail "需要 curl 或 wget 用于下载"
fi

download() {
  "${DOWNLOAD[@]}" "$1"
}

if [[ -z "${VERSION}" ]]; then
  VERSION="$(download "https://api.github.com/repos/${REPOSITORY}/releases/latest" \
    | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
    | head -n 1)"
  [[ -n "${VERSION}" ]] || fail "无法获取最新 Release 版本，请使用 --version 指定版本"
fi

ASSET="file-guard-linux-${ARCH}"
BASE_URL="https://github.com/${REPOSITORY}/releases/download/${VERSION}"
TEMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TEMP_DIR}"' EXIT

echo "下载 file-guard ${VERSION} (${ARCH})..."
download "${BASE_URL}/${ASSET}" > "${TEMP_DIR}/${ASSET}"
download "${BASE_URL}/SHA256SUMS" > "${TEMP_DIR}/SHA256SUMS"

expected_hash="$(awk -v asset="${ASSET}" '$2 == asset {print $1}' "${TEMP_DIR}/SHA256SUMS")"
[[ -n "${expected_hash}" ]] || fail "SHA256SUMS 中未找到 ${ASSET}"
actual_hash="$(sha256sum "${TEMP_DIR}/${ASSET}" | awk '{print $1}')"
[[ "${expected_hash}" == "${actual_hash}" ]] || fail "二进制 SHA256 校验失败"

was_active=0
if systemctl is-active --quiet "${SERVICE_NAME}"; then
  was_active=1
  systemctl stop "${SERVICE_NAME}"
fi

install -d -m 0755 "${INSTALL_DIR}" "${CONFIG_DIR}"
install -m 0755 "${TEMP_DIR}/${ASSET}" "${INSTALL_DIR}/file-guard"

if [[ ! -f "${CONFIG_FILE}" ]]; then
  cat > "${CONFIG_FILE}" <<'EOF'
; file-guard 配置。请创建至少一个项目配置后再启动服务。
notice_level = 3

[example]
log_file = /var/log/example/*.log
match_preg = "(?i)error"
notice_token =
EOF
  chmod 0640 "${CONFIG_FILE}"
  echo "已创建配置模板：${CONFIG_FILE}"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "${SCRIPT_DIR}/../systemd/file-guard.service" ]]; then
  install -m 0644 "${SCRIPT_DIR}/../systemd/file-guard.service" "${SERVICE_FILE}"
else
  cat > "${SERVICE_FILE}" <<'EOF'
[Unit]
Description=file-guard file monitoring service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/file-guard -c /etc/file-guard/config.ini
Restart=on-failure
RestartSec=5s
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF
fi

systemctl daemon-reload
if [[ "${was_active}" -eq 1 || "${START_AFTER_INSTALL}" -eq 1 ]]; then
  systemctl enable --now "${SERVICE_NAME}"
fi

echo "file-guard ${VERSION} 安装完成。"
echo "配置文件：${CONFIG_FILE}"
if [[ "${was_active}" -eq 0 && "${START_AFTER_INSTALL}" -eq 0 ]]; then
  echo "配置完成后执行：systemctl enable --now ${SERVICE_NAME}"
fi
