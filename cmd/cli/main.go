package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/nleiva/cni-plugin/cmd/addr"
)

func main() {
	ipv6 := flag.String("ipv6", "", "Pod IPv6 address to translate. Ex: '2001:db8::ca:fe'")
	ipv4 := flag.String("ipv4", "", "Pod IPv4 address to translate. Ex: '10.240.30.1'")
	pfx := flag.String("pfx", "", "IPv6 prefix assigned. Ex: '2001:db8::/32' ")
	flag.Parse()

	if *ipv4 != "" {
		if *pfx == "" {
			log.Fatalln("Need to specify an IPv6 prefix")
		}
		podip := net.ParseIP(*ipv4)
		prefix, _, err := net.ParseCIDR(*pfx)
		if err != nil {
			log.Fatalf("could not parse the IPv6 prefix: %v\n", err)
		}
		ip, err := addr.GenerateIPv6(prefix, podip, podip[len(podip)-2])
		fmt.Printf("IPv6 is: %s\n", ip.String())
	}
	if *ipv6 != "" {
		pod, ipv4, err := addr.GetIPv4(*ipv6)
		if err != nil {
			log.Fatalf("could not get the IPv4 addr: %v\n", err)
		}
		fmt.Printf("Result\nIPv4: %s, Pod: %s\n", ipv4, pod)
	}

}
