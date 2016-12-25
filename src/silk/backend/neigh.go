package backend

import (
	"fmt"
	"net"
	"syscall"

	"github.com/vishvananda/netlink"
)

// UpsertFDB upserts a forwarding database rule for the vxlan vtep
// The rule is of the form: overlayHardwareAddr --> targetVtepIP
func (dev *vxlanDevice) UpsertFDB(overlayHardwareAddr net.HardwareAddr, targetVtepIP net.IP) error {
	// bridge fdb replace 0a:00:0a:ff:02:00 dev silk.1 dst 10.244.99.11
	return netlink.NeighSet(&netlink.Neigh{
		LinkIndex:    dev.link.Index,
		State:        netlink.NUD_PERMANENT,
		Family:       syscall.AF_BRIDGE,
		Flags:        netlink.NTF_SELF,
		IP:           targetVtepIP,
		HardwareAddr: overlayHardwareAddr,
	})
}

// UpsertARP upserts a neighbor rule to the ARP table
// The rule is of the form: overlay IP --> overlayHardwareAddr
func (dev *vxlanDevice) UpsertARP(overlayIP net.IP, overlayHardwareAddr net.HardwareAddr) error {
	// ip neigh replace to 10.255.1.0 dev silk.1 lladdr 0a:00:0a:ff:01:00
	return netlink.NeighSet(&netlink.Neigh{
		LinkIndex:    dev.link.Index,
		State:        netlink.NUD_PERMANENT,
		Type:         syscall.RTN_UNICAST,
		IP:           overlayIP,
		HardwareAddr: overlayHardwareAddr,
	})
}

// AddRoute installs an IP route that targets the device
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

// PurgeAllRoutes deletes all IP routes targetting the device
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

// PurgeAllAddresses deletes all IP addresses for the device
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
