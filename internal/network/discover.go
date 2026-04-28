package network

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/s00inx/s2board/internal/models"
)

// mDns local net discovering
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
				// first ipv4
				if len(entry.AddrIPv4) > 0 {
					targetip = entry.AddrIPv4[0].String()
				} else if len(entry.AddrIPv6) > 0 { // then ipv6 bc it may be incompatible
					targetip = fmt.Sprintf("[%s]", entry.AddrIPv6[0].String())
				} else {
					continue
				}

				// parse text fields
				var uid, pname string
				for _, f := range entry.Text {
					if len(f) > 4 && f[:4] == "uid=" {
						uid = f[4:]
					}
					if len(f) > 5 && f[:5] == "name=" {
						pname = f[5:]
					}
				}

				if uid == "" || uid == n.UID {
					continue
				}
				if pname == "" {
					pname = "anonymous_peer"
				}

				oldp, exists := n.peers.getpeer(uid)

				newpeer := Peer{
					UID:      uid,
					Name:     pname,
					IP:       targetip,
					Port:     entry.Port,
					LastSeen: time.Now(),
				}

				n.peers.add(newpeer)

				if !exists || time.Since(oldp.LastSeen) > 1*time.Minute {
					log.Printf("[disc] found peer %s (%s:%d)", pname, targetip, entry.Port)
					go func(p Peer) {
						time.Sleep(1 * time.Second)
						n.Syncw(p)
					}(newpeer)
				}
			}
		}
	}()

	err = rs.Browse(ctx, models.ServiceName, "local.", en)
	if err != nil {
		log.Println("[ERR] browse failed:", err)
	}
}

// get available peers
func (n *Node) GetConns() []Peer {
	n.peers.mu.Lock()
	defer n.peers.mu.Unlock()

	plist := make([]Peer, 0, len(n.peers.d))
	for _, v := range n.peers.d {
		plist = append(plist, v)
	}

	return plist
}
