package internal

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
)

type NodeID struct {
	PublicK  ed25519.PublicKey
	PrivateK ed25519.PrivateKey

	UID string // хекс представление публичного ключа
}

// когда нода создается приватный ключ сохраняется в указанную директорию
func newNodeInit(saveto string) (*NodeID, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(saveto, priv, 0600)
	if err != nil {
		return nil, err
	}

	return &NodeID{
		PublicK:  pub,
		PrivateK: priv,
		UID:      hex.EncodeToString(pub),
	}, nil
}

func NodeConnect(privpath string) (*NodeID, error) {
	f, err := os.ReadFile(privpath)

	if err != nil {
		return newNodeInit(privpath)
	}

	priv := ed25519.PrivateKey(f)
	pub := priv.Public().(ed25519.PublicKey)

	return &NodeID{
		PublicK:  pub,
		PrivateK: priv,
		UID:      hex.EncodeToString(pub),
	}, nil
}
