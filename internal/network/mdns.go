// инициализация мднс и локального адреса
package network

import (
	"fmt"
	"net"
	"os"

	"github.com/grandcat/zeroconf"
	"github.com/s00inx/s2board/internal/models"
)

func InitMdns(ip *net.Interface, uid, name string, port int) (string, *zeroconf.Server, error) {
	if ip == nil {
		return "", nil, fmt.Errorf("can't find any valid net interface, please connect to hotspot")
	}

	hostname, _ := os.Hostname()
	iname := fmt.Sprintf("%s_%s", hostname, uid[:8])

	serv, err := zeroconf.Register(
		iname,
		models.ServiceName,
		"local.",
		port,
		[]string{
			"uid=" + uid,
			"name=" + name,
		},
		[]net.Interface{*ip},
	)

	if err != nil {
		return "", nil, fmt.Errorf("mdns error: %w", err)
	}

	return iname, serv, err
}

func GetLocalIface() (*net.Interface, string) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, ""
	}

	for _, ie := range ifaces {
		addrs, _ := ie.Addrs()

		for _, a := range addrs {
			if ie.Flags&net.FlagUp == 0 || ie.Flags&net.FlagLoopback != 0 {
				continue
			}
			if ipmask, ok := a.(*net.IPNet); ok && !ipmask.IP.IsLoopback() {
				if ipmask.IP.To4() != nil {
					return &ie, ipmask.IP.String()
				}
			}
		}
	}
	// ?? maybe panic here
	return nil, "127.0.0.1"
}

// when node leaves the network / recv bye bc packet
func (n *Node) ForgetPeer(peeruid string) {
	h2del := n.filepeers.rm(peeruid)
	n.peers.rm(peeruid)

	for _, h := range h2del {
		n.DbStorage.DeleteMan(h, models.Bucketvirtual)
	}
}
