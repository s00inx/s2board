package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/s00inx/s2board/internal/models"
)

// раздать файл всем кто находится в одной локальной сети
func (n *Node) Broadcast(man *models.Manifest) {
	ps := n.GetConns()

	if len(ps) == 0 {
		log.Println("[BROADCAST] no peers found to broadcast.")
		return
	}

	payload := map[string]any{
		"peer":     models.Peer{UID: n.UID, IP: n.IP, Port: n.Port},
		"manifest": man,
	}
	jsond, err := json.Marshal(payload)

	if err != nil {
		log.Printf("[BROADCAST] marshal error: %v", err)
		return
	}

	log.Printf("[BROADCAST] sending update to %d peers...\n", len(ps))
	for _, p := range ps {
		go func(peer models.Peer) {
			c := http.Client{Timeout: 5 * time.Second}
			resp, err := c.Post(fmt.Sprintf("http://%s:%d/api/recv", peer.IP, peer.Port), "application/json", bytes.NewBuffer(jsond))
			if err != nil {
				log.Printf("[BROADCAST] failed to send to %s: %v", peer.UID[:8], err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				log.Printf("[BROADCAST] delivered to %s", peer.UID[:8])
			}
		}(p)
	}
}

func (n *Node) NodeBye(c http.Client) {
	cconns := n.GetConns()

	log.Printf("[Bye] notifying to %d peers", len(cconns))

	var actc int
	for _, p := range cconns {
		_, err := c.Get(fmt.Sprintf("http://%s:%d/api/bye/%s", p.IP, p.Port, n.UID))
		if err != nil {
			continue
		}
		actc++
	}

	log.Printf("[Bye] said bye to %d/%d peers", actc, len(cconns))
}

func (n *Node) RecvBye(peerid string) {
	n.RmPeer(peerid)
}
