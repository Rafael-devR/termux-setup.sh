# Roda como root: atualização completa, com reparo e limpeza.
dpkg --configure -a || true
apt-get update || exit 1
# shellcheck disable=SC2086
apt-get $APT_O --fix-broken install || true
# shellcheck disable=SC2086
apt-get $APT_O full-upgrade || exit 1
apt-get -y autoremove --purge
apt-get clean
echo "Ubuntu atualizado: $(grep PRETTY_NAME /etc/os-release | cut -d= -f2 | tr -d '"')"
