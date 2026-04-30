// all p2p node as SERVER logic
// recv request from client node -> send response

package network

import (
	"fmt"
	"log"
	"time"

	"github.com/s00inx/s2board/internal/models"
)

// receive file from p2ppacket from net
func (n *Node) Recvf(m *models.Manifest) {
	if !m.Verify() {
		return
	}

	// save to virtual
	n.DbStorage.Save2db(*m, models.Bucketvirtual)

	// if filesize is small, save to local
	if m.FileSize < mindlsize || !n.FileStorage.FileExists(m.FileHash) {
		n.FileStorage.Save2disk(m.FileHash)
		n.DbStorage.Save2db(*m, models.Bucketlocal)
	}
}

type HelloPayload struct {
	Name   string   `json:"name"`
	UID    string   `json:"uid"`
	Hashes []string `json:"hashes"`
}

// receive a hello packet -> send ack packet, finalize handshake
// (rmaddr and port of dst node)
func (n *Node) RecvHellof(reqp *models.P2PPacket, rmaddr string, port int) ([]byte, error) {
	var reqpl HelloPayload
	if err := n.Codec.Decode(reqp.Payload, &reqpl); err != nil {
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

	myhashes, _ := n.getSyncList()
	resppl, _ := n.Codec.Encode(HelloPayload{
		Name:   n.PubName,
		Hashes: myhashes,
	})

	log.Printf("[sync] synced with peer %s:%d", rmaddr, port)

	go n.syncvirtual()

	return resppl, nil
}

// receive ActReqM packet and send list of mans
func (n *Node) recvfetchmans(incp *models.P2PPacket) (*models.P2PPacket, error) {
	if incp.Action != models.ActReqM {
		return nil, fmt.Errorf("")
	}

	var want []string
	if err := n.Codec.Decode(incp.Payload, &want); err != nil {
		return nil, fmt.Errorf("")
	}

	found := make([]*models.Manifest, 0, len(want))
	for _, h := range want {
		raw, _ := n.DbStorage.GetManh(h, models.Bucketlocal)
		if raw == nil {
			raw, _ = n.DbStorage.GetManh(h, models.Bucketvirtual)
		}

		if raw != nil {
			found = append(found, raw)
		}
	}

	resppl, _ := n.Codec.Encode(found)
	respp := models.NewPacket(resppl, models.ActRespM, n.UID, n.PrivateK)

	return respp, nil
}
