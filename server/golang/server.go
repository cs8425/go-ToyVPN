package main

import (
	"encoding/binary"
	"errors"
	"net"
	"io"
	"log"
	"flag"
//	"fmt"
	"runtime"
	"strconv"
	"os"

	"./ippacket"
	"./water"
)


// There are several ways to play with this program. Here we just give an
// example for the simplest scenario. Let us say that a Linux box has a
// public IPv4 address on eth0. Please try the following steps and adjust
// the parameters when necessary.
//
// # Enable IP forwarding
// echo 1 > /proc/sys/net/ipv4/ip_forward
//
// # Pick a range of private addresses and perform NAT over eth0.
// iptables -t nat -A POSTROUTING -s 10.0.0.0/8 -o eth0 -j MASQUERADE
//
// # Create a TUN interface.
// ip tuntap add dev tun2 mode tun
//
// # ** if your box didn't support tuntap, use this command instead
// tunctl -n -t tun2
//
// # Set the addresses and bring up the interface.
// ifconfig tun2 10.0.0.0/8 up
//
// # Create a server on port 23456 with shared secret "test123456".
// ./server -bind ":23456" -tun tun2 -m 1400 -s test123456
//
// This program allow multiple sessions.
// Since this program is designed for demonstration purpose, it performs neither strong
// authentication nor encryption. DO NOT USE IT IN PRODUCTION!


const (
	headerSize   = 2
)

var (
    subnet    = flag.String("local", "10.0.0", "Local tun interface IP prefix like 10.0.0")
    DNS       = flag.String("d", "8.8.8.8", "DNS server IP like 8.8.8.8")
    bind      = flag.String("bind", ":23456", "bind port for communication")

	DEV       = flag.String("tun", "tun2", "tun path")
	MTU       = flag.Int("m", 1400, "MTU size")

	secret    = flag.String("s", "test123456", "Shared Secret")

	verbosity = flag.Int("v", 2, "verbosity")
)

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU() + 2)

	nat := ippacket.NewNAT(*subnet)

	iface, err := water.NewTUN(*DEV)
	if err != nil {
		log.Fatalln("Unable to allocate TUN interface:", err)
	}
    Vlogln(1, "Interface allocated:", iface.Name())

	var packets = make(chan []byte, 1024)
	go func() {
		for {
			buf := <- packets
			// write to TUN interface
			iface.Write(buf)
		}
	}()

	go func() {
		packet := make([]byte, *MTU)
		for {
			n, err := iface.Read(packet)
			if err != nil {
				break
			}

			var header = (ippacket.Packet)(packet[:n])
			Vlogf(3, "-------Send %d bytes\n", n)
			Vlogf(3, "Dst: %s\n", header.Dst())
			Vlogf(3, "Src: %s\n", header.Src())
			Vlogf(3, "Protocol: % x\n", header.Protocol())

			// real send
			conn := nat.GetClientRaw(header.DstRaw())
			if conn != nil {
				buf := []byte{byte(n & 0xFF), byte((n >> 8) & 0xFF)}
				conn.Write(append(buf, packet[:n]...))
			} else {
//				Vlogf(4, "Payload: % x\n", header.Payload())
				Vlogln(2, "IP not found:", header.Dst())
			}
			Vlogf(3, "-------\n")
		}
	}()

	// Listen for incoming connections.
	l, err := net.Listen("tcp", *bind)
	if err != nil {
		Vlogln(1, "Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	log.Println("Listening on " + *bind)
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			Vlogln(2, "Error accepting: ", err.Error())
			//os.Exit(1)
		}
		Vlogln(2, "accepting:", conn.RemoteAddr())
		go handleRequest(conn, nat, packets)
	}

}

func handleRequest(conn net.Conn, nat *ippacket.NAT, queue chan []byte) {
	defer conn.Close()
	ok := handshake(conn, nat)
	if ok != true {
		Vlogln(2, "Error handshake!")
		return
	}
	defer func() {
		nat.RemoveClient(conn)
		Vlogln(2, "conn close:", conn.RemoteAddr())
	}()

	buf := make([]byte, *MTU + headerSize)
	for {
		if f, err := readFrame(conn, buf); err == nil {
			queue <- f
		} else {
			Vlogln(2, "Error readFrame()", err)
			return
		}
	}
}

func readFrame(conn net.Conn, buffer []byte) (f []byte, err error) {
	if _, err := io.ReadFull(conn, buffer[:headerSize]); err != nil {
		return f, errors.New("readFrame header: " + err.Error())
	}

	if length := binary.LittleEndian.Uint16(buffer[0:]); length > 0 {
		if _, err := io.ReadFull(conn, buffer[headerSize:headerSize+length]); err != nil {
			return f, errors.New("readFrame data: " + err.Error())
		}
		f = buffer[headerSize : headerSize+length]
	}
	return f, nil
}

func handshake(conn net.Conn, nat *ippacket.NAT) (bool) {
	buf := make([]byte, 2)
	n, err := conn.Read(buf)
	if err != nil {
		Vlogln(2, "Error handshake reading:", err.Error())
		return false
	}

	secretlen := len(*secret)
	keylen := int(buf[1])
	if buf[0] != 0 || keylen != secretlen {
		Vlogln(2, "Error handshake secret length!", keylen)
		return false
	}

	buf = make([]byte, secretlen)
	n, err = conn.Read(buf)
	if err != nil {
		Vlogln(2, "Error handshake secret reading:", n, err.Error())
		return false
	}

	cmpstr := string(buf)
	Vlogln(2, "handshake secret:", cmpstr, cmpstr == *secret)
	if cmpstr == *secret {
		ip, err := nat.NewClient(conn)
		if err != nil {
			Vlogln(2, "Error allocate!", err)
			return false
		}

		parm := build_parameters(ip)
		Vlogln(2, "allocate IP:", ip)
		Vlogln(2, "parm:", len(parm), parm)
		conn.Write(parm)
		return true
	}
	return false
}

func build_parameters(ip string) ([]byte) {
	str := "m," + strconv.Itoa(*MTU)
	str += " d," + *DNS
	str += " r,0.0.0.0,0"
	str += " a," + ip + ",32"

	buf := []byte(str)
	buflen := byte(len(buf))
	buf = append([]byte{0, buflen}, buf...)
	return buf
}

func Vlogf(level int, format string, v ...interface{}) {
	if level <= *verbosity {
		log.Printf(format, v...)
//		fmt.Printf(format, v...)
	}
}
func Vlog(level int, v ...interface{}) {
	if level <= *verbosity {
		log.Print(v...)
//		fmt.Print(v...)
	}
}
func Vlogln(level int, v ...interface{}) {
	if level <= *verbosity {
		log.Println(v...)
//		fmt.Println(v...)
	}
}


