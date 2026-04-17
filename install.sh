#!/bin/sh
# jotter install script — fetches the latest release binary for the current
# platform and installs it to $JOTTER_INSTALL_DIR (default: $HOME/.local/bin).
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/sebjacobs/jotter/main/install.sh | sh
#
# Overrides:
#   JOTTER_INSTALL_DIR=/path/to/bin  — target directory (must be writable)
#   JOTTER_VERSION=v0.1.0            — pin a specific release (default: latest)

set -eu

REPO="sebjacobs/jotter"
INSTALL_DIR="${JOTTER_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${JOTTER_VERSION:-}"

log() { printf '%s\n' "$*" >&2; }
fail() { log "error: $*"; exit 1; }

banner() {
  cat >&2 <<'BANNER'
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣠⣤⣶⣾⣿⣿⣷⣶⣶⣾⣿⣿⣷⣶⣤⣄⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣤⣶⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣶⣤⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⣴⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⣦⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢰⣿⠋⢉⣻⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣟⡉⠙⣿⡆⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠘⣿⣄⣐⣿⣿⣿⣿⣿⣿⡿⢋⡟⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⢻⡝⢿⣿⣿⣿⣿⣿⣿⣂⣠⣿⠃⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡼⠙⣿⣿⣿⣿⣿⣿⣿⣇⠀⢀⣿⣿⣿⣿⣿⠟⠉⠀⠀⠉⠻⣿⣿⣿⣿⣿⡀⠀⣸⣿⣿⣿⣿⣿⣿⣿⠛⢣⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣤⣚⣀⠌⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠋⠀⠀⠀⠀⠀⠀⠙⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⠁⣀⣓⣤⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⢪⣮⠆⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠃⠀⠀⠀⠀⠀⠀⠀⠀⠸⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠀⣝⣵⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⡳⠋⣾⣀⣸⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣧⡀⠀⠀⠀⠀⠀⠀⢀⣼⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠏⣀⣾⡙⢿⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣸⡞⣲⢫⠀⠀⢻⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⣄⣀⣀⣠⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡏⢁⠠⣻⣖⢳⣇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⢀⠤⠤⠒⠒⢅⡇⠁⠇⢤⣚⠀⠹⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠟⠸⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⢋⠁⣾⣷⡱⠈⢸⣈⠔⠒⠢⠤⢄⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⡃ ⠀⠀⠀⠀⠀⠀⠀⠘⢿⢣⡐⡄⢙⠿⣿⣿⣿⣿⣿⣿⣿⣿⣶⣧⣄⣀⣀⣤⣤⣶⣿⣿⣿⣿⣿⣿⣿⠿⠿⠁⠈⡗⣰⣿⣿⡧⠀⠀⠀⠀ ⠀⠀⠀⡇⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⡜⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠘⡟⣦⣻⣇⠀⡰⠁⠉⠛⠛⠿⠻⠿⠟⠻⠿⠿⠿⠿⠟⠿⠟⠻⠿⠟⠋⠉⠈⢂⠀⣱⣿⣾⣿⡿⢱⠁⠀⠀⠀⠀ ⠀⠀⠀⠱⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⢇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠘⠷⣽⣧⡟⢷⣤⡀⢄⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⡠⢒⢠⣸⣾⣿⡿⢻⠟⠁⠂⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠈⢄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠙⠻⣿⡿⣿⣷⣿⣷⣄⠀⠠⡀⠀⠀⠀⣠⠆⠀⠐⣚⣁⣠⣭⠉⠀⢋⠡⠂⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⠆⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠈⢆⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⡄⠙⠿⡎⢿⠷⠆⣛⣀⣠⣤⣴⣶⣾⣿⣿⣿⣿⣿⣷⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⠃⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⢻⠆⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣀⣠⣤⣶⣶⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠻⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⢀⣣⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡰⣶⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣰⡅⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⢱⡁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣷⢸⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣧⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⡕⠁⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠪⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢹⣧⢻⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣧⠀⠀⠀⠀⠀⠀⠀⠀⢀⠔⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠑⠤⡀⠀⠀⠀⠀⠀⠀⠀⠀⢻⣇⢻⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣧⠀⠀⠀⠀⢀⡠⠒⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⢂⠀⠀⠀⠀⠀⠀⠀⠈⢿⡎⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣧⠀⠀⢠⠋⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⠀⠀⠀⠀⠀⠀⠀⠀⠈⣿⡜⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣧⠀⠇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⢦⠀⠀⠀⠀⠀⠀⠀⠀⠀⠘⣷⡘⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠿⠿⢛⣛⠁⢜⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠘⡏⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠹⣧⠹⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠿⢟⣛⣫⢭⣕⡶⣮⣿⢷⡏⠀⠘⡃⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢰⢁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢻⣧⢻⣿⣿⣿⣿⠿⠿⢟⣛⣭⣭⣶⢾⣽⣿⣾⣿⠿⠿⠛⠛⠉⠉⠀⠀⢀⢃⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠸⠛⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⢿⡆⣋⣭⣕⣶⣶⣛⣷⣿⡻⠶⠝⠓⠋⠉⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠘⡺⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠘⠇⣻⠷⠟⠛⠊⠉⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
                         _       _   _
                        | |     | | | |
                        | | ___ | |_| |_ ___ _ __
                    _   | |/ _ \| __| __/ _ \ '__|
                   | |__| | (_) | |_| ||  __/ |
                    \____/ \___/ \__|\__\___|_|

BANNER
}

banner

# --- 1. Detect OS + arch ---
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *)
    fail "unsupported arch: $ARCH
fallback: go install github.com/$REPO@latest"
    ;;
esac
case "$OS" in
  darwin|linux) ;;
  *)
    fail "unsupported OS: $OS
fallback: go install github.com/$REPO@latest"
    ;;
esac

# --- 2. Resolve version ---
if [ -z "$VERSION" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | \
    awk -F'"' '/"tag_name":/ {print $4; exit}')
  [ -n "$VERSION" ] || fail "could not determine latest release tag"
fi
log "Installing jotter $VERSION ($OS/$ARCH) to $INSTALL_DIR"

# --- 3. Pick a sha256 verifier (Linux: sha256sum; macOS: shasum -a 256) ---
if command -v sha256sum >/dev/null 2>&1; then
  SHA256_CHECK='sha256sum -c -'
elif command -v shasum >/dev/null 2>&1; then
  SHA256_CHECK='shasum -a 256 -c -'
else
  fail "neither sha256sum nor shasum available — cannot verify checksum"
fi

# --- 4. Download archive + checksums ---
ARCHIVE="jotter_${VERSION#v}_${OS}_${ARCH}.tar.gz"
URL_BASE="https://github.com/$REPO/releases/download/$VERSION"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT INT TERM

curl -fsSL -o "$TMPDIR/$ARCHIVE" "$URL_BASE/$ARCHIVE"
curl -fsSL -o "$TMPDIR/checksums.txt" "$URL_BASE/checksums.txt"

# --- 5. Verify SHA-256 ---
(
  cd "$TMPDIR"
  grep " $ARCHIVE\$" checksums.txt | $SHA256_CHECK >/dev/null \
    || fail "checksum verification failed for $ARCHIVE"
)
log "Checksum verified."

# --- 6. Extract + install ---
(cd "$TMPDIR" && tar -xzf "$ARCHIVE")
mkdir -p "$INSTALL_DIR"
install -m 0755 "$TMPDIR/jotter" "$INSTALL_DIR/jotter"
log "Installed $INSTALL_DIR/jotter"

# --- 7. PATH check ---
case ":$PATH:" in
  *":$INSTALL_DIR:"*)
    log ""
    log "Next: run 'jotter --version' to confirm, then see:"
    log "  https://github.com/$REPO#setup"
    ;;
  *)
    log ""
    log "'$INSTALL_DIR' is not on your PATH — add this to your shell rc file:"
    log "  export PATH=\"$INSTALL_DIR:\$PATH\""
    log ""
    log "Then run 'jotter --version' to confirm, and see:"
    log "  https://github.com/$REPO#setup"
    ;;
esac
