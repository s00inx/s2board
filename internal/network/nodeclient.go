// all p2p node as CLIENT logic
// send packet -> recv response / return error/nil

package network

import (
	"bytes"
	"encoding/hex"
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

	mh, err := n.InternalStorage.GetManfh(fhash, models.Bucketvirtual)
	if err != nil {
		log.Printf("no man for fh")
		return fmt.Errorf("file not found by file hash")
	}

	peers, ok := n.mpeers.getpeerlist(mh.Hash)
	if !ok {
		return fmt.Errorf("file not found in network")
	}

	// get first available peer for MVP
	var p Peer
	for _, pr := range peers {
		p, ok = n.peers.getpeer(pr)
		if ok {
			break
		}
	}

	log.Printf("[download] req file %s from %s", fhash[:8], p.Name)

	newreq := filereqpl{
		Fh:     fhash,
		Offset: 0,
	}

	reqpl, _ := n.Codec.Encode(newreq)
	reqp := models.NewPacket(reqpl, models.ActReqF, n.UID, n.PrivateK)

	respp, err := n.Transport.Sendp(p.IP, p.Port, reqp)
	if err != nil {
		log.Println(err)
		return err
	}

	if !respp.Verify() {
		log.Printf("[sec] invalid sig from %s:%d -> drop handshake", p.IP, p.Port)
		return fmt.Errorf("invalid signature")
	}

	fb := fileresppl{}
	err = n.Codec.Decode(respp.Payload, &fb)
	if err != nil {
		return err
	}

	err = n.FileStorage.SaveFile(fhash, bytes.NewReader(fb.Bytes))
	if err != nil {
		return err
	}

	dlppl, _ := n.Codec.Encode(dlpl{
		Mhash: mh.Hash,
	})

	dlp := models.NewPacket(dlppl, models.ActDl, n.UID, n.PrivateK)
	n.Transport.Broadcastp(dlp, n.getConns())

	return nil
}

// upload file from dev TO local network -> broadcast packet
func (n *Node) Createf(src, title, desc string) error {
	var fhash string
	var fsize int64
	var err error

	if src != "" {
		fhash, fsize, err = n.FileStorage.Save2disk(src)
		if err != nil {
			return err
		}
	}

	man := models.NewMan(title, desc, n.UID, n.PubName, fhash, filepath.Base(src), fsize)
	if err := man.Sign(n.PrivateK); err != nil {
		return err
	}
	man.Hash = hex.EncodeToString(man.CalcID())

	n.InternalStorage.Save2db(*man, models.Bucketvirtual)
	if err = n.InternalStorage.Save2db(*man, models.Bucketlocal); err != nil {
		return err
	}

	mb, _ := n.Codec.Encode(man)
	np := models.NewPacket(mb, models.ActCreate, n.UID, n.PrivateK)
	n.Transport.Broadcastp(np, n.getConns())

	return nil
}

// node <-> node first data exchange (pub key, name and local hashes list)
func (n *Node) Handshakew(ip string, port int, action models.Actcode) error {
	// building self hello packet
	hsbytes, _ := n.getsynclist()
	pl := syncpl{
		Name:   n.PubName,
		UID:    n.UID,
		Hashes: hsbytes,
	}
	pl.Port = n.Port

	pl2send, _ := n.Codec.Encode(pl)
	hellop := models.NewPacket(pl2send, action, n.UID, n.PrivateK)

	hellopacket, err := n.Transport.Sendp(ip, port, hellop)
	if err != nil {
		log.Println(err)
		return err
	}

	if !hellopacket.Verify() {
		log.Printf("[sec] invalid sig from %s:%d -> drop handshake", ip, port)
		return err
	}

	// waiting for helloack, so drop other packets
	if hellopacket.Action != models.ActRespHello {
		log.Println("[sync] invalid packet: want RespHello")
		return err
	}

	var hellopl syncpl
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

	log.Printf("[sync] process handshake with %s:%d (%s)", ip, port, hellopl.Name)
	go func() {
		time.Sleep(100 * time.Millisecond)
		n.synchello()
	}()

	return nil
}

func (n *Node) Delf(mhash string) error {
	delppl, err := n.Codec.Encode(delpl{
		Mhash: mhash,
	})

	if err != nil {
		return fmt.Errorf("")
	}

	mh, err := n.InternalStorage.GetManh(mhash, models.Bucketvirtual)
	if err != nil {
		return fmt.Errorf("")
	}

	if mh.AuthorUID != n.UID {
		return fmt.Errorf("[delete] user is not author -> forbidden")
	}

	delp := models.NewPacket(delppl, models.ActDelete, n.UID, n.PrivateK)
	n.Transport.Broadcastp(delp, n.getConns())

	n.deletef(mhash, n.UID)

	return nil
}

// send Bye packet to all peers (not wait for ack, only leave the network)
func (n *Node) Byew() error {
	byeconns := n.getConns()

	log.Printf("[sync] saying Bye to %d peers", len(byeconns))

	byep := models.NewPacket([]byte{}, models.ActBye, n.UID, n.PrivateK)
	n.Transport.Broadcastp(byep, byeconns)

	return nil
}
