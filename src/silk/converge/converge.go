package converge

import (
	"fmt"
	"net"
	"os"
	"time"

	"code.cloudfoundry.org/lager"

	"silk-controller/client"
	"silk-controller/models"

	"silk/backend"
)

type flannelFileWriter interface {
	Write(fullNet, localSubnet net.IPNet, mtu int) error
}

type vtepFactory interface {
	NewVtep(localConfig models.Route) (backend.Vtep, error)
}

type Poller struct {
	Logger       lager.Logger
	PollInterval time.Duration

	ControllerClient client.ControllerClient

	VtepFactory       vtepFactory
	FlannelFileWriter flannelFileWriter

	lease  *models.Lease
	routes []models.Route
	vtep   backend.Vtep
}

func (m *Poller) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	if err := m.setup(); err != nil {
		m.Logger.Error("converge-setup", err)
		return err
	}
	close(ready)

	for {
		select {
		case <-signals:
			return m.teardown()
		case <-time.After(m.PollInterval):
			if err := m.renewAndUpdateRoutes(); err != nil {
				m.Logger.Error("converge-cycle", err)
				continue
			}
		}
	}
}

func (m *Poller) updateRoutes(routes []models.Route) error {
	defer m.Logger.Debug("install-routes-done")

	err := m.vtep.InstallRoutes(routes)
	if err != nil {
		m.Logger.Error("install-routes", err, lager.Data{"old": m.routes, "new": routes})
		return fmt.Errorf("installing routes: %s", err)
	}

	m.routes = routes
	return nil
}

func (m *Poller) setup() error {
	defer m.Logger.Info("setup-complete")
	m.Logger.Info("acquiring-lease")
	resp, err := m.ControllerClient.Acquire()
	if err != nil {
		return fmt.Errorf("acquiring subnet lease: %s", err)
	}
	m.Logger.Info("acquired-lease", lager.Data{"lease": resp})

	m.vtep, err = m.VtepFactory.NewVtep(resp.Self.Route)
	if err != nil {
		return fmt.Errorf("initializing vxlan vtep: %s", err)
	}
	m.Logger.Info("initialized-vxlan-vtep")

	err = m.FlannelFileWriter.Write(
		m.vtep.FullNetwork(),
		m.vtep.LocalSubnet(),
		m.vtep.OverlayMTU(),
	)
	if err != nil {
		return fmt.Errorf("writing flannel file: %s", err)
	}
	m.Logger.Info("wrote-flannel-subnet-file")

	m.lease = &resp.Self

	if err := m.updateRoutes(resp.Routes); err != nil {
		return err
	}

	return nil
}

func (m *Poller) teardown() error {
	defer m.Logger.Info("teardown-done")

	m.Logger.Info("vtep-teardown")
	if err := m.vtep.Teardown(); err != nil {
		m.Logger.Error("vtep-teardown", err)
	}

	if m.lease == nil {
		return nil
	}

	m.Logger.Info("teardown.delete-lease", lager.Data{"lease": m.lease})
	err := m.ControllerClient.Delete(m.lease.ID)
	if err != nil {
		m.Logger.Error("teardown.delete-lease", err)
		return err
	}
	return nil
}

func (m *Poller) renewAndUpdateRoutes() error {
	m.Logger.Info("renewing-lease", lager.Data{"lease": m.lease})
	resp, err := m.ControllerClient.Renew(*m.lease)
	if err != nil {
		return fmt.Errorf("renewing subnet lease: %s", err)
	}
	m.Logger.Info("renewed-lease", lager.Data{"lease": resp})

	if resp.Self.Subnet != m.lease.Subnet {
		err := fmt.Errorf("renewal returned different subnet than one already configured")
		m.Logger.Error("renewal-subnet-mismatch", err, lager.Data{"new": resp.Self.Subnet, "original": m.lease.Subnet})
		return err
	}

	if err := m.updateRoutes(resp.Routes); err != nil {
		return err
	}

	return nil
}
