package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/s00inx/s2board/internal/models"
)

// broadcast models/bcpacket to all known peers in local network
func (n *Node) Broadcast(p *models.BCPacket) {
	p2send, err := json.Marshal(p)
	if err != nil {
		log.Printf("[broadcast] marshal error: %v -> ignored", err)
		return
	}

	ps := n.GetConns()
	if len(ps) == 0 {
		log.Println("[broadcast] no peers -> ok")
		return
	}

	for _, dstpeer := range ps {
		go func(p Peer) {
			var dsturl string = fmt.Sprintf("http://%s:%d/api/p2p", p.IP, p.Port)
			resp, err := n.client.Post(dsturl, "application/json", bytes.NewReader(p2send))
			if err != nil {
				log.Printf("[broadcast] failed to send to %s", p.UID[:8])
				return
			}

			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Printf("[broadcast] code %d from %s", resp.StatusCode, p.UID[:8])
			} else {
				log.Printf("[broadcast] delivered to %s", p.UID[:8])
			}
		}(dstpeer)
	}
}
