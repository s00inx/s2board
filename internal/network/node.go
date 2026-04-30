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
	"github.com/s00inx/s2board/internal/transport"
)

// node is abstraction above THIS device in the network
type Node struct {
	// crypto-identity
	PublicK  ed25519.PublicKey
	PrivateK ed25519.PrivateKey
	UID      string

	IP      string
	Port    int
	PubName string

	DbStorage   nodeInternalStorage
	FileStorage nodeExternalStorage
	Codec       codec.Codec
	Transport   transport.Transport

	peers     *peermap
	filepeers *fpeermap
	mpeers    *mpeertable
}

// unified process logic for p2p packets
func (n *Node) ProcessPacket(incp *models.P2PPacket) (*models.P2PPacket, error) {
	if !incp.Verify() {
		return nil, fmt.Errorf("invalid signature")
	}

	peer, _ := n.peers.getpeer(incp.Senderuid)

	switch incp.Action {
	case models.ActHello:
		pl, err := n.RecvHellof(incp, peer.IP, peer.Port)
		if err != nil {
			return nil, err
		}

		return models.NewPacket(pl, models.ActHelloAck, n.UID, n.PrivateK), nil

	case models.Actsave:
		var man models.Manifest
		n.Codec.Decode(incp.Payload, &man)
		n.Recvf(&man)
		return nil, nil

	case models.ActReqM:
		return n.recvfetchmans(incp)

	default:
		return nil, fmt.Errorf("unknown action")
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

	log.Printf("[init] node connected")
	return &Node{
		PublicK:   pub,
		PrivateK:  priv,
		UID:       hex.EncodeToString(pub),
		Port:      port,
		PubName:   name,
		peers:     newpeermap(),
		filepeers: newfilepeermap(),
		mpeers:    newmpeertable(),
	}, nil
}
