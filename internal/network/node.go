// центральный узел сети
package network

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"
)

// структура для именно этого устройства в сети
type Node struct {
	// ключи
	PublicK  ed25519.PublicKey
	PrivateK ed25519.PrivateKey
	UID      string // строковое представление публичного ключа

	IP   string
	Port int

	// мапа пиров (активных участников сети)
	peermap peermap
	// мапа файлов, которых нету у ноды (хешфайла - список пиров)
	fpeers fpeermap

	// интерфейс для работы с памятью конкретной ноды
	Storage nodeStorage
	// общий клиент
	client http.Client
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
		peermap:  *newpeermap(),
		fpeers:   *newfilepeermap(),
	}, nil
}

// забыть о конкретном пире (когда тот вышел из сети)
func (n *Node) RmPeer(peeruid string) {
	n.fpeers.rmpeer(peeruid)
	n.peermap.rm(peeruid)
}
