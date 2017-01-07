package ippacket

import (
	"net"
	"sync"
	"strconv"
	"errors"
)


const (
	MAXClient   = 200
)

const (
	IPStart     = 2
	IPEnd       = (IPStart + MAXClient)
)

type NAT struct {
	sync.Mutex
	iptable       map[RawIPv4]net.Conn
	ippool        map[net.Conn]RawIPv4
	allocCount    int
	prefix        string
}

func NewNAT(subnet string) (*NAT) {
	nat := NAT{
		iptable:       make(map[RawIPv4]net.Conn, MAXClient),
		ippool:        make(map[net.Conn]RawIPv4, MAXClient),
		allocCount:    0,
		prefix:        subnet,
	}

	return &nat
}


func (nat *NAT) NewClient(conn net.Conn) (string, error) {
	nat.Lock()
	defer nat.Unlock()

	if nat.allocCount >= MAXClient {
		return "", errors.New("No more IP for allocate")
	}
	alloc := IPStart + nat.allocCount
	nat.allocCount++

	ip := nat.prefix + "." + strconv.Itoa(alloc)
	rawip := parseIP(ip)
	nat.ippool[conn] = rawip
	nat.iptable[rawip] = conn

	return ip, nil
}

func (nat *NAT) RemoveClient(conn net.Conn) () {
	nat.Lock()
	defer nat.Unlock()

	rawip, ok := nat.ippool[conn]
	if ok {
		delete(nat.ippool, conn)
		delete(nat.iptable, rawip)
		nat.allocCount--
	}
}

func (nat *NAT) GetClientRaw(rawip RawIPv4) (net.Conn) {
	conn, ok := nat.iptable[rawip]
	if ok {
		return conn
	}
	return nil
}

func parseIP(s string) (RawIPv4) {
	var ip uint32

	addr := net.ParseIP(s)
	p := ([]byte)(addr)

	ip = uint32(p[12])
	ip |= uint32(p[13]) << 8
	ip |= uint32(p[14]) << 16
	ip |= uint32(p[15]) << 24

	return (RawIPv4)(ip)
}


