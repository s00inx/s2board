package network

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/s00inx/s2board/internal/models"
)

// unified broadcast packet above tcp for scalable file sharing
type BCPacket struct {
	Senderuid string         `json:"sender"`
	Payload   []byte         `json:"payload"`
	Signature string         `json:"sig"`
	Action    models.Actcode `json:"action"`
}

// build a new packet for broadcast
func (n *Node) NewBCp(man *models.Manifest, action models.Actcode) *BCPacket {
	jd, _ := json.Marshal(man)
	pk := BCPacket{
		Senderuid: n.UID,
		Payload:   jd,
		Action:    models.Actcode(action),
	}

	// sign: man json + action
	hdata := append(jd, byte(action))
	pk.Signature = hex.EncodeToString(ed25519.Sign(n.PrivateK, hdata))

	return &pk
}

// verify packet (authencity and integrity)
func (p *BCPacket) Verify() bool {
	pub, err := hex.DecodeString(p.Senderuid)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return false
	}

	sig, err := hex.DecodeString(p.Signature)
	if err != nil {
		return false
	}

	hdata := append(p.Payload, byte(p.Action))
	return ed25519.Verify(pub, hdata, sig)
}

// broadcast bcpacket to all known peers in local network
func (n *Node) Broadcast(p *BCPacket) {
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
		go func(p models.Peer) {
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
