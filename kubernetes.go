package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/containernetworking/plugins/plugins/ipam/host-local/backend/allocator"
	"github.com/ericchiang/k8s"
	"github.com/golang/glog"
	dhcp "github.com/krolaw/dhcp4"
)

var (
	k8sNetwork = flag.String("k8s.dhcp_network", "", "Kubernetes CNI network (ns/net or just net) to run the DHCP server on (this network should have appropriate labels set)")
)

const (
	envVarHost = "KUBERNETES_SERVICE_HOST"
	envVarPort = "KUBERNETES_SERVICE_PORT"
)

func inK8sCluster() bool {
	return os.Getenv(envVarHost) != "" && os.Getenv(envVarPort) != ""
}

func maybeInitFromKubernetesEnvironment(ctx context.Context, handler *DHCPHandler) (bool, error) {
	if *k8sNetwork == "" {
		return false, nil
	}

	cli, err := k8s.NewInClusterClient()
	if err != nil {
		return false, err
	}

	namespace, network, err := splitNamespaceNetwork(*k8sNetwork)
	if err != nil {
		return false, err
	}
	glog.V(2).Infof("getting configuration for CNI network %q", namespace+"/"+network)

	attachment, err := getAttachment(namespace, network)
	if err != nil {
		return false, err
	}

	handler.iface = attachment.Interface
	handler.ip = parseIP4(attachment.IPs[0])

	var cniNet netAttachDef
	if err := cli.Get(ctx, namespace, network, &cniNet); err != nil {
		return false, fmt.Errorf("could not net network %q: %v", namespace+"/"+network, err)
	}

	start, ok := cniNet.ObjectMeta.Labels["etcdhcp.start"]
	if !ok {
		return false, fmt.Errorf("no etcdhcp.start label on network \"%v\"", *dockerNetwork)
	}
	handler.start = parseIP4(start)

	end, ok := cniNet.ObjectMeta.Labels["etcdhcp.end"]
	if !ok {
		return false, fmt.Errorf("no etcdhcp.end label on network \"%v\"", *dockerNetwork)
	}
	handler.end = parseIP4(end)

	dns, ok := cniNet.ObjectMeta.Labels["etcdhcp.dns"]
	if !ok {
		return false, fmt.Errorf("no etcdhcp.dns label on network \"%v\"", *dockerNetwork)
	}
	handler.options[dhcp.OptionDomainNameServer] = parseIP4(dns)

	var ipamCfg *allocator.IPAMConfig

	type pluginChain struct {
		Plugins []interface{} `json:"plugins"`
	}
	var pc pluginChain
	json.Unmarshal([]byte(cniNet.Spec.Config), &pc)
	if len(pc.Plugins) != 0 {
		for _, p := range pc.Plugins {
			raw, err := json.Marshal(p)
			if err != nil {
				return false, fmt.Errorf("error marshaling unmarshaled data: %w", err)
			}
			ipamCfg, _, err = allocator.LoadIPAMConfig(raw, "")
			if err == nil {
				break
			}
		}

		if err != nil {
			return false, fmt.Errorf("error parsing CNI config (the IPAM bit): %v", err)
		}
	} else {
		var err error
		ipamCfg, _, err = allocator.LoadIPAMConfig([]byte(cniNet.Spec.Config), "")
		if err != nil {
			glog.Errorf("bad CNI config from k8s API server: %q", cniNet.Spec.Config)
			return false, fmt.Errorf("error parsing CNI config (the IPAM bit): %v", err)
		}
	}

	r, ok := firstRange(ipamCfg)
	if !ok {
		glog.Errorf("bad IPAMConfig: %#v", ipamCfg)
		return false, fmt.Errorf("could not get first range from IPAM config for CNI network %q", namespace+"/"+network)
	}

	handler.options[dhcp.OptionSubnetMask] = r.Subnet.Mask
	handler.options[dhcp.OptionRouter] = r.Gateway

	return true, nil
}

func firstRange(cfg *allocator.IPAMConfig) (r *allocator.Range, ok bool) {
	if cfg.Range != nil {
		r, ok = cfg.Range, true
	} else if len(cfg.Ranges) > 0 && len(cfg.Ranges[0]) > 0 {
		r, ok = &cfg.Ranges[0][0], true
	}
	return
}
