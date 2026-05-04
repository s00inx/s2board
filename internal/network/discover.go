package network

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/grandcat/zeroconf"
	"github.com/s00inx/s2board/internal/models"
)

// discover local net -> handshake
func (n *Node) Discover(ctx context.Context) {
	rs, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Printf("[err] failed to init resolver: %v", err)
		return
	}

	// create entry channel
	en := make(chan *zeroconf.ServiceEntry)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case entry, ok := <-en:
				if !ok {
					return
				}

				// parse IP fields of zeroconf entry
				var targetip string
				if len(entry.AddrIPv4) > 0 {
					targetip = entry.AddrIPv4[0].String()
				} else {
					continue
				}

				if targetip == n.IP && entry.Port == n.Port {
					continue
				}

				log.Printf("[net] found on %s:%d", targetip, entry.Port)

				// most naive determinated decision of double-handshake problem -> compare addr strings
				urldst, urlnode := fmt.Sprintf("%s:%d", targetip, entry.Port), fmt.Sprintf("%s:%d", n.IP, n.Port)
				if urldst > urlnode {
					log.Printf("node is handshake senior -> init handshake...")
					go n.Dialp(targetip, entry.Port, models.ActHelloSyn)
				} else {
					log.Printf("node is not senior -> waiting for hello packet...")
				}

			}
		}
	}()

	err = rs.Browse(ctx, models.ServiceName, "local.", en)
	if err != nil {
		log.Println("[ERR] browse failed:", err)
	}
}

func (n *Node) getConns() []string {
	n.peers.mu.Lock()
	defer n.peers.mu.Unlock()

	addrs := make([]string, 0, len(n.peers.d))
	for _, p := range n.peers.d {
		if p.UID == n.UID {
			continue
		}

		addr := fmt.Sprintf("%s:%d", p.IP, p.Port)
		addrs = append(addrs, addr)
	}

	return addrs
}
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

func (n *Node) GetConnsF() []Peer {
	peers := n.getConns()

	result := make([]Peer, 0, len(peers))
	for _, p := range peers {
		result = append(result, n.peers.d[p])
	}

	return result
}
