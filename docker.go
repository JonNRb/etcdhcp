package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"

	"docker.io/go-docker"
	"docker.io/go-docker/api/types"
	"github.com/golang/glog"
	dhcp "github.com/krolaw/dhcp4"
)

var (
	dockerNetwork = flag.String("docker.dhcp_network", "", "Container network to run the DHCP server on (this network should have appropriate labels set)")
)

func maybeInitFromDockerEnvironment(ctx context.Context, handler *DHCPHandler) (bool, error) {
	if *dockerNetwork == "" {
		return false, nil
	}

	cli, err := docker.NewEnvClient()
	if err != nil {
		return false, err
	}
	defer cli.Close()

	glog.V(2).Info("connected to docker")

	id, err := os.Hostname()
	if err != nil {
		return false, err
	}

	containerJSON, err := cli.ContainerInspect(ctx, id)
	if err != nil {
		return false, err
	}

	epInfo, ok := containerJSON.NetworkSettings.Networks[*dockerNetwork]
	if !ok {
		return false, fmt.Errorf("network \"%v\" not in container info", *dockerNetwork)
	}

	iface, err := ifaceForIP(epInfo.IPAddress, epInfo.IPPrefixLen)
	if err != nil {
		return false, err
	} else if iface == "" {
		return false, fmt.Errorf("no interface found on container network \"%v\" matching address \"%v/%d\"",
			*dockerNetwork, epInfo.IPAddress, epInfo.IPPrefixLen)
	}
	handler.iface = iface

	handler.ip = parseIP4(epInfo.IPAddress)
	handler.options[dhcp.OptionSubnetMask] = cidrToMask(epInfo.IPPrefixLen)
	handler.options[dhcp.OptionRouter] = parseIP4(epInfo.Gateway)

	networkJSON, err := cli.NetworkInspect(ctx, epInfo.NetworkID, types.NetworkInspectOptions{})
	if err != nil {
		return false, err
	}

	start, ok := networkJSON.Labels["etcdhcp.start"]
	if !ok {
		return false, fmt.Errorf("no etcdhcp.start label on network \"%v\"", *dockerNetwork)
	}
	handler.start = parseIP4(start)

	end, ok := networkJSON.Labels["etcdhcp.end"]
	if !ok {
		return false, fmt.Errorf("no etcdhcp.end label on network \"%v\"", *dockerNetwork)
	}
	handler.end = parseIP4(end)

	dns, ok := networkJSON.Labels["etcdhcp.dns"]
	if !ok {
		return false, fmt.Errorf("no etcdhcp.dns label on network \"%v\"", *dockerNetwork)
	}
	handler.options[dhcp.OptionDomainNameServer] = parseIP4(dns)

	return true, nil
}

func ifaceForIP(ip string, prefix int) (string, error) {
	fullIP := fmt.Sprintf("%v/%v", ip, prefix)
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		glog.Infof("looking at %v", iface.Name)
		addrs, err := iface.Addrs()
		if err != nil {
			glog.Infof("error enumerating addresses: %v", err)
			continue
		}
		for _, addr := range addrs {
			if addr.String() == fullIP {
				return iface.Name, nil
			}
			glog.Infof("%v was not a match", addr)
		}
	}
	return "", nil
}

func maskSingle(p int) byte {
	switch {
	case p < 0:
		return 0
	case p < 8:
		return ^((1 << (8 - uint(p))) - 1)
	default:
		return 255
	}
}

func cidrToMask(prefix int) net.IP {
	a := maskSingle(prefix)
	b := maskSingle(prefix - 8)
	c := maskSingle(prefix - 16)
	d := maskSingle(prefix - 24)

	// TODO(jonnrb): get rid of `parseIP4()`
	return parseIP4(fmt.Sprintf("%d.%d.%d.%d", a, b, c, d))
}
