package main
// #cgo CFLAGS: -I/apk/android-ndk-r13b/platforms/android-14/arch-arm/usr/include
// #cgo LDFLAGS: -L/apk/android-ndk-r13b/platforms/android-14/arch-arm/usr/lib -llog -landroid
// #include <stdio.h>
// #include <stdlib.h>
// #include <jni.h>
// #include <android/log.h>
// #include "VPNClient.h"
import "C"


import (
	"encoding/binary"
	"errors"
	"net"
	"io"
	"os"
	"syscall"
	"sync"
	"time"
	"runtime"
	"fmt"
//	"log"

	"strings"
	"strconv"

	"unsafe"
)

const (
	headerSize   = 2
)

const TAG = "ToyVpn"
const verbosity = 0

var tunFD int
var socketFD int
var MTU int = 1400

var running bool = false
var stateMux sync.Mutex

var shutdown chan struct{}

var conn net.Conn
var iface *os.File
//var iface net.Conn

func getGoString(env *C.JNIEnv, javaString C.jstring) (string) {
	cstr := C.jniGetStringUTFChars(env, javaString)
	gostr := C.GoString(cstr)
	C.jniReleaseStringUTFChars(env, javaString, cstr)
	return gostr
}

func getJavaString(env *C.JNIEnv, gostr string) (C.jstring) {
	cstr := C.CString(gostr)
	defer C.free(unsafe.Pointer(cstr))
	ret := C.jniNewStringUTF(env, cstr)
	return ret
}

//export Java_cs8425_vpn_VpnServiceNative_Multiply
func Java_cs8425_vpn_VpnServiceNative_Multiply(env *C.JNIEnv, clazz C.jclass, x C.jlong, y C.jlong) C.jlong {
	return x * y + (C.jlong)(tunFD)
}

//export Java_cs8425_vpn_VpnServiceNative_SetTunFD
func Java_cs8425_vpn_VpnServiceNative_SetTunFD(env *C.JNIEnv, clazz C.jclass, tunfd C.jint) () {
	stateMux.Lock()
	if running {
		Vlogln(3, env, "is running!! can't set TunFD")
		stateMux.Unlock()
		return
	}
	stateMux.Unlock()

	tunFD = int(tunfd)
	iface = os.NewFile(uintptr(tunFD), "tun")
	Vlogln(2, env, "Got TunFD", tunFD)

/*	var err error
	iface, err = net.FileConn(os.NewFile(uintptr(tunFD), "tun"))
	if err != nil {
		Vlogln(2, env, "Error Open TunFD:", err.Error())
	}*/

    return
}

//export Java_cs8425_vpn_VpnServiceNative_SetSocketFD
func Java_cs8425_vpn_VpnServiceNative_SetSocketFD(env *C.JNIEnv, clazz C.jclass, fd C.jint) () {
	stateMux.Lock()
	if running {
		Vlogln(3, env, "is running!! can't set SocketFD")
		stateMux.Unlock()
		return
	}
	stateMux.Unlock()

	socketFD = int(fd)
	Vlogln(2, env, "Got SocketFD", socketFD)

	var err error
	conn, err = net.FileConn(os.NewFile(uintptr(socketFD), "conn"))
	if err != nil {
		Vlogln(2, env, "Error Open socketFD:", err.Error())
	}

	return
}

//export Java_cs8425_vpn_VpnServiceNative_Dump
func Java_cs8425_vpn_VpnServiceNative_Dump(env *C.JNIEnv, clazz C.jclass) (C.jint) {
	return (C.jint)(tunFD)
}

//export Java_cs8425_vpn_VpnServiceNative_Stop
func Java_cs8425_vpn_VpnServiceNative_Stop(env *C.JNIEnv, clazz C.jclass) () {
	stateMux.Lock()
	if !running {
		Vlogln(3, env, "not running!!")
		stateMux.Unlock()
		return
	}
	running = false
	defer stateMux.Unlock()

	if shutdown != nil {
		close(shutdown)
	}

	if conn != nil {
		Vlogln(2, env, "conn close:", conn)
		conn.Close()
	}

	if iface != nil {
		Vlogln(2, env, "iface close:", iface)
		iface.Close()
	}
}

//export Java_cs8425_vpn_VpnServiceNative_Handshake
func Java_cs8425_vpn_VpnServiceNative_Handshake(env *C.JNIEnv, clazz C.jclass, parm C.jstring) (C.jstring) {
	Vlogln(4, env, "Handshake() start")

	key := getGoString(env, parm)
	kbuf := append([]byte{0, byte(len(key))}, []byte(key)...)
	conn.Write(kbuf)

	//Important !! need to cancel Deadline!!
	defer conn.SetReadDeadline(time.Time{})

	buf := make([]byte, 256)
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	n, err := conn.Read(buf[:2])
	if err != nil{
		Vlogln(3, env, "Error Handshake reading:", err.Error())
		return getJavaString(env, "")
	}
	if buf[0] != 0 && n != 2 {
		Vlogln(3, env, "Error Handshake header! 1")
		return getJavaString(env, "")
	}

	parmlen := int(buf[1])
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	n, err = io.ReadFull(conn, buf[:parmlen])
	if err != nil || n != parmlen {
		Vlogln(3, env, "Error Handshake header! 2", err)
		return getJavaString(env, "")
	}

	config := string(buf[:parmlen])
	params := strings.Fields(config)
	for _, param := range params {
		fields := strings.Split(param, ",")
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSpace(fields[0])
		value := strings.TrimSpace(fields[1])

		switch key {
		case "m":
			t, err := strconv.ParseInt(value, 10, 16)
			if err != nil {
				Vlogln(4, env, "MTU size error", key, value, err)
				break
			}
			MTU = int(t)
		}
	}

	Vlogln(4, env, "Handshake() end")
	return getJavaString(env, config)
}

//export Java_cs8425_vpn_VpnServiceNative_Loop
func Java_cs8425_vpn_VpnServiceNative_Loop(env *C.JNIEnv, clazz C.jclass) () {
	runtime.GOMAXPROCS(runtime.NumCPU() + 2)
	Vlogln(4, env, "Loop() start")

	stateMux.Lock()
	if running {
		Vlogln(3, env, "multi call Loop()!!")
		stateMux.Unlock()
		return
	}
	running = true
	stateMux.Unlock()

	wg := new(sync.WaitGroup)
	shutdown = make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		runtime.LockOSThread()

		buf := make([]byte, MTU + headerSize)
		i := 0
		for {
			Vlogln(6, env, i, "conn.Read:")
			i++

			select {
			case <-shutdown:
				//Vlogln(3, env, i, "conn Reader shutdown!")
				break

			default:
				if f, err := readFrame(conn, buf); err == nil {
					Vlogln(5, env, "readFrame()", len(f))
					iface.Write(f)
				} else {
					e, ok := err.(net.Error)
					if !ok || !e.Temporary() {
						Vlogln(2, env, "Error reading Frame", err)
						select {
						case <-shutdown:

						default:
							close(shutdown)
						}
						return
					}
					time.Sleep(time.Millisecond * 20)
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runtime.LockOSThread()

		buf := make([]byte, MTU)
		i := 0
		for {
			Vlogln(6, env, i, "iface.Read:")
			i++

			select {
			case <-shutdown:
				//Vlogln(3, env, i, "iface Reader shutdown!")
				return

			default:
				// Read the incoming packet from the tunnel.
				n, err := iface.Read(buf)
				if err != nil{
					e, ok := err.(*os.PathError)
					if ok && e.Err == syscall.EAGAIN {
						time.Sleep(time.Millisecond * 20)
						continue
					}
					Vlogln(2, env, "Error iface reading2:", n, e, ok, err)
					return
				}
				if n > 0 {
					Vlogln(6, env, "conn.Write:", n)
					header := []byte{byte(n & 0xFF), byte((n >> 8) & 0xFF)}
					conn.Write(append(header, buf[:n]...))
				}

			}
			//time.Sleep(time.Millisecond * 500)
		}
	}()

	wg.Wait()
	Vlogln(4, env, "Loop() end")
}

func readFrame(conn net.Conn, buffer []byte) (f []byte, err error) {
	if n, err := io.ReadFull(conn, buffer[:headerSize]); err != nil {
		return f, errors.New(fmt.Sprint("readFrame header:", n, err))
	}

	if length := binary.LittleEndian.Uint16(buffer[0:]); length > 0 {
		if n, err := io.ReadFull(conn, buffer[headerSize:headerSize+length]); err != nil {
			return f, errors.New(fmt.Sprint("readFrame data:", n, err))
		}
		f = buffer[headerSize : headerSize+length]
	}
	return f, nil
}

// main function is required, don't know why!
func main() {} // a dummy function

func Vlogf(level int, env *C.JNIEnv, format string, v ...interface{}) {
	if level <= verbosity {
		str := fmt.Sprintf(format, v...)
		andLog2(env, str)
	}
}
func Vlog(level int, env *C.JNIEnv, v ...interface{}) {
	if level <= verbosity {
		str := fmt.Sprint(v...)
		andLog2(env, str)
	}
}
func Vlogln(level int, env *C.JNIEnv, v ...interface{}) {
	if level <= verbosity {
		str := fmt.Sprintln(v...)
		andLog2(env, str)
	}
}

func andLog(env *C.JNIEnv, str string) () {
	tagstr := C.CString(TAG)
//	defer C.free(unsafe.Pointer(tagstr))

	infostr := C.CString(str)
//	defer C.free(unsafe.Pointer(infostr))

	C.jniLog(env, tagstr, infostr)
}

// call too fast will crash
// JNI ERROR (app bug): local reference table overflow (max=512)
func andLog2(env *C.JNIEnv, str string) () {
	tagstr := C.CString(TAG)
//	defer C.free(unsafe.Pointer(tagstr))
	jtagstr := C.jniNewStringUTF(env, tagstr)

	infostr := C.CString(str)
//	defer C.free(unsafe.Pointer(infostr))
	jinfostr := C.jniNewStringUTF(env, infostr)

	C.jniLog2(env, jtagstr, jinfostr)

	C.jniReleaseStringUTFChars(env, jinfostr, infostr)
	C.jniReleaseStringUTFChars(env, jtagstr, tagstr)

//	C.free(unsafe.Pointer(infostr))
//	C.free(unsafe.Pointer(tagstr))
}



