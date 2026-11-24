# Roda como root. Variáveis: P (pacotes; "a|b" = o primeiro que existir).
# shellcheck disable=SC2154
# só baixa as listas se tiverem mais de 1 hora
[ -n "$(find /var/lib/apt/lists -maxdepth 1 -name '*Packages*' -mmin -60 2>/dev/null | head -n1)" ] || apt-get update
list=""; miss=""
for tok in $P; do
  ok=""
  IFS="|"; for alt in $tok; do apt-cache show "$alt" >/dev/null 2>&1 && { ok=$alt; break; }; done; unset IFS
  if [ -n "$ok" ]; then list="$list $ok"; else miss="$miss $tok"; fi
done
[ -n "$miss" ] && echo "[aviso] indisponível no Ubuntu:$miss"
[ -z "$list" ] && exit 0
echo "Instalando no Ubuntu:$list"
# shellcheck disable=SC2086
apt-get install -y --no-install-recommends $list
