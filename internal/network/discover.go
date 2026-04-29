package network

import (
	"context"
	"fmt"
	"log"

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
				// first ipv4
				if len(entry.AddrIPv4) > 0 {
					targetip = entry.AddrIPv4[0].String()
				} else if len(entry.AddrIPv6) > 0 { // then ipv6 bc it may be incompatible
					targetip = fmt.Sprintf("[%s]", entry.AddrIPv6[0].String())
				} else {
					continue
				}

				log.Printf("found peer on %s:%d", targetip, entry.Port)
				go n.Handshakew(targetip, entry.Port, models.ActHello)
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
