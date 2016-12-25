package backend

import (
	"fmt"
	"net"
	"silk-controller/models"

	"github.com/vishvananda/netlink"

	multierror "github.com/hashicorp/go-multierror"
)

type VtepFactory struct {
	VNI         int
	Port        int
	FullNetwork net.IPNet
}

func (f *VtepFactory) NewVtep(localConfig models.Route) (Vtep, error) {
	conf, err := netHostFromControllerRoute(localConfig)
	if err != nil {
		return nil, err
	}
	return f.newVtep(conf)
}

func (f *VtepFactory) newVtep(localConfig *NetHost) (Vtep, error) {
	vtepName := fmt.Sprintf("%s.%d", "flannel", f.VNI)
	bridgeName := fmt.Sprintf("cni-%s0", "flannel")
	cont := &vtep{
		dependentLinks:     []string{vtepName, bridgeName},
		localVtepOverlayIP: localConfig.VtepOverlayIP,
		fullNetwork:        f.FullNetwork,
		localSubnet:        localConfig.OverlaySubnet,
	}

	err := cont.deleteDependentLinks()
	if err != nil {
		return nil, fmt.Errorf("cleanup links: %s", err)
	}

	cont.device, err = newVXLANDevice(vtepName, f.VNI, localConfig.PublicIP, f.Port)
	if err != nil {
		return nil, err
	}

	if err := cont.device.Configure(
		localConfig.VtepOverlayIP,
		f.FullNetwork.Mask,
		localConfig.VtepOverlayMAC); err != nil {
		return nil, fmt.Errorf("configuring vxlan device: %s", err)
	}

	return cont, nil
}

type Vtep interface {
	OverlayMTU() int
	InstallRoutes(remoteHosts []models.Route) error
	Teardown() error
	FullNetwork() net.IPNet
	LocalSubnet() net.IPNet
}

type vtep struct {
	device             *vxlanDevice
	dependentLinks     []string
	localVtepOverlayIP net.IP
	fullNetwork        net.IPNet
	localSubnet        net.IPNet
}

func (c *vtep) InstallRoutes(remoteHosts []models.Route) error {
	hosts, err := netHostsFromControllerRoutes(remoteHosts)
	if err != nil {
		return err
	}
	return c.installRoutes(hosts)
}

func (c *vtep) installRoutes(remoteHosts []*NetHost) error {
	for _, host := range remoteHosts {
		err := c.device.UpsertARP(host.VtepOverlayIP, host.VtepOverlayMAC)
		if err != nil {
			return fmt.Errorf("set l3: %s", err)
		}
		err = c.device.UpsertFDB(host.VtepOverlayMAC, host.PublicIP)
		if err != nil {
			return fmt.Errorf("set l2: %s", err)
		}
		err = c.device.AddRoute(
			&host.OverlaySubnet, host.OverlaySubnet.IP,
			netlink.SCOPE_UNIVERSE, c.localVtepOverlayIP,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *vtep) Teardown() error {
	return c.deleteDependentLinks()
}

func (c *vtep) FullNetwork() net.IPNet {
	return c.fullNetwork
}

func (c *vtep) LocalSubnet() net.IPNet {
	return c.localSubnet
}

func (c *vtep) OverlayMTU() int {
	return c.device.MTU()
}

func (c *vtep) deleteDependentLinks() error {
	var result error
	for _, linkName := range c.dependentLinks {
		if err := removeLinkByName(linkName); err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result
}
