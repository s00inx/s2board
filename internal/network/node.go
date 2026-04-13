package network

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"
)

// node is abstrcation above THIS exact device in the network
type Node struct {
	PublicK  ed25519.PublicKey
	PrivateK ed25519.PrivateKey
	UID      string

	IP      string
	Port    int
	PubName string
	Storage nodeStorage
	client  http.Client

	peermap peermap
	fpeers  fpeermap
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
		PublicK:  pub,
		PrivateK: priv,
		UID:      hex.EncodeToString(pub),
		Port:     port,
		PubName:  name,
		peermap:  *newpeermap(),
		fpeers:   *newfilepeermap(),
	}, nil
}

func (n *Node) Forget(peeruid string) {
	n.fpeers.rmpeer(peeruid)
	n.peermap.rm(peeruid)
}
