package panel

const installScript = `#!/usr/bin/env bash
set -euo pipefail

SERVER="{{BASE_URL}}"
TOKEN=""
NAME=""
INSTALL_DIR="/opt/gost-pool-agent"
SERVICE_NAME="gost-pool-agent"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --server) SERVER="$2"; shift 2 ;;
    --token) TOKEN="$2"; shift 2 ;;
    --name) NAME="$2"; shift 2 ;;
    *) echo "Unknown argument: $1"; exit 1 ;;
  esac
done

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "This agent only supports Linux nodes."
  exit 1
fi

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) BIN="gost-pool-agent-linux-amd64" ;;
  aarch64|arm64) BIN="gost-pool-agent-linux-arm64" ;;
  *) echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

if [[ "$EUID" -ne 0 ]]; then
  echo "Please run as root."
  exit 1
fi

mkdir -p "$INSTALL_DIR"
EXISTING_AGENT=0
if [[ -s "$INSTALL_DIR/agent.json" ]] && grep -q '"nodeId"' "$INSTALL_DIR/agent.json" && grep -q '"agentToken"' "$INSTALL_DIR/agent.json"; then
  EXISTING_AGENT=1
fi

if [[ -z "$TOKEN" && "$EXISTING_AGENT" != "1" ]]; then
  echo "--token is required"
  exit 1
fi

if [[ "$EXISTING_AGENT" == "1" ]]; then
  echo "Existing agent identity found; skipping register token check for in-place upgrade."
else
  echo "Checking register token"
  TOKEN_CHECK="$(curl -sS -w '\n%{http_code}' "$SERVER/api/agent/register-token/check?token=$TOKEN")"
  TOKEN_CODE="$(printf '%s\n' "$TOKEN_CHECK" | tail -n 1)"
  TOKEN_BODY="$(printf '%s\n' "$TOKEN_CHECK" | sed '$d')"
  if [[ "$TOKEN_CODE" != "200" ]]; then
    echo "Register token is not available. Generate a new token in the panel."
    echo "$TOKEN_BODY"
    exit 1
  fi
fi

echo "Downloading agent from $SERVER/downloads/$BIN"
TMP_AGENT="$(mktemp "$INSTALL_DIR/gost-pool-agent.XXXXXX")"
cleanup_tmp_agent() {
  rm -f "$TMP_AGENT"
}
trap cleanup_tmp_agent EXIT
curl -fsSL "$SERVER/downloads/$BIN" -o "$TMP_AGENT"
chmod +x "$TMP_AGENT"
echo "Downloaded agent version: $("$TMP_AGENT" --version 2>/dev/null || echo unknown)"
mv -f "$TMP_AGENT" "$INSTALL_DIR/gost-pool-agent"
trap - EXIT

cat > "$INSTALL_DIR/agent.env" <<EOF
GPP_SERVER=$SERVER
GPP_REGISTER_TOKEN=$TOKEN
GPP_NODE_NAME=$NAME
GPP_CONFIG=$INSTALL_DIR/agent.json
EOF

cat > "/etc/systemd/system/$SERVICE_NAME.service" <<EOF
[Unit]
Description=GOST Pool Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=$INSTALL_DIR/agent.env
ExecStart=$INSTALL_DIR/gost-pool-agent --server \$GPP_SERVER --token \$GPP_REGISTER_TOKEN --name "\$GPP_NODE_NAME" --config \$GPP_CONFIG
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl restart "$SERVICE_NAME"
echo "Agent installed. Check status: systemctl status $SERVICE_NAME"
`
