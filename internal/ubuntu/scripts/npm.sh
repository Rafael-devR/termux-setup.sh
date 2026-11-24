# Roda como o usuário comum.
# Variáveis: PKG (pacote npm global) e CMD (comando que ele instala, opcional).
# shellcheck disable=SC2154
set -e
mkdir -p ~/.npm-global
npm config set prefix ~/.npm-global
export PATH="$HOME/.npm-global/bin:$PATH"
npm_global "$PKG"

# Confere se o programa funciona de verdade. Instalar o pacote não basta: o
# postinstall pode ter falhado em silêncio e deixado um aviso no lugar do
# programa (foi o que aconteceu com o opencode no celular).
if [ -n "${CMD:-}" ]; then
  if ! "$CMD" --version; then
    echo "[aviso] $CMD não respondeu; rodando o postinstall na mão"
    name="$(printf '%s' "$PKG" | sed -E 's/^(@?[^@]+)@.*/\1/')"
    dir="$(npm root -g)/$name"
    if [ -f "$dir/postinstall.mjs" ]; then (cd "$dir" && node postinstall.mjs) || true; fi
    if ! "$CMD" --version; then
      echo "[erro] $CMD não funciona depois de instalar $PKG"
      exit 1
    fi
  fi
fi
