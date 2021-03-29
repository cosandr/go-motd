# Maintainer: Andrei Costescu <andrei@costescu.no>

# shellcheck shell=bash

pkgname=go-motd-git
_pkgname="${pkgname%-git}"
_config_path="etc/${_pkgname}/config.yaml"
pkgver=f07595b
pkgrel=1
pkgdesc="Dynamic MOTD written in Go"
arch=("x86_64")
url="https://github.com/cosandr/go-motd"
license=("MIT")
provides=("${_pkgname}")
conflicts=("${_pkgname}")
optdepends=(
    'zfs-utils: ZFS pool status'
    'docker: Docker container status'
    'hddtemp: Disk temperatures'
    'go-check-updates: Pending updates'
    'lm_sensors: CPU temperatures'
)
makedepends=("git" "go")
source=("git+$url")
md5sums=("SKIP")
backup=("$_config_path")

pkgver() {
    cd "${_pkgname}"
  ( set -o pipefail
    git describe --long 2>/dev/null | sed 's/\([^-]*-g\)/r\1/;s/-/./g' ||
    printf "r%s.%s" "$(git rev-list --count HEAD)" "$(git rev-parse --short HEAD)"
  )
}

build() {
    cd "${_pkgname}"
    go mod vendor
    go build -a -ldflags "-X main.defaultCfgPath=/${_config_path}" -o "${_pkgname}"
    # Generate default config
    ./"${_pkgname}" --config /dev/null --dump-config "default-config.yaml" > /dev/null
}

package() {
    cd "${_pkgname}"
    install -dm 755 "${pkgdir}/etc/${_pkgname}"
    install -Dm 644 "default-config.yaml" "${pkgdir}/${_config_path}"
    install -Dm 755 "${_pkgname}" "${pkgdir}/usr/bin/${_pkgname}"
    install -Dm 644 LICENSE "${pkgdir}/usr/share/licenses/${_pkgname}/LICENSE"
}
