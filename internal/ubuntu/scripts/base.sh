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
# Node "linux": o Node do proot pode se achar "android" (veja common.sh)
write_node_shim "/home/$U/.termux-setup"
chown -R "$U:$U" "/home/$U/.termux-setup"
grep -q "termux-setup node" "/home/$U/.profile" || cat >> "/home/$U/.profile" <<'EOP'
# termux-setup node
case " ${NODE_OPTIONS:-} " in *node-linux.js*) ;; *) [ -f "$HOME/.termux-setup/node-linux.js" ] && export NODE_OPTIONS="${NODE_OPTIONS:+$NODE_OPTIONS }--require=$HOME/.termux-setup/node-linux.js" ;; esac
EOP
chown "$U:$U" "/home/$U/.profile"
echo "Usuário $U pronto (casa: /home/$U)"
