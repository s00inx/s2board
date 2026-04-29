// all sync nodes in local network logic
// hello and bye packets
package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/s00inx/s2board/internal/models"
)

type HelloPayload struct {
	Name   string   `json:"name"`
	UID    string   `json:"uid"`
	Hashes []string `json:"hashes"`
}

// node <-> node first data exchange (pub key, name and hash list)
func (n *Node) Handshakew(ip string, port int, action models.Actcode) {
	// building self hello packet
	hsbytes, _ := n.GetHashes()

	pl := HelloPayload{
		Name:   n.PubName,
		UID:    n.UID,
		Hashes: hsbytes,
	}
	pl2send, _ := json.Marshal(pl)

	hellop := models.NewPacket(pl2send, action, n.UID, n.PrivateK)
	sehello, err := json.Marshal(hellop)

	if err != nil {
		return
	}

	hellopacket, err := n.sendp(ip, port, sehello)
	if err != nil {
		return
	}

	// waiting for helloack, so drop other packets
	if hellopacket.Action != models.ActHelloAck {
		return
	}

	var hellopl HelloPayload
	json.Unmarshal(hellopacket.Payload, &hellopl)

	nei := Peer{
		UID:      hellopacket.Senderuid,
		Name:     hellopl.Name,
		IP:       ip,
		Port:     port,
		LastSeen: time.Now(),
	}

	n.peers.add(nei)
	for _, h := range hellopl.Hashes {
		n.filepeers.add(h, nei)
	}
}

// receive a hello packet -> send ack packet, finalize handshake
func (n *Node) RecvHellof(reqp *models.P2PPacket, rmaddr string, port int) ([]byte, error) {
	var reqpl HelloPayload
	if err := json.Unmarshal(reqp.Payload, &reqpl); err != nil {
		return nil, fmt.Errorf("[handshake] invalid req packet payload")
	}

	nei := Peer{
		UID:      reqp.Senderuid,
		Name:     reqpl.Name,
		IP:       rmaddr,
		Port:     port,
		LastSeen: time.Now(),
	}

	n.peers.add(nei)
	for _, h := range reqpl.Hashes {
		n.filepeers.add(h, nei)
	}

	myhashes, _ := n.GetHashes()
	resppl, _ := json.Marshal(HelloPayload{
		Name:   n.PubName,
		Hashes: myhashes,
	})

	return resppl, nil
}

// send a p2p packet to exact ip:port
func (n *Node) sendp(ip string, port int, packet2send []byte) (*models.P2PPacket, error) {
	// sending packet to addr
	resp, err := n.client.Post(fmt.Sprintf("http://%s:%d/api/p2p", ip, port), "application/json", bytes.NewReader(packet2send))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	// process response packet
	var respp models.P2PPacket
	if err := json.NewDecoder(resp.Body).Decode(&respp); err != nil {
		return nil, err
	}

	if !respp.Verify() {
		return nil, fmt.Errorf("invalid signature from %s:%d", ip, port)
	}

	return &respp, nil
}

// assymetrical sync with node and all peers
func (n *Node) Syncallw() {
	dstp := n.GetConns()

	myhashes, _ := n.GetHashes()
	payload, _ := json.Marshal(HelloPayload{
		Name:   n.PubName,
		Hashes: myhashes,
	})

	for _, p := range dstp {
		go func(p Peer) {
			outp, _ := json.Marshal(models.NewPacket(payload, models.ActSync, n.UID, n.PrivateK))
			n.sendp(p.IP, p.Port, outp)
		}(p)
	}
}

// get list if all hashes
func (n *Node) GetHashes() ([]string, error) {
	return n.DbStorage.GetHashesList()
}
