#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/file-guard"
CONFIG_FILE="${CONFIG_DIR}/config.ini"
SERVICE_FILE="/etc/systemd/system/file-guard.service"
SERVICE_NAME="file-guard.service"

BINARY_PATH=""
START_AFTER_INSTALL=0
UNINSTALL=0
PURGE=0
TARGET_TMP=""

usage() {
  cat <<'EOF'
用法：
  sudo bash install-local.sh <二进制路径> [--start]
  sudo bash install-local.sh --uninstall [--purge]

选项：
  --start       安装完成后立即启动服务并设置开机自启。
  --uninstall   删除二进制文件和 systemd 服务。
  --purge       仅与 --uninstall 一起使用，同时删除 /etc/file-guard。
  -h, --help    显示帮助。
EOF
}

fail() {
  echo "错误：$*" >&2
  exit 1
}

need_command() {
  command -v "$1" >/dev/null 2>&1 || fail "缺少命令：$1"
}

cleanup() {
  if [[ -n "${TARGET_TMP}" ]]; then
    rm -f "${TARGET_TMP}"
  fi
}

trap cleanup EXIT

while [[ $# -gt 0 ]]; do
  case "$1" in
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
    --*)
      fail "未知参数：$1"
      ;;
    *)
      [[ -z "${BINARY_PATH}" ]] || fail "只能指定一个二进制文件"
      BINARY_PATH="$1"
      shift
      ;;
  esac
done

[[ "${EUID}" -eq 0 ]] || fail "请使用 root 或 sudo 执行"

if [[ "${PURGE}" -eq 1 && "${UNINSTALL}" -ne 1 ]]; then
  fail "--purge 必须与 --uninstall 一起使用"
fi

if [[ "${UNINSTALL}" -eq 1 ]]; then
  [[ -z "${BINARY_PATH}" ]] || fail "卸载时不能指定二进制路径"
  [[ "${START_AFTER_INSTALL}" -eq 0 ]] || fail "--start 不能与 --uninstall 一起使用"
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

[[ -n "${BINARY_PATH}" ]] || fail "请指定本地二进制文件路径"
[[ -f "${BINARY_PATH}" ]] || fail "二进制文件不存在或不是普通文件：${BINARY_PATH}"
[[ -r "${BINARY_PATH}" ]] || fail "二进制文件不可读：${BINARY_PATH}"
[[ -s "${BINARY_PATH}" ]] || fail "二进制文件为空：${BINARY_PATH}"

need_command uname
need_command install
need_command systemctl
need_command mv

case "$(uname -s)" in
  Linux) ;;
  *) fail "仅支持 Linux 系统" ;;
esac

case "$(uname -m)" in
  x86_64|amd64) EXPECTED_ARCH="x86-64|x86_64|amd64" ;;
  aarch64|arm64) EXPECTED_ARCH="aarch64|arm64" ;;
  *) fail "不支持的 CPU 架构：$(uname -m)" ;;
esac

if command -v file >/dev/null 2>&1; then
  FILE_INFO="$(file -b "${BINARY_PATH}")"
  [[ "${FILE_INFO}" == *ELF* ]] || fail "文件不是 Linux ELF 二进制：${BINARY_PATH}"
  echo "${FILE_INFO}" | grep -Eiq "${EXPECTED_ARCH}" || \
    fail "二进制架构与当前服务器不匹配：${FILE_INFO}"
fi

if command -v sha256sum >/dev/null 2>&1; then
  echo "本地二进制 SHA256：$(sha256sum "${BINARY_PATH}" | awk '{print $1}')"
fi

# 先完成本地复制，避免源文件或磁盘异常时提前停止现有服务。
install -d -m 0755 "${INSTALL_DIR}" "${CONFIG_DIR}"
TARGET_TMP="${INSTALL_DIR}/.file-guard.new.$$"
install -m 0755 "${BINARY_PATH}" "${TARGET_TMP}"

was_active=0
if systemctl is-active --quiet "${SERVICE_NAME}"; then
  was_active=1
  systemctl stop "${SERVICE_NAME}"
fi

mv -f "${TARGET_TMP}" "${INSTALL_DIR}/file-guard"
TARGET_TMP=""

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
  chmod 0644 "${SERVICE_FILE}"
fi

systemctl daemon-reload
if [[ "${was_active}" -eq 1 || "${START_AFTER_INSTALL}" -eq 1 ]]; then
  systemctl enable --now "${SERVICE_NAME}"
fi

echo "file-guard 本地安装完成。"
echo "二进制文件：${INSTALL_DIR}/file-guard"
echo "配置文件：${CONFIG_FILE}"
if [[ "${was_active}" -eq 0 && "${START_AFTER_INSTALL}" -eq 0 ]]; then
  echo "配置完成后执行：systemctl enable --now ${SERVICE_NAME}"
fi
