package ippacket

import (
	"net"
)

type Packet []byte

type RawIPv4 uint32

func (p Packet) Dst() (net.IP) {
	return net.IPv4(p[16], p[17], p[18], p[19])
}

func (p Packet) DstRaw() (RawIPv4) {
	var ip uint32
	ip = uint32(p[16])
	ip |= uint32(p[17]) << 8
	ip |= uint32(p[18]) << 16
	ip |= uint32(p[19]) << 24
	return (RawIPv4)(ip)
}

func (p Packet) Src() (net.IP) {
	return net.IPv4(p[12], p[13], p[14], p[15])
}

func (p Packet) SrcRaw() (RawIPv4) {
	var ip uint32
	ip = uint32(p[12])
	ip |= uint32(p[13]) << 8
	ip |= uint32(p[14]) << 16
	ip |= uint32(p[15]) << 24
	return (RawIPv4)(ip)
}

func (p Packet) Protocol() (int) {
	return int(p[9])
}

func (p Packet) Payload() ([]byte) {
	hdrlen := int(p[0]&0x0f) << 2

	return p[hdrlen:]
}



