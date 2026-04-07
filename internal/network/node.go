// центральный узел сети
package network

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/s00inx/s2board/internal/models"
)

// структура для именно этого устройства в сети
type Node struct {
	// публичная инфа
	Name string // короткое имя для удобства (потом)

	// ключи
	PublicK  ed25519.PublicKey
	PrivateK ed25519.PrivateKey
	UID      string // строковое представление публичного ключа

	// параметры конкретного экземпляра
	Port int

	peers   PeerMap
	Storage nodeStorage
}

// интерфейс для локального хранилища конкретной ноды
type nodeStorage interface {
	// saving files (node.go -> ProcessFile)
	RegisterFile(src string) (string, int64, error)
	SaveManifest(man models.NoteManifest) error

	// sync files (sync.go)
	GetHashes() ([]string, error)
	GetManifest(hash string) (*models.NoteManifest, error)
	HasNote(hash string) bool

	// downloading files (node.go -> DownloadBlob)
	FileExists(fhash string) bool // проверить, есть ли файл на диске
	SaveBlob(fhash string, r io.Reader) error
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
		peers:    *NewPM(),
	}, nil
}

// заполнить форму на фронтенде -> добавить файл в бд -> сохранить -> вернуть json байтами
func (n *Node) ProcessFile(src, title, desc string) []byte {
	fhash, fsize, err := n.Storage.RegisterFile(src)
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
	man.Hash = hex.EncodeToString(man.CalcID())

	n.Storage.SaveManifest(*man)

	manjson, _ := json.MarshalIndent(man, " ", " ")
	return manjson
}

// скачать файл у пира
func (n *Node) DlBlob(p models.Peer, fhash string) error {
	c := http.Client{Timeout: 30 * time.Second}
	resp, err := c.Get(fmt.Sprintf("http://%s:%d/api/dl/%s", p.IP, p.Port, fhash))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer returned status: %d", resp.StatusCode)
	}

	return n.Storage.SaveBlob(fhash, resp.Body)
}

// запустить цикл очистки старых пиров
func (n *Node) StartClean(ctx context.Context) {
	ti := time.NewTicker(1 * time.Minute)
	defer ti.Stop()

	for {
		select {
		case <-ti.C:
			count := n.peers.Cleanup(5 * time.Minute)
			if count > 0 {
				log.Printf("[PEERS] cleaned up %d peers", count)
			}
		case <-ctx.Done():
			log.Println("[INFO] cleanup worker stopped")
			return
		}
	}
}
