package nebula

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
)

type udpAddr struct {
	IP   net.IP
	Port uint16
}

func NewUDPAddr(ip net.IP, port uint16) *udpAddr {
	addr := udpAddr{IP: make([]byte, len(ip)), Port: port}
	copy(addr.IP, ip)
	return &addr
}

func NewUDPAddrFromString(s string) *udpAddr {
	ip, port, err := parseIPAndPort(s)
	//TODO: handle err
	_ = err
	return &udpAddr{IP: ip, Port: port}
}

func (ua *udpAddr) Equals(t *udpAddr) bool {
	if t == nil || ua == nil {
		return t == nil && ua == nil
	}
	return ua.IP.Equal(t.IP) && ua.Port == t.Port
}

func (ua *udpAddr) String() string {
	return net.JoinHostPort(ua.IP.String(), fmt.Sprintf("%v", ua.Port))
}

func (ua *udpAddr) MarshalJSON() ([]byte, error) {
	return json.Marshal(m{"ip": ua.IP, "port": ua.Port})
}

func (ua *udpAddr) Copy() *udpAddr {
	nu := udpAddr{
		Port: ua.Port,
		IP:   make(net.IP, len(ua.IP)),
	}

	copy(nu.IP, ua.IP)
	return &nu
}

func parseIPAndPort(s string) (net.IP, uint16, error) {
	rIp, sPort, err := net.SplitHostPort(s)
	if err != nil {
		return nil, 0, err
	}

	addr, err := net.ResolveIPAddr("ip", rIp)
	if err != nil {
		return nil, 0, err
	}

	iPort, err := strconv.Atoi(sPort)
	if err != nil {
		return nil, 0, err
	}

	return addr.IP, uint16(iPort), nil
}
