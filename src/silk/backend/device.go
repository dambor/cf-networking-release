package backend

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

type vxlanDevice struct {
	link *netlink.Vxlan
}

func newVXLANDevice(name string, vni int, vtepAddress net.IP, vtepPort int) (*vxlanDevice, error) {
	vtepDevice, err := locateInterface(vtepAddress)
	if err != nil {
		return nil, fmt.Errorf("locating external interface: %s", err)
	}

	link := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name: name,
		},
		VxlanId:      vni,
		VtepDevIndex: vtepDevice.Index,
		SrcAddr:      vtepAddress,
		Port:         vtepPort,
		Learning:     false,
		GBP:          true,
	}

	if err = addLink(link); err != nil {
		return nil, err
	}

	return &vxlanDevice{
		link: link,
	}, nil
}

func (dev *vxlanDevice) Configure(vtepOverlayIP net.IP, fullOverlayMask net.IPMask, hwAddr net.HardwareAddr) error {
	// vxlan's subnet is that of the whole overlay network (e.g. /16)
	// and not that of the individual host (e.g. /24)
	addr := &net.IPNet{
		IP:   vtepOverlayIP,
		Mask: fullOverlayMask,
	}

	if err := netlink.LinkSetHardwareAddr(dev.link, hwAddr); err != nil {
		return fmt.Errorf("configuring vtep hw addr: %s", err)
	}

	if err := dev.setAddr4(addr); err != nil {
		return fmt.Errorf("configuring vtep ip address: %s", err)
	}

	if err := netlink.LinkSetUp(dev.link); err != nil {
		return fmt.Errorf("failed to set interface %s to UP state: %s", dev.link.Attrs().Name, err)
	}

	if err := dev.PurgeAllRoutes(); err != nil {
		return err
	}

	// fully mask the address before adding the wide route
	routeSubnet := &net.IPNet{
		IP:   addr.IP.Mask(addr.Mask),
		Mask: addr.Mask,
	}
	return dev.AddRoute(routeSubnet, nil, netlink.SCOPE_LINK, nil)
}

// sets IP4 addr on link removing any existing ones first
func (dev *vxlanDevice) setAddr4(ipn *net.IPNet) error {
	if err := dev.PurgeAllAddresses(); err != nil {
		return err
	}

	addr := netlink.Addr{IPNet: ipn, Label: ""}
	if err := netlink.AddrAdd(dev.link, &addr); err != nil {
		return fmt.Errorf("failed to add IP address %s to %s: %s", ipn.String(), dev.link.Attrs().Name, err)
	}

	return nil
}

func (dev *vxlanDevice) MTU() int {
	return dev.link.Attrs().MTU
}
