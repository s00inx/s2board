package network

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/s00inx/s2board/internal/models"
)

// потокобезопасная мапа для активных соединений, но без ненужных интерфейсов
// map[uid]Peer_struct
type peermap struct {
	d  map[string]models.Peer
	mu sync.Mutex
}

func NewPM() *peermap {
	return &peermap{
		d: make(map[string]models.Peer, 0),
	}
}

// добавить значение в мапу потокобезопасно
func (pm *peermap) Add(p models.Peer) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.d[p.UID] = p
}

// удалить значение из мапы
func (pm *peermap) Remove(uid string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.d, uid)
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
			var targetip string
			if len(entry.AddrIPv4) > 0 {
				targetip = entry.AddrIPv4[0].String()
			} else if len(entry.AddrIPv6) > 0 {
				targetip = fmt.Sprintf("[%s]", entry.AddrIPv6[0].String())
			} else {
				continue
			}

			var uid, pname string

			for _, f := range entry.Text {
				if len(f) > 4 && f[:4] == "uid=" {
					uid = f[4:]
				}
				if f[:5] == "name=" {
					pname = f[5:]
				}
			}

			if uid != "" && uid != n.UID {
				n.peermap.Add(models.Peer{
					UID:      uid,
					Name:     pname,
					IP:       targetip,
					Port:     entry.Port,
					LastSeen: time.Now(),
				})
				log.Printf("\n[NEW PEER] %s at %s:%d - %s\n", uid[:8], targetip, entry.Port, pname)
			}
		}
	}(en)

	err = rs.Browse(ctx, "_s2board._tcp", "local.", en)
	if err != nil {
		log.Println("resolver: failed to browse:", err)

		close(en)
	}
}

// найти активные соединения на момент вызова функции
func (n *Node) GetConns() []models.Peer {
	n.peermap.mu.Lock()
	defer n.peermap.mu.Unlock()

	plist := make([]models.Peer, 0, len(n.peermap.d))
	for _, v := range n.peermap.d {
		plist = append(plist, v)
	}

	return plist
}
