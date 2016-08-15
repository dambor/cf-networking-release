package backend

import (
	"fmt"
	"net"
	"syscall"

	"github.com/vishvananda/netlink"
)

type L2Neigh struct {
	OverlayMAC net.HardwareAddr
	UnderlayIP net.IP
}

func (dev *vxlanDevice) SetL2(n L2Neigh) error {
	// bridge fdb replace 0a:00:0a:ff:02:00 dev silk.1 dst 10.244.99.11
	fmt.Printf("calling L2 NeighSet: %s to %s", n.OverlayMAC.String(), n.UnderlayIP.String())
	return netlink.NeighSet(&netlink.Neigh{
		LinkIndex:    dev.link.Index,
		State:        netlink.NUD_PERMANENT,
		Family:       syscall.AF_BRIDGE,
		Flags:        netlink.NTF_SELF,
		IP:           n.UnderlayIP,
		HardwareAddr: n.OverlayMAC,
	})
}

type L3Neigh struct {
	OverlayMAC net.HardwareAddr
	OverlayIP  net.IP
}

func (dev *vxlanDevice) SetL3(n L3Neigh) error {
	// ip neigh replace to 10.255.1.0 dev silk.1 lladdr 0a:00:0a:ff:01:00
	fmt.Printf("calling L3 NeighSet: %s to %s", n.OverlayIP.String(), n.OverlayMAC.String())
	return netlink.NeighSet(&netlink.Neigh{
		LinkIndex:    dev.link.Index,
		State:        netlink.NUD_PERMANENT,
		Type:         syscall.RTN_UNICAST,
		IP:           n.OverlayIP,
		HardwareAddr: n.OverlayMAC,
	})
}

func (dev *vxlanDevice) AddRoute(destNet *net.IPNet, gateway net.IP, scope netlink.Scope, src net.IP) error {
	// ip route add 10.255.2.0/24 via 10.255.2.0 dev silk.1
	err := netlink.RouteAdd(&netlink.Route{
		LinkIndex: dev.link.Index,
		Scope:     scope,
		Dst:       destNet,
		Gw:        gateway,
		Src:       src,
	})
	if err != nil && err != syscall.EEXIST {
		return fmt.Errorf("ip route add %s via %s dev %s scope %d src %s: %s",
			destNet.String(), gateway.String(), dev.link.Attrs().Name, scope, src.String(), err)
	}
	return nil
}

func (dev *vxlanDevice) PurgeAllRoutes() error {
	existingRoutes, err := netlink.RouteList(dev.link, syscall.AF_INET)
	if err != nil {
		return fmt.Errorf("listing routes: %s", err)
	}

	for _, route := range existingRoutes {
		if err := netlink.RouteDel(&route); err != nil {
			return fmt.Errorf("removing route: %s", err)
		}
	}

	return nil
}

func (dev *vxlanDevice) PurgeAllAddresses() error {
	addrs, err := netlink.AddrList(dev.link, syscall.AF_INET)
	if err != nil {
		return fmt.Errorf("listing addrs: %s", err)
	}

	for _, addr := range addrs {
		if err = netlink.AddrDel(dev.link, &addr); err != nil {
			return fmt.Errorf("deleting addr %s from %s", addr.String(), dev.link.Attrs().Name)
		}
	}
	return nil
}
