package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"unsafe"
)

const (
	afVsock      = 40 // syscall.AF_VSOCK
	vmaddrCidAny = 0xffffffff
)

type rawSockaddrVM struct {
	family    uint16
	reserved1 uint16
	port      uint32
	cid       uint32
	flags     uint8
	zero      [3]uint8
}

type sockaddrVM struct {
	cid   uint32
	port  uint32
	flags uint8
	raw   rawSockaddrVM
}

func (sa *sockaddrVM) sockaddr() unsafe.Pointer {
	sa.raw.family = afVsock
	sa.raw.cid = sa.cid
	sa.raw.port = sa.port
	sa.raw.flags = sa.flags
	return unsafe.Pointer(&sa.raw)
}

func listenVsock(port uint32) (net.Listener, error) {
	fd, err := syscall.Socket(afVsock, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}
	sa := &sockaddrVM{cid: vmaddrCidAny, port: port}
	_, _, errno := syscall.RawSyscall(syscall.SYS_BIND, uintptr(fd), uintptr(sa.sockaddr()), unsafe.Sizeof(sa.raw))
	if errno != 0 {
		syscall.Close(fd)
		return nil, errno
	}
	_, _, errno = syscall.RawSyscall(syscall.SYS_LISTEN, uintptr(fd), 1, 0)
	if errno != 0 {
		syscall.Close(fd)
		return nil, errno
	}
	f := os.NewFile(uintptr(fd), fmt.Sprintf("vsock:%d", port))
	ln, err := net.FileListener(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	return ln, nil
}

func main() {
	portFlag := flag.Uint("port", 5005, "vsock port")
	flag.Parse()

	ln, err := listenVsock(uint32(*portFlag))
	if err != nil {
		log.Fatalf("listen vsock: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{Handler: mux}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	srv.Close()
}
