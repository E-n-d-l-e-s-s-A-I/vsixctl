#!/bin/sh
set -e

REPO="E-n-d-l-e-s-s-A-I/vsixctl"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        *)       echo "unsupported" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64)  echo "amd64" ;;
        aarch64) echo "arm64" ;;
        *)       echo "unsupported" ;;
    esac
}

main() {
    os="$(detect_os)"
    arch="$(detect_arch)"

    if [ "$os" = "unsupported" ] || [ "$arch" = "unsupported" ]; then
        echo "Error: unsupported platform: $(uname -s) $(uname -m)" >&2
        exit 1
    fi

    echo "Detecting latest version..."
    version="$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d '"' -f 4)"

    if [ -z "$version" ]; then
        echo "Error: failed to detect latest version" >&2
        exit 1
    fi

    # GoReleaser создаёт архивы без префикса "v" в версии
    archive="vsixctl_${version#v}_${os}_${arch}.tar.gz"
    url="https://github.com/${REPO}/releases/download/${version}/${archive}"

    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' EXIT

    echo "Downloading ${archive}..."
    http_code="$(curl -sSL -o "${tmpdir}/${archive}" -w "%{http_code}" "$url")"
    if [ "$http_code" != "200" ]; then
        echo "Error: download failed (HTTP ${http_code}): ${url}" >&2
        exit 1
    fi

    echo "Extracting to ${INSTALL_DIR}..."
    mkdir -p "$INSTALL_DIR"
    tar -xzf "${tmpdir}/${archive}" -C "$tmpdir"
    mv "${tmpdir}/vsixctl" "$INSTALL_DIR/vsixctl"

    echo "vsixctl ${version} installed to ${INSTALL_DIR}/vsixctl"

    if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
        echo ""
        echo "Warning: ${INSTALL_DIR} is not in PATH"
        echo "Add this to your shell profile:"
        echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
    fi
}

main
