package addr

import (
	"fmt"
	"net"
	// "github.com/containernetworking/cni/pkg/types"
)

// GenerateIPv6 is a function that autogenerates an IPv6 address based on the IPv4 one and the pod number.
func GenerateIPv6(ipv6, ipv4 net.IP, pod byte) (ip net.IPNet, err error) {
	len4 := len(ipv4)
	len6 := 16
	newaddr := make(net.IP, len6)

	// IPv6 prefix, 8 first octects.
	copy(newaddr[:7], ipv6[:7])
	// POD number
	newaddr[7] = pod
	// Host portion of the IPv6 address, last 4 octects -> IPv4
	copy(newaddr[len6-4:len6], ipv4[len4-4:len4])

	ip = net.IPNet{
		IP:   newaddr,
		Mask: net.CIDRMask(64, 128),
	}
	return ip, nil
}

// GetIPv4 is a funciton that takes a string IPv6 address and returns the correspondig
// pod number and IPv4 address as strings.
func GetIPv4(s string) (pod, ipv4 string, err error) {
	ipv6 := net.ParseIP(s)
	l := len(ipv6)
	if ipv6 == nil || l != 16 {
		return pod, ipv4, fmt.Errorf("could not parse %s", s)
	}
	pod = fmt.Sprintf("%v", ipv6[7])
	ipv4 = fmt.Sprintf("%v", ipv6[l-4:l])
	return pod, ipv4, nil
}
