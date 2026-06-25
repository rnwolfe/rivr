#!/bin/sh
# rivr installer — https://rivr.sh/install.sh
#
#   curl -fsSL https://rivr.sh/install.sh | sh
#
# Env:
#   RIVR_INSTALL_DIR   where to install (default: $HOME/.local/bin)
#   RIVR_VERSION       version to install, e.g. v0.3.0 (default: latest release)
#
# Safety: downloads a signed release tarball from GitHub over HTTPS and VERIFIES its SHA-256
# against the release `checksums.txt` before installing. The whole script is wrapped in a
# function invoked on the last line, so a truncated download can't execute a partial script.
# Prefer `go install` / `brew` if you'd rather not pipe to a shell; or download + inspect:
#   curl -fsSL https://rivr.sh/install.sh -o install.sh  # read it, then: sh install.sh
set -eu

REPO="rnwolfe/rivr"
BIN="rivr"

err() { echo "rivr install: error: $*" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

verify_sha256() { # <file> <expected-hex>
  _f="$1"; _want="$2"
  if have sha256sum; then _got=$(sha256sum "$_f" | awk '{print $1}')
  elif have shasum; then _got=$(shasum -a 256 "$_f" | awk '{print $1}')
  else err "need sha256sum or shasum to verify the download"; fi
  [ "$_got" = "$_want" ] || err "checksum mismatch for $(basename "$_f") (got $_got, want $_want)"
}

main() {
  have curl || err "curl is required"
  have tar || err "tar is required"

  os=$(uname -s); arch=$(uname -m)
  case "$os" in
    Linux) os=linux ;;
    Darwin) os=darwin ;;
    *) err "unsupported OS: $os (use 'go install' or download a binary from GitHub Releases)" ;;
  esac
  case "$arch" in
    x86_64|amd64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *) err "unsupported arch: $arch" ;;
  esac

  ver="${RIVR_VERSION:-}"
  if [ -z "$ver" ]; then
    ver=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
      | sed -n 's/.*"tag_name"[ ]*:[ ]*"\([^"]*\)".*/\1/p' | head -n1)
  fi
  [ -n "$ver" ] || err "could not determine the latest version"

  tarball="${BIN}_${ver#v}_${os}_${arch}.tar.gz"
  base="https://github.com/$REPO/releases/download/$ver"

  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT INT TERM

  echo "rivr install: downloading $tarball ($ver)"
  curl -fsSL "$base/$tarball" -o "$tmp/$tarball" || err "download failed: $base/$tarball"
  curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt" || err "could not fetch checksums.txt"

  want=$(grep " $tarball\$" "$tmp/checksums.txt" | awk '{print $1}' | head -n1)
  [ -n "$want" ] || err "no checksum found for $tarball"
  verify_sha256 "$tmp/$tarball" "$want"

  tar -xzf "$tmp/$tarball" -C "$tmp" || err "extract failed"
  [ -f "$tmp/$BIN" ] || err "binary $BIN not found in archive"

  dir="${RIVR_INSTALL_DIR:-$HOME/.local/bin}"
  mkdir -p "$dir"
  if ! install -m 0755 "$tmp/$BIN" "$dir/$BIN" 2>/dev/null; then
    mv "$tmp/$BIN" "$dir/$BIN" && chmod 0755 "$dir/$BIN"
  fi

  echo "rivr install: installed $ver to $dir/$BIN"
  case ":$PATH:" in
    *":$dir:"*) : ;;
    *) echo "rivr install: note: add $dir to your PATH (e.g. export PATH=\"$dir:\$PATH\")" ;;
  esac
  echo "rivr install: run 'rivr --help' to get started."
}

main "$@"
