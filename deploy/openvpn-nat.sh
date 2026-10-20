#!/bin/bash
iface=$(ip -4 route show default | awk '{print $5; exit}')
iptables -t nat -C POSTROUTING -s 10.8.0.0/24 -o "$iface" -j MASQUERADE 2>/dev/null || \
  iptables -t nat -A POSTROUTING -s 10.8.0.0/24 -o "$iface" -j MASQUERADE

if iptables -nL DOCKER-USER >/dev/null 2>&1; then
  iptables -C DOCKER-USER -i tun0 -j ACCEPT 2>/dev/null || \
    iptables -I DOCKER-USER 1 -i tun0 -j ACCEPT
  iptables -C DOCKER-USER -o tun0 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || \
    iptables -I DOCKER-USER 1 -o tun0 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
fi

iptables -C FORWARD -i tun0 -j ACCEPT 2>/dev/null || iptables -I FORWARD 1 -i tun0 -j ACCEPT
iptables -C FORWARD -o tun0 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || \
  iptables -I FORWARD 1 -o tun0 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
