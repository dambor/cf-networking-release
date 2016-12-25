package backend

import (
	"encoding/json"
	"fmt"
	"net"
	"silk-controller/models"
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

func netHostFromControllerRoute(route models.Route) (*NetHost, error) {
	return NewNetHost(route.VtepIP, route.Subnet)
}
func netHostsFromControllerRoutes(routes []models.Route) ([]*NetHost, error) {
	var err error
	nhs := make([]*NetHost, len(routes))
	for i := 0; i < len(routes); i++ {
		nhs[i], err = NewNetHost(routes[i].VtepIP, routes[i].Subnet)
		if err != nil {
			return nil, err
		}
	}
	return nhs, nil
}

func NewNetHost(publicIP string, overlayCIDR string) (*NetHost, error) {
	host := &NetHost{}
	host.PublicIP = net.ParseIP(publicIP)
	if host.PublicIP == nil {
		return nil, fmt.Errorf("error parsing %q as an IP")
	}
	var err error
	_, overlaySubnet, err := net.ParseCIDR(overlayCIDR)
	if err != nil {
		return nil, fmt.Errorf("parsing %q as overlay cidr: %s", overlayCIDR, err)
	}
	host.OverlaySubnet = *overlaySubnet
	host.VtepOverlayIP = host.OverlaySubnet.IP
	host.VtepOverlayMAC = macAddressFromIP(host.VtepOverlayIP)
	return host, nil
}

// MACAddressFromIP generates 48 bit virtual mac addresses based on the IP4 input.
func macAddressFromIP(ip net.IP) net.HardwareAddr {
	if ip.To4() == nil {
		panic("only ipv4 is supported")
	}

	PrivateMACPrefix := []byte{0x0a, 0x00}
	const ipRelevantByteLen = 4
	ipByteLen := len(ip)
	return (net.HardwareAddr)(
		append(PrivateMACPrefix, ip[ipByteLen-ipRelevantByteLen:ipByteLen]...),
	)
}
