package network

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/s00inx/s2board/internal/models"
)

// unified broadcast packet
type BCPacket struct {
	Senderuid string         `json:"sender"`
	Payload   []byte         `json:"payload"`
	Signature string         `json:"sig"`
	Action    models.Actcode `json:"action"`
}

// build a new packet for broadcast
func (n *Node) Newpacket(man *models.Manifest, action byte) *BCPacket {
	jd, _ := json.Marshal(man)
	pk := BCPacket{
		Senderuid: n.UID,
		Payload:   jd,
		Action:    models.Actcode(action),
	}

	hdata := append(jd, action)
	pk.Signature = hex.EncodeToString(ed25519.Sign(n.PrivateK, hdata))

	return &pk
}

func (n *Node) Broadcastt(p *BCPacket) {
	ps := n.GetConns()
	if len(ps) == 0 {
		log.Println("[broadcast] no peers -> ok")
		return
	}

	data2send, err := json.Marshal(p)
	if err != nil {
		log.Printf("[broadcast] marshal error: %v -> ignored", err)
		return
	}

	// parsing logic
}

// broadcast message (the bcpacket) to all peers in local network
// func (n *Node) Broadcast(man *models.Manifest, action byte) {

// 	payload := map[string]any{
// 		"peer":     models.Peer{UID: n.UID, IP: n.IP, Port: n.Port},
// 		"manifest": man,
// 	}
// 	jsond, err := json.Marshal(payload)

// 	log.Printf("[broadcast] found %d peers -> sending", len(ps))
// 	for _, p := range ps {
// 		go func(peer models.Peer) {
// 			var durl string
// 			if action == byte(models.Actsave) {
// 				durl = fmt.Sprintf("http://%s:%d/api/recv", peer.IP, peer.Port)
// 			} else {
// 				durl = fmt.Sprintf("http://%s:%d/api/del", peer.IP, peer.Port)
// 			}

// 			resp, err := n.client.Post(durl, "application/json", bytes.NewBuffer(jsond))
// 			if err != nil {
// 				log.Printf("[broadcast] faild to send to %s: %v -> ignored", peer.UID[:8], err)
// 				return
// 			}

// 			defer resp.Body.Close()

// 			if resp.StatusCode == http.StatusOK {
// 				log.Printf("[broadcast] ok -> delivered to %s", peer.UID[:8])
// 			} else {
// 				log.Printf("[broadcast] code %d for peer %s", resp.StatusCode, peer.UID[:8])
// 			}
// 		}(p)
// 	}
// }

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
