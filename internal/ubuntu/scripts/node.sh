# Roda como root. Variáveis: URL (instalador do NodeSource).
# shellcheck disable=SC2154
set -e
if command -v node >/dev/null 2>&1 && [ "$(node -v | cut -d. -f1 | tr -d v)" -ge 20 ]; then
  echo "node ok: $(node -v)"; exit 0
fi
f="$(mktemp)"
fetch_check "$URL" "$f"
bash "$f"; rm -f "$f"
apt-get install -y nodejs
node -v; npm -v
