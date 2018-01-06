#!/bin/sh

set -e

if [ -n "${DHCP_AUTO_DOCKER_NETWORK}" ]; then
  container="$(hostname)"
  DHCP_ROUTER="$(/docker network inspect "${DHCP_AUTO_DOCKER_NETWORK}" -f "{{ (index .IPAM.Config 0).Gateway }}")"
  subnet="$(/docker network inspect "${DHCP_AUTO_DOCKER_NETWORK}" -f "{{ (index .IPAM.Config 0).Subnet }}")"
  if [ "$(echo "${subnet}" |cut -d "/" -f 2)" != "16" ]; then
    echo "error: docker network ${DHCP_AUTO_DOCKER_NETWORK} doesn't have a 16-bit subnet mask and this whole script is kinda riding on that..."
    exit 1
  fi
  DHCP_SERVER_IF="$(ip route |grep "${subnet}" |cut -d " " -f 3)"
  DHCP_SERVER_IP="$(/docker inspect "${container}" -f "{{ (index .NetworkSettings.Networks \"${DHCP_AUTO_DOCKER_NETWORK}\").IPAddress }}")"
  DHCP_SUBNET_MASK="255.255.255.0"
  subnet_prefix="$(echo "${subnet}" |cut -d "." -f 1-2)"
  DHCP_ISSUE_FROM="${subnet_prefix}.1.1"
  DHCP_ISSUE_TO="${subnet_prefix}.1.254"
fi

/etcdhcp                                                  \
  -etcd.discovery.endpoints "${ETCD_DISCOVERY_ENDPOINTS}" \
  -etcd.prefix              "${ETCD_PREFIX}"              \
  -dhcp.router              "${DHCP_ROUTER}"              \
  -dhcp.dns                 "${DHCP_DNS}"                 \
  -dhcp.server-if           "${DHCP_SERVER_IF}"           \
  -dhcp.server-ip           "${DHCP_SERVER_IP}"           \
  -dhcp.subnet-mask         "${DHCP_SUBNET_MASK}"         \
  -dhcp.issue-from          "${DHCP_ISSUE_FROM}"          \
  -dhcp.issue-to            "${DHCP_ISSUE_TO}"            \
  -v=2                                                    \
  "$@"
