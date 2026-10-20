#!/bin/bash
set -euo pipefail

cd /opt/aws-hub
exec 9>/opt/aws-hub/.deploy.lock
flock 9

compose=(docker compose -f docker-compose.prod.yaml)
service="${1:-}"

wait_for_vpn() {
  for _ in $(seq 1 30); do
    if ip -4 addr show dev tun0 2>/dev/null | grep -q '10.8.0.1'; then
      return 0
    fi
    sleep 2
  done
  echo "tun0 has no 10.8.0.1; OpenVPN is not ready" >&2
  return 1
}

case "$service" in
  api)
    "${compose[@]}" pull api
    "${compose[@]}" up -d --wait postgres
    "${compose[@]}" run --rm --no-deps migrations
    "${compose[@]}" up -d --no-deps --wait --force-recreate api
    ;;
  frontend)
    wait_for_vpn
    "${compose[@]}" pull frontend
    "${compose[@]}" up -d --no-deps --wait --force-recreate frontend
    ;;
  *)
    echo "usage: deploy.sh api|frontend" >&2
    exit 1
    ;;
esac

if [ -f /etc/systemd/system/aws-hub.service ]; then
  sudo systemctl enable aws-hub.service
fi
