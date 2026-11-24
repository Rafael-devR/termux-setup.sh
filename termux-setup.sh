#!/data/data/com.termux/files/usr/bin/bash
# =====================================================================
#  TERMUX SETUP v4.0
#  Painel fixo, menu por setas, estimativas, log final, Ubuntu (proot)
#  + Claude Code, Codex CLI e Gemini CLI (instaladores oficiais)
#  + Aparência: barra de teclas, tema VS Code, fonte JetBrains Mono
#  + GitHub (login por token) e Root (tsu) opcionais.
#  Uso:  bash termux-setup.sh            (modo normal)
#        bash termux-setup.sh --teste    (simulação, não instala nada)
#  NÃO use "source". Nunca fecha o Termux sozinho.
# =====================================================================

if [ -n "${BASH_SOURCE[0]:-}" ] && [ "${BASH_SOURCE[0]}" != "$0" ]; then
  echo "Execute com:  bash ${BASH_SOURCE[0]##*/}   (não use 'source' nem '.')"
  return 1 2>/dev/null
fi
if [ -z "${BASH_VERSINFO:-}" ] || [ "${BASH_VERSINFO[0]}" -lt 4 ]; then
  echo "Este script precisa de bash 4 ou superior."; exit 1
fi

VERSION="4.0"
SELF="$(readlink -f "$0" 2>/dev/null || echo "$0")"
DRY=0
for a in "$@"; do [ "$a" = "--teste" ] && DRY=1; done

if [ "$DRY" = 1 ]; then
  PREFIX="$(mktemp -d)"; HOME="$(mktemp -d)"
  mkdir -p "$PREFIX/etc/apt/sources.list.d" "$PREFIX/tmp"
  echo "deb https://velho.exemplo/termux-main stable main" > "$PREFIX/etc/apt/sources.list"
fi
PREFIX="${PREFIX:-/data/data/com.termux/files/usr}"
export DEBIAN_FRONTEND=noninteractive

# ---------------------------------------------------------------- cores
C_N=$'\033[0m'; C_B=$'\033[1m'; C_D=$'\033[2m'
C_R=$'\033[1;31m'; C_G=$'\033[1;32m'; C_Y=$'\033[1;33m'
C_CY=$'\033[1;36m'; C_M=$'\033[1;35m'
SPIN=('⠋' '⠙' '⠹' '⠸' '⠼' '⠴' '⠦' '⠧' '⠇' '⠏')
ROWS=24; COLS=40; RULE=""

rep() { local s="" i; for ((i=0; i<$2; i++)); do s+="$1"; done; printf '%s' "$s"; }
term_size() {
  COLS=$(tput cols 2>/dev/null || echo 40); ROWS=$(tput lines 2>/dev/null || echo 24)
  [[ $COLS =~ ^[0-9]+$ ]] || COLS=40
  [[ $ROWS =~ ^[0-9]+$ ]] || ROWS=24
  RULE="$(rep '─' $(( COLS > 61 ? 60 : COLS - 1 )))"
}
term_size
L() { printf '%s\033[K\n' "$*"; }
hide_cur() { printf '\033[?25l'; }
show_cur() { printf '\033[?25h'; }

# ---------------------------------------------------------------- estado
LOGDIR="$HOME/termux-setup-logs"; mkdir -p "$LOGDIR"
WORK="$(mktemp -d "$PREFIX/tmp/tsetup.XXXXXX" 2>/dev/null || mktemp -d)"
TS=""; LOG=""; SUMMARY=""
INSTALL_STARTED=0; FINISHED=0; INTERRUPTED=0
DONE=0; TOTAL=1; TOTAL_W=1; START=0; START_DATE=""; STEP_T0=0
CUR_LABEL=""; RUN_PID=""; LIVE=""; DASH_TITLE=""
FREE_BEFORE=0; STATUS_TXT=""; N_OK=0; N_FAIL=0; N_PEND=0
PLAN_LABEL=(); PLAN_FN=(); PLAN_ARG=(); PLAN_W=()
RES_STAT=(); RES_TIME=()
GIT_NAME=""; GIT_MAIL=""; DO_STORAGE=0; KEYS_PRESET=1
MENU_ITEMS=(); MENU_KEYS=(); MENU_PRE=(); MENU_IDX=-1; MENU_START=0; KEY=""
EST_DL=0; EST_DISK=0
declare -A SEL

APT_OPTS=(-y
  -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold
  -o Acquire::Retries=3 -o Acquire::http::Timeout=25 -o Acquire::https::Timeout=25)

# Espelhos da wiki oficial do Termux (o script testa todos e usa o melhor)
MIRRORS=(
  "https://packages-cf.termux.dev/apt/termux-main"
  "https://packages.termux.dev/apt/termux-main"
  "https://mirror.accum.se/mirror/termux.dev/termux-main"
  "https://mirrors.sahilister.in/termux/termux-main"
  "https://mirror.mwt.me/termux/main"
  "https://mirror.vern.cc/termux/termux-main"
  "https://mirror.fcix.net/termux/termux-main"
  "https://mirror.csclub.uwaterloo.ca/termux/termux-main"
  "https://mirror.freedif.org/termux/termux-main"
  "https://mirror.leitecastro.com/termux/termux-main"
  "https://termux.librehat.com/apt/termux-main"
  "https://mirrors.utermux.dev/termux/termux-main"
)

# ---------------------------------------------------------------- módulos
MOD_ORDER=(base build python node java go rust web db net extras term dev2 api ubuntu claude codex gemini hush theme font keys root)
declare -A MOD_NAME=(
  [base]="Base: git ssh vim zip"
  [build]="Compiladores: clang make cmake"
  [python]="Python + pip + venv"
  [node]="Node.js LTS + yarn + tsc"
  [java]="Java 21 + gradle + maven"
  [go]="Go"
  [rust]="Rust"
  [web]="PHP, Ruby, Perl"
  [db]="SQLite + Redis"
  [net]="Rede: nmap rsync netcat"
  [extras]="Extras: tmux jq rg htop"
  [term]="Terminal bonito: starship eza"
  [dev2]="Dev+: gh lazygit neovim"
  [api]="Termux:API (notificações)"
  [ubuntu]="Ubuntu 26.04 LTS (proot)"
  [claude]="Claude Code (oficial)"
  [codex]="Codex CLI (OpenAI)"
  [gemini]="Gemini CLI (Google)"
  [hush]="Silenciar mensagem de boas-vindas"
  [theme]="Tema escuro do VS Code"
  [font]="Fonte JetBrains Mono"
  [keys]="Barra de teclas personalizada"
  [root]="Root: tsu (só se tiver root)"
)
# "a|b" = usa o primeiro que existir no espelho
declare -A MOD_PKGS=(
  [base]="git curl wget nano vim openssh zip unzip tar which man termux-tools termux-exec findutils grep sed file tree less"
  [build]="clang make cmake pkg-config binutils autoconf automake libtool ninja patch libffi openssl libxml2 libxslt libjpeg-turbo libpng zlib"
  [python]="python python-pip"
  [node]="nodejs-lts|nodejs"
  [java]="openjdk-21|openjdk-17 gradle maven"
  [go]="golang"
  [rust]="rust"
  [web]="php ruby perl"
  [db]="sqlite redis"
  [net]="nmap sshpass netcat-openbsd dnsutils traceroute net-tools rsync"
  [extras]="htop fastfetch|neofetch tmux jq ripgrep fd bat fzf ncdu zsh"
  [term]="starship zoxide eza"
  [dev2]="gh lazygit neovim"
  [api]="termux-api"
)
# peso (segundos estimados) usado na barra de progresso; download/disco em MB (aprox.)
declare -A MOD_W=(  [base]=12 [build]=45 [python]=20 [node]=25 [java]=120 [go]=30 [rust]=70 [web]=50 [db]=15 [net]=40 [extras]=40 [term]=15 [dev2]=40 [api]=6 [ubuntu]=200 [claude]=50 [codex]=30 [gemini]=35 [hush]=1 [theme]=1 [font]=8 [keys]=1 [root]=12 )
declare -A MOD_DL=( [base]=6  [build]=70 [python]=35 [node]=12 [java]=280 [go]=90 [rust]=150 [web]=80 [db]=2  [net]=9  [extras]=11 [term]=15 [dev2]=60 [api]=1 [ubuntu]=160 [claude]=270 [codex]=40 [gemini]=80 [hush]=0 [theme]=0 [font]=2 [keys]=0 [root]=2 )
declare -A MOD_DISK=([base]=25 [build]=400 [python]=150 [node]=60 [java]=460 [go]=400 [rust]=900 [web]=300 [db]=6  [net]=44 [extras]=42 [term]=40 [dev2]=200 [api]=1 [ubuntu]=500 [claude]=300 [codex]=120 [gemini]=250 [hush]=0 [theme]=0 [font]=2 [keys]=0 [root]=5 )

# ================================================================= UI
bar() { # bar PCT LARGURA
  local pct=$1 w=${2:-10} f i s=""
  f=$(( pct * w / 100 ))
  for ((i=0; i<w; i++)); do (( i < f )) && s+="█" || s+="░"; done
  printf '%s' "$s"
}
fmt_t() { printf '%02d:%02d' $(( $1 / 60 )) $(( $1 % 60 )); }
free_kb() {
  local v; v="$(df -Pk "$HOME" 2>/dev/null | tail -n1 | tr -s ' ' | cut -d' ' -f4)"
  [[ $v =~ ^[0-9]+$ ]] || v=0; echo "$v"
}
banner() { # $1 = subtítulo
  L " ${C_CY}${C_B}TERMUX SETUP${C_N} ${C_D}v$VERSION${C_N}${1:+  ${C_M}$1${C_N}}"
  L "${C_D}${RULE}${C_N}"
}
splash() {
  printf '\033[2J\033[H'; hide_cur
  echo
  L " ${C_CY} ▀█▀ █▀▀ █▀█ █▀▄▀█ █ █ ▀▄▀${C_N}"; sleep 0.12
  L " ${C_M}  █  ██▄ █▀▄ █ ▀ █ █▄█ █ █${C_N}"; sleep 0.12
  echo
  L " ${C_D}setup v$VERSION · configure o Termux do zero${C_N}"; sleep 0.45
  show_cur
}
current_mirror() {
  grep -hE '^(deb |URIs:)' "$PREFIX/etc/apt/sources.list" \
    "$PREFIX"/etc/apt/sources.list.d/*.sources 2>/dev/null | head -n1 \
    | sed -E 's/^(deb|URIs:) +//; s/ .*//; s#^[a-z]+://##; s#/.*##'
}
pause() { show_cur; echo; read -r -p " Enter para voltar... " _ || true; }

read_key() { # define KEY: up down left right enter space esc eof ou o caractere
  local k rest
  IFS= read -rsn1 k || { KEY=eof; return; }
  case "$k" in
    $'\033')
      IFS= read -rsn2 -t 0.05 rest
      case "$rest" in
        '[A') KEY=up ;; '[B') KEY=down ;; '[C') KEY=right ;; '[D') KEY=left ;;
        *) KEY=esc ;;
      esac ;;
    '') KEY=enter ;;
    ' ') KEY=space ;;
    *) KEY="$k" ;;
  esac
}

menu_select() { # $1 título; usa MENU_ITEMS MENU_KEYS MENU_PRE MENU_START -> MENU_IDX (-1 = voltar)
  local title="$1" cur="${MENU_START:-0}" n=${#MENU_ITEMS[@]} h first i k
  local pre=${#MENU_PRE[@]}
  MENU_IDX=-1
  term_size; printf '\033[2J'; hide_cur
  while true; do
    h=$(( ROWS - 6 - pre )); [ $h -lt 3 ] && h=3; [ $h -gt $n ] && h=$n
    first=$(( cur - h / 2 )); [ $first -lt 0 ] && first=0
    [ $(( first + h )) -gt $n ] && first=$(( n - h ))
    printf '\033[H'
    banner "$title"
    for i in "${!MENU_PRE[@]}"; do L " ${MENU_PRE[i]}"; done
    L ""
    for ((i=first; i<first+h; i++)); do
      k="${MENU_KEYS[i]:-}"
      if [ "$i" -eq "$cur" ]; then L " ${C_CY}▸${C_N} ${C_B}${k:+$k) }${MENU_ITEMS[i]}${C_N}"
      else L "   ${k:+$k) }${MENU_ITEMS[i]}"; fi
    done
    L ""
    L " ${C_D}↑↓ mover · Enter ok · número = atalho · q sair${C_N}"
    printf '\033[J'
    read_key
    case "$KEY" in
      up|k)   cur=$(( (cur - 1 + n) % n )) ;;
      down|j) cur=$(( (cur + 1) % n )) ;;
      enter|right) MENU_IDX=$cur; break ;;
      q|esc|eof) MENU_IDX=-1; break ;;
      *) for ((i=0; i<n; i++)); do
           if [ -n "${MENU_KEYS[i]:-}" ] && [ "${MENU_KEYS[i]}" = "$KEY" ]; then MENU_IDX=$i; break 2; fi
         done ;;
    esac
  done
  MENU_PRE=(); MENU_START=0; show_cur
}

menu_check() { # ids...  -> 0 continuar / 1 voltar
  local ids=("$@") cur=0 n=$# h first i m mark ptr
  term_size; printf '\033[2J'; hide_cur
  while true; do
    h=$(( ROWS - 6 )); [ $h -lt 3 ] && h=3; [ $h -gt $n ] && h=$n
    first=$(( cur - h / 2 )); [ $first -lt 0 ] && first=0
    [ $(( first + h )) -gt $n ] && first=$(( n - h ))
    printf '\033[H'
    banner "${CHECK_TITLE:-Personalizada}"
    L " ${C_D}↑↓ mover · espaço marca · a todos · n nenhum${C_N}"
    for ((i=first; i<first+h; i++)); do
      m="${ids[i]}"
      if [ "${SEL[$m]:-0}" = 1 ]; then mark="${C_G}[x]${C_N}"; else mark="[ ]"; fi
      if [ "$i" -eq "$cur" ]; then ptr="${C_CY}▸${C_N}"; else ptr=" "; fi
      L " $ptr $mark ${MOD_NAME[$m]} ${C_D}~${MOD_DL[$m]}MB${C_N}"
    done
    L " ${C_D}Enter continua · q volta${C_N}"
    printf '\033[J'
    read_key
    case "$KEY" in
      up|k)   cur=$(( (cur - 1 + n) % n )) ;;
      down|j) cur=$(( (cur + 1) % n )) ;;
      space)  m="${ids[cur]}"; if [ "${SEL[$m]:-0}" = 1 ]; then SEL[$m]=0; else SEL[$m]=1; fi ;;
      a|A)    for m in "${ids[@]}"; do SEL[$m]=1; done ;;
      n|N)    for m in "${ids[@]}"; do SEL[$m]=0; done ;;
      enter)  show_cur; return 0 ;;
      q|esc|eof) show_cur; return 1 ;;
    esac
  done
}

new_run() {
  TS="$(date +%Y%m%d-%H%M%S)"
  LOG="$LOGDIR/setup-$TS.log"
  SUMMARY="$LOGDIR/resumo-$TS.txt"
  : > "$LOG"; : > "$WORK/unavailable"; : > "$WORK/failed_pkgs"; : > "$WORK/tried"
  rm -f "$WORK/mirror_used"
}

# ================================================================= PASSOS
apt_i() { apt-get install "${APT_OPTS[@]}" "$@"; }

pick_pkg() { # "a|b" -> primeiro disponível
  local alt IFS='|'
  for alt in $1; do
    apt-cache show "$alt" >/dev/null 2>&1 && { echo "$alt"; return 0; }
  done
  return 1
}

install_mod() {
  local id="$1" tok p list=() rc=0
  for tok in ${MOD_PKGS[$id]}; do
    if p="$(pick_pkg "$tok")"; then list+=("$p")
    else echo "$tok" >> "$WORK/unavailable"; echo "[aviso] indisponível: $tok"; fi
  done
  [ ${#list[@]} -eq 0 ] && return 0
  echo "Instalando: ${list[*]}"
  apt_i "${list[@]}" && return 0
  echo "[aviso] instalação em bloco falhou; tentando um a um"
  for p in "${list[@]}"; do
    apt_i "$p" || { echo "$p" >> "$WORK/failed_pkgs"; rc=1; }
  done
  return $rc
}

probe_mirror() { # url arquivo_saida -> "idade_horas tempo url"
  local url="$1" out="$2" f t d epoch age
  f="$WORK/rel.$RANDOM"
  t="$(curl -fsSL --max-time 8 -o "$f" -w '%{time_total}' "$url/dists/stable/Release" 2>/dev/null)" \
    || { rm -f "$f"; return 1; }
  d="$(sed -n 's/^Date: *//p' "$f" | head -n1)"
  epoch="$(date -d "$d" +%s 2>/dev/null || echo 0)"
  if [ "$epoch" = 0 ]; then age=9999; else age=$(( ($(date +%s) - epoch) / 3600 )); fi
  echo "$age $t $url" > "$out"; rm -f "$f"
}

rank_mirrors() {
  local age t url fresh="" stale=""
  : > "$WORK/candidates"; : > "$WORK/ranking"
  while read -r age t url; do
    [ -n "$url" ] || continue
    echo "$age $t $url" >> "$WORK/ranking"
    if [ "$age" -le 48 ] 2>/dev/null; then fresh+="$t $url"$'\n'
    else stale+="$age $url"$'\n'; fi
  done < <(cat "$WORK"/probe.* 2>/dev/null)
  { printf '%s' "$fresh" | sort -n | cut -d' ' -f2
    printf '%s' "$stale" | sort -n | cut -d' ' -f2; } | grep . > "$WORK/candidates"
}

step_mirrors() {
  local i=0 m
  rm -f "$WORK"/probe.*
  for m in "${MIRRORS[@]}"; do
    probe_mirror "$m" "$WORK/probe.$i" &
    i=$((i+1))
  done
  wait
  rank_mirrors
  echo "--- ranking (idade_h tempo_s url) ---"; cat "$WORK/ranking"
  [ -s "$WORK/candidates" ] || { echo "Nenhum espelho respondeu (sem internet?)"; return 1; }
}

apply_mirror() {
  local url="${1%/}" f changed=0 bk="$LOGDIR/backup-apt-$TS"
  [ -n "$url" ] || return 1
  mkdir -p "$bk"
  cp -a "$PREFIX/etc/apt/sources.list" "$bk/" 2>/dev/null
  cp -a "$PREFIX/etc/apt/sources.list.d" "$bk/" 2>/dev/null
  # formato novo (deb822): troca só a linha URIs do repositório "stable"
  for f in "$PREFIX"/etc/apt/sources.list.d/*.sources; do
    [ -f "$f" ] || continue
    if grep -qE '^Suites: *stable' "$f" && grep -q '^URIs:' "$f"; then
      sed -i "s|^URIs:.*|URIs: $url|" "$f"; changed=1
    fi
  done
  if [ "$changed" = 1 ]; then
    [ -f "$PREFIX/etc/apt/sources.list" ] && sed -i 's/^deb /#deb /' "$PREFIX/etc/apt/sources.list"
  else
    printf 'deb %s stable main\n' "$url" > "$PREFIX/etc/apt/sources.list"
  fi
  echo "$url" > "$WORK/mirror_used"; echo "$url" >> "$WORK/tried"
  echo "Espelho aplicado: $url (backup em $bk)"
}

step_apply_mirror() {
  local url
  url="$(sed -n '1p' "$WORK/candidates" 2>/dev/null)"
  if [ -z "$url" ]; then echo "Sem candidato; mantendo espelho atual."; return 1; fi
  apply_mirror "$url"
}

step_update() {
  local url n=0
  while true; do
    apt-get update "${APT_OPTS[@]}" && return 0
    n=$((n+1)); [ $n -gt 4 ] && return 1
    url="$(grep -vxFf "$WORK/tried" "$WORK/candidates" 2>/dev/null | head -n1)"
    [ -z "$url" ] && return 1
    echo "[aviso] apt update falhou; tentando outro espelho: $url"
    apply_mirror "$url"
  done
}

step_upgrade() {
  apt-get full-upgrade "${APT_OPTS[@]}" && return 0
  echo "[aviso] upgrade falhou; reparando e tentando de novo"
  dpkg --configure -a
  apt-get --fix-broken install "${APT_OPTS[@]}"
  apt-get update "${APT_OPTS[@]}"
  apt-get full-upgrade "${APT_OPTS[@]}"
}

step_repair() {
  dpkg --configure -a
  apt-get --fix-broken install "${APT_OPTS[@]}"
  apt-get update "${APT_OPTS[@]}"
  apt-get full-upgrade "${APT_OPTS[@]}"
}

step_pyvenv() {
  command -v python >/dev/null 2>&1 || { echo "python não encontrado"; return 1; }
  [ -d "$HOME/venv" ] || python -m venv "$HOME/venv" || return 1
  "$HOME/venv/bin/python" -m pip install --upgrade pip || true
  "$HOME/venv/bin/pip" install requests rich
}

step_npm() {
  command -v npm >/dev/null 2>&1 || { echo "npm não encontrado"; return 1; }
  npm install -g yarn typescript
}

step_git() {
  command -v git >/dev/null 2>&1 || { echo "git não encontrado"; return 1; }
  [ -n "$GIT_NAME" ] && git config --global user.name "$GIT_NAME"
  [ -n "$GIT_MAIL" ] && git config --global user.email "$GIT_MAIL"
  git config --global init.defaultBranch main
  return 0
}

step_storage() { termux-setup-storage; sleep 2; return 0; }
step_cleanup() { apt-get autoremove -y; apt-get clean; return 0; }

# ---------------------------------------------------------------- Ubuntu + IAs
UB=ubuntu
ub_has()  { proot-distro login "$UB" -- true >/dev/null 2>&1; }
ub_root() { proot-distro login "$UB" -- bash -lc "$1"; }
ub_dev()  { proot-distro login "$UB" --user dev -- bash -lc "$1"; }

step_proot_pkgs() { apt_i proot proot-distro; }

ub_ver() { # versão do Ubuntu instalado (ex.: "Ubuntu 26.04.1 LTS")
  proot-distro login "$UB" -- grep PRETTY_NAME /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"'
}

step_ubuntu_install() {
  command -v proot-distro >/dev/null 2>&1 || { echo "proot-distro ausente"; return 1; }
  if ub_has; then echo "Ubuntu já instalado: $(ub_ver). Mantido (não é substituído)."; return 0; fi
  # tenta a LTS mais nova (26.04); se o proot-distro não tiver, cai para 24.04
  proot-distro install ubuntu:26.04 || proot-distro install ubuntu:24.04 || proot-distro install ubuntu
  ub_has && echo "Instalado: $(ub_ver)"
}

step_ubuntu_base() {
  ub_has || { echo "Ubuntu não encontrado"; return 1; }
  ub_root '
set -e
export DEBIAN_FRONTEND=noninteractive
echo "APT::Sandbox::User \"root\";" > /etc/apt/apt.conf.d/99proot
apt-get update
apt-get -y upgrade
apt-get install -y --no-install-recommends curl git ca-certificates gnupg bash ripgrep unzip xz-utils less nano locales
id dev >/dev/null 2>&1 || useradd -m -s /bin/bash dev
mkdir -p /home/dev/projetos
chown dev:dev /home/dev/projetos
touch /home/dev/.profile
grep -q "termux-setup" /home/dev/.profile || cat >> /home/dev/.profile <<EOP
# termux-setup
export PATH="\$HOME/.local/bin:\$HOME/.npm-global/bin:\$PATH"
EOP
chown dev:dev /home/dev/.profile
'
}

step_ubuntu_node() {
  ub_has || { echo "Ubuntu não encontrado"; return 1; }
  ub_root '
set -e
export DEBIAN_FRONTEND=noninteractive
if command -v node >/dev/null 2>&1 && [ "$(node -v | cut -d. -f1 | tr -d v)" -ge 20 ]; then
  echo "node ok: $(node -v)"; exit 0
fi
curl -fsSL https://deb.nodesource.com/setup_lts.x | bash -
apt-get install -y nodejs
node -v; npm -v
'
}

step_claude() {
  ub_has || { echo "Ubuntu não encontrado"; return 1; }
  # instalador oficial da Anthropic (Linux ARM64, dentro do Ubuntu)
  ub_dev 'set -e; curl -fsSL https://claude.ai/install.sh | bash; export PATH="$HOME/.local/bin:$PATH"; claude --version'
}

step_ub_npm() { # $1 = pacote npm
  ub_has || { echo "Ubuntu não encontrado"; return 1; }
  ub_dev "set -e; mkdir -p ~/.npm-global; npm config set prefix ~/.npm-global; export PATH=\"\$HOME/.npm-global/bin:\$PATH\"; npm install -g $1"
}

step_ub_upgrade() {
  ub_has || { echo "Ubuntu não encontrado"; return 1; }
  ub_root 'export DEBIAN_FRONTEND=noninteractive; apt-get update && apt-get -y upgrade'
}

step_ia_update() {
  ub_has || { echo "Ubuntu não encontrado"; return 1; }
  ub_dev 'export PATH="$HOME/.local/bin:$HOME/.npm-global/bin:$PATH"
command -v claude >/dev/null 2>&1 && claude update
command -v codex  >/dev/null 2>&1 && npm install -g @openai/codex@latest
command -v gemini >/dev/null 2>&1 && npm install -g @google/gemini-cli@latest
true'
}

step_wrappers() {
  mkdir -p "$HOME/bin" "$HOME/projetos"
  cat > "$HOME/bin/ia-run" <<'EOS'
#!/data/data/com.termux/files/usr/bin/bash
# uso: ia-run <comando> [args] — roda no Ubuntu (usuário dev) na pasta atual do Termux.
# A pasta atual aparece dentro do Ubuntu como /work/<nome-da-pasta>.
HOST_DIR="$PWD"
NAME="$(basename "$HOST_DIR")"
case "$NAME" in ""|"/") NAME="raiz" ;; esac
NAME="${NAME// /_}"
DEST="/work/$NAME"
command -v termux-wake-lock >/dev/null 2>&1 && termux-wake-lock 2>/dev/null
proot-distro login ubuntu --user dev --bind "$HOST_DIR:$DEST" --work-dir "$DEST" \
  -- bash -lc 'exec "$@"' _ "$@"
rc=$?
command -v termux-wake-unlock >/dev/null 2>&1 && termux-wake-unlock 2>/dev/null
exit $rc
EOS
  for c in claude codex gemini; do
    printf '#!/data/data/com.termux/files/usr/bin/bash\nexec "$HOME/bin/ia-run" %s "$@"\n' "$c" > "$HOME/bin/$c"
  done
  printf '#!/data/data/com.termux/files/usr/bin/bash\nexec "$HOME/bin/ia-run" bash -l\n' > "$HOME/bin/ubuntu"
  printf '#!/data/data/com.termux/files/usr/bin/bash\nexec proot-distro login ubuntu\n' > "$HOME/bin/ubuntu-root"
  cat > "$HOME/bin/ia-update" <<'EOS'
#!/data/data/com.termux/files/usr/bin/bash
proot-distro login ubuntu -- bash -lc 'export DEBIAN_FRONTEND=noninteractive; apt-get update && apt-get -y upgrade'
"$HOME/bin/ia-run" bash -lc 'command -v claude >/dev/null && claude update; command -v codex >/dev/null && npm install -g @openai/codex@latest; command -v gemini >/dev/null && npm install -g @google/gemini-cli@latest; true'
EOS
  chmod +x "$HOME"/bin/* 2>/dev/null
  echo "Comandos: claude codex gemini ubuntu ubuntu-root ia-run ia-update"
}

step_commands() {
  mkdir -p "$HOME/bin" "$HOME/projetos"
  cat > "$HOME/bin/atualizar" <<'EOS'
#!/data/data/com.termux/files/usr/bin/bash
export DEBIAN_FRONTEND=noninteractive
O=(-y -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold)
apt-get update && apt-get full-upgrade "${O[@]}" && apt-get autoremove -y
echo "✔ Tudo atualizado."
EOS
  cat > "$HOME/bin/limpar" <<'EOS'
#!/data/data/com.termux/files/usr/bin/bash
apt-get autoremove -y && apt-get clean
echo "✔ Cache limpo."
EOS
  cat > "$HOME/bin/info" <<'EOS'
#!/data/data/com.termux/files/usr/bin/bash
echo "== Termux =="
echo "Arq: $(uname -m) | Android $(getprop ro.build.version.release) | $(getprop ro.product.model)"
echo "Espelho: $(grep -hE '^(deb |URIs:)' "$PREFIX"/etc/apt/sources.list "$PREFIX"/etc/apt/sources.list.d/*.sources 2>/dev/null | head -n1)"
df -h "$HOME" | tail -n1
for c in python node java go rustc clang git; do
  command -v "$c" >/dev/null 2>&1 || continue
  if [ "$c" = go ]; then v="$(go version 2>&1 | head -n1)"; else v="$($c --version 2>&1 | head -n1)"; fi
  printf '  %-7s %s\n' "$c" "$v"
done
EOS
  cat > "$HOME/bin/novo" <<'EOS'
#!/data/data/com.termux/files/usr/bin/bash
# uso: novo <nome> [python|node|c]   (cria em ~/projetos)
n="$1"; t="${2:-python}"
[ -z "$n" ] && { echo "uso: novo <nome> [python|node|c]"; exit 1; }
mkdir -p "$HOME/projetos/$n" && cd "$HOME/projetos/$n" || exit 1
git init -q 2>/dev/null
case "$t" in
  python) echo 'print("olá, Termux!")' > main.py ;;
  node)   echo 'console.log("olá, Termux!")' > index.js; npm init -y >/dev/null 2>&1 ;;
  c)      printf '#include <stdio.h>\nint main(){puts("ola, Termux!");return 0;}\n' > main.c ;;
esac
echo "# $n" > README.md
echo "✔ Projeto criado em $(pwd)"
EOS
  [ -f "$SELF" ] && cp "$SELF" "$HOME/bin/termux-setup"
  chmod +x "$HOME"/bin/* 2>/dev/null
  touch "$HOME/.bashrc"
  sed -i '/# >>> termux-setup >>>/,/# <<< termux-setup <<</d' "$HOME/.bashrc"
  cat >> "$HOME/.bashrc" <<'EOS'
# >>> termux-setup >>>
export PATH="$HOME/bin:$PATH"
alias ..='cd ..'
alias gs='git status'
alias gp='git pull'
alias p='cd ~/projetos'
alias venv='source ~/venv/bin/activate'
if command -v eza >/dev/null 2>&1; then alias ll='eza -lah --git'; else alias ll='ls -lah --color=auto'; fi
command -v starship >/dev/null 2>&1 && eval "$(starship init bash)"
command -v zoxide   >/dev/null 2>&1 && eval "$(zoxide init bash)"
# <<< termux-setup <<<
EOS
  echo "Comandos criados: atualizar limpar info novo termux-setup (projetos em ~/projetos)"
}

# ---------------------------------------------------------------- Aparência
backup_termux() { # guarda uma cópia de ~/.termux antes de mudar qualquer coisa
  local bk="$LOGDIR/backup-termux-$TS"
  [ -d "$HOME/.termux" ] || return 0
  mkdir -p "$bk"; cp -a "$HOME/.termux/." "$bk/" 2>/dev/null
  return 0
}

step_hush() {
  touch "$HOME/.hushlogin" && echo "Mensagem de boas-vindas silenciada (~/.hushlogin)"
}

step_theme() {
  mkdir -p "$HOME/.termux"; backup_termux
  cat > "$HOME/.termux/colors.properties" <<'EOS'
background=#1e1e1e
foreground=#cccccc
cursor=#aeafad
color0=#000000
color1=#cd3131
color2=#0dbc79
color3=#e5e510
color4=#2472c8
color5=#bc3fbc
color6=#11a8cd
color7=#e5e5e5
color8=#666666
color9=#f14c4c
color10=#23d18b
color11=#f5f543
color12=#3b8eea
color13=#d670d6
color14=#29b8db
color15=#e5e5e5
EOS
  echo "Tema aplicado: cores do VS Code (fundo #1e1e1e)"
}

step_font() {
  local d="$WORK/font" f
  local url="https://github.com/JetBrains/JetBrainsMono/releases/download/v2.304/JetBrainsMono-2.304.zip"
  command -v curl  >/dev/null 2>&1 || { echo "curl ausente"; return 1; }
  command -v unzip >/dev/null 2>&1 || { echo "unzip ausente"; return 1; }
  rm -rf "$d"; mkdir -p "$d" "$HOME/.termux"
  curl -fL --max-time 180 -o "$d/jb.zip" "$url" || { echo "Download da fonte falhou: $url"; return 1; }
  unzip -o -q "$d/jb.zip" -d "$d/x" || { echo "unzip falhou"; return 1; }
  f="$(find "$d/x" -name 'JetBrainsMono-Regular.ttf' 2>/dev/null | head -n1)"
  [ -n "$f" ] || { echo "JetBrainsMono-Regular.ttf não encontrado no zip"; return 1; }
  backup_termux
  cp "$f" "$HOME/.termux/font.ttf" && echo "Fonte JetBrains Mono instalada"
}

keys_line() { # $1 = preset (1 compacta, 2 grande)
  case "$1" in
    1) cat <<'EOS'
extra-keys = [[{key: 'ESC', popup: {macro: "claude ENTER", display: "claude"}},{key: 'TAB', popup: {macro: "codex ENTER", display: "codex"}},{key: '/', popup: {macro: "gemini ENTER", display: "gemini"}},'PASTE','ENTER','UP','DRAWER'], ['CTRL','ALT',{macro: "CTRL c", display: "^C", popup: {macro: "ubuntu ENTER", display: "ubuntu"}},{macro: "CTRL d", display: "^D", popup: {macro: "gs ENTER", display: "gs"}},'LEFT','DOWN','RIGHT']]
EOS
    ;;
    2) cat <<'EOS'
extra-keys = [['ESC','TAB','CTRL','ALT',{macro: "CTRL c", display: "^C"},{macro: "CTRL d", display: "^D"},{macro: "CTRL l", display: "^L"},{macro: "CTRL z", display: "^Z"}], [{key: '/', popup: {macro: "|", display: "|"}},{key: '_', popup: {macro: "-", display: "-"}},{key: ':', popup: {macro: "~", display: "~"}},'PASTE','ENTER','KEYBOARD',{key: 'UP', popup: {macro: "PGUP", display: "PgUp"}},'DRAWER'], ['=','$','*','&',{macro: "CTRL r", display: "^R"},{key: 'LEFT', popup: {macro: "HOME", display: "Home"}},{key: 'DOWN', popup: {macro: "PGDN", display: "PgDn"}},{key: 'RIGHT', popup: {macro: "END", display: "End"}}], [{macro: "claude ENTER", display: "claude"},{macro: "codex ENTER", display: "codex"},{macro: "gemini ENTER", display: "gemini"},{macro: "ubuntu ENTER", display: "ubuntu"},{macro: "gs ENTER", display: "gs"},{macro: "gp ENTER", display: "gp"},{macro: "atualizar ENTER", display: "atualizar"},{macro: "p ENTER", display: "p"}]]
EOS
    ;;
  esac
}

step_keys() {
  local f="$HOME/.termux/termux.properties" line
  mkdir -p "$HOME/.termux"; backup_termux
  [ -f "$f" ] || : > "$f"
  line="$(keys_line "${KEYS_PRESET:-1}")"
  # troca só a linha extra-keys; as outras configurações do arquivo ficam intactas
  grep -v -E '^[[:space:]]*extra-keys[[:space:]]*=' "$f" > "$f.new"; mv "$f.new" "$f"
  if [ -n "$line" ]; then printf '%s\n' "$line" >> "$f"; echo "Barra de teclas: preset ${KEYS_PRESET:-1}"
  else echo "Barra de teclas: padrão do Termux restaurado"; fi
}

step_reload() {
  if command -v termux-reload-settings >/dev/null 2>&1; then termux-reload-settings; fi
  echo "Configurações recarregadas (se a fonte não mudar, feche e abra o Termux)"
  return 0
}

step_root() {
  apt_i root-repo || return 1
  apt-get update "${APT_OPTS[@]}" || true
  apt_i tsu
}

# ================================================================= EXECUÇÃO
live_log() { # última linha útil do log, só ASCII
  tail -c 500 "$LOG" 2>/dev/null | tr '\r' '\n' | tr -cd '\11\12\40-\176\200-\377' \
    | sed -e '/^[[:space:]]*$/d' -e '/^=====/d' | tail -n1
}

draw_dash() { # $1 = índice do spinner
  local i=$1 el=$((SECONDS - START)) w_done=0 j cur_w part pct eta="" bw h first n=$TOTAL lbl maxl t shown
  for ((j=0; j<DONE && j<n; j++)); do w_done=$(( w_done + ${PLAN_W[j]:-1} )); done
  cur_w=${PLAN_W[DONE]:-1}; [ "$cur_w" -lt 1 ] && cur_w=1
  part=$(( (SECONDS - STEP_T0) * 100 )); [ $part -gt $(( cur_w * 90 )) ] && part=$(( cur_w * 90 ))
  [ "$DONE" -ge "$TOTAL" ] && part=0
  pct=$(( (w_done * 100 + part) / TOTAL_W )); [ $pct -gt 100 ] && pct=100
  if [ $pct -ge 8 ] && [ $pct -lt 100 ]; then eta="$(fmt_t $(( el * (100 - pct) / pct )))"; fi
  bw=$(( COLS - 10 )); [ $bw -gt 34 ] && bw=34; [ $bw -lt 10 ] && bw=10
  maxl=$(( COLS - 12 )); [ $maxl -lt 10 ] && maxl=10
  shown=$(( DONE + 1 )); [ $shown -gt $TOTAL ] && shown=$TOTAL
  printf '\033[H'
  banner "$DASH_TITLE"
  L " ${C_G}$(bar "$pct" "$bw")${C_N} ${C_B}${pct}%${C_N}"
  L " ${C_D}passo ${shown}/${TOTAL} · $(fmt_t "$el")${eta:+ · ~$eta restam}${C_N}"
  L ""
  lbl="${PLAN_LABEL[DONE]:-}"
  L " ${C_CY}${SPIN[i%10]}${C_N} ${C_B}${lbl:0:$maxl}${C_N}"
  L "   ${C_D}${LIVE:0:$((COLS-5))}${C_N}"
  L ""
  h=$(( ROWS - 10 )); [ $h -lt 3 ] && h=3; [ $h -gt 16 ] && h=16; [ $h -gt $n ] && h=$n
  first=$(( DONE - h / 2 )); [ $first -lt 0 ] && first=0
  [ $(( first + h )) -gt $n ] && first=$(( n - h ))
  for ((j=first; j<first+h; j++)); do
    lbl="${PLAN_LABEL[j]:0:$((COLS-14))}"
    if [ "$j" -lt "$DONE" ]; then
      t="${RES_TIME[j]:-}"
      if [ "${RES_STAT[j]:-}" = OK ]; then L " ${C_G}✔${C_N} ${lbl} ${C_D}${t}${C_N}"
      else L " ${C_R}✘${C_N} ${lbl} ${C_D}${t}${C_N}"; fi
    elif [ "$j" -eq "$DONE" ]; then L " ${C_CY}▶${C_N} ${C_B}${lbl}${C_N} ${C_D}$(( SECONDS - STEP_T0 ))s${C_N}"
    else L " ${C_D}· ${lbl}${C_N}"; fi
  done
  printf '\033[J'
}

run_step() {
  local idx=$DONE label fn arg pid i=0 rc dur st
  label="${PLAN_LABEL[idx]}"; fn="${PLAN_FN[idx]}"; arg="${PLAN_ARG[idx]}"
  CUR_LABEL="$label"; STEP_T0=$SECONDS; LIVE=""
  printf '\n===== [%s/%s] %s (%s) =====\n' "$((idx+1))" "$TOTAL" "$label" "$(date +%T)" >> "$LOG"
  "$fn" "$arg" </dev/null >>"$LOG" 2>&1 &
  pid=$!; RUN_PID=$pid
  while kill -0 "$pid" 2>/dev/null; do
    [ $(( i % 4 )) -eq 0 ] && LIVE="$(live_log)"
    draw_dash "$i"
    i=$((i+1)); sleep 0.15
  done
  wait "$pid"; rc=$?
  RUN_PID=""; dur=$(( SECONDS - STEP_T0 ))
  if [ $rc -eq 0 ]; then st="OK"; else st="FALHOU"; fi
  RES_STAT[idx]="$st"; RES_TIME[idx]="${dur}s"
  DONE=$(( idx + 1 )); CUR_LABEL=""
  draw_dash "$i"
}

add_step() { PLAN_LABEL+=("$1"); PLAN_FN+=("$2"); PLAN_ARG+=("${3:-}"); PLAN_W+=("${4:-10}"); }
reset_plan() { PLAN_LABEL=(); PLAN_FN=(); PLAN_ARG=(); PLAN_W=(); }

diagnose() {
  grep -qE 'NO_PUBKEY|is not signed' "$LOG" && \
    echo "- Chave GPG ausente: provável Termux da Play Store. Use o app do F-Droid/GitHub."
  grep -q 'unmet dependencies' "$LOG" && \
    echo "- Dependências conflitantes: espelho dessincronizado. Use Espelhos e Reparar."
  grep -qE 'Could not resolve|Temporary failure|Failed to fetch|timed out|Err:[0-9]' "$LOG" && \
    echo "- Falha de rede/espelho: tente outro espelho ou outra rede."
  grep -q 'No space left' "$LOG" && echo "- Sem espaço no armazenamento."
  grep -qE 'dpkg was interrupted|dpkg --configure' "$LOG" && \
    echo "- dpkg interrompido: use o menu Reparar (ou: dpkg --configure -a)."
  grep -q 'Unable to locate package' "$LOG" && \
    echo "- Algum pacote não existe neste espelho/arquitetura."
  grep -qE 'Killed|signal 9' "$LOG" && \
    echo "- Android matou um processo (memória). Feche apps; Opções do desenvolvedor > limite de processos."
  grep -qi 'permission denied' "$LOG" && echo "- Erro de permissão em algum arquivo."
  return 0
}

write_summary() {
  local i st dur=$(( SECONDS - START )) free_now used_mb
  N_OK=0; N_FAIL=0; N_PEND=0
  for ((i=0; i<TOTAL; i++)); do
    st="${RES_STAT[i]:-}"
    if [ -z "$st" ]; then
      if [ "$i" -eq "$DONE" ] && [ -n "$CUR_LABEL" ]; then st="INTERROMPIDO"; else st="NAO EXECUTADO"; fi
      RES_STAT[i]="$st"; RES_TIME[i]="-"
    fi
    case "$st" in
      OK) N_OK=$((N_OK+1)) ;;
      FALHOU) N_FAIL=$((N_FAIL+1)) ;;
      *) N_PEND=$((N_PEND+1)) ;;
    esac
  done
  if [ "$INTERRUPTED" = 1 ] || [ "$N_PEND" -gt 0 ]; then STATUS_TXT="INCOMPLETO"
  elif [ "$N_FAIL" -gt 0 ]; then STATUS_TXT="CONCLUIDO COM FALHAS"
  else STATUS_TXT="COMPLETO"; fi
  free_now="$(free_kb)"; used_mb=$(( (FREE_BEFORE - free_now) / 1024 )); [ $used_mb -lt 0 ] && used_mb=0
  USED_MB=$used_mb; ELAPSED=$dur
  {
    echo "=== RESUMO TERMUX SETUP v$VERSION ==="
    echo "Status : $STATUS_TXT"
    echo "Início : $START_DATE"
    echo "Fim    : $(date '+%F %T') (${dur}s)"
    echo "Disp.  : $(uname -m) | Android $(getprop ro.build.version.release 2>/dev/null) | $(getprop ro.product.model 2>/dev/null)"
    echo "Espelho: $(cat "$WORK/mirror_used" 2>/dev/null || echo '(não alterado)')"
    echo "Espaço : ~${used_mb} MB usados | livre: $(( free_now / 1024 )) MB"
    echo "Passos : $N_OK ok, $N_FAIL falha(s), $N_PEND pendente(s) de $TOTAL"
    echo
    echo "--- Passos ---"
    for ((i=0; i<TOTAL; i++)); do
      printf '[%s] %s (%s)\n' "${RES_STAT[i]}" "${PLAN_LABEL[i]}" "${RES_TIME[i]}"
    done
    if [ -s "$WORK/unavailable" ]; then
      echo; echo "--- Pacotes indisponíveis (ignorados) ---"; sort -u "$WORK/unavailable" | tr '\n' ' '; echo
    fi
    if [ -s "$WORK/failed_pkgs" ]; then
      echo; echo "--- Pacotes que falharam ---"; sort -u "$WORK/failed_pkgs" | tr '\n' ' '; echo
    fi
    local d; d="$(diagnose)"
    if [ -n "$d" ]; then echo; echo "--- Diagnóstico automático ---"; echo "$d"; fi
    echo; echo "--- Últimas linhas de erro do log ---"
    grep -iE '(^E:|^Err:|error|failed|unable|cannot|denied|unmet)' "$LOG" | tail -n 15 | cut -c1-110
    echo; echo "Log completo: $LOG"
    echo "Resumo      : $SUMMARY"
  } > "$SUMMARY" 2>&1
  if [ -d "$HOME/storage/downloads" ]; then
    cp "$SUMMARY" "$LOG" "$HOME/storage/downloads/" 2>/dev/null
  fi
}

show_card() {
  local i col d
  term_size; printf '\033[2J\033[H'
  banner "Resultado"
  case "$STATUS_TXT" in
    COMPLETO) col="$C_G" ;;
    INCOMPLETO) col="$C_R" ;;
    *) col="$C_Y" ;;
  esac
  L " ${col}${C_B}${STATUS_TXT}${C_N}"
  L ""
  L " ${C_G}✔ ${N_OK} ok${C_N}   ${C_R}✘ ${N_FAIL} falhas${C_N}   ${C_Y}· ${N_PEND} pendentes${C_N}"
  L " ${C_D}tempo $(fmt_t "$ELAPSED") · espaço usado ~${USED_MB} MB · livre $(( $(free_kb) / 1024 )) MB${C_N}"
  L " ${C_D}espelho: $(current_mirror)${C_N}"
  for ((i=0; i<TOTAL; i++)); do
    case "${RES_STAT[i]}" in
      FALHOU) L " ${C_R}✘${C_N} ${PLAN_LABEL[i]:0:$((COLS-6))}" ;;
      INTERROMPIDO) L " ${C_Y}■${C_N} ${PLAN_LABEL[i]:0:$((COLS-6))} (interrompido)" ;;
    esac
  done
  d="$(diagnose)"
  if [ -n "$d" ]; then L ""; while IFS= read -r line; do L " ${C_Y}${line:0:$((COLS-3))}${C_N}"; done <<< "$d"; fi
  L ""
  if [ "$STATUS_TXT" = "INCOMPLETO" ]; then
    L " ${C_B}O que fazer${C_N}"
    L "  ${C_D}Rode o script de novo: ele pula o que${C_N}"
    L "  ${C_D}já está instalado e continua.${C_N}"
  else
    L " ${C_B}Próximos passos${C_N}"
    L "  ${C_CY}source ~/.bashrc${C_N} ${C_D}(ou feche e abra o Termux)${C_N}"
    [ "${SEL[claude]:-0}" = 1 ] && L "  ${C_CY}claude${C_N}  ${C_D}login pelo navegador${C_N}"
    [ "${SEL[codex]:-0}" = 1 ]  && L "  ${C_CY}codex${C_N}   ${C_D}Sign in with ChatGPT${C_N}"
    [ "${SEL[gemini]:-0}" = 1 ] && L "  ${C_CY}gemini${C_N}  ${C_D}login Google ou GEMINI_API_KEY${C_N}"
    [ "${SEL[ubuntu]:-0}${SEL[claude]:-0}${SEL[codex]:-0}${SEL[gemini]:-0}" != "0000" ] && \
      L "  ${C_CY}ubuntu${C_N}  ${C_D}shell Ubuntu na pasta atual${C_N}"
    { [ "${SEL[keys]:-0}" = 1 ] || [ "${SEL[theme]:-0}" = 1 ] || [ "${SEL[font]:-0}" = 1 ]; } && \
      L "  ${C_D}Barra/tema/fonte: se não mudarem, feche e abra o Termux${C_N}"
    [ "${SEL[dev2]:-0}" = 1 ] && L "  ${C_CY}GitHub${C_N} ${C_D}use o menu GitHub (entrar com token)${C_N}"
    L "  ${C_CY}termux-setup${C_N} ${C_D}reabre este menu${C_N}"
  fi
  L ""
  L " ${C_D}log: ${SUMMARY/#$HOME/~}${C_N}"
  [ -d "$HOME/storage/downloads" ] && L " ${C_D}(cópia em Downloads)${C_N}"
  printf '\033[J'
}

notify_done() {
  command -v termux-notification >/dev/null 2>&1 || return 0
  ( timeout 6 termux-notification --title "Termux Setup" --content "$1" >/dev/null 2>&1
    command -v termux-vibrate >/dev/null 2>&1 && timeout 4 termux-vibrate -d 250 >/dev/null 2>&1 ) &
  return 0
}

finish() {
  [ "$INSTALL_STARTED" = 1 ] || return 0
  [ "$FINISHED" = 1 ] && return 0
  FINISHED=1
  show_cur
  if [ -n "$RUN_PID" ]; then
    pkill -P "$RUN_PID" 2>/dev/null; kill "$RUN_PID" 2>/dev/null; RUN_PID=""
  fi
  write_summary
  CUR_LABEL=""
  show_card
  [ "$INTERRUPTED" = 1 ] || notify_done "$STATUS_TXT: $N_OK ok, $N_FAIL falhas"
}

on_int() { INTERRUPTED=1; echo; echo "${C_Y}Interrompido pelo usuário.${C_N}"; exit 130; }
on_exit() { finish; show_cur; rm -rf "$WORK"; }
trap on_int INT TERM
trap on_exit EXIT

execute_plan() { # [quiet]
  local quiet="${1:-}" w
  TOTAL=${#PLAN_LABEL[@]}; [ "$TOTAL" -lt 1 ] && return 0
  term_size; new_run
  DONE=0; RES_STAT=(); RES_TIME=(); TOTAL_W=0
  for w in "${PLAN_W[@]}"; do TOTAL_W=$(( TOTAL_W + w )); done
  [ $TOTAL_W -lt 1 ] && TOTAL_W=1
  START=$SECONDS; START_DATE="$(date '+%F %T')"; FREE_BEFORE="$(free_kb)"
  STEP_T0=$SECONDS; LIVE=""
  if [ -z "$quiet" ]; then DASH_TITLE="Instalando"; INSTALL_STARTED=1; FINISHED=0; INTERRUPTED=0
  else DASH_TITLE="Trabalhando"; fi
  printf '\033[2J\033[H'; hide_cur
  while [ "$DONE" -lt "$TOTAL" ]; do run_step; done
  if [ -z "$quiet" ]; then
    finish; INSTALL_STARTED=0; pause
  else
    show_cur
  fi
}

# ================================================================= PLANOS
clear_sel()  { local m; for m in "${MOD_ORDER[@]}"; do SEL[$m]=0; done; }
select_all() { local m; for m in "${MOD_ORDER[@]}"; do SEL[$m]=1; done; SEL[root]=0; }
select_dev() { local m; clear_sel; for m in base build python node extras term dev2 api hush theme font keys; do SEL[$m]=1; done; }
select_ia()  { local m; clear_sel; for m in base api ubuntu claude codex gemini; do SEL[$m]=1; done; }

add_look_steps() {
  local any=0
  [ "${SEL[hush]:-0}" = 1 ]  && add_step "Silenciar boas-vindas" step_hush "" 1 && any=1
  [ "${SEL[theme]:-0}" = 1 ] && add_step "Tema VS Code" step_theme "" 1 && any=1
  [ "${SEL[font]:-0}" = 1 ]  && add_step "Fonte JetBrains Mono" step_font "" 8 && any=1
  [ "${SEL[keys]:-0}" = 1 ]  && add_step "Barra de teclas" step_keys "" 1 && any=1
  [ $any = 1 ] && add_step "Aplicar aparência" step_reload "" 1
  return 0
}

build_plan_look() { reset_plan; add_look_steps; }

ask_keys_preset() {
  [ "${SEL[keys]:-0}" = 1 ] || return 0
  MENU_ITEMS=("Compacta (2 linhas, recomendada)" "Grande (4 linhas)" "Padrão do Termux")
  MENU_KEYS=(1 2 3)
  MENU_PRE=("${C_B}Compacta${C_N}${C_D}  ESC TAB / PASTE ENTER ↑ DRAWER${C_N}"
            "${C_D}          CTRL ALT ^C ^D ← ↓ →${C_N}"
            "${C_D}Deslizar p/ cima: ESC=claude TAB=codex${C_N}"
            "${C_D}/=gemini ^C=ubuntu ^D=gs${C_N}"
            "${C_B}Grande${C_N}${C_D} 4 linhas, mais símbolos e comandos${C_N}")
  menu_select "Barra de teclas"
  case "$MENU_IDX" in 1) KEYS_PRESET=2 ;; 2) KEYS_PRESET=3 ;; *) KEYS_PRESET=1 ;; esac
}

ask_options() {
  local r
  GIT_NAME=""; GIT_MAIL=""; DO_STORAGE=0
  term_size; printf '\033[2J\033[H'; banner "Opções"; show_cur
  if [ -z "$(git config --global user.name 2>/dev/null)" ]; then
    read -r -p " Nome p/ Git (Enter pula): " GIT_NAME || true
    [ -n "$GIT_NAME" ] && { read -r -p " E-mail p/ Git: " GIT_MAIL || true; }
  fi
  if [ ! -d "$HOME/storage" ]; then
    read -r -p " Liberar acesso ao armazenamento? [S/n] " r || true
    [[ ! $r =~ ^[nN] ]] && DO_STORAGE=1
  fi
  ask_keys_preset
}

build_plan() { # modo: full | update
  local m need_ub=0 need_node=0
  reset_plan
  add_step "Testar espelhos" step_mirrors "" 4
  add_step "Aplicar melhor espelho" step_apply_mirror "" 1
  add_step "Atualizar listas (apt)" step_update "" 5
  add_step "Atualizar pacotes" step_upgrade "" 35
  if [ "$1" = full ]; then
    for m in "${MOD_ORDER[@]}"; do
      [ "${SEL[$m]:-0}" = 1 ] || continue
      case "$m" in
        ubuntu|claude|codex|gemini|hush|theme|font|keys|root) ;;
        *) add_step "${MOD_NAME[$m]}" install_mod "$m" "${MOD_W[$m]}" ;;
      esac
    done
    [ "${SEL[python]:-0}" = 1 ] && add_step "Python: venv + libs" step_pyvenv "" 25
    [ "${SEL[node]:-0}" = 1 ]   && add_step "Node: yarn + typescript" step_npm "" 15
    for m in ubuntu claude codex gemini; do [ "${SEL[$m]:-0}" = 1 ] && need_ub=1; done
    for m in codex gemini; do [ "${SEL[$m]:-0}" = 1 ] && need_node=1; done
    if [ $need_ub = 1 ]; then
      add_step "Ubuntu: proot-distro" step_proot_pkgs "" 10
      add_step "Ubuntu 26.04 LTS: instalar" step_ubuntu_install "" 90
      add_step "Ubuntu: base + usuário dev" step_ubuntu_base "" 60
      [ $need_node = 1 ] && add_step "Ubuntu: Node.js LTS" step_ubuntu_node "" 45
      [ "${SEL[claude]:-0}" = 1 ] && add_step "Claude Code (oficial)" step_claude "" 50
      [ "${SEL[codex]:-0}" = 1 ]  && add_step "Codex CLI (OpenAI)" step_ub_npm "@openai/codex" 30
      [ "${SEL[gemini]:-0}" = 1 ] && add_step "Gemini CLI (Google)" step_ub_npm "@google/gemini-cli" 35
      add_step "Criar comandos claude/codex/gemini" step_wrappers "" 1
    fi
    { [ -n "$GIT_NAME" ] || [ -n "$GIT_MAIL" ]; } && add_step "Configurar Git" step_git "" 1
    [ "$DO_STORAGE" = 1 ] && add_step "Acesso ao armazenamento" step_storage "" 3
    add_step "Criar comandos e aliases" step_commands "" 1
    add_look_steps
    [ "${SEL[root]:-0}" = 1 ] && add_step "Root: tsu" step_root "" 12
  fi
  add_step "Limpeza" step_cleanup "" 3
}

est_totals() {
  local m need_node=0
  EST_DL=30; EST_DISK=60
  [ "${PLAN_FN[0]:-}" = step_mirrors ] || { EST_DL=0; EST_DISK=0; }
  for m in "${MOD_ORDER[@]}"; do
    [ "${SEL[$m]:-0}" = 1 ] || continue
    EST_DL=$(( EST_DL + MOD_DL[$m] )); EST_DISK=$(( EST_DISK + MOD_DISK[$m] ))
  done
  for m in codex gemini; do [ "${SEL[$m]:-0}" = 1 ] && need_node=1; done
  [ $need_node = 1 ] && { EST_DL=$(( EST_DL + 30 )); EST_DISK=$(( EST_DISK + 150 )); }
  return 0
}

confirm_run() {
  local free_mb w secs=0
  est_totals
  free_mb=$(( $(free_kb) / 1024 ))
  for w in "${PLAN_W[@]}"; do secs=$(( secs + w )); done
  MENU_PRE=(
    "${C_B}Passos:${C_N} ${#PLAN_LABEL[@]}   ${C_B}Tempo:${C_N} ~$(fmt_t $secs) ${C_D}(estimado)${C_N}"
    "${C_B}Download:${C_N} ~${EST_DL} MB   ${C_B}Disco:${C_N} ~${EST_DISK} MB"
    "${C_B}Livre:${C_N} ${free_mb} MB ${C_D}(valores aproximados)${C_N}"
  )
  if [ "$free_mb" -lt $(( EST_DISK * 12 / 10 + 300 )) ]; then
    MENU_PRE+=("${C_R}Pouco espaço livre para esta seleção!${C_N}")
  fi
  MENU_PRE+=("${C_D}Recomendado: Wi-Fi (downloads grandes).${C_N}")
  if [ "${SEL[claude]:-0}" = 1 ]; then
    MENU_PRE+=("${C_D}Claude Code: RAM >= 4 GB (requisito oficial).${C_N}")
  fi
  MENU_ITEMS=("Iniciar instalação" "Voltar")
  MENU_KEYS=(1 0)
  menu_select "Confirmar"
  [ "$MENU_IDX" = 0 ]
}

run_plan_flow() { # após selecionar módulos
  ask_options; build_plan full
  if confirm_run; then execute_plan; fi
}

menu_mirror() {
  local url line age t host tag i=0
  reset_plan; add_step "Testando espelhos" step_mirrors "" 6
  execute_plan quiet
  term_size; printf '\033[2J\033[H'
  if [ ! -s "$WORK/candidates" ]; then
    banner "Espelhos"; L " ${C_R}Nenhum espelho respondeu.${C_N} Verifique a internet."
    pause; return
  fi
  MENU_ITEMS=(); MENU_KEYS=()
  while read -r url; do
    line="$(grep -F " $url" "$WORK/ranking" | head -n1)"
    age="${line%% *}"; t="$(echo "$line" | cut -d' ' -f2)"
    host="$(echo "${url#*://}" | cut -d/ -f1)"
    if [ "$age" -le 48 ] 2>/dev/null; then tag="${C_G}fresco${C_N}"; else tag="${C_R}velho${C_N}"; fi
    MENU_ITEMS+=("$(printf '%-22s %.4ss %sh' "${host:0:22}" "$t" "$age") $tag")
    if [ $i -lt 9 ]; then MENU_KEYS+=($((i+1))); else MENU_KEYS+=(""); fi
    i=$((i+1))
  done < "$WORK/candidates"
  MENU_ITEMS+=("Voltar"); MENU_KEYS+=(0)
  MENU_PRE=("${C_D}Escolha o espelho (o 1º é o melhor)${C_N}")
  menu_select "Espelhos"
  [ "$MENU_IDX" -lt 0 ] && return
  url="$(sed -n "$((MENU_IDX+1))p" "$WORK/candidates")"
  [ -z "$url" ] && return
  reset_plan
  add_step "Aplicando espelho" apply_mirror "$url" 1
  add_step "apt update" step_update "" 5
  execute_plan quiet
  term_size; printf '\033[2J\033[H'; banner "Espelhos"
  if [ "${RES_STAT[1]:-}" = OK ]; then L " ${C_G}✔${C_N} Espelho aplicado: ${url#*://}"
  else L " ${C_R}✘${C_N} apt update falhou (veja o log)"; fi
  L " ${C_D}log: ${LOG/#$HOME/~}${C_N}"
  pause
}

verify_install() {
  local c v out st name rest
  term_size; printf '\033[2J\033[H'; banner "Verificar instalação"
  L " ${C_B}Termux${C_N}"
  for c in git python pip node npm java go rustc clang cmake tmux jq rg gh nvim starship; do
    if command -v "$c" >/dev/null 2>&1; then
      case "$c" in
        go)   v="$(timeout 5 go version 2>&1 | head -n1)" ;;
        java) v="$(timeout 8 java -version 2>&1 | head -n1)" ;;
        *)    v="$(timeout 5 "$c" --version 2>&1 | head -n1)" ;;
      esac
      L " ${C_G}✔${C_N} $(printf '%-8s' "$c") ${C_D}${v:0:$((COLS-14))}${C_N}"
    else
      L " ${C_D}· $(printf '%-8s' "$c") não instalado${C_N}"
    fi
  done
  if command -v proot-distro >/dev/null 2>&1 && ub_has; then
    L ""; L " ${C_B}Ubuntu (proot)${C_N}"
    out="$(ub_dev 'export PATH="$HOME/.local/bin:$HOME/.npm-global/bin:$PATH"; for c in node claude codex gemini; do if command -v $c >/dev/null 2>&1; then echo "OK $c $($c --version 2>&1 | head -n1)"; else echo "NO $c"; fi; done' 2>/dev/null)"
    while read -r st name rest; do
      [ -n "$name" ] || continue
      if [ "$st" = OK ]; then L " ${C_G}✔${C_N} $(printf '%-8s' "$name") ${C_D}${rest:0:$((COLS-14))}${C_N}"
      else L " ${C_D}· $(printf '%-8s' "$name") não instalado${C_N}"; fi
    done <<< "$out"
  else
    L ""; L " ${C_D}Ubuntu: não instalado${C_N}"
  fi
  pause
}

menu_log() {
  local f
  term_size; printf '\033[2J\033[H'; banner "Último log"
  f="$(ls -t "$LOGDIR"/resumo-*.txt 2>/dev/null | head -n1)"
  if [ -n "$f" ]; then echo " ${f/#$HOME/~}"; echo; cat "$f"
  else
    f="$(ls -t "$LOGDIR"/setup-*.log 2>/dev/null | head -n1)"
    if [ -n "$f" ]; then tail -n 40 "$f"; else echo " Nenhum log ainda."; fi
  fi
  pause
}

run_repair() {
  reset_plan
  add_step "Reparar pacotes quebrados" step_repair "" 30
  add_step "Limpeza" step_cleanup "" 3
  clear_sel; execute_plan
}

menu_ia() {
  local m
  while true; do
    MENU_ITEMS=("Instalar tudo (Ubuntu + 3 IAs)" "Escolher quais instalar" "Atualizar Ubuntu e IAs" "Entrar no Ubuntu (shell)" "Voltar")
    MENU_KEYS=(1 2 3 4 0)
    MENU_PRE=("${C_D}Roda dentro do Ubuntu (proot), que é Linux${C_N}"
              "${C_D}ARM64, suportado pelo Claude Code.${C_N}"
              "${C_D}O login de cada IA é feito no navegador.${C_N}")
    menu_select "IA & Ubuntu"
    case "$MENU_IDX" in
      0) select_ia; run_plan_flow ;;
      1) clear_sel; SEL[base]=1; SEL[api]=1
         CHECK_TITLE="IA & Ubuntu"
         if menu_check ubuntu claude codex gemini; then CHECK_TITLE=""; run_plan_flow; fi
         CHECK_TITLE="" ;;
      2) reset_plan
         add_step "Ubuntu: atualizar pacotes" step_ub_upgrade "" 40
         add_step "Atualizar Claude/Codex/Gemini" step_ia_update "" 40
         clear_sel; execute_plan ;;
      3) term_size; printf '\033[2J\033[H'; show_cur
         if command -v proot-distro >/dev/null 2>&1 && ub_has; then
           proot-distro login "$UB" --user dev -- bash -l
         else
           banner "Ubuntu"; L " ${C_Y}Ubuntu ainda não instalado.${C_N}"; pause
         fi ;;
      *) return ;;
    esac
  done
}

menu_appearance() {
  clear_sel
  SEL[hush]=1; SEL[theme]=1; SEL[font]=1; SEL[keys]=1
  CHECK_TITLE="Aparência"
  if menu_check hush theme font keys; then
    CHECK_TITLE=""
    ask_keys_preset
    build_plan_look
    if [ ${#PLAN_LABEL[@]} -gt 0 ] && confirm_run; then execute_plan; fi
  fi
  CHECK_TITLE=""
}

menu_github() {
  local r tok
  term_size; printf '\033[2J\033[H'; banner "GitHub"; show_cur
  if ! command -v gh >/dev/null 2>&1; then
    L " ${C_Y}O gh não está instalado.${C_N}"
    read -r -p " Instalar agora? [S/n] " r || true
    [[ ! $r =~ ^[nN] ]] && apt-get install "${APT_OPTS[@]}" gh
    command -v gh >/dev/null 2>&1 || { L " ${C_R}gh indisponível.${C_N}"; pause; return; }
  fi
  if gh auth status >/dev/null 2>&1; then
    L " ${C_G}✔ Já autenticado:${C_N}"; gh auth status 2>&1 | head -n 6
    read -r -p " Entrar de novo com outro token? [s/N] " r || true
    [[ $r =~ ^[sSyY] ]] || { pause; return; }
  fi
  echo
  L " O login pelo navegador trava no Termux."
  L " Use um token (não aparece ao colar):"
  L "  1. abra ${C_CY}github.com/settings/tokens${C_N}"
  L "  2. crie um token clássico: ${C_B}repo${C_N} + ${C_B}read:org${C_N}"
  L "  3. copie e cole abaixo"
  echo
  read -rsp " Token: " tok || true; echo
  if [ -z "$tok" ]; then L " Cancelado."; pause; return; fi
  if printf '%s\n' "$tok" | gh auth login --hostname github.com --git-protocol https --with-token; then
    gh auth setup-git && L " ${C_G}✔ Git configurado para usar o gh${C_N}"
    gh auth status 2>&1 | head -n 6
  else
    L " ${C_R}✘ Token recusado.${C_N} Confira os escopos e a validade."
  fi
  tok=""; pause
}

main_menu() {
  while true; do
    MENU_ITEMS=("Instalação completa (tudo + IA)" "Dev padrão (recomendado)" "IA & Ubuntu (Claude, Codex, Gemini)" "Aparência (teclas, tema, fonte)" "Personalizada (escolher módulos)" "GitHub (entrar com token)" "Só atualizar o sistema" "Espelhos: testar / trocar" "Reparar pacotes quebrados" "Verificar instalação" "Ver último log" "Sair")
    MENU_KEYS=(1 2 3 4 5 6 7 8 9 v l 0)
    MENU_PRE=("${C_D}Android $(getprop ro.build.version.release 2>/dev/null || echo ?) · $(getprop ro.product.model 2>/dev/null) · $(uname -m)${C_N}"
              "${C_D}livre $(( $(free_kb) / 1024 )) MB · espelho: $(current_mirror | cut -c1-28)${C_N}")
    menu_select "Menu principal"
    case "$MENU_IDX" in
      0) select_all; run_plan_flow ;;
      1) select_dev; run_plan_flow ;;
      2) menu_ia ;;
      3) menu_appearance ;;
      4) clear_sel; select_dev
         if menu_check "${MOD_ORDER[@]}"; then run_plan_flow; fi ;;
      5) menu_github ;;
      6) clear_sel; build_plan update
         if confirm_run; then execute_plan; fi ;;
      7) menu_mirror ;;
      8) run_repair ;;
      9) verify_install ;;
      10) menu_log ;;
      *) break ;;
    esac
  done
  printf '\033[2J\033[H'
  echo " Até mais! Logs em ~/termux-setup-logs"
}

# ================================================================= MODO TESTE
if [ "$DRY" = 1 ]; then
  apt-get() {
    echo "[teste] apt-get $*"; sleep 0.4
    if [ -n "${TEST_FAIL:-}" ] && [[ "$*" == *"$TEST_FAIL"* ]]; then echo "E: simulado"; return 100; fi
    return 0
  }
  apt-cache() { case "$2" in rust|nmap) return 1;; esac; return 0; }
  dpkg() { echo "[teste] dpkg $*"; return 0; }
  probe_mirror() { echo "$((RANDOM%60)) 0.$((RANDOM%900+100)) $1" > "$2"; }
  termux-setup-storage() { echo "[teste] storage"; }
  proot-distro() { echo "[teste] proot-distro $*"; sleep 0.3; return 0; }
  termux-notification() { echo "[teste] notificacao: $*" >> "$HOME/notif.txt"; }
  termux-vibrate() { echo "[teste] vibrar" >> "$HOME/notif.txt"; }
  getprop() { echo "teste"; }
  step_pyvenv() { echo "[teste] venv"; sleep 0.8; }
  step_npm() { echo "[teste] npm"; sleep 0.6; }
  step_font() { echo "[teste] baixando fonte"; sleep 0.8; mkdir -p "$HOME/.termux"; echo FAKE > "$HOME/.termux/font.ttf"; }
  termux-reload-settings() { echo "[teste] reload-settings"; }
fi

main() {
  command -v curl >/dev/null 2>&1 || echo "${C_Y}Aviso: curl não encontrado; o teste de espelhos pode falhar.${C_N}"
  new_run
  splash
  main_menu
}
main "$@"
