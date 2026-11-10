package capacity

import "os"

// Orbit bridge network defaults (technical-design.md §6).
// 172.28.0.0/16: RFC1918, not Docker's default bridge (172.17), and an
// uncommon home-LAN range.
const (
	NetName       = "orbit-net"
	DefaultSubnet = "172.28.0.0/16"
	SubnetEnvVar  = "ORBIT_NET_SUBNET"
)

// Subnet returns the configured bridge subnet.
func Subnet() string {
	if s := os.Getenv(SubnetEnvVar); s != "" {
		return s
	}
	return DefaultSubnet
}
