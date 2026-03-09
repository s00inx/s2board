package internal

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
)

// узел нашей сети это компьютер, подключенный к общему хостпоту, на котором запущен бинарник,
// нода имеет свой публичный и приватный ключи для идентификации (потому что сеть децентрализована)
// а также UID для удобного представления

// экземпляр узла (ноды)
type Node struct {
	AliasName string // короткое имя для удобства (потом)2

	PublicK  ed25519.PublicKey
	PrivateK ed25519.PrivateKey

	UID string // строковое представление публичного ключа
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
