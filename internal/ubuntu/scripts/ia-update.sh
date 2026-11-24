# Roda como o usuário comum: atualiza as IAs e ferramentas do usuário.
export PATH="$HOME/.local/bin:$HOME/.npm-global/bin:$PATH"
command -v claude >/dev/null 2>&1 && claude update
command -v codex  >/dev/null 2>&1 && npm install -g @openai/codex@latest
command -v gemini >/dev/null 2>&1 && npm install -g @google/gemini-cli@latest
command -v opencode >/dev/null 2>&1 && npm install -g opencode-ai@latest
[ -x ~/venv/bin/pip ] && ~/venv/bin/pip install -q --upgrade pip
command -v pipx >/dev/null 2>&1 && pipx upgrade-all
true
