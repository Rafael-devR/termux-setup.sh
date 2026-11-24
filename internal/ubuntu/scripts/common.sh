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
