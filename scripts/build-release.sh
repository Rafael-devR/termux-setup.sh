#!/usr/bin/env bash
# Gera os binários de release em dist/ (um .tar.gz por alvo + checksums.txt).
# Uso: scripts/build-release.sh v6.0.0
#
# android/arm64 é nativo do Android (quase todos os celulares). Os outros
# alvos precisam de CGO para GOOS=android, então saem como linux estático,
# que também roda no Termux (o fuso horário é corrigido em sys.FixTimezone).
set -euo pipefail
VERSION="${1:?uso: $0 vX.Y.Z}"
MOD="github.com/Rafael-devR/termux-setup.sh"
cd "$(dirname "$0")/.."
mkdir -p dist
BUILD="$(mktemp -d)"
trap 'rm -rf "$BUILD"' EXIT
for target in android/arm64 linux/arm linux/amd64 linux/386; do
  os="${target%/*}"; arch="${target#*/}"
  out="$BUILD/${os}_${arch}"
  mkdir -p "$out"
  echo "› $os/$arch"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" GOARM=7 go build -trimpath \
    -ldflags "-s -w -X $MOD/internal/app.Version=$VERSION" -o "$out/termux-setup" ./cmd/termux-setup
  cp README.md "$out/"
  tar -czf "dist/termux-setup_${os}_${arch}.tar.gz" -C "$out" termux-setup README.md
done
(cd dist && sha256sum ./*.tar.gz | sed 's| \./| |' > checksums.txt)
cat dist/checksums.txt
