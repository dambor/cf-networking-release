package main

import (
	"flag"
	"fmt"
	"lib/flannel"
	"net"
	"os"
	"silk-controller/client"
	"silk/backend"
	"silk/config"
	"silk/converge"

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
	flag.StringVar(&configFilePath, "config-file", "", "path to config file")
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

	convergePoller := &converge.Poller{
		Logger:           logger.Session("poller"),
		PollInterval:     parsedConfig.PollInterval,
		ControllerClient: client.New(logger, parsedConfig.ControllerBaseURL, parsedConfig.VtepIP.String()),
		FlannelFileWriter: &flannelFileWriter{
			FilePath: rawConfig.FlannelSubnetFilePath,
		},
		VtepFactory: &backend.VtepFactory{
			Port:        parsedConfig.VtepPort,
			VNI:         parsedConfig.VNI,
			FullNetwork: parsedConfig.FullNetwork,
		},
	}

	members := grouper.Members{{"silk-upkeep", convergePoller}}

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))
	logger.Info("starting")
	err = <-monitor.Wait()
	if err != nil {
		logger.Fatal("ifrit monitor", err)
	}

	return nil
}

type flannelFileWriter struct {
	FilePath string
}

func (f *flannelFileWriter) Write(fullNet, localNet net.IPNet, mtu int) error {
	flannelSubnetFileContents := &flannel.SubnetFileInfo{
		FullNetwork: fullNet,
		Subnet:      getFirstAllocatableAddress(localNet),
		MTU:         mtu,
		IPMasq:      false,
	}
	return flannelSubnetFileContents.WriteFile(f.FilePath)
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
