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
