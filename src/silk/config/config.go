package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"silk/subnet"
	"time"
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
	VNI                   int    `json:"vni"`
	VtepPort              int    `json:"vtep_port"`
	VtepIP                string `json:"vtep_ip"`
	PollInterval          int    `json:"poll_interval"`
	Network               string `json:"network"`
	FlannelSubnetFilePath string `json:"flannel_subnet_file"`
	Controller            struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
}

type Parsed struct {
	VNI               int
	VtepPort          int
	VtepIP            net.IP
	FullNetwork       net.IPNet
	ControllerBaseURL string
	PollInterval      time.Duration
}

func Parse(conf *Silk) (*Parsed, error) {
	parsed := &Parsed{
		VNI:      conf.VNI,
		VtepPort: conf.VtepPort,
		VtepIP:   net.ParseIP(conf.VtepIP),
	}
	if conf.VNI < 0 || conf.VNI >= (1<<24) {
		return nil, fmt.Errorf("VNI must be between 0 and 2^24-1")
	}
	if conf.VtepPort <= 0 || conf.VtepPort >= (1<<16) {
		return nil, fmt.Errorf("VtepPort must be between 0 and 2^16-1")
	}
	if parsed.VtepIP == nil {
		return nil, fmt.Errorf("failed parsing VtepIP")
	}

	if conf.Controller.Host == "" || conf.Controller.Port == 0 {
		return nil, fmt.Errorf("controller host and port are required")
	}
	parsed.ControllerBaseURL = fmt.Sprintf("http://%s:%d", conf.Controller.Host, conf.Controller.Port)

	if conf.PollInterval < 1 {
		return nil, fmt.Errorf("controller poll interval must be at least 1 second")
	}
	parsed.PollInterval = time.Duration(conf.PollInterval) * time.Second

	partition, err := subnet.NewPartition(conf.Network)
	if err != nil {
		return nil, fmt.Errorf("partitioning subnets: %s", err)
	}
	parsed.FullNetwork = partition.FullNetwork()
	return parsed, nil
}
