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

// инициализируем новую пир-мап (один раз при ините ноды)
func newpeermap() *peermap {
	return &peermap{
		d: make(map[string]models.Peer, 0), // инициализируем мапу
	}
}

// добавить пир в мапу \ сделать его активным
func (pm *peermap) add(p models.Peer) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.d[p.UID] = p
}

// удалить неактивный (или недоступный) пир из мапы
func (pm *peermap) rm(uid string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.d, uid)
}

// запустить поиск других узлов в сети и синхронизироваться с ними
func (n *Node) Discover(ctx context.Context) {
	rs, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Printf("[ERR] failed to init resolver: %v", err)
		return
	}

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
					if len(f) > 5 && f[:5] == "name=" {
						pname = f[5:]
					}
				}

				if uid == "" || uid == n.UID {
					continue
				}
				if pname == "" {
					pname = "anonymous"
				}

				n.peermap.mu.Lock()
				oldp, exists := n.peermap.d[uid]
				n.peermap.mu.Unlock()

				newpeer := models.Peer{
					UID:      uid,
					Name:     pname,
					IP:       targetip,
					Port:     entry.Port,
					LastSeen: time.Now(),
				}

				n.peermap.add(newpeer)

				if !exists || time.Since(oldp.LastSeen) > 1*time.Minute {
					log.Printf("[DISCOVERY] Found peer %s (%s:%d)", pname, targetip, entry.Port)
					go func(p models.Peer) {
						time.Sleep(1 * time.Second)
						n.Syncw(p)
					}(newpeer)
				}
			}
		}
	}()

	err = rs.Browse(ctx, service_name, "local.", en)
	if err != nil {
		log.Println("[ERR] browse failed:", err)
	}
}

// найти активные соединения на момент вызова функции (просто прочитать пирмап)
func (n *Node) GetConns() []models.Peer {
	n.peermap.mu.Lock()
	defer n.peermap.mu.Unlock()

	plist := make([]models.Peer, 0, len(n.peermap.d))
	for _, v := range n.peermap.d {
		plist = append(plist, v)
	}

	return plist
}
