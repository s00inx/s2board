// all p2p node as SERVER logic
// recv request from client node -> send response

package network

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/s00inx/s2board/internal/models"
)

const mindlsize = 1

// receive file from p2ppacket from net
func (n *Node) recvf(incp *models.P2PPacket) error {
	m := models.Manifest{}
	if err := n.Codec.Decode(incp.Payload, &m); err != nil {
		return err
	}

	if !m.Verify() {
		return fmt.Errorf("")
	}

	// save to virtual
	n.InternalStorage.Save2db(m, models.Bucketvirtual)

	// if filesize is small, save to local
	if m.FileSize < mindlsize || !n.FileStorage.FileExists(m.FileHash) {
		n.FileStorage.Save2disk(m.FileHash)
		n.InternalStorage.Save2db(m, models.Bucketlocal)
	}

	return nil
}

// receive a hello packet -> send ack packet, finalize handshake
// (rmaddr and port of dst node)
func (n *Node) recvHandshakef(reqp *models.P2PPacket, rmaddr string, port int) (*models.P2PPacket, error) {
	log.Printf("[sync] handshake req from %s -> respond", reqp.Senderuid[:8])
	var reqpl syncpl
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
		n.mpeers.add(h, nei.UID)
	}

	myhashes, _ := n.getSyncList()
	resppl, _ := n.Codec.Encode(syncpl{
		Name:   n.PubName,
		Hashes: myhashes,
	})

	n.syncvirtual()
	return models.NewPacket(resppl, models.ActRespHello, n.UID, n.PrivateK), nil
}

// receive ActReqM packet and send list of mans
func (n *Node) recvFetch(incp *models.P2PPacket) (*models.P2PPacket, error) {
	var want []string
	if err := n.Codec.Decode(incp.Payload, &want); err != nil {
		return nil, err
	}

	found := make([]*models.Manifest, 0, len(want))
	for _, h := range want {
		raw, _ := n.InternalStorage.GetManh(h, models.Bucketlocal)
		if raw == nil {
			raw, _ = n.InternalStorage.GetManh(h, models.Bucketvirtual)
		}

		if raw != nil {
			found = append(found, raw)
		}
	}

	resppl, err := n.Codec.Encode(found)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	respp := models.NewPacket(resppl, models.ActRespM, n.UID, n.PrivateK)

	return respp, nil
}

// receive ActReqF packet
func (n *Node) recvDlf(reqp *models.P2PPacket, addr string) (*models.P2PPacket, error) {
	pl := filereqpl{}
	err := n.Codec.Decode(reqp.Payload, &pl)

	if err != nil {
		return nil, err
	}

	file, err := os.Open(n.FileStorage.Fhash2path(pl.Fh))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// for MVP
	fbytes, _ := io.ReadAll(file)

	new := fileresppl{
		Bytes: fbytes,
	}
	respp, err := n.Codec.Encode(&new)
	if err != nil {
		return nil, err
	}

	return models.NewPacket(respp, models.ActRespF, n.UID, n.PrivateK), nil
}

// recv ActDel packet
func (n *Node) recvDelf(incp *models.P2PPacket) error {
	pl := delpl{}
	if err := n.Codec.Decode(incp.Payload, &pl); err != nil {
		return fmt.Errorf("")
	}

	return n.deletef(pl.Mhash, incp.Senderuid)
}

// receive Bye Packet, only remove peer from all peer lists
func (n *Node) recvByep(incp *models.P2PPacket) error {
	n.forgetpeer(incp.Senderuid)

	return nil
}
