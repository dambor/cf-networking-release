package backend

import (
	"fmt"
	"net"
	"silk/models"

	"github.com/vishvananda/netlink"
)

type Controller interface {
	OverlayMTU() int
	ConfigureDevice() error
	InstallRoutes(remoteHosts []*models.NetHost) error
}

type controller struct {
	ExternalInterface net.Interface
	Device            *vxlanDevice
	LocalConfig       *models.NetHost
	FullOverlay       net.IPNet
	devAttrs          *vxlanDeviceAttrs
}

func (c *controller) OverlayMTU() int {
	return c.ExternalInterface.MTU - 48
}

func New(vni int, port int, localConfig *models.NetHost, fullOverlay net.IPNet) (Controller, error) {
	externalIP := localConfig.PublicIP
	externalInterface, err := locateInterface(externalIP)
	if err != nil {
		return nil, fmt.Errorf("locating external interface: %s", err)
	}

	cont := &controller{
		ExternalInterface: externalInterface,
		LocalConfig:       localConfig,
		FullOverlay:       fullOverlay,
		devAttrs: &vxlanDeviceAttrs{
			vni:       vni,
			name:      fmt.Sprintf("silk.%v", vni),
			vtepIndex: externalInterface.Index,
			vtepAddr:  externalIP,
			vtepPort:  port,
		},
	}

	return cont, nil
}

func (c *controller) ConfigureDevice() error {
	dev, err := newVXLANDevice(c.devAttrs)
	if err != nil {
		return err
	}
	c.Device = dev

	if err := c.Device.Configure(
		c.LocalConfig.VtepOverlayIP,
		c.FullOverlay.Mask,
		c.LocalConfig.VtepOverlayMAC); err != nil {
		return fmt.Errorf("configuring vxlan device: %s", err)
	}

	return nil
}

func (c *controller) InstallRoutes(remoteHosts []*models.NetHost) error {
	for _, host := range remoteHosts {
		l3neigh := L3Neigh{
			OverlayIP:  host.VtepOverlayIP,
			OverlayMAC: host.VtepOverlayMAC,
		}
		l2neigh := L2Neigh{
			OverlayMAC: host.VtepOverlayMAC,
			UnderlayIP: host.PublicIP,
		}
		err := c.Device.SetL3(l3neigh)
		if err != nil {
			return fmt.Errorf("set l3: %s", err)
		}
		err = c.Device.SetL2(l2neigh)
		if err != nil {
			return fmt.Errorf("set l2: %s", err)
		}
		err = c.Device.AddRoute(
			&host.OverlaySubnet, host.OverlaySubnet.IP, netlink.SCOPE_UNIVERSE, c.LocalConfig.VtepOverlayIP)
		if err != nil {
			return err
		}
	}
	return nil
}
