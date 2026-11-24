# Roda como o usuário comum: venv pronto (o Ubuntu novo bloqueia pip global).
set -e
[ -d ~/venv ] || python3 -m venv ~/venv
~/venv/bin/pip install -q --upgrade pip
grep -q "alias venv=" ~/.bashrc 2>/dev/null || echo 'alias venv="source ~/venv/bin/activate"' >> ~/.bashrc
pipx ensurepath >/dev/null 2>&1 || true
echo "venv: ~/venv ($(~/venv/bin/python --version)) · ative com: venv"
