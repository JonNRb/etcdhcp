package main

import (
	"bytes"
	"context"
	"net"
	"sync"
	"syscall"

	"github.com/golang/glog"
	"github.com/mdlayher/arp"
)

type ConflictDetector struct {
	c        *arp.Client
	detected map[string]net.HardwareAddr
	mu       sync.Mutex
}

func newConflictDetector(iface string) (*ConflictDetector, error) {
	i, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, err
	}
	c, err := arp.Dial(i)
	if err != nil {
		return nil, err
	}
	return &ConflictDetector{
		c:        c,
		detected: make(map[string]net.HardwareAddr),
	}, nil
}

func (c *ConflictDetector) WouldConflict(ctx context.Context, ip net.IP, mac net.HardwareAddr) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.setDeadline(ctx)
	existing := c.resolveOrNil(ip)
	return existing != nil && !bytes.Equal(mac, existing)
}

func (c *ConflictDetector) setDeadline(ctx context.Context) {
	d, _ := ctx.Deadline()
	c.c.SetDeadline(d)
}

func (c *ConflictDetector) resolveOrNil(ip net.IP) net.HardwareAddr {
	mac, err := c.c.Resolve(ip)

	switch err {
	case nil:
		return mac
	case syscall.EAGAIN:
		return nil
	default:
		glog.Warningf("error resolving %q: %v", ip, err)
		return nil
	}
}
