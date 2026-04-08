// центральный узел сети
package network

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
)

// структура для именно этого устройства в сети
type Node struct {
	// ключи
	PublicK  ed25519.PublicKey
	PrivateK ed25519.PrivateKey
	UID      string // строковое представление публичного ключа

	// параметры конкретного экземпляра
	Port int

	peermap peermap
	fpeers  fpeermap
	Storage nodeStorage
}

// функция которая вызывается при подключении ноды к локальной сети
func ConnNode(prkpath string, port int) (*Node, error) {
	_, err := os.Stat(prkpath)

	var (
		pub  ed25519.PublicKey
		priv ed25519.PrivateKey
	)

	if err == nil {
		f, err := os.ReadFile(prkpath)
		if err != nil {
			log.Printf("[NODE] error configuring node")
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

	return &Node{
		PublicK:  pub,
		PrivateK: priv,
		UID:      hex.EncodeToString(pub),
		Port:     port,
		peermap:  *NewPM(),
	}, nil
}

func (n *Node) RmPeer(pid string) {
	n.fpeers.rmPeer(pid)
	n.peermap.Remove(pid)
}
