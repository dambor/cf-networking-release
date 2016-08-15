package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"silk/hwaddr"
	"silk/models"
	"silk/subnet"
)

func ReadFile(configFilePath string) (*Silk, error) {
	if configFilePath == "" {
		return nil, fmt.Errorf("no path provided for config file")
	}

	configBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}

	conf := &Silk{}
	err = json.Unmarshal(configBytes, conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}

type Silk struct {
	Hosts []struct {
		Index    int    `json:"index"`
		PublicIP string `json:"public_ip"`
	} `json:"hosts"`
	ThisIndex             int    `json:"this_index"`
	VNI                   int    `json:"vni"`
	Port                  int    `json:"port"`
	Network               string `json:"network"`
	FlannelSubnetFilePath string `json:"flannel_subnet_file"`
}

type Parsed struct {
	VNI         int
	Port        int
	FullNetwork net.IPNet
	RemoteHosts []*models.NetHost
	ThisHost    *models.NetHost
}

func parseHost(publicIP string, partition subnet.Partition, hostIndex int) (*models.NetHost, error) {
	host := &models.NetHost{}
	host.PublicIP = net.ParseIP(publicIP)
	if host.PublicIP == nil {
		return nil, fmt.Errorf("error parsing %q as an IP")
	}
	var err error
	host.OverlaySubnet, err = partition.IndexToSubnet(hostIndex)
	if err != nil {
		return nil, fmt.Errorf("partition subnet: %s", err)
	}
	host.VtepOverlayIP = host.OverlaySubnet.IP
	host.VtepOverlayMAC = hwaddr.MACAddressFromIP(host.VtepOverlayIP)
	return host, nil
}

func Parse(conf *Silk) (*Parsed, error) {
	parsed := &Parsed{
		VNI:  conf.VNI,
		Port: conf.Port,
	}
	if conf.VNI < 0 || conf.VNI >= (1<<24) {
		return nil, fmt.Errorf("VNI must be between 0 and 2^24-1")
	}
	if conf.Port <= 0 || conf.Port >= (1<<16) {
		return nil, fmt.Errorf("Port must be between 0 and 2^16-1")
	}
	partition, err := subnet.NewPartition(conf.Network)
	if err != nil {
		return nil, fmt.Errorf("partitioning subnets: %s", err)
	}
	parsed.FullNetwork = partition.FullNetwork()
	for _, confHost := range conf.Hosts {
		parsedHost, err := parseHost(confHost.PublicIP, partition, confHost.Index)
		if err != nil {
			return nil, fmt.Errorf("parsing host: %s", err)
		}
		if confHost.Index == conf.ThisIndex {
			parsed.ThisHost = parsedHost
		} else {
			parsed.RemoteHosts = append(parsed.RemoteHosts, parsedHost)
		}
	}
	return parsed, nil
}
