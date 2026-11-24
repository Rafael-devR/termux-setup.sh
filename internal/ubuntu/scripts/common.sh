# Funções comuns, coladas antes de cada script deste diretório.
# fetch_check URL ARQUIVO: baixa e confere se é um script antes de executar
# (no lugar de "curl | bash": se a rede cair no meio, nada meio-baixado roda).
fetch_check() {
  curl -fsSL --retry 2 --max-time 120 -o "$2" "$1" || { echo "Download falhou: $1"; return 1; }
  [ -s "$2" ] || { echo "Arquivo vazio: $1"; return 1; }
  head -n1 "$2" | grep -q '^#!' || { echo "Não parece um script (rede bloqueada?): $1"; return 1; }
}
export DEBIAN_FRONTEND=noninteractive
# shellcheck disable=SC2034 # usado pelos outros scripts
APT_O="-y -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold"

# write_node_shim DIR: cria DIR/node-linux.js.
# Dentro do proot o Node pode se achar "android" (process.platform e
# os.platform()). Como o Ubuntu é linux de verdade, isso quebra o npm
# (EBADPLATFORM) e, pior, os postinstall que escolhem o binário pela
# plataforma (opencode-ai procura "opencode-android-arm64", que não existe).
# Este arquivo, carregado com --require, faz o Node dizer "linux".
write_node_shim() {
  mkdir -p "$1"
  cat > "$1/node-linux.js" <<'EOJS'
// termux-setup: no proot o Node pode se achar "android"; aqui é Ubuntu (linux).
if (process.platform === 'android') {
  Object.defineProperty(process, 'platform', { value: 'linux' });
  require('os').platform = () => 'linux';
}
EOJS
}

# node_linux: liga o arquivo acima nesta sessão (vale também para os
# processos filhos, como o postinstall do npm). Entra DEPOIS das opções que
# já existiam, para ter a última palavra.
node_linux() {
  write_node_shim "$HOME/.termux-setup"
  case " ${NODE_OPTIONS:-} " in
    *node-linux.js*) ;;
    *) export NODE_OPTIONS="${NODE_OPTIONS:+$NODE_OPTIONS }--require=$HOME/.termux-setup/node-linux.js" ;;
  esac
}

# npm_global PACOTE...: instala/atualiza pacotes npm globais.
# Com o Node "linux" (node_linux) o npm escolhe os binários certos e roda os
# postinstall direito. Se ainda assim recusar a plataforma, repete com
# --force. Por fim roda "npm rebuild -g" nos pacotes: conserta instalações
# antigas que ficaram sem o postinstall (o npm diria "já está instalado").
npm_global() {
  node_linux
  if ! npm install -g --os=linux "$@"; then
    echo "[aviso] npm recusou; tentando de novo com --force"
    npm install -g --force --os=linux "$@" || return 1
  fi
  for p in "$@"; do
    npm rebuild -g "$(printf '%s' "$p" | sed -E 's/^(@?[^@]+)@.*/\1/')" || true
  done
}
