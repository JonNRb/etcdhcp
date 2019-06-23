package main

import (
	"bytes"
	"context"
	"net"
	"sync"

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
	return !bytes.Equal(mac, c.resolveOrNil(ip))
}

func (c *ConflictDetector) setDeadline(ctx context.Context) {
	d, _ := ctx.Deadline()
	c.c.SetDeadline(d)
}

func (c *ConflictDetector) resolveOrNil(ip net.IP) (mac net.HardwareAddr) {
	var err error
	mac, err = c.c.Resolve(ip)
	if err != nil {
		glog.Warningf("Could not resolve %q. This may result in a conflict! Err: %v", ip, err)
	}
	return
}
