package subnet

import (
	"fmt"
	"net"
	"strings"
)

type Partition interface {
	IndexToSubnet(index int) (net.IPNet, error)
	FullNetwork() net.IPNet
}

type partition16to24 struct {
	firstOctet  byte
	secondOctet byte
	mask        net.IPMask
}

func NewPartition(slash16CIDR string) (Partition, error) {
	_, fullNetwork, err := net.ParseCIDR(slash16CIDR)
	if err != nil {
		return nil, fmt.Errorf("parsing cidr %q: %s", slash16CIDR, err)
	}
	if len(fullNetwork.IP) != net.IPv4len {
		return nil, fmt.Errorf("expecting v4 address")
	}
	if fullNetwork.IP[3] != 0 {
		return nil, fmt.Errorf("expecting fourth octet to be 0")
	}
	if fullNetwork.IP[2] != 0 {
		return nil, fmt.Errorf("expecting 3rd octet to be 0")
	}
	if !strings.HasSuffix(slash16CIDR, "/16") {
		return nil, fmt.Errorf("expecting %q to be a /16", slash16CIDR)
	}

	return &partition16to24{
		firstOctet:  fullNetwork.IP[0],
		secondOctet: fullNetwork.IP[1],
		mask:        net.CIDRMask(24, 32),
	}, nil
}

func (p *partition16to24) IndexToSubnet(index int) (net.IPNet, error) {
	if index < 0 || index > 254 {
		return net.IPNet{}, fmt.Errorf("index must be between 0 and 254")
	}
	thirdOctet := 1 + byte(index)
	return net.IPNet{
		IP:   net.IPv4(p.firstOctet, p.secondOctet, thirdOctet, 0),
		Mask: p.mask,
	}, nil
}

func (p *partition16to24) FullNetwork() net.IPNet {
	return net.IPNet{
		IP:   net.IPv4(p.firstOctet, p.secondOctet, 0, 0),
		Mask: net.CIDRMask(16, 32),
	}
}
