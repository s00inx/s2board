// центральный узел сети
package network

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"os"
	"path/filepath"

	"github.com/s00inx/stdesk/internal/models"
)

/*
	POST /api/create (json) --> some logic --> ProcessFile
*/

// структура для именно этого устройства в сети
type Node struct {
	// публичная инфа
	Name string // короткое имя для удобства (потом)

	// ключи и crypto
	PublicK  ed25519.PublicKey
	PrivateK ed25519.PrivateKey
	UID      string // строковое представление публичного ключа

	Storage nodeStorage

	// параметры конкретного экземпляра
	iface *net.Interface
	ip    [4]byte
	port  int
	peers PeerMap
}

type nodeStorage interface {
	RegisterFile(src, title, desc string) (string, int64, error)
	SaveFile(man models.NoteManifest) error
}

func (n *Node) ProcessFile(src, title, desc string) []byte {
	fhash, fsize, err := n.Storage.RegisterFile(src, title, desc)
	if err != nil {
		return []byte{}
	}
	man := models.NewNote(
		title,
		filepath.Base(src),
		desc,
		fhash,
		fsize,
		models.FileType,
	)

	man.AuthorUID = n.UID
	man.Sign(n.PrivateK)
	man.Hash = hex.EncodeToString(man.CalculateID())

	n.Storage.SaveFile(*man)

	manjson, _ := json.MarshalIndent(man, " ", " ")
	return manjson
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
