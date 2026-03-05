#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi

APP_NAME="Personal Agent Daemon"
BUNDLE_ID="com.personalagent.daemon"
VERSION="1.0.0"
EXECUTABLE_NAME="personal-agent-daemon"
OUTPUT_APP="$HOME/Applications/${APP_NAME}.app"
SOURCE_BINARY=""
GO_BIN="${GO_BIN:-go}"
SIGN_IDENTITY="-"
ENTITLEMENTS_FILE=""
SKIP_SIGN=false

usage() {
  cat <<'EOF'
Usage: package_daemon_app_macos.sh [options]

Options:
  --output-app <path>          Output .app path (default: ~/Applications/Personal Agent Daemon.app)
  --app-name <name>            App display name (default: Personal Agent Daemon)
  --bundle-id <id>             Bundle identifier (default: com.personalagent.daemon)
  --version <version>          Bundle version string (default: 1.0.0)
  --source-binary <path>       Existing daemon binary to package (default: build from source/services/daemon-go/cmd/personal-agent-daemon)
  --go-bin <path>              Go binary for build step (default: go)
  --sign-identity <id>         codesign identity (default: '-' ad-hoc)
  --entitlements <path>        Entitlements plist for codesign
  --skip-sign                  Skip codesign step
  --help                       Show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --output-app)
      OUTPUT_APP="${2:-}"
      shift 2
      ;;
    --app-name)
      APP_NAME="${2:-}"
      shift 2
      ;;
    --bundle-id)
      BUNDLE_ID="${2:-}"
      shift 2
      ;;
    --version)
      VERSION="${2:-}"
      shift 2
      ;;
    --source-binary)
      SOURCE_BINARY="${2:-}"
      shift 2
      ;;
    --go-bin)
      GO_BIN="${2:-}"
      shift 2
      ;;
    --sign-identity)
      SIGN_IDENTITY="${2:-}"
      shift 2
      ;;
    --entitlements)
      ENTITLEMENTS_FILE="${2:-}"
      shift 2
      ;;
    --skip-sign)
      SKIP_SIGN=true
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "${OUTPUT_APP// }" ]]; then
  echo "--output-app cannot be empty" >&2
  exit 2
fi
if [[ -z "${APP_NAME// }" ]]; then
  echo "--app-name cannot be empty" >&2
  exit 2
fi
if [[ -z "${BUNDLE_ID// }" ]]; then
  echo "--bundle-id cannot be empty" >&2
  exit 2
fi
if [[ -z "${VERSION// }" ]]; then
  echo "--version cannot be empty" >&2
  exit 2
fi

APP_CONTENTS="${OUTPUT_APP}/Contents"
APP_MACOS="${APP_CONTENTS}/MacOS"
APP_RESOURCES="${APP_CONTENTS}/Resources"
APP_EXECUTABLE="${APP_MACOS}/${EXECUTABLE_NAME}"
INFO_PLIST="${APP_CONTENTS}/Info.plist"
BUILD_DIR="${ROOT}/out/dist/macos-daemon-package"
STAGING_BINARY="${BUILD_DIR}/${EXECUTABLE_NAME}"

mkdir -p "${BUILD_DIR}"
if [[ -n "${SOURCE_BINARY// }" ]]; then
  if [[ ! -f "${SOURCE_BINARY}" ]]; then
    echo "source binary not found: ${SOURCE_BINARY}" >&2
    exit 1
  fi
  cp "${SOURCE_BINARY}" "${STAGING_BINARY}"
else
  (
    cd "${ROOT}/source/services/daemon-go"
    "${GO_BIN}" build -o "${STAGING_BINARY}" ./cmd/personal-agent-daemon
  )
fi
chmod +x "${STAGING_BINARY}"

rm -rf "${OUTPUT_APP}"
mkdir -p "${APP_MACOS}" "${APP_RESOURCES}"
cp "${STAGING_BINARY}" "${APP_EXECUTABLE}"
chmod +x "${APP_EXECUTABLE}"

cat > "${INFO_PLIST}" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key>
  <string>en</string>
  <key>CFBundleDisplayName</key>
  <string>${APP_NAME}</string>
  <key>CFBundleExecutable</key>
  <string>${EXECUTABLE_NAME}</string>
  <key>CFBundleIdentifier</key>
  <string>${BUNDLE_ID}</string>
  <key>CFBundleInfoDictionaryVersion</key>
  <string>6.0</string>
  <key>CFBundleName</key>
  <string>${APP_NAME}</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>${VERSION}</string>
  <key>CFBundleVersion</key>
  <string>${VERSION}</string>
  <key>LSBackgroundOnly</key>
  <true/>
  <key>NSAppleEventsUsageDescription</key>
  <string>Personal Agent Daemon needs Automation access to control Mail, Calendar, Messages, and Safari on your behalf.</string>
</dict>
</plist>
EOF

resolved_entitlements="${ENTITLEMENTS_FILE}"
cleanup_entitlements=false
if [[ "${SKIP_SIGN}" == "false" ]]; then
  if ! command -v codesign >/dev/null 2>&1; then
    echo "codesign not found; rerun with --skip-sign or install Xcode command line tools." >&2
    exit 1
  fi
  if [[ -z "${resolved_entitlements// }" ]]; then
    resolved_entitlements="${BUILD_DIR}/daemon-automation.entitlements"
    cleanup_entitlements=true
    cat > "${resolved_entitlements}" <<'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>com.apple.security.automation.apple-events</key>
  <true/>
</dict>
</plist>
EOF
  fi
  if [[ ! -f "${resolved_entitlements}" ]]; then
    echo "entitlements file not found: ${resolved_entitlements}" >&2
    exit 1
  fi
  codesign --force --deep --sign "${SIGN_IDENTITY}" --timestamp=none --entitlements "${resolved_entitlements}" "${OUTPUT_APP}"
  codesign --verify --deep --strict "${OUTPUT_APP}"
else
  echo "[package-daemon-app] signing skipped (--skip-sign)."
fi

if [[ "${cleanup_entitlements}" == "true" ]]; then
  rm -f "${resolved_entitlements}"
fi

echo "packaged daemon app: ${OUTPUT_APP}"
echo "daemon executable: ${APP_EXECUTABLE}"
echo "bundle identifier: ${BUNDLE_ID}"
echo "next step: ./tools/scripts/install_daemon_service_macos.sh --daemon-app \"${OUTPUT_APP}\" --auth-token-file <path>"
