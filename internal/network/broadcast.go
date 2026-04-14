package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/s00inx/s2board/internal/models"
)

// broadcast message to all peers in local network
func (n *Node) Broadcast(man *models.Manifest, action byte) {
	if action != models.BroadcastSave && action != models.BroadcastDel {
		log.Printf("[broadcast] invalid action -> skipped")
		return
	}

	ps := n.GetConns()
	if len(ps) == 0 {
		log.Println("[broadcast] no peers -> ok")
		return
	}

	payload := map[string]any{
		"peer":     models.Peer{UID: n.UID, IP: n.IP, Port: n.Port},
		"manifest": man,
	}
	jsond, err := json.Marshal(payload)

	if err != nil {
		log.Printf("[broadcast] marshal error: %v -> ignored", err)
		return
	}

	log.Printf("[broadcast] found %d peers -> sending", len(ps))
	dcnt := 0
	for _, p := range ps {
		go func(peer models.Peer) {
			var durl string
			if action == models.BroadcastSave {
				durl = fmt.Sprintf("http://%s:%d/api/recv", peer.IP, peer.Port)
			} else {
				durl = fmt.Sprintf("http://%s:%d/api/del", peer.IP, peer.Port)
			}

			resp, err := n.client.Post(durl, "application/json", bytes.NewBuffer(jsond))
			if err != nil {
				log.Printf("[broadcast] faild to send to %s: %v -> ignored", peer.UID[:8], err)
				return
			}

			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				log.Printf("[broadcast] ok -> delivered to %s", peer.UID[:8])
				dcnt++
			} else {
				log.Printf("[broadcast] code %d for peer %s", resp.StatusCode, peer.UID[:8])
			}
		}(p)
	}

	log.Printf("[broadcast] ok -> %d/%d peers", dcnt, len(ps))
}

func (n *Node) NodeBye(c http.Client) {
	cconns, actc := n.GetConns(), 0

	for _, p := range cconns {
		_, err := c.Get(fmt.Sprintf("http://%s:%d/api/bye/%s", p.IP, p.Port, n.UID))
		if err != nil {
			continue
		}
		actc++
	}

	log.Printf("[nodebye] said bye to %d/%d peers -> goodbye!", actc, len(cconns))
}

func (n *Node) RecvBye(peerid string) {
	n.Forget(peerid)
}
