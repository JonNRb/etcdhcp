package main

import (
	dhcp "github.com/krolaw/dhcp4"
)

type ClientInfo struct {
	Hostname    string
	VendorClass string
}

func clientInfo(opts dhcp.Options) ClientInfo {
	var client ClientInfo

	client.Hostname = string(opts[dhcp.OptionHostName])
	client.VendorClass = string(opts[dhcp.OptionVendorClassIdentifier])

	return client
}
