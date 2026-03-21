package network

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"net"
	"os"
)

// экземпляр узла (ноды)
type Node struct {
	Name     string // короткое имя для удобства (потом)
	PublicK  ed25519.PublicKey
	PrivateK ed25519.PrivateKey
	UID      string // строковое представление публичного ключа

	iface *net.Interface
	ip    [4]byte
	port  int

	peers PeerMap
}

// когда нода создается приватный ключ сохраняется в указанную директорию
func newNodeInit(saveto string) (*Node, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(saveto, priv, 0600)
	if err != nil {
		return nil, err
	}

	return &Node{
		PublicK:  pub,
		PrivateK: priv,
		UID:      hex.EncodeToString(pub),
	}, nil
}

func NodeConnect(privpath string) (*Node, error) {
	f, err := os.ReadFile(privpath)

	if err != nil {
		return newNodeInit(privpath)
	}

	priv := ed25519.PrivateKey(f)
	pub := priv.Public().(ed25519.PublicKey)

	return &Node{
		PublicK:  pub,
		PrivateK: priv,
		UID:      hex.EncodeToString(pub),
	}, nil
}
