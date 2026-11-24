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

# npm_global PACOTE...: instala/atualiza pacotes npm globais.
# Dentro do proot o Node pode se achar "android" (process.platform). Aí o
# npm recusa pacotes que só listam linux/darwin/win32 (EBADPLATFORM) e, pior,
# pula os binários por plataforma (opencode-linux-arm64, codex-linux-arm64):
# o pacote "instala" mas fica sem o programa. O Ubuntu é linux de verdade,
# então pedimos --os=linux; se o npm ainda recusar, repete com --force.
npm_global() {
  npm install -g --os=linux "$@" && return 0
  echo "[aviso] npm recusou a plataforma; tentando de novo com --force"
  npm install -g --force --os=linux "$@"
}
