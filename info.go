package main

import (
	dhcp "github.com/krolaw/dhcp4"

	pb "go.jonnrb.io/etcdhcp/proto"
)

func clientInfo(opts dhcp.Options) *pb.ClientInfo {
	return &pb.ClientInfo{
		Hostname:    string(opts[dhcp.OptionHostName]),
		VendorClass: string(opts[dhcp.OptionVendorClassIdentifier]),
	}
}
