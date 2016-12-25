package backend

import (
	"fmt"
	"net"
	"strings"

	"github.com/vishvananda/netlink"
)

func locateInterface(toFind net.IP) (net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return net.Interface{}, err
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return net.Interface{}, err
		}

		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				return net.Interface{}, err
			}
			if ip.String() == toFind.String() {
				return iface, nil
			}
		}
	}

	return net.Interface{}, fmt.Errorf("no interface with address %s", toFind.String())
}

func isNotFoundError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
}

func removeLinkByName(name string) error {
	existing, err := netlink.LinkByName(name)
	if isNotFoundError(err) {
		return nil
	}
	if err != nil {
		return err
	}

	return netlink.LinkDel(existing)
}

func addLink(vxlan *netlink.Vxlan) error {
	if err := netlink.LinkAdd(vxlan); err != nil {
		return fmt.Errorf("adding vxlan link: %s", err)
	}

	// re-find the link to get extra metadata
	foundLink, err := netlink.LinkByIndex(vxlan.Index)
	if err != nil {
		return fmt.Errorf("can't locate created vxlan device")
	}

	foundVxlanLink, ok := foundLink.(*netlink.Vxlan)
	if !ok {
		return fmt.Errorf("created device is not vxlan")
	}

	*vxlan = *foundVxlanLink
	return nil
}
