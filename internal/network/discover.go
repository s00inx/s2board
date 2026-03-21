package network

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

// потокобезопасная мапа для активных соединений, но без ненужных интерфейсов
type PeerMap struct {
	d  map[string]Peer
	mu sync.Mutex
}

// добавить значение в мапу потокобезопасно
func (pm *PeerMap) Add(p Peer) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.d[p.UID] = p
}

// найти узел в сети
func (n *Node) Discover(ctx context.Context) {
	rs, err := zeroconf.NewResolver(nil)
	if err != nil {
		panic(err)
	}

	en := make(chan *zeroconf.ServiceEntry)
	go func(res <-chan *zeroconf.ServiceEntry) {
		for entry := range res {
			var uid string
			for _, f := range entry.Text {
				if len(f) > 4 && f[:4] == "uid=" {
					uid = f[4:]
				}
			}

			if uid != "" && uid != n.UID {
				n.peers.Add(Peer{
					UID:      uid,
					IP:       entry.AddrIPv4[0].String(),
					Port:     entry.Port,
					LastSeen: time.Now(),
				})

				fmt.Printf("\n[NEW PEER] Found node: %s at %s:%d\n", uid[:8], entry.AddrIPv4[0], entry.Port)
			}
		}
	}(en)

	err = rs.Browse(ctx, "_stdesk._tcp", "local.", en)
	if err != nil {
		log.Println("resolver: failed to browse:", err)
	}
}

// найти активные соединения на момент вызова функции
func (n *Node) GetConns() []Peer {
	n.peers.mu.Lock()
	defer n.peers.mu.Unlock()

	plist := make([]Peer, 0, len(n.peers.d))
	for _, v := range n.peers.d {
		plist = append(plist, v)
	}

	return plist
}
