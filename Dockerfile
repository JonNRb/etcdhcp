from golang:alpine

run apk update && apk add git docker
add . /go/src/github.com/jonnrb/etcdhcp
run cd /go/src/github.com/jonnrb/etcdhcp \
 && go get -u github.com/golang/dep/cmd/dep \
 && dep ensure \
 && go install

add entrypoint.sh /entrypoint.sh

expose 9842

# this be hacky for me
env DHCP_AUTO_DOCKER_NETWORK ""

env ETCD_DISCOVERY_ENDPOINTS localhost:2379
env DHCP_ROUTER 10.6.9.1
env DHCP_DNS 8.8.8.8
env DHCP_SERVER_IF eth0
env DHCP_SERVER_IP 10.6.9.1
env DHCP_SUBNET_MASK 255.255.255.0
env DHCP_ISSUE_FROM 10.6.9.10
env DHCP_ISSUE_TO 10.6.9.100

entrypoint ["/entrypoint.sh"]
