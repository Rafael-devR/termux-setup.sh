# Roda como o usuário comum. Variáveis: URL (instalador oficial da Anthropic).
# shellcheck disable=SC2154
set -e
f="$(mktemp)"
fetch_check "$URL" "$f"
bash "$f"; rm -f "$f"
export PATH="$HOME/.local/bin:$PATH"
claude --version
