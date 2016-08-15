package flannel

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
)

const (
	flannelSubnetRegex  = `FLANNEL_SUBNET=((?:[0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2})`
	flannelNetworkRegex = `FLANNEL_NETWORK=((?:[0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2})`
)

type NetworkInfo struct {
	FlannelSubnetFilePath string
}

func (l *NetworkInfo) DiscoverNetworkInfo() (string, string, error) {
	fileContents, err := ioutil.ReadFile(l.FlannelSubnetFilePath)
	if err != nil {
		return "", "", err
	}

	subnetMatches := regexp.MustCompile(flannelSubnetRegex).FindStringSubmatch(string(fileContents))
	if len(subnetMatches) < 2 {
		return "", "", fmt.Errorf("unable to parse flannel subnet file")
	}

	networkMatches := regexp.MustCompile(flannelNetworkRegex).FindStringSubmatch(string(fileContents))
	if len(networkMatches) < 2 {
		return "", "", fmt.Errorf("unable to parse flannel network from subnet file")
	}

	return subnetMatches[1], networkMatches[1], nil
}

type SubnetFileInfo struct {
	FullNetwork net.IPNet
	Subnet      net.IPNet
	MTU         int
	IPMasq      bool
}

func (info *SubnetFileInfo) MarshalJSON() ([]byte, error) {
	toMarshal := struct {
		FullNetwork string
		Subnet      string
		MTU         int
		IPMasq      bool
	}{
		FullNetwork: info.FullNetwork.String(),
		Subnet:      info.Subnet.String(),
		MTU:         info.MTU,
		IPMasq:      info.IPMasq,
	}
	return json.Marshal(toMarshal)
}

func (info *SubnetFileInfo) String() string {
	bytes, _ := info.MarshalJSON()
	return string(bytes)
}

func (info *SubnetFileInfo) WriteFile(path string) error {
	contents := fmt.Sprintf(`FLANNEL_NETWORK=%s
FLANNEL_SUBNET=%s
FLANNEL_MTU=%d
FLANNEL_IPMASQ=%v
`, info.FullNetwork.String(), info.Subnet.String(), info.MTU, info.IPMasq)

	dir, _ := filepath.Split(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	tempFile := filepath.Join(dir, "temp")
	err = ioutil.WriteFile(tempFile, []byte(contents), 0644)
	if err != nil {
		return err
	}

	// rename(2) the temporary file to the desired location so that it becomes
	// atomically visible with the contents
	return os.Rename(tempFile, path)
}
