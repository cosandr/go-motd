#!/bin/bash

set -e -o pipefail -o noclobber -o nounset

! getopt --test > /dev/null
if [[ ${PIPESTATUS[0]} -ne 4 ]]; then
    echo '`getopt --test` failed in this environment.'
    exit 1
fi

OPTIONS=h
LONGOPTS=help,pkg-name:,cfg-file:,bin-path:

! PARSED=$(getopt --options=${OPTIONS} --longoptions=${LONGOPTS} --name "$0" -- "$@")
if [[ ${PIPESTATUS[0]} -ne 0 ]]; then
    exit 2
fi

eval set -- "${PARSED}"

### DEFAULTS ###

PKG_NAME="go-motd"
BIN_PATH="/usr/bin"
CFG_FILE="/etc/${PKG_NAME}/config.yaml"

function print_help () {
# Using a here doc with standard out.
cat <<-END
Usage $0: COMMAND [OPTIONS]

Commands:
install               Build and install binary
pacman-build          Copy required files to build a pacman package from local files

Options:
-h    --help            Show this message
      --pkg-name        Change package name (default ${PKG_NAME})
      --bin-path        Path where the binary is installed (default ${BIN_PATH})
      --cfg-path        Path to config (default ${CFG_FILE})
END
}

while true; do
    case "$1" in
        -h|--help)
            print_help
            exit 0
            ;;
        --pkg-name)
            PKG_NAME="$2"
            shift 2
            ;;
        --cfg-file)
            CFG_FILE="$2"
            shift 2
            ;;
        --bin-path)
            BIN_PATH="$2"
            shift 2
            ;;
        --)
            shift
            break
            ;;
        *)
            echo "Programming error"
            exit 3
            ;;
    esac
done

if [[ $# -ne 1 ]]; then
    echo "$0: A command is required."
    exit 4
fi

PKG_PATH="$BIN_PATH/$PKG_NAME"

case "$1" in
    install)
        mkdir -p $(dirname "$CFG_FILE")
        go build -o "$PKG_NAME" -ldflags "-X main.defaultCfgPath=${CFG_FILE}"
        if [[ ! -f "$CFG_FILE" ]]; then
            ./"${PKG_NAME}" -cfg /dev/null -dump-config "$CFG_FILE"
        fi
        install -Dm 755 "$PKG_NAME" "$PKG_PATH"
        ;;
    pacman-build)
        rm -rf ./build
        mkdir -p ./build/src/"$PKG_NAME"
        rsync -a ./ ./build/src/"$PKG_NAME" --exclude build --exclude PKGBUILD
        cp -f ./PKGBUILD ./build/
        cd ./build
        makepkg --noextract
        ;;
    *)
        echo "Unrecognized command: $1"
        print_help
        exit 2
        ;;
esac
