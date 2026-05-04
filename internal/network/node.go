package network

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/s00inx/s2board/internal/codec"
	"github.com/s00inx/s2board/internal/models"
	"github.com/s00inx/s2board/internal/storage"
	"github.com/s00inx/s2board/internal/transport"
)

// node is abstraction above THIS device in the network
type Node struct {
	// crypto-identity
	PublicK  ed25519.PublicKey
	PrivateK ed25519.PrivateKey
	UID      string

	// node public properties
	IP      string
	Port    int
	PubName string

	// node modules
	InternalStorage storage.NodeInternalStorage
	FileStorage     storage.NodeExternalStorage
	Codec           codec.Codec
	Transport       transport.Transport

	// in-memory maps for peers
	peers  *peertable
	mpeers *mhtable
}

// unified process logic for p2p packets (in -> process -> out/error)
// this func is only business-logic dispatcher
func (n *Node) ProcessPacket(incp *models.P2PPacket, rmaddr string) (*models.P2PPacket, error) {
	if !incp.Verify() {
		return nil, fmt.Errorf("invalid sig -> denied")
	}

	if incp.Action == models.ActHelloSyn {
		p, err := n.recvDialp(incp, rmaddr)
		go n.synchello()
		return p, err
	}

	switch incp.Action {

	case models.ActCreate:
		return nil, n.recvCreatef(incp)

	case models.ActReqM:
		return n.recvFetch(incp)

	case models.ActReqF:
		return n.recvDlf(incp)

	case models.ActDel:
		return nil, n.recvDelf(incp)

	case models.ActBye:
		log.Printf("recv bye packet from %s", incp.Senderuid[:8])
		return nil, n.recvByep(incp)

	case models.ActDl:
		return nil, n.recvDl(incp)

	default:
		return nil, fmt.Errorf("unknown act -> denied")
	}
}

func ConnNode(prkpath string, port int, name string) (*Node, error) {
	_, err := os.Stat(prkpath)

	var (
		pub  ed25519.PublicKey
		priv ed25519.PrivateKey
	)

	if err == nil {
		f, err := os.ReadFile(prkpath)
		if err != nil {
			log.Printf("[init] error configuring node")
			return nil, err
		}

		priv = ed25519.PrivateKey(f)
		pub = priv.Public().(ed25519.PublicKey)
	} else {
		pub, priv, err = ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}

		err = os.WriteFile(prkpath, priv, 0600)
		if err != nil {
			return nil, err
		}
	}

	// log.Printf("[init] node connected")
	return &Node{
		PublicK:  pub,
		PrivateK: priv,
		UID:      hex.EncodeToString(pub),
		Port:     port,
		PubName:  name,
		peers:    newpt(),
		mpeers:   newmpeertable(),
	}, nil
}

// delete manifest and file safe
func (n *Node) deletef(mhash, senderuid string) error {
	mh, err := n.InternalStorage.GetManh(mhash, models.Bucketvirtual)
	if err != nil || mh == nil {
		return err
	}

	if senderuid != mh.AuthorUID {
		return fmt.Errorf("[del] user is not author")
	}

	err = n.InternalStorage.DeleteMan(mhash, models.Bucketlocal)
	err = n.InternalStorage.DeleteMan(mhash, models.Bucketvirtual)
	if err != nil {
		return err
	}

	n.mpeers.dropfh(mhash)

	if mh.FileHash != "" {
		return n.FileStorage.DeleteFile(mh.FileHash)
	}
	return nil
}

func (n *Node) GetMpeers() map[string][]string {
	return n.mpeers.d
}
