# Maintainer: Andrei Costescu <andrei@costescu.no>

# shellcheck shell=bash

pkgname=go-motd-git
_pkgname="${pkgname%-git}"
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

_config_path="/etc/${_pkgname}/config.yaml"

pkgver() {
    cd "${srcdir}/${_pkgname}"
    git describe --always
}

build() {
    cd "${_pkgname}"
    go get -d
    # Static linked binary
    # export CGO_ENABLED=0
    # export GOOS=linux
    # export GOARCH=amd64
    go build -a -ldflags "-X main.defaultCfgPath=${_config_path}" -o "${_pkgname}" 
}

package() {
    cd "${_pkgname}"
    # install -dm 755 "${pkgdir}/etc/${_pkgname}"
    install -Dm 644 "config.yaml" "${pkgdir}/${_config_path}"
    install -Dm 755 "${_pkgname}" "${pkgdir}/usr/bin/${_pkgname}"
    install -Dm 644 LICENSE "${pkgdir}/usr/share/licenses/${_pkgname}/LICENSE"
}
