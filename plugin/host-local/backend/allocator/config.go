// Copyright 2015 CNI authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package allocator

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/containernetworking/cni/pkg/types"
	types020 "github.com/containernetworking/cni/pkg/types/020"
)

// The top-level network config, just so we can get the IPAM block
type Net struct {
	Name       string      `json:"name"`
	CNIVersion string      `json:"cniVersion"`
	IPAM       *IPAMConfig `json:"ipam"`
	Args       *struct {
		A *IPAMArgs `json:"cni"`
	} `json:"args"`
}

// IPAMConfig represents the IP related network configuration.
// This nests Range because we initially only supported a single
// range directly, and wish to preserve backwards compatability
type IPAMConfig struct {
	*Range
	Name       string
	Type       string         `json:"type"`
	Routes     []*types.Route `json:"routes"`
	DataDir    string         `json:"dataDir"`
	ResolvConf string         `json:"resolvConf"`
	Ranges     []RangeSet     `json:"ranges"`
	IPArgs     []net.IP       `json:"-"` // Requested IPs from CNI_ARGS and args
	IPv6       *string        `json:"ipv6,omitempty"`
}

type IPAMEnvArgs struct {
	types.CommonArgs
	IP net.IP `json:"ip,omitempty"`
}

type IPAMArgs struct {
	IPs []net.IP `json:"ips"`
}

type RangeSet []Range

type Range struct {
	RangeStart net.IP      `json:"rangeStart,omitempty"` // The first ip, inclusive
	RangeEnd   net.IP      `json:"rangeEnd,omitempty"`   // The last ip, inclusive
	Subnet     types.IPNet `json:"subnet"`
	Gateway    net.IP      `json:"gateway,omitempty"`
}

// NewIPAMConfig creates a NetworkConfig from the given network name.
func LoadIPAMConfig(bytes []byte, envArgs string) (*IPAMConfig, string, error) {
	n := Net{}
	if err := json.Unmarshal(bytes, &n); err != nil {
		//log.Printf("error decoding the config: %v", err)
		//if e, ok := err.(*json.SyntaxError); ok {
		//	log.Printf("syntax error at byte offset %d", e.Offset)
		//}
		//log.Printf("the config: %q", bytes)
		return nil, "", err
	}

	if n.IPAM == nil {
		return nil, "", fmt.Errorf("IPAM config missing 'ipam' key")
	}

	// Parse custom IP from both env args *and* the top-level args config
	if envArgs != "" {
		e := IPAMEnvArgs{}
		err := types.LoadArgs(envArgs, &e)
		if err != nil {
			return nil, "", err
		}

		if e.IP != nil {
			n.IPAM.IPArgs = []net.IP{e.IP}
		}
	}

	if n.Args != nil && n.Args.A != nil && len(n.Args.A.IPs) != 0 {
		n.IPAM.IPArgs = append(n.IPAM.IPArgs, n.Args.A.IPs...)
	}

	for idx, _ := range n.IPAM.IPArgs {
		if err := canonicalizeIP(&n.IPAM.IPArgs[idx]); err != nil {
			return nil, "", fmt.Errorf("cannot understand ip: %v", err)
		}
	}

	// If a single range (old-style config) is specified, prepend it to
	// the Ranges array
	if n.IPAM.Range != nil && n.IPAM.Range.Subnet.IP != nil {
		n.IPAM.Ranges = append([]RangeSet{{*n.IPAM.Range}}, n.IPAM.Ranges...)
	}
	n.IPAM.Range = nil

	if len(n.IPAM.Ranges) == 0 {
		return nil, "", fmt.Errorf("no IP ranges specified")
	}

	// Validate all ranges
	numV4 := 0
	numV6 := 0
	for i, _ := range n.IPAM.Ranges {
		if err := n.IPAM.Ranges[i].Canonicalize(); err != nil {
			return nil, "", fmt.Errorf("invalid range set %d: %s", i, err)
		}

		if n.IPAM.Ranges[i][0].RangeStart.To4() != nil {
			numV4++
		} else {
			numV6++
		}
	}

	// CNI spec 0.2.0 and below supported only one v4 and v6 address.
	// We now automatically assign the ipv6 subnet, so don't accept manual config.
	if numV4 > 1 || numV6 > 0 {
		for _, v := range types020.SupportedVersions {
			if n.CNIVersion == v {
				return nil, "", fmt.Errorf("CNI version %v does not support more than 1 address per family", n.CNIVersion)
			}
		}
	}
	if n.IPAM.IPv6 == nil {
		// Default prefix
		dpfx := "2001:db8::/32"
		n.IPAM.IPv6 = &dpfx
	}

	ipv6, _, err := net.ParseCIDR(*n.IPAM.IPv6)
	if err != nil {
		fmt.Println(err)
	}
	// We know Ranges is not equal to 0 and no IPv6 per previous checks
	ipv4 := n.IPAM.Ranges[0][0].Subnet.IP

	newIP, err := GenerateIPv6(ipv6, ipv4, ipv4[len(ipv4)-2])
	if err != nil {
		fmt.Println(err)
	}

	// Automatically assign an IPv6 prefix
	rs := RangeSet{
		Range{
			// Subnet: mustSubnet(n.IPAM.IPv6),
			Subnet: newIP,
		},
	}
	err = rs[0].Canonicalize()
	if err != nil {
		fmt.Printf("could not canonicalize range: %v", err)
	}
	n.IPAM.Ranges = append(n.IPAM.Ranges, rs)

	// Add IPv6 default route
	// v6gw, err := GenerateIPv6(ipv6, net.IP{0, 0, 0, 1}, ipv4[len(ipv4)-2])
	// if err != nil {
	// 	fmt.Println(err)
	// }

	r := &types.Route{
		Dst: net.IPNet{
			IP:   net.ParseIP("::"),
			Mask: net.CIDRMask(0, 128),
		},
		// GW: v6gw.IP,
	}
	n.IPAM.Routes = append(n.IPAM.Routes, r)

	// Check for overlaps
	l := len(n.IPAM.Ranges)
	for i, p1 := range n.IPAM.Ranges[:l-1] {
		for j, p2 := range n.IPAM.Ranges[i+1:] {
			if p1.Overlaps(&p2) {
				return nil, "", fmt.Errorf("range set %d overlaps with %d", i, (i + j + 1))
			}
		}
	}

	// Copy net name into IPAM so not to drag Net struct around
	n.IPAM.Name = n.Name

	return n.IPAM, n.CNIVersion, nil
}

// GenerateIPv6 is a function that autogenerates an IPv6 address based on the IPv4 one and the pod number.
func GenerateIPv6(ipv6, ipv4 net.IP, pod byte) (ip types.IPNet, err error) {
	len4 := len(ipv4)
	len6 := 16
	newaddr := make(net.IP, len6)

	// IPv6 prefix, 8 first octects.
	copy(newaddr[:7], ipv6[:7])
	// POD number
	newaddr[7] = pod
	// Host portion of the IPv6 address, last 4 octects -> IPv4
	copy(newaddr[len6-4:len6], ipv4[len4-4:len4])

	ip = types.IPNet{
		IP:   newaddr,
		Mask: net.CIDRMask(64, 128),
	}
	return ip, nil
}
