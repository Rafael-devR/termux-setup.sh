#!/data/data/com.termux/files/usr/bin/bash
# =====================================================================
#  Instalador do termux-setup
#  Baixa o binário da última versão (GitHub Releases), confere o SHA256 e
#  instala em $PREFIX/bin. Sem versão publicada (ou com --fonte), compila
#  do código-fonte com o Go do Termux.
#
#  curl -fsSL https://raw.githubusercontent.com/Rafael-devR/termux-setup.sh/main/install.sh | bash
#  bash install.sh [--fonte] [--versao vX.Y.Z] [--ref RAMO]
# =====================================================================
set -eu

REPO="Rafael-devR/termux-setup.sh"
PREFIX="${PREFIX:-/data/data/com.termux/files/usr}"
BIN="$PREFIX/bin/termux-setup"
FROM_SOURCE=0
VERSION=""
REF="main"

while [ $# -gt 0 ]; do
  case "$1" in
    --fonte) FROM_SOURCE=1 ;;
    --versao) VERSION="${2:?falta a versão}"; shift ;;
    --ref) REF="${2:?falta o ramo}"; shift ;;
    -h|--ajuda) sed -n '3,10p' "$0"; exit 0 ;;
    *) echo "opção desconhecida: $1"; exit 1 ;;
  esac
  shift
done

say()  { printf '\033[1;36m›\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31m✗\033[0m %s\n' "$*" >&2; exit 1; }

[ -d "$PREFIX/bin" ] || die "isto é para o Termux (não achei $PREFIX/bin)"
command -v curl >/dev/null 2>&1 || { say "instalando curl"; pkg install -y curl; }

case "$(uname -m)" in
  aarch64|arm64)        ASSET="android_arm64" ;;
  armv7l|armv8l|arm*)   ASSET="linux_arm" ;;
  x86_64|amd64)         ASSET="linux_amd64" ;;
  i686|i386|x86)        ASSET="linux_386" ;;
  *) die "arquitetura não suportada: $(uname -m)" ;;
esac

TMP="$(mktemp -d "$PREFIX/tmp/tsetup-inst.XXXXXX")"
trap 'rm -rf "$TMP"' EXIT

from_release() {
  if [ -z "$VERSION" ]; then
    VERSION="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null \
      | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)" || true
  fi
  [ -n "$VERSION" ] || return 1
  base="https://github.com/$REPO/releases/download/$VERSION"
  file="termux-setup_${ASSET}.tar.gz"
  say "baixando termux-setup $VERSION ($ASSET)"
  curl -fsSL -o "$TMP/$file" "$base/$file" || return 1
  curl -fsSL -o "$TMP/checksums.txt" "$base/checksums.txt" || return 1
  (cd "$TMP" && grep " $file\$" checksums.txt | sha256sum -c --quiet -) || die "SHA256 não confere: download corrompido"
  tar -xzf "$TMP/$file" -C "$TMP"
  install -m 755 "$TMP/termux-setup" "$BIN"
}

from_source() {
  say "compilando do código-fonte ($REF)"
  pkg install -y golang git
  git clone --depth 1 --branch "$REF" "https://github.com/$REPO" "$TMP/src"
  ver="$(git -C "$TMP/src" describe --tags --always 2>/dev/null || echo dev)"
  (cd "$TMP/src" && CGO_ENABLED=0 GOTOOLCHAIN=local go build -trimpath \
    -ldflags "-s -w -X github.com/$REPO/internal/app.Version=$ver" -o "$BIN" ./cmd/termux-setup)
}

if [ "$FROM_SOURCE" = 1 ] || ! from_release; then
  [ "$FROM_SOURCE" = 1 ] || say "nenhuma versão publicada; vou compilar"
  from_source
fi

say "pronto: $("$BIN" --version)"
printf '\n  \033[1mtermux-setup\033[0m          abre o menu\n'
printf '  \033[1mtermux-setup --teste\033[0m  simulação (não instala nada)\n'
printf '  \033[1mtermux-setup --help\033[0m   todos os comandos\n\n'
