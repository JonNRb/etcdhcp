package main

import (
	"context"
	"net"
	"time"

	"github.com/golang/glog"
	dhcp "github.com/krolaw/dhcp4"
	"github.com/pkg/errors"
	etcd "go.etcd.io/etcd/clientv3"
)

type DHCPHandler struct {
	client  *etcd.Client
	prefix  string
	timeout time.Duration

	iface         string
	ip            net.IP
	options       dhcp.Options
	start         net.IP
	end           net.IP
	leaseDuration time.Duration

	conflictDetector *ConflictDetector
}

func (h *DHCPHandler) ServeDHCP(p dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options) dhcp.Packet {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	switch msgType {
	case dhcp.Discover:
		glog.Infof("handling discover")
		nic := p.CHAddr().String()
		ip, err := h.handleDiscover(ctx, nic)
		if err != nil {
			glog.Errorf("failed to respond to discover request: %v", err)
			return nil
		}
		glog.Infof("offering %v to %v", ip, nic)

		return dhcp.ReplyPacket(p, dhcp.Offer, h.ip, ip, h.leaseDuration,
			h.options.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]))

	case dhcp.Request:
		if server, ok := options[dhcp.OptionServerIdentifier]; ok && !net.IP(server).Equal(h.ip) {
			return nil // Message not for this dhcp server
		}
		reqIP := net.IP(options[dhcp.OptionRequestedIPAddress])
		if reqIP == nil {
			reqIP = net.IP(p.CIAddr())
		}

		nic := p.CHAddr().String()
		ip, err := h.handleRequest(ctx, reqIP, nic)
		if err != nil {
			glog.Errorf("could not lease: %v", err)
			return dhcp.ReplyPacket(p, dhcp.NAK, h.ip, nil, 0, nil)
		}
		opts := p.ParseOptions()
		glog.V(2).Infof("client provided options %v", opts)
		if h.recordClientInfo(ctx, nic, clientInfo(opts)) != nil {
			glog.Warningf("error recording client options for %v", nic)
		}
		glog.Infof("leased %v to %v", reqIP, nic)
		return dhcp.ReplyPacket(p, dhcp.ACK, h.ip, ip, h.leaseDuration,
			h.options.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]))

	case dhcp.Release, dhcp.Decline:
		err := h.revokeLease(ctx, p.CHAddr().String())
		if err != nil {
			glog.Errorf("could not revoke lease for %v: %v", p.CHAddr().String(), err)
		}
	}
	return nil
}

func (h *DHCPHandler) handleDiscover(ctx context.Context, nic string) (net.IP, error) {
	ip, err := h.nicLeasedIP(ctx, nic)
	if err != nil {
		return nil, errors.Wrap(err, "could not lookup existing nic lease")
	}
	var filter []net.IP
	if ip != nil {
		if h.wouldConflict(ctx, nic, ip) {
			filter = append(filter, ip)
		} else {
			glog.Infof("found previous lease for %v", nic)
			return ip, nil
		}
	}
	for {
		ip, err = h.freeIP(ctx, filter)
		if err != nil {
			return nil, errors.Wrap(err, "could not find next free ip")
		}
		if h.wouldConflict(ctx, nic, ip) {
			filter = append(filter, ip)
			continue
		}
		return ip, nil
	}
}

func (h *DHCPHandler) wouldConflict(ctx context.Context, nic string, ip net.IP) bool {
	if h.conflictDetector == nil {
		return false
	}

	mac, err := net.ParseMAC(nic)
	if err != nil {
		glog.Fatalf("could not parse MAC that we String()'d ourselves %q: %v", nic, err)
	}
	return h.conflictDetector.WouldConflict(ctx, ip, mac)
}

func (h *DHCPHandler) handleRequest(ctx context.Context, ip net.IP, nic string) (net.IP, error) {
	glog.Infof("handling request for %v from %v", ip, nic)
	if len(ip) != 4 || ip.Equal(net.IPv4zero) {
		return nil, errors.New("invalid ip requested")
	}

	err := h.leaseIP(ctx, ip, nic, h.leaseDuration*2)
	if err != nil {
		return nil, errors.Wrap(err, "could not update lease")
	}
	return ip, nil
}
