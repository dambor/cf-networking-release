package main

import (
	"flag"
	"fmt"
	"lib/flannel"
	"net"
	"os"
	"silk/backend"
	"silk/config"

	"code.cloudfoundry.org/lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

func main() {
	logger := lager.NewLogger("silk")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

	if err := mainWithError(logger); err != nil {
		logger.Error("fatal-error", err)
		os.Exit(1)
	}
}

func mainWithError(logger lager.Logger) error {
	var configFilePath string
	var runOnce bool
	flag.StringVar(&configFilePath, "config-file", "", "path to config file")
	flag.BoolVar(&runOnce, "run-once", false, "run once and then quit.")
	flag.Parse()

	rawConfig, err := config.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("reading config: %s", err)
	}
	logger.Info("read-config", lager.Data{"config": rawConfig})

	parsedConfig, err := config.Parse(rawConfig)
	if err != nil {
		return fmt.Errorf("parsing config: %s", err)
	}
	logger.Info("parsed-config", lager.Data{"config": parsedConfig})

	controller, err := backend.New(
		parsedConfig.VNI,
		parsedConfig.Port,
		parsedConfig.ThisHost,
		parsedConfig.FullNetwork,
	)
	if err != nil {
		return fmt.Errorf("initializing vxlan controller: %s", err)
	}
	logger.Info("initialized-vxlan-controller", lager.Data{"controller": controller})

	flannelSubnetFileContents := &flannel.SubnetFileInfo{
		FullNetwork: parsedConfig.FullNetwork,
		MTU:         controller.OverlayMTU(),
		IPMasq:      false,
		Subnet:      getFirstAllocatableAddress(parsedConfig.ThisHost.OverlaySubnet),
	}
	err = flannelSubnetFileContents.WriteFile(rawConfig.FlannelSubnetFilePath)
	if err != nil {
		return fmt.Errorf("writing flannel file: %s", err)
	}
	logger.Info("wrote-flannel-subnet-file")

	err = controller.ConfigureDevice()
	if err != nil {
		return fmt.Errorf("configuring vxlan device: %s", err)
	}
	logger.Info("configured vxlan device")

	err = controller.InstallRoutes(parsedConfig.RemoteHosts)
	if err != nil {
		return fmt.Errorf("installing routes: %s", err)
	}
	logger.Info("installed routes")

	if runOnce {
		return nil
	}

	upkeep := ifrit.RunFunc(func(sigChan <-chan os.Signal, ready chan<- struct{}) error {
		close(ready)

		<-sigChan

		return nil
	})
	members := grouper.Members{{"silk-upkeep", upkeep}}

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))
	logger.Info("starting")
	err = <-monitor.Wait()
	if err != nil {
		logger.Fatal("ifrit monitor", err)
	}

	return nil
}

func getFirstAllocatableAddress(subnet net.IPNet) net.IPNet {
	newIP := make(net.IP, len(subnet.IP))
	copy(newIP, subnet.IP)
	i := len(subnet.IP) - 1
	newIP[i]++
	return net.IPNet{
		IP:   newIP,
		Mask: subnet.Mask,
	}
}
