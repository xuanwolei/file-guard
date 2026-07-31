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
# 未指定代理时使用的公共 GitHub 中转站。所有中转失败后会直连 GitHub。
DEFAULT_PROXIES=(
  "https://gh-proxy.com/"
  "https://ghproxy.net/"
  "https://ghfast.top/"
)
CUSTOM_PROXIES=()
PROXIES=()

usage() {
  cat <<'EOF'
用法：
  sudo bash install.sh [--version <版本>] [--proxy <中转前缀>] [--start]
  sudo bash install.sh --uninstall [--purge]

选项：
  --version <版本>  安装指定 Release 标签，例如 1.1.0；默认安装最新版本。
  --proxy <前缀>    GitHub 中转前缀，可重复指定；例如 https://gh-proxy.com/
                    也可通过 FILE_GUARD_PROXYES 以英文逗号或空格分隔配置多个前缀。
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
    --proxy)
      [[ $# -ge 2 ]] || fail "--proxy 需要中转前缀"
      CUSTOM_PROXIES+=("$2")
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
  DOWNLOAD_TOOL="curl"
elif command -v wget >/dev/null 2>&1; then
  DOWNLOAD_TOOL="wget"
else
  fail "需要 curl 或 wget 用于下载"
fi

# 将代理前缀规范为可直接拼接原始 GitHub URL 的形式。
normalize_proxy() {
  local proxy="$1"
  [[ "${proxy}" =~ ^https?:// ]] || fail "代理地址必须以 http:// 或 https:// 开头：${proxy}"
  [[ "${proxy}" == */ ]] || proxy="${proxy}/"
  printf '%s\n' "${proxy}"
}

add_proxy() {
  local proxy
  proxy="$(normalize_proxy "$1")"
  local existing
  for existing in "${PROXIES[@]}"; do
    [[ "${existing}" == "${proxy}" ]] && return
  done
  PROXIES+=("${proxy}")
}

configure_proxies() {
  local -a configured=()
  local proxy
  if [[ "${#CUSTOM_PROXIES[@]}" -gt 0 ]]; then
    configured=("${CUSTOM_PROXIES[@]}")
  elif [[ -n "${FILE_GUARD_PROXYES:-}" ]]; then
    # 同时接受 README 中使用的逗号分隔形式和空格分隔形式。
    local proxy_list="${FILE_GUARD_PROXYES//,/ }"
    read -r -a configured <<< "${proxy_list}"
  else
    configured=("${DEFAULT_PROXIES[@]}")
  fi

  for proxy in "${configured[@]}"; do
    add_proxy "${proxy}"
  done
}

download_to() {
  local url="$1"
  local output="$2"
  if [[ "${DOWNLOAD_TOOL}" == "curl" ]]; then
    curl -fsSL --connect-timeout 10 --max-time 180 --retry 3 --retry-delay 1 \
      -o "${output}" "${url}"
  else
    wget -q --timeout=10 --tries=3 -O "${output}" "${url}"
  fi
}

candidate_url() {
  local proxy="$1"
  local github_url="$2"
  if [[ -n "${proxy}" ]]; then
    printf '%s%s\n' "${proxy}" "${github_url}"
  else
    printf '%s\n' "${github_url}"
  fi
}

fetch_latest_version() {
  local github_url="https://api.github.com/repos/${REPOSITORY}/releases/latest"
  local proxy candidate version

  for proxy in "${PROXIES[@]}" ""; do
    candidate="$(candidate_url "${proxy}" "${github_url}")"
    rm -f "${LATEST_FILE}"
    if ! download_to "${candidate}" "${LATEST_FILE}"; then
      echo "下载失败，尝试下一个来源：${candidate}" >&2
      continue
    fi

    version="$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "${LATEST_FILE}" \
      | head -n 1)"
    if [[ -n "${version}" ]]; then
      printf '%s\n' "${version}"
      return 0
    fi
    echo "最新版本响应无效，尝试下一个来源：${candidate}" >&2
  done
  fail "无法获取最新 Release 版本，请使用 --version 指定版本"
}

extract_expected_hash() {
  local sums_file="$1"
  local asset="$2"
  local line hash name extra matched_hash="" count=0

  # 不依赖不同 awk 实现对字符类的兼容性，直接用 Bash 解析校验清单。
  while IFS= read -r line || [[ -n "${line}" ]]; do
    line="${line%$'\r'}"
    read -r hash name extra <<< "${line}"
    [[ "${name:-}" == "${asset}" && -z "${extra:-}" ]] || continue
    [[ "${hash:-}" =~ ^[0-9a-fA-F]{64}$ ]] || return 1
    matched_hash="${hash,,}"
    count=$((count + 1))
  done < "${sums_file}"

  [[ "${count}" -eq 1 ]] || return 1
  printf '%s\n' "${matched_hash}"
}

# 二进制和校验清单必须来自同一个候选来源。代理返回 HTTP 200 错误页时，
# 校验清单格式无效会尝试下一个来源；二进制哈希不匹配则立即终止。
download_release() {
  local github_base_url="$1"
  local asset="$2"
  local asset_file="$3"
  local sums_file="$4"
  local proxy candidate expected_hash actual_hash

  for proxy in "${PROXIES[@]}" ""; do
    candidate="$(candidate_url "${proxy}" "${github_base_url}")"
    rm -f "${asset_file}" "${sums_file}"
    if ! download_to "${candidate}/SHA256SUMS" "${sums_file}"; then
      echo "下载失败，尝试下一个来源：${candidate}" >&2
      continue
    fi

    if ! expected_hash="$(extract_expected_hash "${sums_file}" "${asset}")"; then
      echo "校验清单无效，尝试下一个来源：${candidate}" >&2
      continue
    fi

    if ! download_to "${candidate}/${asset}" "${asset_file}"; then
      echo "下载失败，尝试下一个来源：${candidate}" >&2
      continue
    fi

    actual_hash="$(sha256sum "${asset_file}" | awk '{print $1}')"
    [[ "${expected_hash}" == "${actual_hash}" ]] || fail "二进制 SHA256 校验失败（来源：${candidate}）"
    return 0
  done
  fail "无法下载有效的 Release 文件：${github_base_url}"
}

configure_proxies

if [[ -z "${VERSION}" ]]; then
  LATEST_FILE="$(mktemp)"
  VERSION="$(fetch_latest_version)"
fi

ASSET="file-guard-linux-${ARCH}"
BASE_URL="https://github.com/${REPOSITORY}/releases/download/${VERSION}"
TEMP_DIR="$(mktemp -d)"
trap 'rm -f "${LATEST_FILE:-}"; rm -rf "${TEMP_DIR}"' EXIT

echo "下载 file-guard ${VERSION} (${ARCH})..."
download_release "${BASE_URL}" "${ASSET}" "${TEMP_DIR}/${ASSET}" "${TEMP_DIR}/SHA256SUMS"

was_active=0
if systemctl is-active --quiet "${SERVICE_NAME}"; then
  was_active=1
  systemctl stop "${SERVICE_NAME}"
fi

install -d -m 0755 "${INSTALL_DIR}" "${CONFIG_DIR}"
install -m 0755 "${TEMP_DIR}/${ASSET}" "${INSTALL_DIR}/file-guard"

if [[ ! -f "${CONFIG_FILE}" ]]; then
  cat > "${CONFIG_FILE}" <<'EOF'
; file-guard 配置模板
; 全局配置写在项目段之前，项目段配置优先于全局配置。
; 至少保留一个项目段，并填写有效的 log_file 后再启动服务。

; -------------------- 全局配置 --------------------
; 通知级别：1-8，数值越小通知越频繁。
notice_level = 3

; 钉钉机器人配置。多个项目共用机器人时可填写在这里。
; notice_token =
; notice_mobile =

; 日志驱动类型，当前通常使用 error。
; log_driver = error

; 是否递归查找目录：1 开启，0 关闭。
; log_recursive_find = 0

; 用于相同日志限流标识的截取长度和跳过字节数。
; log_check_length = 30
; log_skip_length = 0

; 命中日志后是否定时扫描并重载文件：1 开启，0 关闭。
; auto_reload = 0
; auto_reload_interval = 3600

; 多行堆栈采集：命中后收集前置上下文和后续堆栈行。
; multiline_enabled = 0
; multiline_context_before_lines = 20
; multiline_continue_preg = "^(\\s+at\\s|\\s*Caused by:|\\s*#\\d+|\\s*goroutine\\s|\\s+File\\s|\\s*Traceback|\\s+)"
; multiline_flush_timeout_ms = 1000
; multiline_max_lines = 120
; multiline_max_bytes = 65536

; 钉钉 Markdown 通知的 UTF-8 字节限制和元数据预留空间。
; notice_max_bytes = 12000
; notice_reserved_bytes = 1024

; -------------------- 项目配置 --------------------
; 项目配置会覆盖同名全局配置。
[example]

; 监控文件路径，支持 * 匹配。请按实际日志路径修改。
log_file = /var/log/example/*.log

; 正则匹配规则，匹配后发送通知。
match_preg = "(?i)(error|exception|fatal|panic)"

; 过滤规则：匹配到的内容不会通知；无需过滤时保持为空。
filter_preg =

; 项目独立的钉钉机器人 token；为空时继承全局配置。
notice_token =

; 以下配置可按项目需要覆盖全局配置：
; notice_level = 3
; notice_mobile =
; log_recursive_find = 0
; log_check_length = 30
; log_skip_length = 0
; auto_reload = 0
; auto_reload_interval = 3600
; multiline_enabled = 0
; multiline_context_before_lines = 20
; multiline_continue_preg = "^(\\s+at\\s|\\s*Caused by:|\\s*#\\d+|\\s*goroutine\\s|\\s+File\\s|\\s*Traceback|\\s+)"
; multiline_flush_timeout_ms = 1000
; multiline_max_lines = 120
; multiline_max_bytes = 65536
; notice_max_bytes = 12000
; notice_reserved_bytes = 1024
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
