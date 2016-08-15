package hwaddr

import "net"

// MACAddressFromIP generates 48 bit virtual mac addresses based on the IP4 input.
func MACAddressFromIP(ip net.IP) net.HardwareAddr {
	if ip.To4() == nil {
		panic("only ipv4 is supported")
	}

	PrivateMACPrefix := []byte{0x0a, 0x00}
	const ipRelevantByteLen = 4
	ipByteLen := len(ip)
	return (net.HardwareAddr)(
		append(PrivateMACPrefix, ip[ipByteLen-ipRelevantByteLen:ipByteLen]...),
	)
}
