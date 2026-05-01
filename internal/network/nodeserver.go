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

// receive a hello packet -> send ack packet, finalize handshake
// (rmaddr and port of dst node)
func (n *Node) RecvHandshakef(reqp *models.P2PPacket, rmaddr string, port int) (*models.P2PPacket, error) {
	if reqp.Action != models.ActHello {
		return nil, fmt.Errorf("")
	}

	var reqpl Hspl
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
	resppl, _ := n.Codec.Encode(Hspl{
		Name:   n.PubName,
		Hashes: myhashes,
	})

	go n.syncvirtual()

	return models.NewPacket(resppl, models.ActHelloAck, n.UID, n.PrivateK), nil
}

// receive ActReqM packet and send list of mans
func (n *Node) recvfetchmans(incp *models.P2PPacket) (*models.P2PPacket, error) {
	if incp.Action != models.ActReqM {
		log.Printf("want: ReqM, recv: %d", incp.Action)
		return nil, fmt.Errorf("")
	}

	var want []string
	if err := n.Codec.Decode(incp.Payload, &want); err != nil {
		return nil, err
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

	resppl, err := n.Codec.Encode(found)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	respp := models.NewPacket(resppl, models.ActRespM, n.UID, n.PrivateK)

	return respp, nil
}

// node A req file -> respond
type FileResppl struct {
	Fh    string
	Bytes []byte
}

// receive ActReqF packet
func (n *Node) RecvDlf(reqp *models.P2PPacket, addr string) (*models.P2PPacket, error) {
	if reqp.Action != models.ActReqF {
		return nil, fmt.Errorf("")
	}

	pl := FileReqpl{}
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

	new := FileResppl{
		Bytes: fbytes,
	}
	respp, err := n.Codec.Encode(&new)
	if err != nil {
		return nil, err
	}

	return models.NewPacket(respp, models.ActRespF, n.UID, n.PrivateK), nil
}

// recv ActDel packet
func (n *Node) RecvDelf(incp *models.P2PPacket) error {
	if incp.Action != models.Actdel {
		return fmt.Errorf("")
	}

	pl := Delpl{}
	if err := n.Codec.Decode(incp.Payload, &pl); err != nil {
		return fmt.Errorf("")
	}

	mh, _ := n.DbStorage.GetManh(pl.Mhash, models.Bucketvirtual)

	if incp.Senderuid != mh.AuthorUID {
		return fmt.Errorf("403")
	}

	n.DbStorage.DeleteMan(pl.Mhash, models.Bucketlocal)
	n.DbStorage.DeleteMan(pl.Mhash, models.Bucketvirtual)

	n.mpeers.dropfh(pl.Mhash)

	return n.FileStorage.DeleteFile(mh.FileHash)
}

func (n *Node) RecvByep(incp *models.P2PPacket) error {
	if incp.Action != models.Actbye {
		return fmt.Errorf("")
	}

	n.forgetpeer(incp.Senderuid)

	return nil
}
