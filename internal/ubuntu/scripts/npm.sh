# Roda como o usuário comum. Variáveis: PKG (pacote npm global).
# shellcheck disable=SC2154
set -e
mkdir -p ~/.npm-global
npm config set prefix ~/.npm-global
export PATH="$HOME/.npm-global/bin:$PATH"
npm install -g "$PKG"
