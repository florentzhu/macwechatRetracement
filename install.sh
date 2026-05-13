#!/usr/bin/env bash
#
# wechattweak installer
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/florentzhu/macwechatRetracement/main/install.sh | sudo bash
#
# Env vars:
#   PREFIX   install dir, default: /usr/local/bin
#   VERSION  release tag to install, default: latest
#

set -euo pipefail

REPO="florentzhu/macwechatRetracement"
BIN_NAME="wechattweak"
PREFIX="${PREFIX:-/usr/local/bin}"
VERSION="${VERSION:-latest}"

red()    { printf '\033[31m%s\033[0m\n' "$*"; }
green()  { printf '\033[32m%s\033[0m\n' "$*"; }
yellow() { printf '\033[33m%s\033[0m\n' "$*"; }
info()   { printf '==> %s\n' "$*"; }

# 1. OS check
if [[ "$(uname -s)" != "Darwin" ]]; then
  red "This installer only supports macOS."
  exit 1
fi

# 2. Privilege check (need write access to PREFIX)
if [[ ! -w "$PREFIX" ]] && [[ "$(id -u)" -ne 0 ]]; then
  red "No write permission to $PREFIX. Re-run with sudo, e.g.:"
  red "  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh | sudo bash"
  exit 1
fi

# 3. Resolve download URL
if [[ "$VERSION" == "latest" ]]; then
  URL="https://github.com/${REPO}/releases/latest/download/${BIN_NAME}"
else
  URL="https://github.com/${REPO}/releases/download/${VERSION}/${BIN_NAME}"
fi

# 4. Download to a temp file, then atomically move into place
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT
TMP_BIN="${TMP_DIR}/${BIN_NAME}"

info "Downloading ${BIN_NAME} (${VERSION}) from ${URL}"
if ! curl -fsSL --retry 3 -o "$TMP_BIN" "$URL"; then
  red "Download failed. Check that release '${VERSION}' exists at:"
  red "  https://github.com/${REPO}/releases"
  exit 1
fi

# Sanity check: a Mach-O binary should be > 100KB; an HTML 404 page is tiny.
SIZE=$(wc -c < "$TMP_BIN" | tr -d ' ')
if [[ "$SIZE" -lt 102400 ]]; then
  red "Downloaded file looks too small (${SIZE} bytes); aborting."
  exit 1
fi

# 5. Install
mkdir -p "$PREFIX"
DEST="${PREFIX}/${BIN_NAME}"
info "Installing to ${DEST}"
mv "$TMP_BIN" "$DEST"
chmod 0755 "$DEST"
xattr -dr com.apple.quarantine "$DEST" 2>/dev/null || true

# 6. PATH hint
case ":$PATH:" in
  *":$PREFIX:"*) ;;
  *) yellow "Note: ${PREFIX} is not in your PATH. Add it to your shell rc:"
     yellow "  export PATH=\"${PREFIX}:\$PATH\""
     ;;
esac

# 7. Verify
if "$DEST" --help >/dev/null 2>&1; then
  green "Installed: $("$DEST" --help 2>&1 | head -1)"
  green "Run \`${BIN_NAME} versions\` to list supported WeChat versions."
else
  yellow "Installed to ${DEST}, but failed to execute. Try running it manually."
fi
