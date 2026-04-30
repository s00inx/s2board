// all p2p node as CLIENT logic
// send packet -> recv response

package network

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/s00inx/s2board/internal/models"
)

// download file from available peers
func (n *Node) Dlf(fhash string) error {
	if n.FileStorage.FileExists(fhash) {
		return nil
	}

	peers, _ := n.filepeers.getpeerlist(fhash)
	if len(peers) == 0 {
		return fmt.Errorf("file not found in network")
	}

	// get first available peer for MVP
	var p Peer
	var ok bool
	for _, pr := range peers {
		p, ok = n.peers.getpeer(pr)
		if ok {
			break
		}
	}

	log.Printf("[download] req file %s from %s", fhash[:8], p.Name)
	reqpl, _ := n.Codec.Encode(map[string]string{"fhash": fhash})
	reqp := models.NewPacket(reqpl, models.ActReqF, n.UID, n.PrivateK)

	respp, err := n.Transport.Sendp(p.IP, p.Port, reqp)
	if err != nil {
		return err
	}

	// actualHash := n.(respp.Payload)
	// if actualHash != fhash {
	// 	return fmt.Errorf("integrity check failed: hash mismatch")
	// }

	return n.FileStorage.SaveFile(fhash, bytes.NewReader(respp.Payload))
}

// upload file from dev TO local network -> broadcast packet
func (n *Node) Createf(src, title, desc string) (*models.Manifest, error) {
	var fhash string
	var fsize int64
	var err error

	if src != "" {
		fhash, fsize, err = n.FileStorage.Save2disk(src)
		if err != nil {
			return nil, err
		}
	}

	man := models.NewMan(title, desc, n.UID, n.PubName, fhash, filepath.Base(src), fsize)
	if err := man.Sign(n.PrivateK); err != nil {
		return nil, err
	}
	man.Hash = hex.EncodeToString(man.CalcID())

	n.DbStorage.Save2db(*man, models.Bucketvirtual)
	if err = n.DbStorage.Save2db(*man, models.Bucketlocal); err != nil {
		return nil, err
	}

	mb, _ := n.Codec.Encode(man)
	np := models.NewPacket(mb, models.Actsave, n.UID, n.PrivateK)
	go n.Transport.Broadcastp(np, n.GetConns())

	return man, nil
}

// node <-> node first data exchange (pub key, name and lacal hashes list)
func (n *Node) Handshakew(ip string, port int, action models.Actcode) {
	if ip == n.IP && port == n.Port {
		return
	}

	// building self hello packet
	hsbytes, _ := n.getSyncList()

	pl := HelloPayload{
		Name:   n.PubName,
		UID:    n.UID,
		Hashes: hsbytes,
	}
	pl2send, _ := n.Codec.Encode(pl)
	hellop := models.NewPacket(pl2send, action, n.UID, n.PrivateK)

	hellopacket, err := n.Transport.Sendp(ip, port, hellop)
	if err != nil {
		log.Println(err)
		return
	}

	// waiting for helloack, so drop other packets
	if hellopacket.Action != models.ActHelloAck {
		log.Println("[sync] invalid packet: want HelloAck")
		return
	}

	var hellopl HelloPayload
	err = n.Codec.Decode(hellopacket.Payload, &hellopl)
	if err != nil {
		log.Println(err)
	}

	nei := Peer{
		UID:      hellopacket.Senderuid,
		Name:     hellopl.Name,
		IP:       ip,
		Port:     port,
		LastSeen: time.Now(),
	}

	n.peers.add(nei)
	for _, h := range hellopl.Hashes {
		n.mpeers.add(h, nei.UID)
	}

	log.Printf("[sync] handshake with peer %s:%d", ip, port)
	go n.syncvirtual()
}

// fetch batch of manifests from peers
func (n *Node) fetchmans(p Peer, h []string) ([]models.Manifest, error) {
	log.Printf("[sync] fetching %d manifests from peer %s (%s)", len(h), p.UID[:8], p.Name)
	data, err := n.Codec.Encode(h)
	if err != nil {
		log.Println(err)
		return nil, errors.New("[sync] hashlist encode error")
	}

	reqp := models.NewPacket(data, models.ActReqM, n.UID, n.PrivateK)

	inc, _ := n.Transport.Sendp(p.IP, p.Port, reqp)

	var pl []models.Manifest
	if err := n.Codec.Decode(inc.Payload, &pl); err != nil {
		return nil, err
	}

	return pl, nil
}
