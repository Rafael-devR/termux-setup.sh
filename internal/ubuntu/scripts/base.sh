# Roda como root. Variáveis: U (usuário comum).
# shellcheck disable=SC2154
set -e
echo 'APT::Sandbox::User "root";' > /etc/apt/apt.conf.d/99proot
apt-get update
apt-get -y upgrade
apt-get install -y --no-install-recommends curl git ca-certificates gnupg bash ripgrep unzip xz-utils less nano locales
id "$U" >/dev/null 2>&1 || useradd -m -s /bin/bash "$U"
mkdir -p "/home/$U/projetos"
chown "$U:$U" "/home/$U/projetos"
touch "/home/$U/.profile"
grep -q "termux-setup" "/home/$U/.profile" || cat >> "/home/$U/.profile" <<'EOP'
# termux-setup
export PATH="$HOME/.local/bin:$HOME/.npm-global/bin:$PATH"
EOP
chown "$U:$U" "/home/$U/.profile"
echo "Usuário $U pronto (casa: /home/$U)"
