package models

import (
	"encoding/json"
	"net"
)

type NetHost struct {
	PublicIP       net.IP
	OverlaySubnet  net.IPNet
	VtepOverlayIP  net.IP
	VtepOverlayMAC net.HardwareAddr
}

func (n *NetHost) MarshalJSON() ([]byte, error) {
	toMarshal := struct {
		PublicIP       string
		OverlaySubnet  string
		VtepOverlayIP  string
		VtepOverlayMAC string
	}{
		PublicIP:       n.PublicIP.String(),
		OverlaySubnet:  n.OverlaySubnet.String(),
		VtepOverlayIP:  n.VtepOverlayIP.String(),
		VtepOverlayMAC: n.VtepOverlayMAC.String(),
	}
	return json.Marshal(toMarshal)
}

func (n *NetHost) String() string {
	bytes, _ := n.MarshalJSON()
	return string(bytes)
}
