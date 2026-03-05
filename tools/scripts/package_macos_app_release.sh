#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi

APP_PROJECT_PATH="${ROOT}/source/apps/macos/app-host/PersonalAgent.xcodeproj"
APP_SCHEME="PersonalAgent"
APP_NAME="PersonalAgent.app"
APP_EXECUTABLE_NAME="PersonalAgent"
DAEMON_APP_NAME="Personal Agent Daemon.app"
DAEMON_EXECUTABLE_NAME="personal-agent-daemon"

OUTPUT_DIR="${ROOT}/out/dist/macos-release"
DERIVED_DATA_PATH="${ROOT}/out/build/xcode-release-derived-data"
DMG_NAME="PersonalAgent-unsigned.dmg"
DMG_VOLUME_NAME="PersonalAgent"
DMG_BACKGROUND_IMAGE="${ROOT}/tools/assets/dmg/dmg-background.png"
STYLE_DMG_WINDOW=true
DAEMON_SOURCE_BINARY=""
GO_BIN="${GO_BIN:-go}"
XCODEBUILD_BIN="${XCODEBUILD_BIN:-xcodebuild}"
XCODEGEN_BIN="${XCODEGEN_BIN:-xcodegen}"
SKIP_CLEAN=false

usage() {
  cat <<'USAGE'
Usage: package_macos_app_release.sh [options]

Builds an unsigned/ad-hoc macOS release artifact set for local/internal distribution:
- Release PersonalAgent.app
- Embedded daemon helper app inside PersonalAgent.app
- Drag-install DMG with /Applications link
- SHA256 checksums and release manifest

Options:
  --output-dir <path>          Output directory (default: out/dist/macos-release)
  --derived-data-path <path>   Xcode derived data path (default: out/build/xcode-release-derived-data)
  --dmg-name <name>            Output dmg file name (default: PersonalAgent-unsigned.dmg)
  --dmg-volume-name <name>     DMG volume name (default: PersonalAgent)
  --dmg-background-image <p>   DMG background artwork path (default: tools/assets/dmg/dmg-background.png)
  --no-dmg-style               Disable Finder window styling for the DMG
  --daemon-source-binary <p>   Existing daemon binary path to package in helper app
  --go-bin <path>              Go binary for daemon helper packaging (default: go)
  --xcodebuild-bin <path>      xcodebuild binary path (default: xcodebuild)
  --xcodegen-bin <path>        xcodegen binary path (default: xcodegen)
  --skip-clean                 Keep prior output artifacts when possible
  --help                       Show this help
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --output-dir)
      OUTPUT_DIR="${2:-}"
      shift 2
      ;;
    --derived-data-path)
      DERIVED_DATA_PATH="${2:-}"
      shift 2
      ;;
    --dmg-name)
      DMG_NAME="${2:-}"
      shift 2
      ;;
    --dmg-volume-name)
      DMG_VOLUME_NAME="${2:-}"
      shift 2
      ;;
    --dmg-background-image)
      DMG_BACKGROUND_IMAGE="${2:-}"
      shift 2
      ;;
    --no-dmg-style)
      STYLE_DMG_WINDOW=false
      shift
      ;;
    --daemon-source-binary)
      DAEMON_SOURCE_BINARY="${2:-}"
      shift 2
      ;;
    --go-bin)
      GO_BIN="${2:-}"
      shift 2
      ;;
    --xcodebuild-bin)
      XCODEBUILD_BIN="${2:-}"
      shift 2
      ;;
    --xcodegen-bin)
      XCODEGEN_BIN="${2:-}"
      shift 2
      ;;
    --skip-clean)
      SKIP_CLEAN=true
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

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "package_macos_app_release.sh requires macOS (Darwin host)." >&2
  exit 1
fi

for cmd in "${XCODEGEN_BIN}" "${XCODEBUILD_BIN}" hdiutil shasum ditto; do
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "required command not found: ${cmd}" >&2
    exit 1
  fi
done

if [[ ! -d "${APP_PROJECT_PATH}" ]]; then
  echo "app project not found: ${APP_PROJECT_PATH}" >&2
  exit 1
fi

if [[ -n "${DAEMON_SOURCE_BINARY// }" && ! -f "${DAEMON_SOURCE_BINARY}" ]]; then
  echo "daemon source binary not found: ${DAEMON_SOURCE_BINARY}" >&2
  exit 1
fi

if [[ "${STYLE_DMG_WINDOW}" == "true" && ! -f "${DMG_BACKGROUND_IMAGE}" ]]; then
  echo "warning: dmg background image not found (${DMG_BACKGROUND_IMAGE}); continuing without DMG styling" >&2
  STYLE_DMG_WINDOW=false
fi

OUTPUT_APP="${OUTPUT_DIR}/${APP_NAME}"
STAGING_DIR="${OUTPUT_DIR}/staging"
DMG_STAGING_DIR="${OUTPUT_DIR}/dmg-staging"
OUTPUT_DMG="${OUTPUT_DIR}/${DMG_NAME}"
RW_DMG_PATH="${OUTPUT_DIR}/.${DMG_NAME%.dmg}.rw.dmg"
CHECKSUMS_FILE="${OUTPUT_DIR}/SHA256SUMS.txt"
MANIFEST_FILE="${OUTPUT_DIR}/release-manifest.json"

if [[ "${SKIP_CLEAN}" == "false" ]]; then
  rm -rf "${OUTPUT_APP}" "${STAGING_DIR}" "${DMG_STAGING_DIR}" "${OUTPUT_DMG}" "${RW_DMG_PATH}" "${CHECKSUMS_FILE}" "${MANIFEST_FILE}"
fi

mkdir -p "${OUTPUT_DIR}" "${DERIVED_DATA_PATH}" "${STAGING_DIR}" "${DMG_STAGING_DIR}"

(
  cd "${ROOT}/source/apps/macos/app-host"
  "${XCODEGEN_BIN}" generate >/dev/null
)

"${XCODEBUILD_BIN}" \
  -project "${APP_PROJECT_PATH}" \
  -scheme "${APP_SCHEME}" \
  -configuration Release \
  -derivedDataPath "${DERIVED_DATA_PATH}" \
  CODE_SIGNING_ALLOWED=NO \
  build >/dev/null

BUILT_APP="${DERIVED_DATA_PATH}/Build/Products/Release/${APP_NAME}"
if [[ ! -d "${BUILT_APP}" ]]; then
  echo "built app bundle not found: ${BUILT_APP}" >&2
  exit 1
fi

rm -rf "${OUTPUT_APP}"
ditto "${BUILT_APP}" "${OUTPUT_APP}"

STAGED_DAEMON_APP="${STAGING_DIR}/${DAEMON_APP_NAME}"
DAEMON_ARGS=(
  "${ROOT}/tools/scripts/package_daemon_app_macos.sh"
  --output-app "${STAGED_DAEMON_APP}"
  --skip-sign
  --go-bin "${GO_BIN}"
)
if [[ -n "${DAEMON_SOURCE_BINARY// }" ]]; then
  DAEMON_ARGS+=(--source-binary "${DAEMON_SOURCE_BINARY}")
fi
"${DAEMON_ARGS[@]}" >/dev/null

EMBEDDED_DAEMON_DIR="${OUTPUT_APP}/Contents/Resources/Daemon"
EMBEDDED_DAEMON_APP="${EMBEDDED_DAEMON_DIR}/${DAEMON_APP_NAME}"
mkdir -p "${EMBEDDED_DAEMON_DIR}"
rm -rf "${EMBEDDED_DAEMON_APP}"
ditto "${STAGED_DAEMON_APP}" "${EMBEDDED_DAEMON_APP}"

rm -rf "${DMG_STAGING_DIR}"
mkdir -p "${DMG_STAGING_DIR}"
ditto "${OUTPUT_APP}" "${DMG_STAGING_DIR}/${APP_NAME}"
ln -sfn /Applications "${DMG_STAGING_DIR}/Applications"

rm -f "${OUTPUT_DMG}" "${RW_DMG_PATH}"
hdiutil create \
  -volname "${DMG_VOLUME_NAME}" \
  -srcfolder "${DMG_STAGING_DIR}" \
  -ov \
  -format UDRW \
  "${RW_DMG_PATH}" >/dev/null

DMG_MOUNT_DIR=""
DMG_DEVICE=""
cleanup_dmg_artifacts() {
  if [[ -n "${DMG_DEVICE}" ]]; then
    hdiutil detach "${DMG_DEVICE}" -force >/dev/null 2>&1 || true
    DMG_DEVICE=""
  fi
  if [[ -n "${DMG_MOUNT_DIR}" && -d "${DMG_MOUNT_DIR}" ]]; then
    rm -rf "${DMG_MOUNT_DIR}"
  fi
}
trap cleanup_dmg_artifacts EXIT

if [[ "${STYLE_DMG_WINDOW}" == "true" ]]; then
  DMG_MOUNT_DIR="$(mktemp -d "${OUTPUT_DIR}/.dmg-mount.XXXXXX")"
  ATTACH_OUTPUT="$(hdiutil attach -readwrite -noverify -noautoopen -mountpoint "${DMG_MOUNT_DIR}" "${RW_DMG_PATH}")"
  DMG_DEVICE="$(printf '%s\n' "${ATTACH_OUTPUT}" | awk '/^\/dev\// {print $1; exit}')"
  if [[ -z "${DMG_DEVICE}" ]]; then
    echo "warning: unable to determine mounted device for DMG styling; continuing with default Finder layout" >&2
  else
    mkdir -p "${DMG_MOUNT_DIR}/.background"
    ditto "${DMG_BACKGROUND_IMAGE}" "${DMG_MOUNT_DIR}/.background/dmg-background.png"
    chflags hidden "${DMG_MOUNT_DIR}/.background" >/dev/null 2>&1 || true

    if ! osascript - "${DMG_VOLUME_NAME}" "${APP_NAME}" "Applications" "${DMG_MOUNT_DIR}" <<'APPLESCRIPT'
on run argv
  set volumeName to item 1 of argv
  set appName to item 2 of argv
  set applicationsAliasName to item 3 of argv
  set mountPath to item 4 of argv
  set backgroundImageAlias to POSIX file (mountPath & "/.background/dmg-background.png") as alias
  tell application "Finder"
    tell disk volumeName
      open
      delay 0.6
      set current view of container window to icon view
      set toolbar visible of container window to false
      set statusbar visible of container window to false
      set bounds of container window to {180, 120, 860, 560}
      set viewOptions to icon view options of container window
      set arrangement of viewOptions to not arranged
      set icon size of viewOptions to 128
      set text size of viewOptions to 13
      set background picture of viewOptions to backgroundImageAlias
      set position of item appName of container window to {190, 250}
      set position of item applicationsAliasName of container window to {490, 250}
      close
      open
      delay 0.8
      update without registering applications
      delay 0.8
      close
    end tell
  end tell
end run
APPLESCRIPT
    then
      echo "warning: Finder DMG styling command failed; continuing with default Finder layout" >&2
    fi
    sync
    sleep 2
    if [[ ! -f "${DMG_MOUNT_DIR}/.DS_Store" ]]; then
      echo "warning: Finder did not persist DMG window metadata (.DS_Store missing); DMG may open with default layout" >&2
    fi
  fi
fi

sync
cleanup_dmg_artifacts
trap - EXIT

DMG_OUTPUT_TARGET="${OUTPUT_DMG%.dmg}"
hdiutil convert "${RW_DMG_PATH}" -ov -format UDZO -imagekey zlib-level=9 -o "${DMG_OUTPUT_TARGET}" >/dev/null
if [[ "${DMG_OUTPUT_TARGET}.dmg" != "${OUTPUT_DMG}" ]]; then
  mv -f "${DMG_OUTPUT_TARGET}.dmg" "${OUTPUT_DMG}"
fi
rm -f "${RW_DMG_PATH}"

APP_EXECUTABLE_PATH="${OUTPUT_APP}/Contents/MacOS/${APP_EXECUTABLE_NAME}"
EMBEDDED_DAEMON_EXECUTABLE_PATH="${EMBEDDED_DAEMON_APP}/Contents/MacOS/${DAEMON_EXECUTABLE_NAME}"

if [[ ! -x "${APP_EXECUTABLE_PATH}" ]]; then
  echo "packaged app executable missing: ${APP_EXECUTABLE_PATH}" >&2
  exit 1
fi
if [[ ! -x "${EMBEDDED_DAEMON_EXECUTABLE_PATH}" ]]; then
  echo "embedded daemon executable missing: ${EMBEDDED_DAEMON_EXECUTABLE_PATH}" >&2
  exit 1
fi

DMG_SHA256="$(shasum -a 256 "${OUTPUT_DMG}" | awk '{print $1}')"
APP_EXEC_SHA256="$(shasum -a 256 "${APP_EXECUTABLE_PATH}" | awk '{print $1}')"
EMBEDDED_DAEMON_SHA256="$(shasum -a 256 "${EMBEDDED_DAEMON_EXECUTABLE_PATH}" | awk '{print $1}')"

cat > "${CHECKSUMS_FILE}" <<EOF_SUMS
${DMG_SHA256}  ${DMG_NAME}
${APP_EXEC_SHA256}  ${APP_NAME}/Contents/MacOS/${APP_EXECUTABLE_NAME}
${EMBEDDED_DAEMON_SHA256}  ${APP_NAME}/Contents/Resources/Daemon/${DAEMON_APP_NAME}/Contents/MacOS/${DAEMON_EXECUTABLE_NAME}
EOF_SUMS

APP_INFO_PLIST="${OUTPUT_APP}/Contents/Info.plist"
DAEMON_INFO_PLIST="${EMBEDDED_DAEMON_APP}/Contents/Info.plist"

APP_BUNDLE_ID="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "${APP_INFO_PLIST}" 2>/dev/null || echo "com.personalagent.app")"
APP_VERSION="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "${APP_INFO_PLIST}" 2>/dev/null || echo "unknown")"
DAEMON_BUNDLE_ID="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "${DAEMON_INFO_PLIST}" 2>/dev/null || echo "com.personalagent.daemon")"
DAEMON_VERSION="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "${DAEMON_INFO_PLIST}" 2>/dev/null || echo "unknown")"
GIT_REV="$(git -C "${ROOT}" rev-parse --short HEAD 2>/dev/null || echo "unknown")"
GENERATED_AT="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

cat > "${MANIFEST_FILE}" <<EOF_MANIFEST
{
  "generated_at_utc": "${GENERATED_AT}",
  "distribution_mode": "local_internal_unsigned",
  "git_revision": "${GIT_REV}",
  "app": {
    "bundle_path": "${OUTPUT_APP}",
    "bundle_id": "${APP_BUNDLE_ID}",
    "version": "${APP_VERSION}",
    "executable_path": "${APP_EXECUTABLE_PATH}",
    "executable_sha256": "${APP_EXEC_SHA256}"
  },
  "embedded_daemon": {
    "bundle_path": "${EMBEDDED_DAEMON_APP}",
    "bundle_id": "${DAEMON_BUNDLE_ID}",
    "version": "${DAEMON_VERSION}",
    "executable_path": "${EMBEDDED_DAEMON_EXECUTABLE_PATH}",
    "executable_sha256": "${EMBEDDED_DAEMON_SHA256}"
  },
  "dmg": {
    "path": "${OUTPUT_DMG}",
    "sha256": "${DMG_SHA256}",
    "volume_name": "${DMG_VOLUME_NAME}"
  },
  "checksums_file": "${CHECKSUMS_FILE}"
}
EOF_MANIFEST

echo "packaged app: ${OUTPUT_APP}"
echo "embedded daemon helper: ${EMBEDDED_DAEMON_APP}"
echo "packaged dmg: ${OUTPUT_DMG}"
echo "checksums: ${CHECKSUMS_FILE}"
echo "manifest: ${MANIFEST_FILE}"
