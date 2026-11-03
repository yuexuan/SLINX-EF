#!/usr/bin/env bash
set -Eeuo pipefail
umask 077
REPO=yuexuan/SLINX-EF
DIR=/etc/slinx
[[ $EUID == 0 ]] || { echo 'Run as root'; exit 1; }
command -v systemctl >/dev/null || { echo 'systemd required'; exit 1; }
case $(uname -m) in x86_64) ARCH=amd64; CORE_SHA=bae152039a7728f8afad50b4c7e0c58fe84e40d1236329041eade0c4897c4235;; aarch64|arm64) ARCH=arm64; CORE_SHA=ddb236bd6a6bce2c538229f70f910812a09d01560577b89b23e66bad05479468;; *) exit 1;; esac
. /etc/os-release
case "$ID" in debian|ubuntu) apt-get update; apt-get install -y curl ca-certificates gzip sqlite3 util-linux;; rocky|almalinux|fedora|centos|rhel) dnf install -y curl ca-certificates gzip sqlite util-linux;; *) echo "Unsupported distribution: $ID"; exit 1;; esac
mkdir -p /run/lock
exec 9>/run/lock/slinx-ef-install.lock
flock -n 9 || { echo 'Another update is running'; exit 1; }
if [[ -e "$DIR/slinx" && ! -f "$DIR/.slinx-ef" ]]; then echo 'Existing unmanaged SLINX installation: migrate explicitly first'; exit 1; fi
if [[ ! -e "$DIR/.slinx-ef" ]] && systemctl cat slinx.service >/dev/null 2>&1; then echo 'Conflicting slinx service'; exit 1; fi
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
VERSION=${1:-$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')}
[[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'Invalid release'; exit 1; }
BASE="https://github.com/$REPO/releases/download/$VERSION"
curl -fsSL "$BASE/slinx_linux_$ARCH" -o "$TMP/slinx_linux_$ARCH"
curl -fsSL "$BASE/SHA256SUMS" -o "$TMP/SHA256SUMS"
(cd "$TMP"; grep -E "^[a-f0-9]{64}  slinx_linux_$ARCH$" SHA256SUMS > panel.sum; test -s panel.sum; sha256sum -c panel.sum)
curl -fsSL "https://github.com/slinxlink/node/releases/download/sing-box-1.13.12/sing-box_linux_$ARCH.gz" -o "$TMP/core.gz"
echo "$CORE_SHA  $TMP/core.gz" | sha256sum -c -
gzip -dc "$TMP/core.gz" > "$TMP/sing-box"
chmod 700 "$TMP/slinx_linux_$ARCH" "$TMP/sing-box"
"$TMP/sing-box" version | grep -F '1.13.12'
BACKUP="$DIR/backups/$(date -u +%Y%m%dT%H%M%SZ)-$$"
mkdir -p "$BACKUP" "$DIR/bin" "$DIR/data" "$DIR/cert"
HAD_OLD=0
[[ ! -f "$DIR/slinx" ]] || HAD_OLD=1
trap 'if [[ $HAD_OLD == 1 ]]; then systemctl start slinx.service; fi' ERR
systemctl stop slinx.service 2>/dev/null || true
for item in slinx bin data cert .slinx-ef; do [[ ! -e "$DIR/$item" ]] || cp -a "$DIR/$item" "$BACKUP/"; done
[[ ! -f /etc/systemd/system/slinx.service ]] || cp -a /etc/systemd/system/slinx.service "$BACKUP/service"
rollback() {
 trap - ERR
 systemctl stop slinx.service 2>/dev/null || true
 if [[ $HAD_OLD == 1 ]]; then
  for item in slinx bin data cert .slinx-ef; do if [[ -e "$BACKUP/$item" ]]; then rm -rf -- "$DIR/$item"; cp -a "$BACKUP/$item" "$DIR/"; fi; done
  [[ ! -f "$BACKUP/service" ]] || cp -a "$BACKUP/service" /etc/systemd/system/slinx.service
  systemctl daemon-reload; systemctl start slinx.service
 fi
 echo "Installation failed; backup: $BACKUP" >&2
 exit 1
}
trap rollback ERR
install -m 700 "$TMP/slinx_linux_$ARCH" "$DIR/slinx.new"
mv "$DIR/slinx.new" "$DIR/slinx"
install -m 700 "$TMP/sing-box" "$DIR/bin/sing-box"
cat > /etc/systemd/system/slinx.service <<EOF
[Unit]
Description=SLINX-EF
After=network-online.target
Wants=network-online.target
[Service]
Type=simple
WorkingDirectory=/etc/slinx
ExecStart=/etc/slinx/slinx
UMask=0077
Restart=on-failure
RestartSec=5
[Install]
WantedBy=multi-user.target
EOF
printf '%s\n' "$VERSION" > "$DIR/.slinx-ef"
printf '#!/bin/bash\nexec /etc/slinx/slinx cli\n' > /usr/local/bin/slinx
chmod 755 /usr/local/bin/slinx
systemctl daemon-reload
systemctl enable --now slinx.service
sleep 10
systemctl is-active --quiet slinx.service
PANEL_PORT=$(sqlite3 "$DIR/data/slinx.db" 'SELECT port FROM config LIMIT 1;')
PANEL_PATH=$(sqlite3 "$DIR/data/slinx.db" 'SELECT path FROM config LIMIT 1;')
PANEL_DOMAIN=$(sqlite3 "$DIR/data/slinx.db" 'SELECT domain FROM config LIMIT 1;')
if [[ -z "$PANEL_DOMAIN" ]]; then
 curl -fsS --max-time 10 "http://127.0.0.1:$PANEL_PORT$PANEL_PATH" -o /dev/null
else
 curl -fsS --max-time 10 --resolve "$PANEL_DOMAIN:$PANEL_PORT:127.0.0.1" "https://$PANEL_DOMAIN:$PANEL_PORT$PANEL_PATH" -o /dev/null
fi
[[ ! -f "$DIR/data/sing-box.json" ]] || (cd "$DIR"; bin/sing-box check -c data/sing-box.json)
trap - ERR
echo "SLINX-EF $VERSION installed. Backup: $BACKUP"
echo "Panel port: $PANEL_PORT; path: $PANEL_PATH"
echo 'Run slinx to view panel URL and manage credentials. Use an SSH tunnel for initial access.'
echo 'Status: systemctl status slinx; logs: journalctl -u slinx'
