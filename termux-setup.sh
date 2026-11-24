#!/data/data/com.termux/files/usr/bin/bash
# =====================================================================
#  TERMUX SETUP v5.0
#  Visual "hacker": chuva de caracteres, logo que se decifra, glitch,
#  onda senoidal, menus animados (desligue com --sem-animacao).
#  Painel fixo, menu por setas, estimativas, log final, Ubuntu (proot)
#  + Ubuntu: linguagens (Python, Node, Java, C/C++) e atualização completa
#  + Claude Code, Codex CLI e Gemini CLI (instaladores oficiais)
#  + Aparência: barra de teclas, tema VS Code, fonte JetBrains Mono
#  + GitHub (login por token) e Root (tsu) opcionais.
#  Uso:  bash termux-setup.sh               (menu)
#        bash termux-setup.sh --ajuda       (lista todas as opções)
#  NÃO use "source". Nunca fecha o Termux sozinho.
# =====================================================================

if [ -n "${BASH_SOURCE[0]:-}" ] && [ "${BASH_SOURCE[0]}" != "$0" ]; then
  echo "Execute com:  bash ${BASH_SOURCE[0]##*/}   (não use 'source' nem '.')"
  return 1 2>/dev/null
fi
if [ -z "${BASH_VERSINFO:-}" ] || [ "${BASH_VERSINFO[0]}" -lt 4 ]; then
  echo "Este script precisa de bash 4 ou superior."; exit 1
fi

VERSION="5.0"
SELF="$(readlink -f "$0" 2>/dev/null || echo "$0")"

# ================================================================= CONFIGURAÇÃO
# Mude aqui as versões sem precisar mexer no resto do script.
JAVA_PKGS="openjdk-21|openjdk-17"         # "a|b" = usa o primeiro que existir
NODE_PKGS="nodejs-lts|nodejs"
UBUNTU_VERSIONS="26.04 24.04"             # tenta na ordem; a 1ª é a preferida
UB="ubuntu"                               # nome da distro no proot-distro
# Usuário comum criado dentro do Ubuntu. Programas como o Claude Code não
# devem rodar como root, por isso as IAs usam esse usuário (casa: /home/dev).
UB_USER="dev"
NODESOURCE_URL="https://deb.nodesource.com/setup_lts.x"
CLAUDE_URL="https://claude.ai/install.sh"
FONT_VER="2.304"
FONT_URL="https://github.com/JetBrains/JetBrainsMono/releases/download/v${FONT_VER}/JetBrainsMono-${FONT_VER}.zip"
FONT_SHA256="6f6376c6ed2960ea8a963cd7387ec9d76e3f629125bc33d1fdcd7eb7012f7bbf"

# ================================================================= ARGUMENTOS
usage() {
  cat <<EOF
Termux Setup v$VERSION
Uso: bash termux-setup.sh [opção] [--sim] [--teste]

  (sem opção)     abre o menu
  --tudo          instalação completa (tudo + IA)
  --dev           dev padrão (recomendado)
  --ia            Ubuntu (+ ferramentas) + Claude Code, Codex e Gemini
  --atualizar     atualiza Termux, Ubuntu e IAs (sem menu)
  --desfazer      restaura espelho, aparência e .bashrc originais
  --sim           não faz perguntas (usa os padrões)
  --sem-animacao  desliga as animações (celular lento)
  --teste         simulação: não instala nada
  --ajuda         mostra esta ajuda
EOF
}
DRY=0; AUTO=0; ACTION=""; NOANIM=0
for a in "$@"; do
  case "$a" in
    --teste) DRY=1 ;;
    --sim|-y) AUTO=1 ;;
    --sem-animacao) NOANIM=1 ;;
    --tudo|--dev|--ia|--atualizar|--desfazer) ACTION="${a#--}" ;;
    -h|--ajuda|--help) usage; exit 0 ;;
    *) echo "Opção desconhecida: $a"; echo; usage; exit 1 ;;
  esac
done

# só roda dentro do Termux (exceto no modo teste)
if [ "$DRY" = 0 ] && [ -z "${TERMUX_VERSION:-}" ] && [ ! -x /data/data/com.termux/files/usr/bin/bash ]; then
  echo "Este script é só para o Termux (Android). Use --teste para simular."
  exit 1
fi

if [ "$DRY" = 1 ]; then
  PREFIX="$(mktemp -d)"; HOME="$(mktemp -d)"
  mkdir -p "$PREFIX/etc/apt/sources.list.d" "$PREFIX/tmp"
  echo "deb https://velho.exemplo/termux-main stable main" > "$PREFIX/etc/apt/sources.list"
fi
PREFIX="${PREFIX:-/data/data/com.termux/files/usr}"
export DEBIAN_FRONTEND=noninteractive

# ---------------------------------------------------------------- cores
# Paleta de 256 cores (o Termux suporta). PALC = degradê ciano -> magenta.
E=$'\033'
C_N=$'\033[0m'; C_B=$'\033[1m'; C_D=$'\033[2m'
C_R=$'\033[1;38;5;203m'; C_G=$'\033[1;38;5;84m'; C_Y=$'\033[1;38;5;221m'
C_CY=$'\033[1;38;5;51m'; C_M=$'\033[1;38;5;207m'
C_W=$'\033[38;5;255m'; C_TX=$'\033[38;5;250m'; C_DIM3=$'\033[38;5;244m'; C_DK=$'\033[38;5;239m'
C_HEAD=$'\033[1;38;5;231m'; C_ACC=$'\033[1;38;5;207m'; BG_SEL=$'\033[48;5;235m'
PALC=(); for _c in 51 45 39 33 63 99 135 171 207 201; do PALC+=("${E}[38;5;${_c}m"); done
NP=${#PALC[@]}
GR=("${E}[38;5;22m" "${E}[38;5;28m" "${E}[38;5;34m" "${E}[38;5;40m" "${E}[38;5;46m") # verdes da "chuva"
GLY=(0 1 2 3 4 5 6 7 8 9 a b c d e f A B C D E F '#' '$' '%' '&' '*' '+' '=' '<' '>' '/' '\' '|' '{' '}' '[' ']' '?' '!' '~' '^' λ Σ π Δ ∇ ∫ √ ∞ ≈ Ω µ φ θ ψ)
NG=${#GLY[@]}
SINE=(4 5 6 7 7 6 5 4 3 2 1 0 0 1 2 3)    # seno inteiro (16 pontos, 0..7)
SPARK=(▁ ▂ ▃ ▄ ▅ ▆ ▇ █)
SPIN=('⠋' '⠙' '⠹' '⠸' '⠼' '⠴' '⠦' '⠧' '⠇' '⠏')
GRAD_NAME="${PALC[0]}◈ ${C_B}"
for ((_i=0; _i<6; _i++)); do _t=TERMUX; GRAD_NAME+="${PALC[_i+3]}${_t:_i:1}"; done
GRAD_NAME+="${C_N}"
ROWS=24; COLS=40; RULE=""; RULEW=38

# animações: só em terminal de verdade, e nunca no modo --sim
ANIM=1
{ [ -t 0 ] && [ -t 1 ]; } || ANIM=0
[ "$AUTO" = 1 ] && ANIM=0
[ "$NOANIM" = 1 ] && ANIM=0
[ "${TSETUP_ANIM:-1}" = 0 ] && ANIM=0
FX_SKIP=0; FX_INTRO=0; FX_REVEAL=0
# "sleep" sem criar processo (lê com tempo-limite de um pipe que nunca recebe nada)
FXFD=""; exec {FXFD}<> <(:) || FXFD=""

rep() { local s="" i; for ((i=0; i<$2; i++)); do s+="$1"; done; printf '%s' "$s"; }
term_size() {
  COLS=$(tput cols 2>/dev/null || echo 40); ROWS=$(tput lines 2>/dev/null || echo 24)
  [[ $COLS =~ ^[0-9]+$ ]] || COLS=40
  [[ $ROWS =~ ^[0-9]+$ ]] || ROWS=24
  RULEW=$(( COLS > 61 ? 60 : COLS - 2 ))
  RULE="$(rep '─' $(( COLS > 61 ? 60 : COLS - 1 )))"
}
term_size
L() { printf '%s\033[K\033[0m\n' "$*"; [ "$FX_REVEAL" = 1 ] && fx_sleep 0.012; }
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
MOD_ORDER=(base build python node java web db net extras term dev2 api ubuntu ubtools ubbuild ubpy ubnode ubjava claude codex gemini hush theme font keys root)
UB_MODS="ubtools ubbuild ubpy ubnode ubjava"   # módulos instalados DENTRO do Ubuntu
declare -A MOD_NAME=(
  [base]="Base: git ssh vim zip"
  [build]="Compiladores: clang make cmake"
  [python]="Python + pip + venv"
  [node]="Node.js LTS + yarn + tsc"
  [java]="Java 21 + gradle + maven"
  [web]="Ruby + Perl"
  [db]="SQLite + Redis"
  [net]="Rede: nmap rsync netcat"
  [extras]="Extras: tmux jq rg htop"
  [term]="Terminal: eza + zoxide"
  [dev2]="Dev+: gh lazygit neovim"
  [api]="Termux:API (notificações)"
  [ubuntu]="Ubuntu ${UBUNTU_VERSIONS%% *} LTS (proot)"
  [ubtools]="Ubuntu: ferramentas (git jq rg htop)"
  [ubbuild]="Ubuntu: C/C++ (gcc make cmake gdb)"
  [ubpy]="Ubuntu: Python 3 + pip + venv"
  [ubnode]="Ubuntu: Node.js LTS + npm"
  [ubjava]="Ubuntu: Java 21 (JDK) + maven"
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
  [node]="$NODE_PKGS"
  [java]="$JAVA_PKGS gradle maven"
  [web]="ruby perl"
  [db]="sqlite redis"
  [net]="nmap sshpass netcat-openbsd dnsutils traceroute net-tools rsync"
  [extras]="htop fastfetch|neofetch tmux jq ripgrep fd bat fzf ncdu zsh"
  [term]="zoxide eza"
  [dev2]="gh lazygit neovim"
  [api]="termux-api"
)
# pacotes do apt do Ubuntu ("a|b" = o primeiro que existir). Node vem do NodeSource.
declare -A UB_PKGS=(
  [ubtools]="git curl wget zip unzip jq ripgrep fd-find htop tmux vim nano tree less file openssh-client sqlite3 rsync"
  [ubbuild]="build-essential cmake gdb pkg-config autoconf automake libtool ninja-build libssl-dev libffi-dev zlib1g-dev"
  [ubpy]="python3 python3-pip python3-venv python3-dev pipx"
  [ubjava]="openjdk-21-jdk-headless|openjdk-17-jdk-headless maven"
)
# peso (segundos estimados) usado na barra de progresso; download/disco em MB (aprox., base arm64)
declare -A MOD_W=(  [base]=12 [build]=45 [python]=20 [node]=25 [java]=120 [web]=35 [db]=15 [net]=40 [extras]=40 [term]=12 [dev2]=40 [api]=6 [ubuntu]=200 [ubtools]=30 [ubbuild]=60 [ubpy]=40 [ubnode]=45 [ubjava]=90 [claude]=50 [codex]=30 [gemini]=35 [hush]=1 [theme]=1 [font]=8 [keys]=1 [root]=12 )
declare -A MOD_DL=( [base]=6  [build]=70 [python]=35 [node]=12 [java]=280 [web]=45 [db]=2  [net]=9  [extras]=11 [term]=12 [dev2]=60 [api]=1 [ubuntu]=160 [ubtools]=25 [ubbuild]=80 [ubpy]=40 [ubnode]=30 [ubjava]=180 [claude]=270 [codex]=40 [gemini]=80 [hush]=0 [theme]=0 [font]=2 [keys]=0 [root]=2 )
declare -A MOD_DISK=([base]=25 [build]=400 [python]=150 [node]=60 [java]=460 [web]=170 [db]=6  [net]=44 [extras]=42 [term]=30 [dev2]=200 [api]=1 [ubuntu]=500 [ubtools]=90 [ubbuild]=350 [ubpy]=180 [ubnode]=150 [ubjava]=400 [claude]=300 [codex]=120 [gemini]=250 [hush]=0 [theme]=0 [font]=2 [keys]=0 [root]=5 )

# ---------------------------------------------------------------- arquitetura
# Os tamanhos acima são para arm64. Em 32 bits os pacotes são ~20% menores,
# e Claude Code / Codex não têm versão para 32 bits.
ARCH="$(uname -m 2>/dev/null)"
case "$ARCH" in
  aarch64|arm64|x86_64) ARCH_PCT=100; ARCH_64=1 ;;
  *) ARCH_PCT=80; ARCH_64=0 ;;
esac
arch_ok() { # $1 = módulo -> 0 se funciona nesta arquitetura
  case "$1" in claude|codex) [ "$ARCH_64" = 1 ] ;; *) return 0 ;; esac
}

# ================================================================= FX (efeitos)
fx_sleep() { if [ -n "$FXFD" ]; then read -rt "$1" -u "$FXFD" _ 2>/dev/null; else sleep "$1"; fi; return 0; }
fx_wait() { # espera $1 s lendo o teclado: qualquer tecla pula a intro
  [ "$FX_SKIP" = 1 ] && return 1
  IFS= read -rsn1 -t "$1" _ && { FX_SKIP=1; return 1; }
  return 0
}
fx_tick() { # um quadro de animação; 1 = parar
  if [ "$FX_INTRO" = 1 ]; then fx_wait "$1"; return; fi
  read -t 0 && return 1          # tem tecla esperando: não atrasa quem está navegando
  fx_sleep "$1"
}
fx_flush() { while IFS= read -rsn1 -t 0.01 _; do :; done; }

# texto "decifrado": letras aleatórias que vão travando, da esquerda p/ direita
fx_decrypt() { # linha coluna "texto" cor [quadros]   (linha "@" = posição guardada com ESC 7)
  local r=$1 c=$2 t="$3" col="$4" n=${5:-8} f i s ch len=${#3} pos
  if [ "$r" = @ ]; then pos="${E}8${E}7${E}[${c}G"   # volta e guarda de novo (alguns terminais "gastam")
  else printf '%s7' "$E"; pos="${E}[${r};${c}H"; fi   # guarda o cursor (volta no fim)
  if [ "$ANIM" = 1 ]; then
    for ((f=1; f<n; f++)); do
      s=""
      for ((i=0; i<len; i++)); do
        ch="${t:i:1}"
        if [ "$ch" = " " ] || (( i < len * f / n )); then s+="$ch"; else s+="${GLY[RANDOM%NG]}"; fi
      done
      printf '%s%s%s%s' "$pos" "$col" "$s" "$C_N"
      fx_tick 0.022 || break
    done
  fi
  printf '%s%s%s%s' "$pos" "$col" "$t" "$C_N"
  [ "$r" = @ ] || printf '%s8' "$E"
}

# ---------------------------------------------------------------- logo
# Letras desenhadas em grade 5x5 ('#' = pixel). O logo grande usa █ (5 linhas);
# o pequeno junta 2 linhas de pixels em 1 com meio-blocos ▀ ▄ (3 linhas).
declare -A FONT=(
  [T]="#####|  #  |  #  |  #  |  #  "
  [E]="#####|#    |#### |#    |#####"
  [R]="#### |#   #|#### |#  # |#   #"
  [M]="#   #|## ##|# # #|#   #|#   #"
  [U]="#   #|#   #|#   #|#   #| ### "
  [X]="#   #| # # |  #  | # # |#   #"
)
LOGO_BIG=(); LOGO_SMALL=(); LOGO_S_C=()
logo_line() { # linha "plana" (# u d espaço) -> colorida em degradê por coluna
  local p="$1" c n=${#1} s="" ch
  for ((c=0; c<n; c++)); do
    case "${p:c:1}" in '#') ch="█" ;; u) ch="▀" ;; d) ch="▄" ;; *) s+=" "; continue ;; esac
    s+="${PALC[c*NP/n]}$ch"
  done
  printf '%s' "$s${C_N}"
}
logo_build() {
  local w="TERMUX" i r c a b line rows bm=("" "" "" "" "")
  for ((i=0; i<${#w}; i++)); do
    IFS='|' read -ra rows <<< "${FONT[${w:i:1}]}"
    for r in 0 1 2 3 4; do bm[r]+="${rows[r]}"; if (( i < ${#w} - 1 )); then bm[r]+=" "; fi; done
  done
  LOGO_BIG=("${bm[@]}")
  for r in 0 2 4; do
    line=""
    for ((c=0; c<${#bm[0]}; c++)); do
      a="${bm[r]:c:1}"; b=" "; (( r < 4 )) && b="${bm[r+1]:c:1}"
      case "$a$b" in "##") line+="#" ;; "# ") line+="u" ;; " #") line+="d" ;; *) line+=" " ;; esac
    done
    LOGO_SMALL+=("$line"); LOGO_S_C+=("$(logo_line "$line")")
  done
}
logo_build

fx_logo_final() { # sombra (camada de baixo, deslocada 1↘) + logo (camada de cima)
  local -n LGF=$1; local top=$2 left=$3 H=${#LGF[@]} W=${#LGF[0]} r c buf="" nx
  for ((r=0; r<H+1; r++)); do buf+="${E}[$((top+r));$((left-1))H$(printf '%*s' $((W+3)) '')"; done
  for ((r=0; r<H; r++)); do
    for ((c=0; c<W; c++)); do
      [ "${LGF[r]:c:1}" = " " ] && continue
      nx=" "; (( r + 1 < H )) && nx="${LGF[r+1]:c+1:1}"
      [ "$nx" = " " ] || [ -z "$nx" ] && buf+="${E}[$((top+r+1));$((left+c+1))H${C_DK}░"
    done
  done
  for ((r=0; r<H; r++)); do buf+="${E}[$((top+r));${left}H$(logo_line "${LGF[r]}")"; done
  printf '%s' "$buf"
}

fx_formulas() { # camada de fundo: fórmulas e hexadecimais espalhados
  local i r c buf="" fs=("∇·E = ρ/ε" "e^iπ + 1 = 0" "∑ 1/n² = π²/6" "∫ f(x) dx" "λ = h/p" "0xDEADBEEF"
    "ψ(x,t)" "Δx·Δp ≥ ħ/2" "√2 ≈ 1.4142" "SHA-256" "0x7F454C46" "∞" "Ω" "lim x→0" "E = mc²" "∂ψ/∂t")
  for ((i=0; i<${#fs[@]}; i++)); do
    r=$(( RANDOM % ROWS + 1 )); c=$(( RANDOM % (COLS > 18 ? COLS - 16 : 1) + 1 ))
    buf+="${E}[${r};${c}H${C_DK}${fs[i]}"
  done
  printf '%s' "$buf${C_N}"
}

fx_splash() { # intro: chuva de caracteres -> logo se decifra -> glitch -> onda -> boot
  [ "$ANIM" = 1 ] || return 0
  term_size
  local big=1 H W top left f F=40 c r y k rr pc buf s ch line row sub
  local -a Y LEN LK
  if [ "$COLS" -lt 40 ] || [ "$ROWS" -lt 20 ]; then big=0; fi
  local lgn=LOGO_SMALL; [ $big = 1 ] && lgn=LOGO_BIG
  local -n LG=$lgn
  H=${#LG[@]}; W=${#LG[0]}
  top=$(( ROWS / 2 - H - 3 )); [ $top -lt 2 ] && top=2
  left=$(( (COLS - W) / 2 + 1 )); [ $left -lt 2 ] && left=2
  FX_SKIP=0; FX_INTRO=1; hide_cur; printf '%s[0m%s[2J' "$E" "$E"
  fx_formulas
  for ((c=1; c<=COLS; c+=2)); do Y[c]=$(( -(RANDOM % ROWS) )); LEN[c]=$(( RANDOM % 8 + 6 )); done
  for ((c=0; c<W; c++)); do LK[c]=$(( 16 + c * 22 / W + RANDOM % 4 )); done
  # --- camada 1: chuva | camada 2: logo sendo decifrado por cima
  for ((f=0; f<F; f++)); do
    buf=""
    for ((c=1; c<=COLS; c+=2)); do
      y=${Y[c]}; pc=0
      (( f >= 12 && c >= left - 2 && c <= left + W + 1 )) && pc=1
      for k in 0 1 4 "${LEN[c]}"; do
        rr=$(( y - k )); (( rr < 1 || rr > ROWS )) && continue
        (( pc && rr >= top - 1 && rr <= top + H )) && continue
        case $k in
          0) ch="${C_HEAD}${GLY[RANDOM%NG]}" ;; 1) ch="${GR[4]}${GLY[RANDOM%NG]}" ;;
          4) ch="${GR[1]}${GLY[RANDOM%NG]}" ;; *) ch=" " ;;
        esac
        buf+="${E}[${rr};${c}H$ch"
      done
      Y[c]=$(( y + 1 )); (( y - LEN[c] > ROWS )) && Y[c]=$(( -(RANDOM % 6) ))
    done
    (( f == 12 )) && for ((r=top-1; r<=top+H; r++)); do buf+="${E}[${r};$((left-2))H$(printf '%*s' $((W+4)) '')"; done
    if (( f >= 13 )); then
      for ((r=0; r<H; r++)); do
        line="${LG[r]}"; s=""
        for ((c=0; c<W; c++)); do
          ch="${line:c:1}"
          if [ "$ch" = " " ]; then s+=" "; continue; fi
          if (( f >= LK[c] + (r & 1) )); then
            case "$ch" in '#') ch="█" ;; u) ch="▀" ;; d) ch="▄" ;; esac
            s+="${PALC[c*NP/W]}$ch"
          else s+="${GR[3]}${GLY[RANDOM%NG]}"; fi
        done
        buf+="${E}[$((top+r));${left}H$s"
      done
    fi
    printf '%s' "$buf"
    fx_wait 0.03 || break
  done
  # --- varredura: uma linha de luz desce limpando a chuva (o logo fica)
  if [ "$FX_SKIP" = 0 ]; then
    local scan; scan="$(rep '━' "$COLS")"
    for ((r=1; r<=ROWS; r++)); do
      if (( r >= top - 1 && r <= top + H )); then
        buf="${E}[${r};1H$(printf '%*s' $(( left > 3 ? left - 3 : 0 )) '')${E}[${r};$((left+W+2))H${E}[K"
      else buf="${E}[${r};1H${E}[2K"; fi
      (( r < ROWS )) && ! (( r + 1 >= top - 1 && r + 1 <= top + H )) && buf+="${E}[$((r+1));1H${PALC[r*NP/ROWS]}${scan}${C_N}"
      printf '%s' "$buf"
      fx_wait 0.009 || break
    done
  fi
  # --- glitch: linhas do logo "escorregam" e voltam
  if [ "$FX_SKIP" = 0 ]; then
    fx_logo_final "$lgn" "$top" "$left"
    for k in 1 2 3; do
      r=$(( RANDOM % H ))
      printf '%s[%d;%dH%s%s' "$E" $((top+r)) $((left + (k % 2 ? 2 : -1))) "$C_M" "$(logo_line "${LG[r]}" | sed 's/\x1b\[[0-9;]*m//g')"
      fx_wait 0.06 || break
      fx_logo_final "$lgn" "$top" "$left"; fx_wait 0.09 || break
    done
  fi
  # --- subtítulo + onda senoidal (a fórmula embaixo é a da própria onda)
  sub=$(( top + H + 2 ))
  if [ "$FX_SKIP" = 0 ]; then
    s="S E T U P   //   v$VERSION"
    fx_decrypt "$sub" $(( (COLS - ${#s}) / 2 + 1 )) "$s" "$C_W" 10
    s="ƒ(x,t) = ½[sin(x+t) + sin(3x+2t)]"
    (( sub + 3 <= ROWS )) && printf '%s[%d;%dH%s%s' "$E" $((sub+3)) $(( (COLS - ${#s}) / 2 + 1 )) "$C_DK" "$s"
    for ((f=0; f<22; f++)); do
      (( sub + 2 <= ROWS )) || break
      printf '%s[%d;%dH%s' "$E" $((sub+2)) "$left" "$(wave "$W" "$f")"
      fx_wait 0.035 || break
    done
  fi
  # --- boot log digitado
  row=$(( sub + 5 ))
  if [ "$FX_SKIP" = 0 ]; then
    local -a BL=("kernel ......... android $(getprop ro.build.version.release 2>/dev/null || echo ?)"
                 "arquitetura .... $ARCH"
                 "armazenamento .. $(( $(free_kb) / 1024 )) MB livres"
                 "espelho ........ $(current_mirror | cut -c1-24)"
                 "status ......... ONLINE")
    for ((k=0; k<${#BL[@]}; k++)); do
      (( row + k < ROWS )) || break
      s="${BL[k]:0:$((COLS - 12))}"
      printf '%s[%d;%dH%s0x%02X %s›%s ' "$E" $((row+k)) "$left" "$C_DK" $(( k * 31 + 16 )) "${PALC[k*2]}" "$C_N"
      for ((c=0; c<${#s}; c+=3)); do printf '%s%s' "$C_DIM3" "${s:c:3}"; fx_wait 0.008 || break 2; done
      [ $k = 4 ] && printf ' %s✓%s' "$C_G" "$C_N"
    done
    s="▸ ACESSO LIBERADO"
    (( row + 6 <= ROWS )) && fx_decrypt $((row+6)) $(( (COLS - ${#s}) / 2 + 1 )) "$s" "$C_G" 12
    fx_wait 0.6
  fi
  FX_INTRO=0; fx_flush; printf '%s' "$C_N"
}

fx_outro() { # desliga como TV de tubo: a imagem se fecha numa linha, depois num ponto
  [ "$ANIM" = 1 ] || return 0
  local top=1 bot=$ROWS mid w c
  hide_cur
  while (( top < bot )); do
    printf '%s[%d;1H%s[2K%s[%d;1H%s[2K' "$E" $top "$E" "$E" $bot "$E"
    top=$((top + 1)); bot=$((bot - 1))
    if (( top < bot )); then
      printf '%s[%d;1H%s%s%s[%d;1H%s%s' "$E" $top "${PALC[0]}" "$(rep '▔' "$COLS")" "$E" $bot "$(rep '▁' "$COLS")" "$C_N"
    fi
    fx_sleep 0.01
  done
  mid=$(( (ROWS + 1) / 2 )); w=$COLS
  while (( w > 1 )); do
    c=$(( (COLS - w) / 2 + 1 ))
    printf '%s[%d;1H%s[2K%s[%d;%dH%s%s%s' "$E" $mid "$E" "$E" $mid $c "$C_HEAD" "$(rep '━' $w)" "$C_N"
    w=$(( w * 3 / 5 )); fx_sleep 0.025
  done
  c=$(( COLS / 2 ))
  printf '%s[%d;1H%s[2K%s[%d;%dH%s●%s' "$E" $mid "$E" "$E" $mid $c "$C_HEAD" "$C_N"; fx_sleep 0.18
  printf '%s[%d;%dH%s•%s' "$E" $mid $c "${PALC[0]}" "$C_N"; fx_sleep 0.15
  printf '%s[%d;%dH%s·%s' "$E" $mid $c "$C_DK" "$C_N"; fx_sleep 0.15
  printf '%s[2J%s[H' "$E" "$E"
}

# ================================================================= UI
bar() { # bar PCT LARGURA [quadro] -> degradê + um brilho que corre pela parte cheia
  local pct=$1 w=${2:-10} fr=${3:-0} f i s="" sh=-1
  f=$(( pct * w / 100 ))
  [ "$ANIM" = 1 ] && [ $f -gt 2 ] && sh=$(( fr % (f + 8) ))
  for ((i=0; i<w; i++)); do
    if (( i < f )); then
      if (( i == sh )); then s+="${C_HEAD}█"; else s+="${PALC[i*NP/w]}█"; fi
    elif (( i == f && pct < 100 )); then s+="${PALC[NP-1]}▓"
    else s+="${C_DK}░"; fi
  done
  printf '%s' "$s${C_N}"
}
wave() { # wave LARGURA QUADRO -> soma de duas senóides em ▁▂▃▄▅▆▇█
  local w=$1 fr=$2 i v s=""
  for ((i=0; i<w; i++)); do
    v=$(( (SINE[(i + fr) % 16] + SINE[(i * 3 + fr * 2) % 16]) / 2 ))
    s+="${PALC[i*NP/w]}${SPARK[v]}"
  done
  printf '%s' "$s${C_N}"
}
fmt_t() { printf '%02d:%02d' $(( $1 / 60 )) $(( $1 % 60 )); }
free_kb() {
  local v; v="$(df -Pk "$HOME" 2>/dev/null | tail -n1 | tr -s ' ' | cut -d' ' -f4)"
  [[ $v =~ ^[0-9]+$ ]] || v=0; echo "$v"
}
banner() { # $1 = título da tela
  local t="${1^^}" n i s=""
  n=$(( RULEW - ${#t} - 4 )); [ $n -lt 2 ] && n=2
  for ((i=0; i<n; i++)); do s+="${PALC[(NP - 1) - i * (NP - 1) / n]}━"; done
  L " ${GRAD_NAME}${C_DK}//${C_N}${C_W}SETUP${C_N} ${C_DK}v$VERSION${C_N}"
  L " ${PALC[0]}━━${C_N} ${C_W}${C_B}${t}${C_N} $s${C_N}"
}
current_mirror() {
  grep -hE '^(deb |URIs:)' "$PREFIX/etc/apt/sources.list" \
    "$PREFIX"/etc/apt/sources.list.d/*.sources 2>/dev/null | head -n1 \
    | sed -E 's/^(deb|URIs:) +//; s/ .*//; s#^[a-z]+://##; s#/.*##'
}
pause() { show_cur; echo; read -r -p " ${C_DK}⏎${C_N} ${C_DIM3}voltar${C_N} " _ || true; }

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
  local title="$1" cur="${MENU_START:-0}" n=${#MENU_ITEMS[@]} h first i k lbl prev=-1 fresh=1
  local pre=${#MENU_PRE[@]} mw
  MENU_IDX=-1
  term_size; printf '\033[2J'; hide_cur
  mw=$(( COLS - 7 )); [ $mw -lt 8 ] && mw=8
  while true; do
    h=$(( ROWS - 6 - pre )); [ $h -lt 3 ] && h=3; [ $h -gt $n ] && h=$n
    first=$(( cur - h / 2 )); [ $first -lt 0 ] && first=0
    [ $(( first + h )) -gt $n ] && first=$(( n - h ))
    printf '\033[H'
    [ "$fresh" = 1 ] && [ "$ANIM" = 1 ] && FX_REVEAL=1   # 1ª vez: a tela "desce" linha a linha
    banner "$title"
    for i in "${!MENU_PRE[@]}"; do L " ${MENU_PRE[i]}"; done
    L ""
    for ((i=first; i<first+h; i++)); do
      k="${MENU_KEYS[i]:-·}"; lbl="${MENU_ITEMS[i]:0:$mw}"
      if [ "$i" -eq "$cur" ]; then printf '%s7' "$E"; L "${BG_SEL}${C_ACC}▌ ${k}  ${C_W}${C_B}${lbl}${C_N}${BG_SEL}"
      else L "  ${C_DK}${k}${C_N}  ${C_TX}${lbl}${C_N}"; fi
    done
    L ""
    L " ${C_DK}↑↓ navegar · ⏎ ok · q sair${C_N}   ${PALC[0]}$(printf '%02d' $((cur+1)))${C_DK}/$(printf '%02d' "$n")${C_N}"
    printf '\033[J'
    FX_REVEAL=0; fresh=0
    if [ "$ANIM" = 1 ] && [ "$cur" != "$prev" ]; then   # item escolhido se "decifra"
      fx_decrypt @ 6 "${MENU_ITEMS[cur]:0:$mw}" "${BG_SEL}${C_W}${C_B}" 5
    fi
    prev=$cur
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
  local ids=("$@") cur=0 n=$# h first i m mark fresh=1 nsel lbl
  term_size; printf '\033[2J'; hide_cur
  while true; do
    h=$(( ROWS - 6 )); [ $h -lt 3 ] && h=3; [ $h -gt $n ] && h=$n
    first=$(( cur - h / 2 )); [ $first -lt 0 ] && first=0
    [ $(( first + h )) -gt $n ] && first=$(( n - h ))
    printf '\033[H'
    [ "$fresh" = 1 ] && [ "$ANIM" = 1 ] && FX_REVEAL=1
    banner "${CHECK_TITLE:-Personalizada}"
    nsel=0; for m in "${ids[@]}"; do [ "${SEL[$m]:-0}" = 1 ] && nsel=$((nsel+1)); done
    L " ${C_DK}espaço marca · a todos · n nenhum${C_N}  ${PALC[0]}◉ ${nsel}${C_DK}/${n}${C_N}"
    for ((i=first; i<first+h; i++)); do
      m="${ids[i]}"
      if ! arch_ok "$m"; then mark="${C_DK}⊘"
      elif [ "${SEL[$m]:-0}" = 1 ]; then mark="${C_G}◉"; else mark="${C_DK}○"; fi
      if arch_ok "$m"; then
        lbl="${MOD_NAME[$m]:0:$((COLS-13))} ${C_DK}~$(( MOD_DL[$m] * ARCH_PCT / 100 ))MB"
      else lbl="${C_DK}${MOD_NAME[$m]:0:$((COLS-28))} (sem suporte a $ARCH)"; fi
      if [ "$i" -eq "$cur" ]; then L "${BG_SEL}${C_ACC}▌${C_N}${BG_SEL} ${mark}${C_N}${BG_SEL} ${C_W}${C_B}${lbl}${C_N}${BG_SEL}"
      else L "  ${mark}${C_N} ${C_TX}${lbl}${C_N}"; fi
    done
    L " ${C_DK}⏎ continuar · q voltar${C_N}"
    printf '\033[J'
    FX_REVEAL=0; fresh=0
    read_key
    case "$KEY" in
      up|k)   cur=$(( (cur - 1 + n) % n )) ;;
      down|j) cur=$(( (cur + 1) % n )) ;;
      space)  m="${ids[cur]}"; arch_ok "$m" || continue
              if [ "${SEL[$m]:-0}" = 1 ]; then SEL[$m]=0; else SEL[$m]=1; fi ;;
      a|A)    for m in "${ids[@]}"; do arch_ok "$m" && SEL[$m]=1; done ;;
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
  : > "$LOG"; : > "$WORK/unavailable"; : > "$WORK/failed_pkgs"; : > "$WORK/tried"; : > "$WORK/warnings"
  rm -f "$WORK/mirror_used"
}

# ================================================================= PASSOS
apt_i() { apt-get install "${APT_OPTS[@]}" "$@"; }
warn() { echo "[aviso] $*"; echo "$*" >> "$WORK/warnings"; } # vai para o resumo final

# baixa um script de instalação para um arquivo, confere e só depois executa.
# (no lugar de "curl | bash": se a rede cair no meio, nada meio-baixado é executado)
# uso: fetch_check URL ARQUIVO
fetch_check() {
  curl -fsSL --retry 2 --max-time 120 -o "$2" "$1" || { echo "Download falhou: $1"; return 1; }
  [ -s "$2" ] || { echo "Arquivo vazio: $1"; return 1; }
  head -n1 "$2" | grep -q '^#!' || { echo "Não parece um script (rede bloqueada?): $1"; return 1; }
}

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
  if [ ! -d "$bk" ]; then # só na 1ª troca da execução (não sobrescreve o original)
    mkdir -p "$bk"
    cp -a "$PREFIX/etc/apt/sources.list" "$bk/" 2>/dev/null
    cp -a "$PREFIX/etc/apt/sources.list.d" "$bk/" 2>/dev/null
  fi
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
ub_has()  { proot-distro login "$UB" -- true >/dev/null 2>&1; }
ub_root() { proot-distro login "$UB" -- bash -lc "$1"; }
ub_dev()  { proot-distro login "$UB" --user "$UB_USER" -- bash -lc "$1"; }

step_proot_pkgs() { apt_i proot proot-distro; }

ub_ver() { # versão do Ubuntu instalado (ex.: "Ubuntu 26.04.1 LTS")
  proot-distro login "$UB" -- grep PRETTY_NAME /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"'
}

step_ubuntu_install() {
  command -v proot-distro >/dev/null 2>&1 || { echo "proot-distro ausente"; return 1; }
  if ub_has; then echo "Ubuntu já instalado: $(ub_ver). Mantido (não é substituído)."; return 0; fi
  # tenta as versões na ordem de UBUNTU_VERSIONS; avisa se não foi a preferida
  local v want="${UBUNTU_VERSIONS%% *}"
  for v in $UBUNTU_VERSIONS; do
    if proot-distro install "$UB:$v"; then
      [ "$v" != "$want" ] && warn "Ubuntu $want indisponível no proot-distro; instalado o $v."
      echo "Instalado: $(ub_ver)"; return 0
    fi
  done
  if proot-distro install "$UB"; then
    warn "Nenhuma versão de [$UBUNTU_VERSIONS] disponível; instalada a padrão: $(ub_ver)."
    return 0
  fi
  return 1
}

step_ubuntu_base() {
  ub_has || { echo "Ubuntu não encontrado"; return 1; }
  ub_root "U=$(printf '%q' "$UB_USER")"'
set -e
export DEBIAN_FRONTEND=noninteractive
echo "APT::Sandbox::User \"root\";" > /etc/apt/apt.conf.d/99proot
apt-get update
apt-get -y upgrade
apt-get install -y --no-install-recommends curl git ca-certificates gnupg bash ripgrep unzip xz-utils less nano locales
id "$U" >/dev/null 2>&1 || useradd -m -s /bin/bash "$U"
mkdir -p "/home/$U/projetos"
chown "$U:$U" "/home/$U/projetos"
touch "/home/$U/.profile"
grep -q "termux-setup" "/home/$U/.profile" || cat >> "/home/$U/.profile" <<EOP
# termux-setup
export PATH="\$HOME/.local/bin:\$HOME/.npm-global/bin:\$PATH"
EOP
chown "$U:$U" "/home/$U/.profile"
'
}

step_ubuntu_node() {
  ub_has || { echo "Ubuntu não encontrado"; return 1; }
  ub_root "URL=$(printf '%q' "$NODESOURCE_URL"); $(declare -f fetch_check)"'
set -e
export DEBIAN_FRONTEND=noninteractive
if command -v node >/dev/null 2>&1 && [ "$(node -v | cut -d. -f1 | tr -d v)" -ge 20 ]; then
  echo "node ok: $(node -v)"; exit 0
fi
f="$(mktemp)"
fetch_check "$URL" "$f"
bash "$f"; rm -f "$f"
apt-get install -y nodejs
node -v; npm -v
'
}

step_claude() {
  ub_has || { echo "Ubuntu não encontrado"; return 1; }
  # instalador oficial da Anthropic (Linux ARM64, dentro do Ubuntu)
  ub_dev "URL=$(printf '%q' "$CLAUDE_URL"); $(declare -f fetch_check)"'
set -e
f="$(mktemp)"
fetch_check "$URL" "$f"
bash "$f"; rm -f "$f"
export PATH="$HOME/.local/bin:$PATH"
claude --version'
}

step_ub_npm() { # $1 = pacote npm
  ub_has || { echo "Ubuntu não encontrado"; return 1; }
  ub_dev "set -e; mkdir -p ~/.npm-global; npm config set prefix ~/.npm-global; export PATH=\"\$HOME/.npm-global/bin:\$PATH\"; npm install -g $(printf '%q' "$1")"
}

step_ub_mod() { # $1 = módulo de UB_PKGS: instala no Ubuntu o que existir
  ub_has || { echo "Ubuntu não encontrado"; return 1; }
  ub_root "P=$(printf '%q' "${UB_PKGS[$1]}")"'
export DEBIAN_FRONTEND=noninteractive
# só baixa as listas se tiverem mais de 1 hora
[ -n "$(find /var/lib/apt/lists -maxdepth 1 -name "*Packages*" -mmin -60 2>/dev/null | head -n1)" ] || apt-get update
list=""; miss=""
for tok in $P; do
  ok=""; IFS="|"; for alt in $tok; do apt-cache show "$alt" >/dev/null 2>&1 && { ok=$alt; break; }; done; unset IFS
  if [ -n "$ok" ]; then list="$list $ok"; else miss="$miss $tok"; fi
done
[ -n "$miss" ] && echo "[aviso] indisponível no Ubuntu:$miss"
[ -z "$list" ] && exit 0
echo "Instalando no Ubuntu:$list"
apt-get install -y --no-install-recommends $list'
}

step_ub_pyvenv() { # venv pronto para o usuário (o Ubuntu novo bloqueia "pip install" global)
  ub_has || { echo "Ubuntu não encontrado"; return 1; }
  ub_dev 'set -e
[ -d ~/venv ] || python3 -m venv ~/venv
~/venv/bin/pip install -q --upgrade pip
grep -q "alias venv=" ~/.bashrc 2>/dev/null || echo "alias venv=\"source ~/venv/bin/activate\"" >> ~/.bashrc
pipx ensurepath >/dev/null 2>&1 || true
echo "venv: ~/venv ($(~/venv/bin/python --version)) · ative com: venv"'
}

step_ub_upgrade() { # atualização completa do Ubuntu, com reparo e limpeza
  ub_has || { echo "Ubuntu não encontrado"; return 1; }
  ub_root 'export DEBIAN_FRONTEND=noninteractive
O="-y -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold"
dpkg --configure -a || true
apt-get update || exit 1
apt-get $O --fix-broken install || true
apt-get $O full-upgrade || exit 1
apt-get -y autoremove --purge
apt-get clean
echo "Ubuntu atualizado: $(grep PRETTY_NAME /etc/os-release | cut -d= -f2 | tr -d \")"'
}

step_ia_update() {
  ub_has || { echo "Ubuntu não encontrado"; return 1; }
  ub_dev 'export PATH="$HOME/.local/bin:$HOME/.npm-global/bin:$PATH"
command -v claude >/dev/null 2>&1 && claude update
command -v codex  >/dev/null 2>&1 && npm install -g @openai/codex@latest
command -v gemini >/dev/null 2>&1 && npm install -g @google/gemini-cli@latest
[ -x ~/venv/bin/pip ] && ~/venv/bin/pip install -q --upgrade pip
command -v pipx >/dev/null 2>&1 && pipx upgrade-all
true'
}

step_wrappers() {
  mkdir -p "$HOME/bin" "$HOME/projetos"
  cat > "$HOME/bin/ia-run" <<'EOS'
#!/data/data/com.termux/files/usr/bin/bash
# uso: ia-run <comando> [args] — roda no Ubuntu (usuário __UB_USER__) na pasta atual do Termux.
# A pasta atual do Termux é "emprestada" ao Ubuntu e aparece lá como /work/<nome-da-pasta>:
# os arquivos são os MESMOS (o que a IA editar lá, muda aqui).
HOST_DIR="$PWD"
NAME="$(basename "$HOST_DIR")"
case "$NAME" in ""|"/") NAME="raiz" ;; esac
NAME="${NAME// /_}"
DEST="/work/$NAME"
command -v termux-wake-lock >/dev/null 2>&1 && termux-wake-lock 2>/dev/null
proot-distro login __UB__ --user __UB_USER__ --bind "$HOST_DIR:$DEST" --work-dir "$DEST" \
  -- bash -lc 'exec "$@"' _ "$@"
rc=$?
command -v termux-wake-unlock >/dev/null 2>&1 && termux-wake-unlock 2>/dev/null
exit $rc
EOS
  for c in claude codex gemini; do
    printf '#!/data/data/com.termux/files/usr/bin/bash\nexec "$HOME/bin/ia-run" %s "$@"\n' "$c" > "$HOME/bin/$c"
  done
  printf '#!/data/data/com.termux/files/usr/bin/bash\nexec "$HOME/bin/ia-run" bash -l\n' > "$HOME/bin/ubuntu"
  printf '#!/data/data/com.termux/files/usr/bin/bash\nexec proot-distro login %s\n' "$UB" > "$HOME/bin/ubuntu-root"
  cat > "$HOME/bin/ia-update" <<'EOS'
#!/data/data/com.termux/files/usr/bin/bash
proot-distro login __UB__ -- bash -lc 'export DEBIAN_FRONTEND=noninteractive; O="-y -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold"; apt-get update && apt-get $O full-upgrade && apt-get -y autoremove --purge && apt-get clean'
"$HOME/bin/ia-run" bash -lc 'command -v claude >/dev/null && claude update; command -v codex >/dev/null && npm install -g @openai/codex@latest; command -v gemini >/dev/null && npm install -g @google/gemini-cli@latest; true'
EOS
  sed -i "s/__UB_USER__/$UB_USER/g; s/__UB__/$UB/g" "$HOME/bin/ia-run" "$HOME/bin/ia-update"
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
echo "✓ Tudo atualizado."
EOS
  cat > "$HOME/bin/limpar" <<'EOS'
#!/data/data/com.termux/files/usr/bin/bash
apt-get autoremove -y && apt-get clean
echo "✓ Cache limpo."
EOS
  cat > "$HOME/bin/info" <<'EOS'
#!/data/data/com.termux/files/usr/bin/bash
echo "== Termux =="
echo "Arq: $(uname -m) | Android $(getprop ro.build.version.release) | $(getprop ro.product.model)"
echo "Espelho: $(grep -hE '^(deb |URIs:)' "$PREFIX"/etc/apt/sources.list "$PREFIX"/etc/apt/sources.list.d/*.sources 2>/dev/null | head -n1)"
df -h "$HOME" | tail -n1
for c in python node java clang git; do
  command -v "$c" >/dev/null 2>&1 || continue
  if [ "$c" = java ]; then v="$(java -version 2>&1 | head -n1)"; else v="$($c --version 2>&1 | head -n1)"; fi
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
echo "✓ Projeto criado em $(pwd)"
EOS
  [ -f "$SELF" ] && cp "$SELF" "$HOME/bin/termux-setup"
  chmod +x "$HOME"/bin/* 2>/dev/null
  touch "$HOME/.bashrc"
  sed -i '/# >>> termux-setup >>>/,/# <<< termux-setup <<</d' "$HOME/.bashrc"
  cat >> "$HOME/.bashrc" <<'EOS'
# >>> termux-setup >>>
# PATH só recebe ~/bin uma vez, mesmo com vários "source ~/.bashrc"
case ":$PATH:" in *":$HOME/bin:"*) ;; *) export PATH="$HOME/bin:$PATH" ;; esac
alias ..='cd ..'
alias gs='git status'
alias gp='git pull'
alias p='cd ~/projetos'
alias venv='source ~/venv/bin/activate'
if command -v eza >/dev/null 2>&1; then alias ll='eza -lah --git'; else alias ll='ls -lah --color=auto'; fi
# zoxide ("z pasta") só é ativado uma vez por sessão
if [ -z "${__TSETUP_Z:-}" ] && command -v zoxide >/dev/null 2>&1; then eval "$(zoxide init bash)"; __TSETUP_Z=1; fi
# <<< termux-setup <<<
EOS
  echo "Comandos criados: atualizar limpar info novo termux-setup (projetos em ~/projetos)"
}

# ---------------------------------------------------------------- Aparência
backup_termux() { # guarda uma cópia de ~/.termux antes de mudar qualquer coisa
  local bk="$LOGDIR/backup-termux-$TS"
  [ -d "$bk" ] && return 0          # já feito nesta execução: não sobrescreve o original
  mkdir -p "$bk"
  if [ -d "$HOME/.termux" ]; then cp -a "$HOME/.termux/." "$bk/" 2>/dev/null
  else touch "$bk/.vazio"; fi       # ~/.termux não existia (o --desfazer apaga o que foi criado)
  return 0
}

step_hush() {
  touch "$HOME/.hushlogin" && echo "Mensagem de boas-vindas silenciada (~/.hushlogin)"
}

step_theme() {
  backup_termux; mkdir -p "$HOME/.termux"
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
  local d="$WORK/font" f sum
  command -v curl  >/dev/null 2>&1 || { echo "curl ausente"; return 1; }
  command -v unzip >/dev/null 2>&1 || { echo "unzip ausente"; return 1; }
  rm -rf "$d"; mkdir -p "$d"
  curl -fL --retry 2 --max-time 180 -o "$d/jb.zip" "$FONT_URL" || { echo "Download da fonte falhou: $FONT_URL"; return 1; }
  # confere se o arquivo é exatamente o oficial (não corrompido nem trocado)
  if [ -n "$FONT_SHA256" ] && command -v sha256sum >/dev/null 2>&1; then
    sum="$(sha256sum "$d/jb.zip" | cut -d' ' -f1)"
    [ "$sum" = "$FONT_SHA256" ] || { echo "SHA256 da fonte não confere (esperado $FONT_SHA256, veio $sum)"; return 1; }
    echo "SHA256 ok"
  fi
  unzip -o -q "$d/jb.zip" -d "$d/x" || { echo "unzip falhou"; return 1; }
  f="$(find "$d/x" -name 'JetBrainsMono-Regular.ttf' 2>/dev/null | head -n1)"
  [ -n "$f" ] || { echo "JetBrainsMono-Regular.ttf não encontrado no zip"; return 1; }
  backup_termux; mkdir -p "$HOME/.termux"
  cp "$f" "$HOME/.termux/font.ttf" && echo "Fonte JetBrains Mono $FONT_VER instalada"
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
  backup_termux; mkdir -p "$HOME/.termux"
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
  L " $(bar "$pct" "$bw" "$i") ${C_W}${C_B}${pct}%${C_N}"
  L " $(wave "$bw" "$i") ${C_DK}λ${C_N}"
  L " ${C_DIM3}passo ${C_W}${shown}${C_DK}/${TOTAL}${C_DIM3} · $(fmt_t "$el")${eta:+ · ~$eta restam}${C_N}"
  L ""
  lbl="${PLAN_LABEL[DONE]:-}"
  L " ${PALC[i%NP]}${SPIN[i%10]}${C_N} ${C_W}${C_B}${lbl:0:$maxl}${C_N}"
  L "   ${C_DK}0x$(printf '%04X' $(( (i * 40503) & 0xFFFF ))) ›${C_N} ${C_DIM3}${LIVE:0:$((COLS-14))}${C_N}"
  L ""
  h=$(( ROWS - 11 )); [ $h -lt 3 ] && h=3; [ $h -gt 16 ] && h=16; [ $h -gt $n ] && h=$n
  first=$(( DONE - h / 2 )); [ $first -lt 0 ] && first=0
  [ $(( first + h )) -gt $n ] && first=$(( n - h ))
  for ((j=first; j<first+h; j++)); do
    lbl="${PLAN_LABEL[j]:0:$((COLS-14))}"
    if [ "$j" -lt "$DONE" ]; then
      t="${RES_TIME[j]:-}"
      if [ "${RES_STAT[j]:-}" = OK ]; then L " ${C_G}✓${C_N} ${C_DIM3}${lbl}${C_N} ${C_DK}${t}${C_N}"
      else L " ${C_R}✗${C_N} ${C_TX}${lbl}${C_N} ${C_DK}${t}${C_N}"; fi
    elif [ "$j" -eq "$DONE" ]; then L "${BG_SEL}${C_ACC}▌${PALC[(i/2)%NP]}▶${C_N}${BG_SEL} ${C_W}${C_B}${lbl}${C_N}${BG_SEL} ${C_DIM3}$(( SECONDS - STEP_T0 ))s"
    else L " ${C_DK}· ${lbl}${C_N}"; fi
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
    if [ -s "$WORK/warnings" ]; then
      echo; echo "--- Avisos ---"; sort -u "$WORK/warnings"
    fi
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
  local i col d bw icon
  term_size; printf '\033[2J\033[H'
  banner "Resultado"
  case "$STATUS_TXT" in
    COMPLETO) col="$C_G"; icon="◉" ;;
    INCOMPLETO) col="$C_R"; icon="■" ;;
    *) col="$C_Y"; icon="▲" ;;
  esac
  bw=$(( ${#STATUS_TXT} + 10 ))
  L " ${col}╭$(rep '─' $bw)╮${C_N}"
  L " ${col}│${C_N}  ${col}${icon}${C_N}  $(printf '%*s' ${#STATUS_TXT} '')     ${col}│${C_N}"
  L " ${col}╰$(rep '─' $bw)╯${C_N}"
  L " ${C_G}✓ ${N_OK} ok${C_N}   ${C_R}✗ ${N_FAIL} falhas${C_N}   ${C_Y}· ${N_PEND} pendentes${C_N}"
  L " ${C_DIM3}tempo ${C_W}$(fmt_t "$ELAPSED")${C_DIM3} · +${USED_MB} MB · livre $(( $(free_kb) / 1024 )) MB${C_N}"
  L " ${C_D}espelho: $(current_mirror)${C_N}"
  for ((i=0; i<TOTAL; i++)); do
    case "${RES_STAT[i]}" in
      FALHOU) L " ${C_R}✗${C_N} ${PLAN_LABEL[i]:0:$((COLS-6))}" ;;
      INTERROMPIDO) L " ${C_Y}■${C_N} ${PLAN_LABEL[i]:0:$((COLS-6))} (interrompido)" ;;
    esac
  done
  if [ -s "$WORK/warnings" ]; then
    L ""; while IFS= read -r line; do L " ${C_Y}! ${line:0:$((COLS-5))}${C_N}"; done < <(sort -u "$WORK/warnings")
  fi
  d="$(diagnose)"
  if [ -n "$d" ]; then L ""; while IFS= read -r line; do L " ${C_Y}${line:0:$((COLS-3))}${C_N}"; done <<< "$d"; fi
  L ""
  if [ "$STATUS_TXT" = "INCOMPLETO" ]; then
    L " ${C_B}O que fazer${C_N}"
    L "  ${C_D}Rode o script de novo: ele pula o que${C_N}"
    L "  ${C_D}já está instalado e continua.${C_N}"
  else
    L " ${C_B}Próximos passos${C_N}"
    L "  ${C_CY}source ~/.bashrc${C_N} ${C_D}(ou reabra o Termux)${C_N}"
    [ "${SEL[claude]:-0}" = 1 ] && L "  ${C_CY}claude${C_N}  ${C_D}login pelo navegador${C_N}"
    [ "${SEL[codex]:-0}" = 1 ]  && L "  ${C_CY}codex${C_N}   ${C_D}Sign in with ChatGPT${C_N}"
    [ "${SEL[gemini]:-0}" = 1 ] && L "  ${C_CY}gemini${C_N}  ${C_D}login Google ou GEMINI_API_KEY${C_N}"
    [ "${SEL[ubuntu]:-0}${SEL[claude]:-0}${SEL[codex]:-0}${SEL[gemini]:-0}" != "0000" ] && \
      L "  ${C_CY}ubuntu${C_N}  ${C_D}shell Ubuntu na pasta atual${C_N}"
    { [ "${SEL[keys]:-0}" = 1 ] || [ "${SEL[theme]:-0}" = 1 ] || [ "${SEL[font]:-0}" = 1 ]; } && \
      L "  ${C_D}tema/fonte não mudou? reabra o Termux${C_N}"
    [ "${SEL[dev2]:-0}" = 1 ] && L "  ${C_CY}GitHub${C_N} ${C_D}menu 6: entrar com token${C_N}"
    L "  ${C_CY}termux-setup${C_N} ${C_D}reabre este menu${C_N}"
  fi
  L ""
  L " ${C_D}log: ${SUMMARY/#$HOME/~}${C_N}"
  [ -d "$HOME/storage/downloads" ] && L " ${C_D}(cópia em Downloads)${C_N}"
  printf '\033[J'
  # o status "se decifra" dentro da caixa (linha 4, depois de "│  ◉  ")
  [ "$INTERRUPTED" = 1 ] || fx_decrypt 4 8 "$STATUS_TXT" "${col}${C_B}" 14
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
    finish; INSTALL_STARTED=0; [ "$AUTO" = 1 ] || pause
  else
    show_cur
  fi
}

# ================================================================= PLANOS
clear_sel()  { local m; for m in "${MOD_ORDER[@]}"; do SEL[$m]=0; done; }
select_all() { local m; for m in "${MOD_ORDER[@]}"; do SEL[$m]=1; done; SEL[root]=0; }
select_dev() { local m; clear_sel; for m in base build python node extras term dev2 api hush theme font keys; do SEL[$m]=1; done; }
select_ia()  { local m; clear_sel; for m in base api ubuntu ubtools claude codex gemini; do SEL[$m]=1; done; }
select_ub_all() { local m; clear_sel; for m in base api ubuntu $UB_MODS claude codex gemini; do SEL[$m]=1; done; }

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
  [ "$AUTO" = 1 ] && { KEYS_PRESET=1; return 0; } # --sim: sem perguntas
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
  for m in "${MOD_ORDER[@]}"; do arch_ok "$m" || SEL[$m]=0; done # tira o que não roda nesta arquitetura
  add_step "Testar espelhos" step_mirrors "" 4
  add_step "Aplicar melhor espelho" step_apply_mirror "" 1
  add_step "Atualizar listas (apt)" step_update "" 5
  add_step "Atualizar pacotes" step_upgrade "" 35
  if [ "$1" = full ]; then
    for m in "${MOD_ORDER[@]}"; do
      [ "${SEL[$m]:-0}" = 1 ] || continue
      case "$m" in
        ubuntu|ub*|claude|codex|gemini|hush|theme|font|keys|root) ;;
        *) add_step "${MOD_NAME[$m]}" install_mod "$m" "${MOD_W[$m]}" ;;
      esac
    done
    [ "${SEL[python]:-0}" = 1 ] && add_step "Python: venv + libs" step_pyvenv "" 25
    [ "${SEL[node]:-0}" = 1 ]   && add_step "Node: yarn + typescript" step_npm "" 15
    for m in ubuntu $UB_MODS claude codex gemini; do [ "${SEL[$m]:-0}" = 1 ] && need_ub=1; done
    for m in ubnode codex gemini; do [ "${SEL[$m]:-0}" = 1 ] && need_node=1; done
    if [ $need_ub = 1 ]; then
      SEL[ubuntu]=1
      add_step "Ubuntu: proot-distro" step_proot_pkgs "" 10
      add_step "Ubuntu ${UBUNTU_VERSIONS%% *} LTS: instalar" step_ubuntu_install "" 90
      add_step "Ubuntu: base + usuário $UB_USER" step_ubuntu_base "" 60
      for m in ubtools ubbuild ubpy ubjava; do
        [ "${SEL[$m]:-0}" = 1 ] && add_step "${MOD_NAME[$m]}" step_ub_mod "$m" "${MOD_W[$m]}"
      done
      [ "${SEL[ubpy]:-0}" = 1 ] && add_step "Ubuntu: venv do Python" step_ub_pyvenv "" 15
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
  [ "${SEL[ubnode]:-0}" = 1 ] && need_node=0   # já contado no módulo
  [ $need_node = 1 ] && { EST_DL=$(( EST_DL + 30 )); EST_DISK=$(( EST_DISK + 150 )); }
  EST_DL=$(( EST_DL * ARCH_PCT / 100 )); EST_DISK=$(( EST_DISK * ARCH_PCT / 100 ))
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
  [ "$ARCH_64" = 0 ] && MENU_PRE+=("${C_Y}$ARCH (32 bits): Claude Code e Codex foram tirados.${C_N}")
  [ "$AUTO" = 1 ] && return 0 # --sim: confirma sozinho
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
  if [ "${RES_STAT[1]:-}" = OK ]; then L " ${C_G}✓${C_N} Espelho aplicado: ${url#*://}"
  else L " ${C_R}✗${C_N} apt update falhou (veja o log)"; fi
  L " ${C_D}log: ${LOG/#$HOME/~}${C_N}"
  pause
}

verify_install() {
  local c v out st name rest
  term_size; printf '\033[2J\033[H'; banner "Verificar instalação"
  L " ${C_B}Termux${C_N}"
  for c in git python pip node npm java clang cmake ruby tmux jq rg gh nvim eza; do
    if command -v "$c" >/dev/null 2>&1; then
      case "$c" in
        java) v="$(timeout 8 java -version 2>&1 | head -n1)" ;;
        *)    v="$(timeout 5 "$c" --version 2>&1 | head -n1)" ;;
      esac
      L " ${C_G}✓${C_N} $(printf '%-8s' "$c") ${C_D}${v:0:$((COLS-14))}${C_N}"
    else
      L " ${C_D}· $(printf '%-8s' "$c") não instalado${C_N}"
    fi
  done
  if command -v proot-distro >/dev/null 2>&1 && ub_has; then
    L ""; L " ${C_B}Ubuntu (proot)${C_N}"
    out="$(ub_dev 'export PATH="$HOME/.local/bin:$HOME/.npm-global/bin:$PATH"; for c in python3 node npm java gcc cmake git rg claude codex gemini; do if command -v $c >/dev/null 2>&1; then echo "OK $c $($c --version 2>&1 | head -n1)"; else echo "NO $c"; fi; done' 2>/dev/null)"
    while read -r st name rest; do
      [ -n "$name" ] || continue
      if [ "$st" = OK ]; then L " ${C_G}✓${C_N} $(printf '%-8s' "$name") ${C_D}${rest:0:$((COLS-14))}${C_N}"
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

# ---------------------------------------------------------------- desfazer
oldest() { ls -dtr "$LOGDIR"/$1-* 2>/dev/null | head -n1; } # backup mais antigo = estado original

step_undo_apt() {
  local bk; bk="$(oldest backup-apt)"
  [ -n "$bk" ] || { echo "Nenhum backup do espelho (nunca foi trocado)."; return 0; }
  echo "Restaurando espelho de: $bk"
  [ -f "$bk/sources.list" ] && cp -a "$bk/sources.list" "$PREFIX/etc/apt/sources.list"
  [ -d "$bk/sources.list.d" ] && cp -a "$bk/sources.list.d/." "$PREFIX/etc/apt/sources.list.d/"
  apt-get update "${APT_OPTS[@]}" || true
}

step_undo_look() {
  local bk f="$HOME/.termux/termux.properties"; bk="$(oldest backup-termux)"
  [ -n "$bk" ] || { echo "Nenhum backup da aparência (nunca foi mudada)."; return 0; }
  echo "Restaurando aparência de: $bk"
  rm -f "$HOME/.termux/colors.properties" "$HOME/.termux/font.ttf"
  if [ -f "$bk/.vazio" ]; then # ~/.termux não existia: só tira a barra de teclas
    [ -f "$f" ] && { grep -v -E '^[[:space:]]*extra-keys[[:space:]]*=' "$f" > "$f.new"; mv "$f.new" "$f"; }
  else
    mkdir -p "$HOME/.termux"; cp -a "$bk/." "$HOME/.termux/"
  fi
  rm -f "$HOME/.hushlogin"
  command -v termux-reload-settings >/dev/null 2>&1 && termux-reload-settings
  return 0
}

step_undo_bashrc() {
  [ -f "$HOME/.bashrc" ] || return 0
  sed -i '/# >>> termux-setup >>>/,/# <<< termux-setup <<</d' "$HOME/.bashrc"
  echo "Bloco do termux-setup removido do ~/.bashrc (comandos em ~/bin continuam lá)"
}

run_undo() {
  if [ "$AUTO" != 1 ]; then
    MENU_ITEMS=("Restaurar tudo" "Voltar")
    MENU_KEYS=(1 0)
    MENU_PRE=("${C_B}Volta ao estado de antes do 1º uso do script:${C_N}"
              "${C_D}· espelho apt original${C_N}"
              "${C_D}· tema, fonte, barra de teclas e boas-vindas${C_N}"
              "${C_D}· aliases do ~/.bashrc${C_N}"
              "${C_D}Pacotes instalados e o Ubuntu NÃO são removidos.${C_N}")
    menu_select "Desfazer"
    [ "$MENU_IDX" = 0 ] || return 0
  fi
  reset_plan
  add_step "Restaurar espelho original" step_undo_apt "" 5
  add_step "Restaurar aparência" step_undo_look "" 1
  add_step "Limpar ~/.bashrc" step_undo_bashrc "" 1
  clear_sel; execute_plan
}

# atualiza Termux + (se existir) Ubuntu e IAs
run_update_all() {
  clear_sel; build_plan update
  if command -v proot-distro >/dev/null 2>&1 && ub_has; then
    add_step "Ubuntu: atualizar tudo" step_ub_upgrade "" 40
    add_step "Atualizar Claude/Codex/Gemini" step_ia_update "" 40
  fi
  if confirm_run; then execute_plan; fi
}

run_repair() {
  reset_plan
  add_step "Reparar pacotes quebrados" step_repair "" 30
  add_step "Limpeza" step_cleanup "" 3
  clear_sel; execute_plan
}

menu_ia() {
  while true; do
    MENU_ITEMS=("Instalar tudo (linguagens + IAs)" "Escolher o que instalar" "Atualizar Ubuntu (+ limpeza)"
                "Atualizar só as IAs" "Atualizar Ubuntu + IAs" "Entrar no Ubuntu (shell)" "Voltar")
    MENU_KEYS=(1 2 3 4 5 6 0)
    MENU_PRE=("${C_DIM3}Ubuntu real dentro do Termux (proot)${C_N}"
              "${C_DIM3}Python · Node · Java · C/C++ · IAs${C_N}"
              "${C_DK}login das IAs: navegador, no 1º uso${C_N}")
    menu_select "Ubuntu & IA"
    case "$MENU_IDX" in
      0) select_ub_all; run_plan_flow ;;
      1) clear_sel; SEL[base]=1; SEL[api]=1; SEL[ubuntu]=1; SEL[ubtools]=1
         CHECK_TITLE="Ubuntu & IA"
         if menu_check ubuntu $UB_MODS claude codex gemini; then CHECK_TITLE=""; run_plan_flow; fi
         CHECK_TITLE="" ;;
      2|3|4)
         if ! { command -v proot-distro >/dev/null 2>&1 && ub_has; }; then
           term_size; printf '\033[2J\033[H'; banner "Ubuntu"; L " ${C_Y}Ubuntu ainda não instalado.${C_N}"; pause; continue
         fi
         reset_plan
         [ "$MENU_IDX" != 3 ] && add_step "Ubuntu: atualizar tudo" step_ub_upgrade "" 40
         [ "$MENU_IDX" != 2 ] && add_step "Atualizar Claude/Codex/Gemini" step_ia_update "" 40
         clear_sel; execute_plan ;;
      5) term_size; printf '\033[2J\033[H'; show_cur
         if command -v proot-distro >/dev/null 2>&1 && ub_has; then
           proot-distro login "$UB" --user "$UB_USER" -- bash -l
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
    if [[ ! $r =~ ^[nN] ]]; then
      # tenta direto; se falhar (listas velhas), atualiza as listas e tenta de novo
      apt_i gh || { apt-get update "${APT_OPTS[@]}" && apt_i gh; }
    fi
    command -v gh >/dev/null 2>&1 || { L " ${C_R}gh indisponível.${C_N}"; pause; return; }
  fi
  if gh auth status >/dev/null 2>&1; then
    L " ${C_G}✓ Já autenticado:${C_N}"; gh auth status 2>&1 | head -n 6
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
    gh auth setup-git && L " ${C_G}✓ Git configurado para usar o gh${C_N}"
    gh auth status 2>&1 | head -n 6
  else
    L " ${C_R}✗ Token recusado.${C_N} Confira os escopos e a validade."
  fi
  tok=""; pause
}

main_menu() {
  while true; do
    MENU_ITEMS=("Instalação completa (tudo + IA)" "Dev padrão (recomendado)" "Ubuntu & IA (linguagens + IAs)" "Aparência (teclas, tema, fonte)" "Personalizada (escolher módulos)" "GitHub (entrar com token)" "Atualizar tudo (Termux+Ubuntu+IA)" "Espelhos: testar / trocar" "Reparar pacotes quebrados" "Verificar instalação" "Ver último log" "Desfazer (restaurar original)" "Sair")
    MENU_KEYS=(1 2 3 4 5 6 7 8 9 v l d 0)
    MENU_PRE=()
    if [ "$ROWS" -ge 30 ] && [ "$COLS" -ge 38 ]; then MENU_PRE=("${LOGO_S_C[@]}" ""); fi
    local dev mir
    dev="Android $(getprop ro.build.version.release 2>/dev/null || echo ?) · $(getprop ro.product.model 2>/dev/null) · $ARCH"
    mir="$(( $(free_kb) / 1024 )) MB livres · $(current_mirror)"
    MENU_PRE+=("${C_DIM3}${dev:0:$((COLS-2))}${C_N}" "${C_DK}${mir:0:$((COLS-2))}${C_N}")
    menu_select "Menu principal"
    case "$MENU_IDX" in
      0) select_all; run_plan_flow ;;
      1) select_dev; run_plan_flow ;;
      2) menu_ia ;;
      3) menu_appearance ;;
      4) clear_sel; select_dev
         if menu_check "${MOD_ORDER[@]}"; then run_plan_flow; fi ;;
      5) menu_github ;;
      6) run_update_all ;;
      7) menu_mirror ;;
      8) run_repair ;;
      9) verify_install ;;
      10) menu_log ;;
      11) run_undo ;;
      *) break ;;
    esac
  done
  fx_outro
  printf '\033[2J\033[H'
  echo " ${C_DK}conexão encerrada ·${C_N} ${C_DIM3}logs em ~/termux-setup-logs${C_N}"
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
  case "$ACTION" in # atalhos de linha de comando: pulam o menu
    tudo)      select_all; run_plan_flow ;;
    dev)       select_dev; run_plan_flow ;;
    ia)        select_ia;  run_plan_flow ;;
    atualizar) run_update_all ;;
    desfazer)  run_undo ;;
    *)         fx_splash; main_menu ;;
  esac
}
main "$@"
